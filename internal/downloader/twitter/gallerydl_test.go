package twitter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestDetectType(t *testing.T) {
	if got := detectType("photo.JPG"); got != media.Photo {
		t.Fatalf("tipo = %q; esperado foto", got)
	}

	if got := detectType("video.mp4"); got != media.Video {
		t.Fatalf("tipo = %q; esperado vídeo", got)
	}
}

func TestDeduplicateByContent(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.jpg")
	second := filepath.Join(dir, "b.jpg")
	third := filepath.Join(dir, "c.jpg")

	if err := os.WriteFile(first, []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(third, []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}

	items, err := deduplicate([]media.Item{
		{Type: media.Photo, Path: first},
		{Type: media.Photo, Path: second},
		{Type: media.Photo, Path: third},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatalf("esperava 2 itens únicos; recebido %d", len(items))
	}
}
