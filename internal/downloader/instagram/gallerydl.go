package instagram

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

// GalleryDLStrategy usa gallery-dl para baixar publicações do Instagram.
//
// O arquivo de cookies é opcional. Quando informado, deve estar no formato
// Netscape/Mozilla aceito pelo gallery-dl. O caminho pode conter variáveis
// de ambiente, como $HOME.
type GalleryDLStrategy struct {
	path        string
	cookiesFile string
}

func NewGalleryDLStrategy(
	path string,
	cookiesFile string,
) *GalleryDLStrategy {
	return &GalleryDLStrategy{
		path:        path,
		cookiesFile: cookiesFile,
	}
}

func (s *GalleryDLStrategy) Name() string {
	return "gallery-dl-instagram"
}

func (s *GalleryDLStrategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	if detectedPlatform != platform.Instagram {
		return nil, downloader.ErrUnsupported
	}

	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("caminho do gallery-dl não foi definido")
	}

	if strings.TrimSpace(rawURL) == "" {
		return nil, downloader.ErrInvalidURL
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório de saída: %w",
			err,
		)
	}

	args := []string{
		"--no-input",
	}

	cookiesFile := strings.TrimSpace(
		os.ExpandEnv(s.cookiesFile),
	)

	if cookiesFile != "" {
		if _, err := os.Stat(cookiesFile); err != nil {
			return nil, fmt.Errorf(
				"arquivo de cookies do Instagram indisponível %q: %w",
				cookiesFile,
				err,
			)
		}

		args = append(
			args,
			"--cookies",
			cookiesFile,
		)
	}

	args = append(
		args,
		"-D",
		outputDir,
		"--",
		rawURL,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(
		ctx,
		s.path,
		args...,
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}

		if message == "" {
			return nil, fmt.Errorf(
				"gallery-dl falhou: %w",
				err,
			)
		}

		return nil, fmt.Errorf(
			"gallery-dl falhou: %w: %s",
			err,
			message,
		)
	}

	items := itemsFromOutput(
		stdout.String(),
		outputDir,
	)

	if len(items) == 0 {
		var err error

		items, err = scanDownloadedMedia(outputDir)
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return &media.Result{
		Platform: platform.Instagram,
		URL:      rawURL,
		Items:    items,
	}, nil
}

// itemsFromOutput preserva a ordem em que o gallery-dl informou os arquivos.
// Isso é útil para manter a ordem original de carrosséis do Instagram.
func itemsFromOutput(
	output string,
	outputDir string,
) []media.Item {
	scanner := bufio.NewScanner(
		strings.NewReader(output),
	)

	items := make([]media.Item, 0)
	seen := make(map[string]struct{})

	for scanner.Scan() {
		path := strings.TrimSpace(
			scanner.Text(),
		)

		if path == "" {
			continue
		}

		if !filepath.IsAbs(path) {
			path = filepath.Join(
				outputDir,
				path,
			)
		}

		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		mediaType := instagramMediaType(path)
		if mediaType == media.Unknown {
			continue
		}

		if _, exists := seen[path]; exists {
			continue
		}

		seen[path] = struct{}{}

		items = append(
			items,
			media.Item{
				Type:     mediaType,
				Path:     path,
				Filename: filepath.Base(path),
			},
		)
	}

	return items
}

func scanDownloadedMedia(
	outputDir string,
) ([]media.Item, error) {
	items := make([]media.Item, 0)

	err := filepath.WalkDir(
		outputDir,
		func(
			path string,
			entry os.DirEntry,
			err error,
		) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			mediaType := instagramMediaType(path)
			if mediaType == media.Unknown {
				return nil
			}

			items = append(
				items,
				media.Item{
					Type:     mediaType,
					Path:     path,
					Filename: filepath.Base(path),
				},
			)

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao localizar mídias baixadas: %w",
			err,
		)
	}

	sort.Slice(
		items,
		func(i, j int) bool {
			return items[i].Path < items[j].Path
		},
	)

	return items, nil
}

func instagramMediaType(
	path string,
) media.Type {
	switch strings.ToLower(
		filepath.Ext(path),
	) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return media.Photo

	case ".mp4", ".mkv", ".webm", ".mov", ".m4v":
		return media.Video

	default:
		return media.Unknown
	}
}
