package bot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type noMediaDownloader struct {
	err    error
	result *media.Result
}

func (d noMediaDownloader) CanHandle(platform.Platform) bool { return true }
func (d noMediaDownloader) Download(context.Context, string, string) (*media.Result, error) {
	return d.result, d.err
}

func TestProcessDownloadSilencesOnlyNoMedia(t *testing.T) {
	for _, site := range []platform.Platform{platform.Threads, platform.Instagram, platform.Twitter, platform.Reddit, platform.TikTok, platform.YouTube, platform.Erome} {
		for _, tc := range []struct {
			name    string
			err     error
			replies int
		}{
			{"text post", downloader.ErrNoMedia, 0},
			{"fallback empty", errors.Join(downloader.ErrUnsupported, downloader.ErrNoMedia), 0},
			{"blocked fallback", errors.Join(errors.New("HTTP 403 Blocked"), downloader.ErrNoMedia), 1},
			{"authentication", downloader.ErrAuthentication, 1},
			{"empty result", nil, 0},
		} {
			t.Run(string(site)+"/"+tc.name, func(t *testing.T) {
				var messages []string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if strings.HasSuffix(r.URL.Path, "getMe") {
						fmt.Fprint(w, `{"ok":true,"result":{"id":99,"is_bot":true,"first_name":"Bot"}}`)
						return
					}
					r.ParseForm()
					messages = append(messages, r.Form.Get("text"))
					fmt.Fprint(w, `{"ok":true,"result":{"message_id":10,"chat":{"id":1,"type":"private"}}}`)
				}))
				defer server.Close()
				api, err := tgbotapi.NewBotAPIWithClient("fake", server.URL+"/bot%s/%s", server.Client())
				if err != nil {
					t.Fatal(err)
				}
				b := &Bot{api: api, manager: downloader.NewManager(noMediaDownloader{err: tc.err, result: &media.Result{Platform: site, URL: "https://example.com/post"}})}
				b.processDownload(context.Background(), &tgbotapi.Message{MessageID: 1, Chat: &tgbotapi.Chat{ID: 1}}, "https://example.com/post", site)
				if len(messages) != tc.replies {
					t.Fatalf("got messages %v, want %d", messages, tc.replies)
				}
				for _, text := range messages {
					if strings.Contains(text, "Baixando") || strings.Contains(text, "Preparando") || strings.Contains(text, "Não encontrei") {
						t.Fatalf("unexpected message %q", text)
					}
				}
			})
		}
	}
}
