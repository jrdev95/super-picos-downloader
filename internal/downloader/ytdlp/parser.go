package ytdlp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func parseDownloadedFiles(output string) ([]media.Item, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	seen := make(map[string]struct{})
	items := make([]media.Item, 0)

	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())

		if path == "" {
			continue
		}

		if _, exists := seen[path]; exists {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		mediaType := detectMediaType(path)

		if mediaType == media.Unknown {
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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf(
			"falha ao processar saída do yt-dlp: %w",
			err,
		)
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return items, nil
}

func detectMediaType(path string) media.Type {
	extension := strings.ToLower(
		filepath.Ext(path),
	)

	switch extension {
	case ".jpg",
		".jpeg",
		".png",
		".webp":
		return media.Photo

	case ".mp4",
		".mkv",
		".webm",
		".mov",
		".m4v",
		".avi":
		return media.Video

	default:
		return media.Unknown
	}
}
