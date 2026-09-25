package ytdlp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		path     string
		expected media.Type
	}{
		{
			path:     "video.mp4",
			expected: media.Video,
		},
		{
			path:     "video.WEBM",
			expected: media.Video,
		},
		{
			path:     "foto.jpg",
			expected: media.Photo,
		},
		{
			path:     "foto.WEBP",
			expected: media.Photo,
		},
		{
			path:     "arquivo.txt",
			expected: media.Unknown,
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			result := detectMediaType(test.path)

			if result != test.expected {
				t.Fatalf(
					"detectMediaType(%q) = %q; esperado %q",
					test.path,
					result,
					test.expected,
				)
			}
		})
	}
}

func TestParseDownloadedFiles(t *testing.T) {
	dir := t.TempDir()

	videoPath := filepath.Join(dir, "video.mp4")
	photoPath := filepath.Join(dir, "photo.jpg")

	if err := os.WriteFile(
		videoPath,
		[]byte("video"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		photoPath,
		[]byte("photo"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	output := videoPath + "\n" +
		photoPath + "\n"

	items, err := parseDownloadedFiles(output)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf(
			"esperava 2 mídias; recebido %d",
			len(items),
		)
	}

	if items[0].Type != media.Video {
		t.Fatalf(
			"primeiro item deveria ser vídeo; recebido %q",
			items[0].Type,
		)
	}

	if items[1].Type != media.Photo {
		t.Fatalf(
			"segundo item deveria ser foto; recebido %q",
			items[1].Type,
		)
	}
}

func TestParseIgnoresDuplicateFiles(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "video.mp4")

	if err := os.WriteFile(
		path,
		[]byte("video"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	output := path + "\n" +
		path + "\n"

	items, err := parseDownloadedFiles(output)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf(
			"esperava 1 mídia; recebido %d",
			len(items),
		)
	}
}
