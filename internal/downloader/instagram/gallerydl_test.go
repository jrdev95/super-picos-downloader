package instagram

import (
	"context"
	"errors"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestInstagramMediaType(t *testing.T) {
	if got := instagramMediaType("photo.webp"); got != media.Photo {
		t.Fatalf("tipo = %q; esperado foto", got)
	}

	if got := instagramMediaType("video.MP4"); got != media.Video {
		t.Fatalf("tipo = %q; esperado vÃ­deo", got)
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

func TestAnonymousThenCookies(t *testing.T) {
	for _, tc := range []struct {
		name, failure string
		cookies       bool
		calls         int
	}{
		{"public", "", false, 1},
		{"login", "LoginRequired: Please log in", true, 2},
		{"login redirect", "HttpError: '429 Too Many Requests' for 'https://www.instagram.com/accounts/login/?next=/api/v1/media/4010368958171287586/info/'", true, 2},
		{"missing cookies", "LoginRequired", false, 1},
		{"rate limit", "HTTP 429 Too Many Requests", true, 1},
		{"network", "connection timed out", true, 1},
		{"no media", "", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cookies := filepath.Join(dir, "cookies.txt")
			if tc.cookies {
				if err := os.WriteFile(cookies, []byte("# Netscape HTTP Cookie File"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			strategy := NewGalleryDLStrategy("fake", cookies)
			calls := 0
			strategy.run = func(ctx context.Context, args []string) (string, string, error) {
				calls++
				joined := strings.Join(args, " ")
				if !strings.Contains(joined, "--config-ignore") || !strings.Contains(joined, ":memory:") {
					t.Fatal("global session not isolated")
				}
				for _, option := range []string{"--http-timeout 15", "--retries 0", "--sleep-429 0"} {
					if !strings.Contains(joined, option) {
						t.Fatalf("missing bounded request option: %s", option)
					}
				}
				hasCookies := strings.Contains(joined, "--cookies")
				if hasCookies != (calls == 2) {
					t.Fatalf("unexpected cookies on call %d", calls)
				}
				output := ""
				for i, a := range args {
					if a == "-D" {
						output = args[i+1]
					}
				}
				if calls == 1 && tc.failure != "" {
					_ = os.WriteFile(filepath.Join(output, "partial.jpg"), []byte("partial"), 0600)
					return "", tc.failure, errors.New("exit 1")
				}
				if tc.name == "no media" {
					return "", "", nil
				}
				path := filepath.Join(output, "photo.jpg")
				if err := os.WriteFile(path, []byte("photo"), 0600); err != nil {
					t.Fatal(err)
				}
				return path, "", nil
			}
			result, err := strategy.Download(context.Background(), platform.Instagram, "https://www.instagram.com/p/test/", dir)
			if calls != tc.calls {
				t.Fatalf("calls=%d want=%d", calls, tc.calls)
			}
			if tc.name == "public" || tc.name == "login" || tc.name == "login redirect" {
				if err != nil || len(result.Items) != 1 {
					t.Fatalf("result=%v err=%v", result, err)
				}
			} else if err == nil {
				t.Fatal("expected error")
			}
			if tc.name == "no media" && !errors.Is(err, downloader.ErrNoMedia) {
				t.Fatal(err)
			}
		})
	}
}
