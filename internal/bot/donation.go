package bot

import (
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func parseDonation(args string) (int64, string, bool) {
	parts := strings.Fields(args)
	if len(parts) != 2 || len(parts[1]) < 2 || parts[1][0] != '@' {
		return 0, "", false
	}
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return 0, "", false
		}
	}
	for _, c := range parts[1][1:] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return 0, "", false
		}
	}
	amount, err := strconv.ParseInt(parts[0], 10, 64)
	return amount, parts[1], err == nil && amount > 0
}

func (b *Bot) handleDonation(m *tgbotapi.Message, args string) {
	amount, username, ok := parseDonation(args)
	if !ok {
		b.reply(m, "Use /doar quantidade @usuário. Ex.: /doar 5 @pedro")
		return
	}
	id, err := b.game.FindUsername(m.Chat.ID, username)
	if err != nil {
		b.reply(m, gameError(err))
		return
	}
	member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: m.Chat.ID, UserID: id}})
	if err != nil {
		b.reply(m, "Não foi possível verificar o destinatário. Tente novamente.")
		return
	}
	if member.User == nil || member.User.IsBot || member.HasLeft() || member.WasKicked() || (member.Status == "restricted" && !member.IsMember) {
		b.reply(m, "O destinatário precisa ser uma pessoa que participa deste grupo.")
		return
	}
	if !strings.EqualFold(member.User.UserName, strings.TrimPrefix(username, "@")) {
		b.reply(m, "Esse @ mudou. Peça ao destinatário para usar /status e tente novamente.")
		return
	}
	text, err := b.game.Donate(m.Chat.ID, m.From.ID, id, amount, visibleName(m.From), visibleName(member.User))
	if err != nil {
		b.reply(m, gameError(err))
		return
	}
	b.reply(m, text)
}
