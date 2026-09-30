package bot

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

// Optional reduction is best-effort; the mandatory video size check still runs
// afterwards. Skip small files and oversized videos to avoid duplicate work.
func reduceMedia(ctx context.Context, result *media.Result, run videoRunner) (*media.Result, func(), error) {
	prepared := *result
	prepared.Items = append([]media.Item(nil), result.Items...)
	var dirs []string
	cleanup := func() {
		for _, dir := range dirs {
			_ = os.RemoveAll(dir)
		}
	}
	for i, item := range prepared.Items {
		if err := ctx.Err(); err != nil {
			return nil, cleanup, err
		}
		info, err := os.Stat(item.Path)
		if err != nil {
			return nil, cleanup, err
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, cleanup, fmt.Errorf("arquivo de mídia vazio ou inválido")
		}
		if item.Type == media.Video && (info.Size() < 5_000_000 || info.Size() > maxVideoBytes) {
			continue
		}
		if item.Type == media.Photo && info.Size() < 256_000 {
			continue
		}
		if item.Type != media.Video && item.Type != media.Photo {
			continue
		}
		dir, err := os.MkdirTemp(filepath.Dir(item.Path), "telegram-reduced-")
		if err != nil {
			return nil, cleanup, err
		}
		dirs = append(dirs, dir)
		started := time.Now()
		output := filepath.Join(dir, "video.mp4")
		args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", item.Path, "-map_metadata", "-1"}
		if item.Type == media.Video {
			args = append(args, "-map", "0:v:0", "-map", "0:a:0?", "-sn", "-dn", "-vf", reductionScale(1280), "-c:v", "libx264", "-preset", "veryfast", "-crf", "26", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", output)
		} else {
			output = filepath.Join(dir, "photo.jpg")
			// Resize via ffmpeg, then use Go's explicit JPEG quality scale (0..100).
			args = append(args, "-map", "0:v:0", "-frames:v", "1", "-vf", reductionScale(1920), "-c:v", "png", filepath.Join(dir, "resized.png"))
		}
		err = run(ctx, args...)
		if err == nil && item.Type == media.Photo {
			err = encodeReducedJPEG(filepath.Join(dir, "resized.png"), output)
		}
		if ctx.Err() != nil {
			return nil, cleanup, ctx.Err()
		}
		if err != nil {
			slog.Warn("redução opcional falhou; mantendo original", "media_type", item.Type, "error", err)
			continue
		}
		reduced, err := os.Stat(output)
		if err != nil || !reduced.Mode().IsRegular() || reduced.Size() == 0 {
			continue
		}
		// Require at least 10% savings to justify sending a lossy copy.
		if reduced.Size() > info.Size()*9/10 {
			continue
		}
		prepared.Items[i].Path = output
		prepared.Items[i].Filename = filepath.Base(output)
		slog.Info("mídia reduzida para envio", "media_type", item.Type, "original_bytes", info.Size(), "reduced_bytes", reduced.Size(), "elapsed", time.Since(started))
	}
	return &prepared, cleanup, nil
}

func reductionScale(limit int) string {
	return fmt.Sprintf("scale=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2", limit, limit)
}

func encodeReducedJPEG(input, output string) error {
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	img, _, decodeErr := image.Decode(f)
	closeErr := f.Close()
	if decodeErr != nil {
		return decodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	// JPEG has no alpha channel. Composite transparent pixels onto white.
	canvas := image.NewRGBA(img.Bounds())
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), img, img.Bounds().Min, draw.Over)
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	err = jpeg.Encode(out, canvas, &jpeg.Options{Quality: 85})
	closeErr = out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
