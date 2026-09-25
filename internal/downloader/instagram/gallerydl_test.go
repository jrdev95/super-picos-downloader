package instagram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestInstagramMediaType(t *testing.T) {
	if got := instagramMediaType("photo.webp"); got != media.Photo {
		t.Fatalf("tipo = %q; esperado foto", got)
	}

	if got := instagramMediaType("video.MP4"); got != media.Video {
		t.Fatalf("tipo = %q; esperado vídeo", got)
	}
}

func TestItemsFromOutputPreservesOrderAndRemovesPathDuplicates(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "02.jpg")
	second := filepath.Join(dir, "01.jpg")

	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	output := first + "\n" + second + "\n" + first + "\n"
	items := itemsFromOutput(output, dir)

	if len(items) != 2 {
		t.Fatalf("esperava 2 itens; recebido %d", len(items))
	}

	if items[0].Path != first || items[1].Path != second {
		t.Fatalf("ordem inesperada: %#v", items)
	}
}
