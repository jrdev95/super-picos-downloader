package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Strategy struct {
	ytdlpPath  string
	ffmpegPath string
}

func New(ytdlpPath, ffmpegPath string) *Strategy {
	return &Strategy{
		ytdlpPath:  ytdlpPath,
		ffmpegPath: ffmpegPath,
	}
}

func (s *Strategy) Name() string {
	return "yt-dlp"
}

func (s *Strategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	url string,
	outputDir string,
) (*media.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(url) == "" {
		return nil, downloader.ErrInvalidURL
	}

	if outputDir == "" {
		return nil, fmt.Errorf("diretório de saída não definido")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"falha ao preparar diretório de saída: %w",
			err,
		)
	}

	outputTemplate := filepath.Join(
		outputDir,
		"%(id)s_%(autonumber)03d.%(ext)s",
	)

	args := []string{
		"--no-progress",
		"--no-simulate",
		"--ffmpeg-location",
		s.ffmpegPath,
		"--recode-video",
		"mp4",
		"--output",
		outputTemplate,
		"--print",
		"after_move:%(filepath)s",
		"--",
		url,
	}

	cmd := exec.CommandContext(
		ctx,
		s.ytdlpPath,
		args...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	if err != nil {
		return nil, classifyError(
			err,
			stderr.String(),
		)
	}

	items, err := parseDownloadedFiles(stdout.String())
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return &media.Result{
		Platform: detectedPlatform,
		URL:      url,
		Items:    items,
	}, nil
}
