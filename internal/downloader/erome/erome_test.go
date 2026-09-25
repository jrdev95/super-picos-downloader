package erome

import "testing"

func TestExtractURLs(t *testing.T) {
	page := []byte(`<source src="//cdn.example/video.mp4"><img class="img-front" data-src="https://cdn.example/photo.jpg?a=1&amp;b=2">`)

	videos := extractURLs(videoPattern, page)
	if len(videos) != 1 || videos[0] != "https://cdn.example/video.mp4" {
		t.Fatalf("vídeos inesperados: %#v", videos)
	}

	images := extractURLs(imagePattern, page)
	if len(images) != 1 || images[0] != "https://cdn.example/photo.jpg?a=1&b=2" {
		t.Fatalf("imagens inesperadas: %#v", images)
	}
}

func TestUniqueURLsPreservesOrder(t *testing.T) {
	got := uniqueURLs([]string{"a", "b", "a", "c", "b"})
	expected := []string{"a", "b", "c"}

	if len(got) != len(expected) {
		t.Fatalf("len = %d; esperado %d", len(got), len(expected))
	}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("posição %d = %q; esperado %q", i, got[i], expected[i])
		}
	}
}

func TestImageExtension(t *testing.T) {
	if got := imageExtension("https://cdn.example/photo.webp?x=1"); got != ".webp" {
		t.Fatalf("extensão = %q; esperado .webp", got)
	}

	if got := imageExtension("https://cdn.example/photo"); got != ".jpg" {
		t.Fatalf("fallback = %q; esperado .jpg", got)
	}
}
