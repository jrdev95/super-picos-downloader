package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jrdev95/super-picos-downloader/internal/minigame"
)

func WithMinigame(game *minigame.Store) Option { return func(b *Bot) { b.game = game } }
func visibleName(user *tgbotapi.User) string {
	return strings.TrimSpace(user.FirstName + " " + user.LastName)
}
func groupChat(chat *tgbotapi.Chat) bool {
	return chat != nil && (chat.IsGroup() || chat.IsSuperGroup())
}
func gameError(err error) string {
	var rule minigame.RuleError
	if errors.As(err, &rule) {
		return rule.Error()
	}
	slog.Error("falha no minigame", "error", err)
	return "Não foi possível concluir a jogada. Tente novamente."
}
func (b *Bot) handleGameCommand(m *tgbotapi.Message) bool {
	command := m.Command()
	switch command {
	case "grow", "rank", "duelo", "emprestimo", "status", "limpar", "doar":
	default:
		return false
	}
	// Commands addressed to another bot must never mutate this game's state.
	target := strings.SplitN(strings.Fields(m.Text)[0], "@", 2)
	if len(target) == 2 && !strings.EqualFold(target[1], b.api.Self.UserName) {
		return true
	}
	if !groupChat(m.Chat) {
		b.reply(m, "O minigame só funciona em grupos.")
		return true
	}
	if m.From == nil || m.From.IsBot || m.SenderChat != nil {
		b.reply(m, "Use sua conta pessoal para jogar.")
		return true
	}
	if b.game == nil {
		b.reply(m, "O minigame está indisponível.")
		return true
	}
	args := strings.TrimSpace(m.CommandArguments())
	if command == "doar" {
		b.handleDonation(m, args)
		return true
	}
	if command == "limpar" {
		member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: m.Chat.ID, UserID: m.From.ID}})
		if err != nil {
			slog.Error("falha ao verificar administrador", "error", err)
			b.reply(m, "Não foi possível verificar sua permissão de administrador.")
			return true
		}
		if !member.IsCreator() && !member.IsAdministrator() {
			b.reply(m, "Somente administradores do grupo podem limpar o minigame.")
			return true
		}
		if args != "confirmar" {
			b.reply(m, "Use /limpar confirmar para zerar todos os dados do minigame deste grupo.")
			return true
		}
		if err = b.game.Reset(m.Chat.ID); err != nil {
			b.reply(m, gameError(err))
		} else {
			b.reply(m, "Minigame zerado neste grupo. Todo mundo voltou aos 0 cm!")
		}
		return true
	}
	result, err := b.game.Command(m.Chat.ID, m.From.ID, visibleName(m.From), command, args)
	if err != nil {
		b.reply(m, gameError(err))
		return true
	}
	if result.DuelID != 0 {
		reply := tgbotapi.NewMessage(m.Chat.ID, result.Text)
		reply.ReplyToMessageID = m.MessageID
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Ataque!", fmt.Sprintf("minigame:%d", result.DuelID))))
		if _, err = b.api.Send(reply); err != nil {
			slog.Error("falha ao enviar convite do duelo", "error", err)
		}
	} else {
		b.replyGame(m, result.Text)
	}
	return true
}

// Split on line boundaries so large group rankings fit Telegram's UTF-16 limit.
func (b *Bot) replyGame(m *tgbotapi.Message, text string) {
	chunk := ""
	units := 0
	for _, line := range strings.Split(text, "\n") {
		n := 1
		for _, r := range line {
			n++
			if r > 0xffff {
				n++
			}
		}
		if units+n > 3900 {
			b.reply(m, strings.TrimSuffix(chunk, "\n"))
			chunk = ""
			units = 0
		}
		chunk += line + "\n"
		units += n
	}
	if chunk != "" {
		b.reply(m, strings.TrimSuffix(chunk, "\n"))
	}
}
func (b *Bot) handleGameCallback(q *tgbotapi.CallbackQuery) {
	if !strings.HasPrefix(q.Data, "minigame:") {
		return
	}
	notice := ""
	text := ""
	if b.game == nil || q.Message == nil || !groupChat(q.Message.Chat) || q.From == nil || q.From.IsBot {
		notice = "Esse duelo não está disponível."
	} else {
		id, err := strconv.ParseInt(strings.TrimPrefix(q.Data, "minigame:"), 10, 64)
		if err != nil || id < 1 {
			notice = "Duelo inválido."
		} else {
			// Check current membership, including callbacks on old messages.
			member, e := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: q.Message.Chat.ID, UserID: q.From.ID}})
			if e != nil {
				notice = "Não foi possível verificar sua participação no grupo."
			} else if member.HasLeft() || member.WasKicked() || (member.Status == "restricted" && !member.IsMember) {
				notice = "Apenas membros deste grupo podem aceitar."
			} else {
				text, err = b.game.Accept(q.Message.Chat.ID, id, q.From.ID, visibleName(q.From))
				if err != nil {
					notice = gameError(err)
				}
			}
		}
	}
	if _, err := b.api.Request(tgbotapi.NewCallback(q.ID, notice)); err != nil {
		slog.Warn("falha ao responder callback", "error", err)
	}
	if text != "" {
		edit := tgbotapi.NewEditMessageText(q.Message.Chat.ID, q.Message.MessageID, text)
		empty := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		edit.ReplyMarkup = &empty
		if _, err := b.api.Send(edit); err != nil {
			slog.Warn("falha ao editar resultado do duelo", "error", err)
			b.replyGame(q.Message, text)
		}
	}
}
