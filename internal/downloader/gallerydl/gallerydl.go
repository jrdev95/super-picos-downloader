package gallerydl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/executil"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Strategy struct {
	path           string
	browserProfile string
}

func New(path, browserProfile string) *Strategy {
	return &Strategy{
		path:           path,
		browserProfile: browserProfile,
	}
}

func (s *Strategy) Name() string {
	return "gallery-dl"
}

func (s *Strategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	if detectedPlatform != platform.TikTok {
		return nil, downloader.ErrUnsupported
	}

	if !strings.Contains(strings.ToLower(rawURL), "/photo/") {
		return nil, downloader.ErrUnsupported
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório de saída: %w",
			err,
		)
	}

	args := []string{
		"--no-input",

		"-o",
		"browser=" + s.browserProfile,

		"--filter",
		"extension in ('jpg', 'jpeg', 'png', 'webp')",

		"-D",
		outputDir,

		"--",
		rawURL,
	}

	cmd := executil.CommandContext(
		ctx,
		s.path,
		args...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		return nil, fmt.Errorf(
			"gallery-dl falhou: %w: %s",
			err,
			strings.TrimSpace(stderr.String()),
		)
	}

	items, err := scanPhotos(outputDir)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return &media.Result{
		Platform: detectedPlatform,
		URL:      rawURL,
		Items:    items,
	}, nil
}

func scanPhotos(outputDir string) ([]media.Item, error) {
	items := make([]media.Item, 0)

	err := filepath.WalkDir(
		outputDir,
		func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			if !isPhoto(path) {
				return nil
			}

			items = append(
				items,
				media.Item{
					Type:     media.Photo,
					Path:     path,
					Filename: filepath.Base(path),
				},
			)

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao localizar fotos: %w",
			err,
		)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	return items, nil
}

func isPhoto(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true

	default:
		return false
	}
}

func DefaultBrowserProfile() string {
	switch runtime.GOOS {
	case "linux":
		return "chrome:linux"

	case "windows":
		return "chrome:windows"

	case "darwin":
		return "chrome:macos"

	default:
		// Android/Termux e outros ambientes.
		return "chrome"
	}
}
