package twitter

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/executil"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type GalleryDLStrategy struct {
	path string
}

func NewGalleryDLStrategy(
	path string,
) *GalleryDLStrategy {
	return &GalleryDLStrategy{
		path: path,
	}
}

func (s *GalleryDLStrategy) Name() string {
	return "gallery-dl-twitter"
}

func (s *GalleryDLStrategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	if detectedPlatform != platform.Twitter {
		return nil, downloader.ErrUnsupported
	}

	if err := os.MkdirAll(
		outputDir,
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório de saída: %w",
			err,
		)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := executil.CommandContext(
		ctx,
		s.path,
		"--no-input",
		"-D",
		outputDir,
		"--",
		rawURL,
	)

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

	items := itemsFromOutput(
		stdout.String(),
		outputDir,
	)

	if len(items) == 0 {
		var err error

		items, err = scanMedia(outputDir)
		if err != nil {
			return nil, err
		}
	}

	items, err := deduplicate(items)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return &media.Result{
		Platform: platform.Twitter,
		URL:      rawURL,
		Items:    items,
	}, nil
}

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

		mediaType := detectType(path)
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

func scanMedia(
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

			mediaType := detectType(path)
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
		return nil, err
	}

	sort.Slice(
		items,
		func(i, j int) bool {
			return items[i].Path < items[j].Path
		},
	)

	return items, nil
}

func deduplicate(
	items []media.Item,
) ([]media.Item, error) {
	result := make(
		[]media.Item,
		0,
		len(items),
	)

	hashes := make(
		map[[sha256.Size]byte]struct{},
	)

	for _, item := range items {
		file, err := os.Open(item.Path)
		if err != nil {
			return nil, err
		}

		hash := sha256.New()

		_, copyErr := io.Copy(
			hash,
			file,
		)

		closeErr := file.Close()

		if copyErr != nil {
			return nil, copyErr
		}

		if closeErr != nil {
			return nil, closeErr
		}

		var digest [sha256.Size]byte
		copy(
			digest[:],
			hash.Sum(nil),
		)

		if _, exists := hashes[digest]; exists {
			continue
		}

		hashes[digest] = struct{}{}

		result = append(
			result,
			item,
		)
	}

	return result, nil
}

func detectType(
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
