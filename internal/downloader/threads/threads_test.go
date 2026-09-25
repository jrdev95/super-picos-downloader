package threads

import (
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestExtractPostCode(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.threads.com/@user/post/ABC123/", "ABC123"},
		{"https://www.threads.net/@user/post/XYZ987?x=1", "XYZ987"},
		{"https://www.threads.com/share/AAAA/", ""},
	}

	for _, test := range tests {
		if got := extractPostCode(test.url); got != test.expected {
			t.Fatalf("extractPostCode(%q) = %q; esperado %q", test.url, got, test.expected)
		}
	}
}

func TestFindMediaByCodeIgnoresOtherPosts(t *testing.T) {
	wanted := map[string]any{
		"code": "POST123",
		"image_versions2": map[string]any{
			"candidates": []any{},
		},
	}

	payload := map[string]any{
		"reply": map[string]any{
			"code":            "REPLY999",
			"image_versions2": map[string]any{},
		},
		"root": wanted,
	}

	got := findMediaByCode(payload, "POST123")
	if got == nil {
		t.Fatal("mídia principal deveria ter sido encontrada")
	}

	if got["code"] != "POST123" {
		t.Fatalf("code = %v; esperado POST123", got["code"])
	}
}

func TestExtractSingleEntryPrefersVideoOverCover(t *testing.T) {
	value := map[string]any{
		"video_versions": []any{
			map[string]any{"url": "https://cdn.example/video.mp4"},
		},
		"image_versions2": map[string]any{
			"candidates": []any{
				map[string]any{"url": "https://cdn.example/cover.jpg", "width": float64(1080), "height": float64(1920)},
			},
		},
	}

	entry, ok := extractSingleEntry(value)
	if !ok {
		t.Fatal("entrada deveria ser reconhecida")
	}

	if entry.mediaType != media.Video {
		t.Fatalf("tipo = %q; esperado vídeo", entry.mediaType)
	}

	if entry.url != "https://cdn.example/video.mp4" {
		t.Fatalf("URL = %q", entry.url)
	}
}

func TestExtractSingleEntrySelectsLargestImage(t *testing.T) {
	value := map[string]any{
		"image_versions2": map[string]any{
			"candidates": []any{
				map[string]any{"url": "small.jpg", "width": float64(320), "height": float64(320)},
				map[string]any{"url": "large.jpg", "width": float64(1080), "height": float64(1350)},
			},
		},
	}

	entry, ok := extractSingleEntry(value)
	if !ok {
		t.Fatal("entrada deveria ser reconhecida")
	}

	if entry.mediaType != media.Photo || entry.url != "large.jpg" {
		t.Fatalf("entrada inesperada: %#v", entry)
	}
}
