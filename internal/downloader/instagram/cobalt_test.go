package instagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCobaltDownloadsAndCleansFailures(t *testing.T) {
	for _, mode := range []string{"picker", "redirect", "tunnel", "partial", "error", "html"} {
		t.Run(mode, func(t *testing.T) {
			var base string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Cookie") != "" {
					t.Error("unexpected cookies")
				}
				if r.URL.Path == "/" {
					var body map[string]string
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["url"] != "https://www.instagram.com/p/test/" {
						t.Error("bad source")
					}
					switch mode {
					case "picker", "partial":
						second := "/photo"
						if mode == "partial" {
							second = "/bad"
						}
						fmt.Fprintf(w, `{"status":"picker","picker":[{"type":"photo","url":%q},{"type":"photo","url":%q}]}`, base+"/photo", base+second)
					case "error":
						fmt.Fprint(w, `{"status":"error","error":{"code":"fetch.failed"}}`)
					case "html":
						fmt.Fprintf(w, `{"status":"redirect","url":%q}`, base+"/bad")
					default:
						fmt.Fprintf(w, `{"status":%q,"url":%q}`, mode, base+"/photo")
					}
					return
				}
				if r.URL.Path == "/bad" {
					fmt.Fprint(w, "<html>blocked</html>")
					return
				}
				w.Write([]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0})
			}))
			defer server.Close()
			base = server.URL
			dir := t.TempDir()
			s := NewCobaltStrategy(base)
			result, err := s.Download(context.Background(), platform.Instagram, "https://www.instagram.com/p/test/", dir)
			if mode == "partial" || mode == "error" || mode == "html" {
				if err == nil || !downloader.ShouldFallback(err) {
					t.Fatalf("expected fallback: %v", err)
				}
				files, _ := os.ReadDir(dir)
				if len(files) != 0 {
					t.Fatal("partial files retained")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := 1
			if mode == "picker" {
				want = 2
			}
			if len(result.Items) != want {
				t.Fatal(result)
			}
			for i, item := range result.Items {
				if item.Filename != fmt.Sprintf("%03d.jpg", i+1) {
					t.Fatal("order")
				}
				if _, err := os.Stat(item.Path); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestCobaltTimeoutAllowsFallbackAndParentCancellationStops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(60 * time.Millisecond) }))
	defer server.Close()
	s := NewCobaltStrategy(server.URL)
	s.client.Timeout = 20 * time.Millisecond
	_, err := s.Download(context.Background(), platform.Instagram, "https://www.instagram.com/p/test/", t.TempDir())
	if err == nil || !downloader.ShouldFallback(err) {
		t.Fatalf("timeout blocked fallback: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Download(ctx, platform.Instagram, "https://www.instagram.com/p/test/", t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
