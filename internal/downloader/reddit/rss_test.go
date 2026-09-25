package reddit

import (
	"errors"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
)

func TestMakeRSSURL(t *testing.T) {
	got, err := makeRSSURL("https://www.reddit.com/r/test/comments/abc123/title/?share_id=1#x")
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	expected := "https://www.reddit.com/r/test/comments/abc123/title/.rss"
	if got != expected {
		t.Fatalf("URL = %q; esperado %q", got, expected)
	}
}

func TestMakeRSSURLRejectsNonPost(t *testing.T) {
	_, err := makeRSSURL("https://www.reddit.com/r/test/")
	if !errors.Is(err, downloader.ErrInvalidURL) {
		t.Fatalf("erro = %v; esperado ErrInvalidURL", err)
	}
}

func TestFirstImageURLPrefersOriginal(t *testing.T) {
	entry := `https://preview.redd.it/preview.jpg?width=640 https://i.redd.it/original.jpg`
	got := firstImageURL(entry)
	if got != "https://i.redd.it/original.jpg" {
		t.Fatalf("URL = %q", got)
	}
}

func TestGalleryPattern(t *testing.T) {
	if !galleryPattern.MatchString(`href="https://www.reddit.com/gallery/abc123"`) {
		t.Fatal("link de galeria deveria ser detectado")
	}
}
