package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/executil"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Strategy struct {
	ytdlpPath          string
	ffmpegPath         string
	lowMemory          bool
	twitterCookiesFile string
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

func (s *Strategy) SetLowMemory(enabled bool) { s.lowMemory = enabled }

// SetTwitterCookiesFile configures Netscape cookies for X downloads only.
func (s *Strategy) SetTwitterCookiesFile(path string) { s.twitterCookiesFile = strings.TrimSpace(path) }

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
		return nil, fmt.Errorf("diretÃ³rio de saÃ­da nÃ£o definido")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"falha ao preparar diretÃ³rio de saÃ­da: %w",
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

	if detectedPlatform == platform.Instagram {
		args = append([]string{"--ignore-config"}, args...)
	}
	if detectedPlatform == platform.Twitter && s.twitterCookiesFile != "" {
		cookiesFile := strings.TrimSpace(os.ExpandEnv(s.twitterCookiesFile))
		if _, err := os.Stat(cookiesFile); err != nil {
			return nil, fmt.Errorf("arquivo de cookies do X indisponível %q: %w", cookiesFile, err)
		}
		args = append([]string{"--cookies", cookiesFile}, args...)
	}
	if s.lowMemory {
		args = append([]string{"--postprocessor-args", "ffmpeg_i:-threads 1", "--postprocessor-args", "ffmpeg_o:-threads 1 -filter_threads 1 -filter_complex_threads 1"}, args...)
	}
	cmd := executil.CommandContext(
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
