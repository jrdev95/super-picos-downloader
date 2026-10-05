package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestRunDiscardsLateOldMessagesAndKeepsNewMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var methods []string
	var offsets []string
	sent := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		r.ParseForm()
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		methods = append(methods, method)
		switch method {
		case "getMe":
			fmt.Fprint(w, `{"ok":true,"result":{"id":99,"is_bot":true,"first_name":"Bot"}}`)
		case "deleteWebhook":
			if r.Form.Get("drop_pending_updates") != "true" {
				t.Error("pending queue not explicitly cleared")
			}
			fmt.Fprint(w, `{"ok":true,"result":true}`)
		case "getUpdates":
			offsets = append(offsets, r.Form.Get("offset"))
			if r.Form.Get("allowed_updates") != `["message","callback_query"]` {
				t.Error("missing update filter")
			}
			if len(offsets) == 1 {
				fmt.Fprint(w, `{"ok":true,"result":[{"update_id":100}]}`)
				return
			}
			if len(offsets) == 3 {
				cancel()
				fmt.Fprint(w, `{"ok":true,"result":[]}`)
				return
			}
			now := int(time.Now().Unix())
			message := func(text string, date int) *tgbotapi.Message {
				m := &tgbotapi.Message{MessageID: 1, Date: date, Chat: &tgbotapi.Chat{ID: 1, Type: "private"}, Text: text}
				if text == "/start" {
					m.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 6}}
				}
				return m
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": []tgbotapi.Update{
				{UpdateID: 101, Message: message("/start", now-3*3600)},
				{UpdateID: 102, Message: message("https://www.instagram.com/reel/example/", now-3*3600)},
				{UpdateID: 103, Message: message("/start", now)},
				{UpdateID: 104, Message: message("https://www.instagram.com/reel/example/", now)},
			}})
		case "sendMessage":
			sent++
			fmt.Fprint(w, `{"ok":true,"result":{"message_id":5,"chat":{"id":1,"type":"private"}}}`)
		default:
			t.Errorf("unexpected method %s", method)
		}
	}))
	defer server.Close()
	api, err := tgbotapi.NewBotAPIWithClient("fake", server.URL+"/bot%s/%s", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	b := &Bot{api: api, jobs: make(chan downloadJob, 10)}
	if err := b.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if sent != 1 || len(b.jobs) != 1 {
		t.Fatalf("sent=%d queued=%d; only new messages should execute", sent, len(b.jobs))
	}
	if strings.Join(offsets, ",") != "-1,101,105" {
		t.Fatalf("offsets=%v; ignored messages must also be acknowledged", offsets)
	}
	if methods[1] != "deleteWebhook" {
		t.Fatalf("queue cleanup must precede polling: %v", methods)
	}
}

func TestStartupCleanupFailureStopsBot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "getMe") {
			fmt.Fprint(w, `{"ok":true,"result":{"id":99,"is_bot":true,"first_name":"Bot"}}`)
		} else {
			fmt.Fprint(w, `{"ok":false,"error_code":500,"description":"cleanup failed"}`)
		}
	}))
	defer server.Close()
	api, err := tgbotapi.NewBotAPIWithClient("fake", server.URL+"/bot%s/%s", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	b := &Bot{api: api, workers: 1}
	if err := b.Run(context.Background()); err == nil {
		t.Fatal("cleanup failure should stop startup")
	}
}
