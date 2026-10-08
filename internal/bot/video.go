package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jrdev95/super-picos-downloader/internal/executil"
	"github.com/jrdev95/super-picos-downloader/internal/media"
)

// Use decimal MB conservatively; leave room below the official upload limit.
const maxVideoBytes int64 = 50_000_000
const targetVideoBytes int64 = 48_000_000
const compressionTimeout = 10 * time.Minute

type videoSizeError struct {
	size  int64
	cause error
}

func (e *videoSizeError) Error() string {
	return fmt.Sprintf("vídeo com %.1f MB não pôde ser preparado para envio (limite 50 MB, alvo 48 MB): %v", float64(e.size)/1_000_000, e.cause)
}
func (e *videoSizeError) Unwrap() error { return e.cause }

type videoRunner func(context.Context, ...string) error

func lowMemoryFFmpegArgs(args []string) []string {
	// Input decoder options must precede -i; encoder options precede output.
	limited := []string{"-threads", "1", "-filter_threads", "1", "-filter_complex_threads", "1"}
	if len(args) == 0 {
		return limited
	}
	limited = append(limited, args[:len(args)-1]...)
	return append(limited, "-threads", "1", args[len(args)-1])
}

func runFFmpegLowMemory(ctx context.Context, args ...string) error {
	return runFFmpeg(ctx, lowMemoryFFmpegArgs(args)...)
}

func runFFmpeg(ctx context.Context, args ...string) error {
	cmd := executil.CommandContext(ctx, "ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

// Prepare the entire result before uploading any part of an album. Originals
// remain intact and the caller owns cleanup, including on preparation errors.
func prepareVideos(ctx context.Context, result *media.Result, run videoRunner) (*media.Result, func(), error) {
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
		if item.Type != media.Video {
			continue
		}
		info, err := os.Stat(item.Path)
		if err != nil {
			return nil, cleanup, fmt.Errorf("verificar vídeo: %w", err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, cleanup, errors.New("arquivo de vídeo inválido ou vazio")
		}
		if info.Size() <= maxVideoBytes {
			continue
		}
		dir, err := os.MkdirTemp(filepath.Dir(item.Path), "telegram-video-")
		if err != nil {
			return nil, cleanup, err
		}
		dirs = append(dirs, dir)
		output := filepath.Join(dir, "video.mp4")
		size := info.Size()
		var compressErr error
		crf := 28
		// Bounded quality/resolution reductions; never truncate with -fs or -t.
		for attempt, width := range []int{1280, 960, 720} {
			if err := ctx.Err(); err != nil {
				return nil, cleanup, err
			}
			args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", item.Path,
				"-map", "0:v:0", "-map", "0:a:0?", "-map_metadata", "-1", "-sn", "-dn",
				"-vf", fmt.Sprintf("scale=w='min(%d,iw)':h='min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2", width, width),
				"-c:v", "libx264", "-preset", "veryfast", "-crf", strconv.Itoa(crf),
				"-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "96k", "-movflags", "+faststart", output}
			started := time.Now()
			compressErr = run(ctx, args...)
			slog.Info("tentativa de compressão concluída", "attempt", attempt+1, "elapsed", time.Since(started), "crf", crf, "max_dimension", width, "error", compressErr)
			if compressErr != nil {
				break
			}
			compressed, err := os.Stat(output)
			if err != nil {
				compressErr = err
				break
			}
			if !compressed.Mode().IsRegular() || compressed.Size() == 0 {
				compressErr = errors.New("ffmpeg produziu arquivo vazio ou inválido")
				break
			}
			size = compressed.Size()
			slog.Info("tamanho após compressão", "original_bytes", info.Size(), "compressed_bytes", size)
			if size <= targetVideoBytes {
				prepared.Items[i].Path = output
				prepared.Items[i].Filename = "video.mp4"
				break
			}
			crf = nextVideoCRF(crf, size)
		}
		if err := ctx.Err(); err != nil {
			return nil, cleanup, err
		}
		if prepared.Items[i].Path == item.Path {
			if compressErr != nil {
				size = info.Size()
			}
			return nil, cleanup, &videoSizeError{size: size, cause: compressErr}
		}
	}
	return &prepared, cleanup, nil
}

func sendErrorMessage(err error) string {
	var sizeErr *videoSizeError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "⏱️ A preparação do vídeo demorou mais do que o esperado e foi cancelada."
	case errors.Is(err, context.Canceled):
		return "⚠️ O envio da mídia foi cancelado."
	case errors.As(err, &sizeErr):
		if sizeErr.cause != nil {
			return fmt.Sprintf("❌ O vídeo possui %.1f MB e excede o limite de 50 MB do Telegram. Não foi possível recomprimi-lo.", float64(sizeErr.size)/1_000_000)
		}
		return fmt.Sprintf("❌ Mesmo após a compressão, o vídeo ficou com %.1f MB. Não foi possível atingir a margem segura de 48 MB para o limite de 50 MB do Telegram.", float64(sizeErr.size)/1_000_000)
	default:
		return "❌ A mídia foi baixada, mas não foi possível concluir o envio pelo Telegram. Tente novamente em instantes."
	}
}

// Size feedback is a heuristic, not a size guarantee. Always stat the output.
// Larger misses warrant a stronger adjustment instead of another similar encode.
func nextVideoCRF(current int, size int64) int {
	step := int(math.Ceil(6 * math.Log2(float64(size)/float64(targetVideoBytes))))
	if step < 4 {
		step = 4
	}
	if step > 12 {
		step = 12
	}
	return min(current+step, 40)
}
