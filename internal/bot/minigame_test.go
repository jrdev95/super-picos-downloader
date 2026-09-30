package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jrdev95/super-picos-downloader/internal/minigame"
)

func TestMinigameAdminAndCallback(t *testing.T) {
	game, err := minigame.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	role := "member"
	var sent []string
	var edited, answered bool
	var callbackData string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		r.ParseForm()
		switch {
		case strings.HasSuffix(r.URL.Path, "getMe"):
			fmt.Fprint(w, `{"ok":true,"result":{"id":99,"is_bot":true,"first_name":"Bot","username":"testbot"}}`)
		case strings.HasSuffix(r.URL.Path, "getChatMember"):
			fmt.Fprintf(w, `{"ok":true,"result":{"status":%q,"user":{"id":1,"first_name":"A"}}}`, role)
		case strings.HasSuffix(r.URL.Path, "answerCallbackQuery"):
			answered = true
			fmt.Fprint(w, `{"ok":true,"result":true}`)
		default:
			if strings.HasSuffix(r.URL.Path, "editMessageText") {
				edited = true
			}
			sent = append(sent, r.Form.Get("text"))
			if markup := r.Form.Get("reply_markup"); markup != "" {
				var v tgbotapi.InlineKeyboardMarkup
				json.Unmarshal([]byte(markup), &v)
				if len(v.InlineKeyboard) > 0 {
					callbackData = *v.InlineKeyboard[0][0].CallbackData
				}
			}
			fmt.Fprint(w, `{"ok":true,"result":{"message_id":10,"chat":{"id":-1,"type":"supergroup"}}}`)
		}
	}))
	defer server.Close()
	api, err := tgbotapi.NewBotAPIWithClient("fake", server.URL+"/bot%s/%s", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	b := &Bot{api: api, game: game}
	message := func(text string) *tgbotapi.Message {
		n := strings.IndexByte(text, ' ')
		if n < 0 {
			n = len(text)
		}
		return &tgbotapi.Message{MessageID: 1, Chat: &tgbotapi.Chat{ID: -1, Type: "supergroup"}, From: &tgbotapi.User{ID: 1, FirstName: "A"}, Text: text, Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: n}}}
	}
	b.handleGameCommand(message("/grow@testbot"))
	b.handleGameCommand(message("/limpar confirmar"))
	if !strings.Contains(sent[len(sent)-1], "Somente administradores") {
		t.Fatal(sent)
	}
	role = "administrator"
	b.handleGameCommand(message("/limpar"))
	if !strings.Contains(sent[len(sent)-1], "/limpar confirmar") {
		t.Fatal(sent)
	}
	b.handleGameCommand(message("/limpar confirmar"))
	r, err := game.Command(-1, 1, "A", "status", "")
	if err != nil || !strings.Contains(r.Text, "Tamanho: 0 cm") {
		t.Fatal(r, err)
	}
	// Loan may draw zero, so retry until both users have a balance.
	for _, id := range []int64{1, 2} {
		if _, err := game.Command(-1, id, "Pessoa", "grow", ""); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 100; i++ {
			r, e := game.Command(-1, id, "Pessoa", "emprestimo", "")
			if e != nil {
				break
			}
			if !strings.Contains(r.Text, "Saiu 0") {
				break
			}
		}
	}
	b.handleGameCommand(message("/duelo 1"))
	if callbackData == "" {
		t.Fatal("missing attack button")
	}
	b.handleGameCallback(&tgbotapi.CallbackQuery{ID: "cb", From: &tgbotapi.User{ID: 2, FirstName: "B"}, Message: message(""), Data: callbackData})
	if !answered || !edited {
		t.Fatalf("answered=%v edited=%v sent=%v", answered, edited, sent)
	}
	before := len(sent)
	b.handleGameCommand(message("/grow@anotherbot"))
	if len(sent) != before {
		t.Fatal("handled another bot's command")
	}
}
