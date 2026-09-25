package bot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"github.com/jrdev95/super-picos-downloader/internal/urlutil"
	"github.com/jrdev95/super-picos-downloader/internal/workspace"
)

const downloadTimeout = 3 * time.Minute

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if message == nil {
		return
	}

	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	text := strings.TrimSpace(message.Text)
	if text == "" {
		text = strings.TrimSpace(message.Caption)
	}

	link, found := urlutil.ExtractFirst(text)
	if !found {
		return
	}

	b.handleURL(message, link)
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.handleStart(message)
	}
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	b.reply(
		message,
		"👋 Super Picos Downloader está online.\n\n"+
			"Envie um link de mídia para começar.",
	)
}

func (b *Bot) handleURL(message *tgbotapi.Message, link string) {
	detectedPlatform := platform.Detect(link)

	if detectedPlatform == platform.Unknown {
		slog.Info(
			"plataforma não suportada",
			"url", link,
			"chat_id", message.Chat.ID,
		)

		b.reply(
			message,
			"⚠️ O link foi detectado, mas essa plataforma ainda não é suportada.",
		)
		return
	}

	slog.Info(
		"URL recebida",
		"url", link,
		"platform", detectedPlatform,
		"chat_id", message.Chat.ID,
	)

	b.enqueueDownload(
		message,
		link,
		detectedPlatform,
	)
}

func (b *Bot) enqueueDownload(
	message *tgbotapi.Message,
	link string,
	detectedPlatform platform.Platform,
) {
	job := downloadJob{
		message:  message,
		url:      link,
		platform: detectedPlatform,
	}

	select {
	case b.jobs <- job:
		slog.Info(
			"download adicionado à fila",
			"platform", detectedPlatform,
			"url", link,
			"queue_size", len(b.jobs),
		)

	default:
		slog.Warn(
			"fila de downloads cheia",
			"platform", detectedPlatform,
			"url", link,
		)

		b.reply(
			message,
			"⚠️ Há muitos downloads na fila no momento. Tente novamente em instantes.",
		)
	}
}

func (b *Bot) processDownload(
	parentCtx context.Context,
	message *tgbotapi.Message,
	link string,
	detectedPlatform platform.Platform,
) {
	statusMessage, statusErr := b.sendStatus(
		message,
		"⏳ Baixando mídia...",
	)
	if statusErr != nil {
		slog.Error(
			"falha ao enviar status de download",
			"error", statusErr,
			"chat_id", message.Chat.ID,
		)
	}

	if statusMessage.MessageID != 0 {
		defer b.deleteMessage(
			message.Chat.ID,
			statusMessage.MessageID,
		)
	}

	ws, err := workspace.New()
	if err != nil {
		slog.Error(
			"falha ao criar workspace",
			"error", err,
		)

		b.reply(
			message,
			"❌ Não foi possível preparar o download.",
		)
		return
	}

	defer func() {
		if err := ws.Cleanup(); err != nil {
			slog.Error(
				"falha ao limpar workspace",
				"error", err,
				"path", ws.Path(),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(
		parentCtx,
		downloadTimeout,
	)
	defer cancel()

	slog.Info(
		"iniciando download",
		"platform", detectedPlatform.DisplayName(),
		"url", link,
		"workspace", ws.Path(),
	)

	result, err := b.manager.Download(
		ctx,
		detectedPlatform,
		link,
		ws.Path(),
	)
	if err != nil {
		slog.Error(
			"download falhou",
			"platform", detectedPlatform,
			"url", link,
			"error", err,
		)

		b.reply(
			message,
			downloadErrorMessage(err),
		)
		return
	}

	slog.Info(
		"download concluído",
		"platform", result.Platform,
		"items", len(result.Items),
		"album", result.IsAlbum(),
	)

	if err := b.sendResult(message, result); err != nil {
		slog.Error(
			"falha ao enviar mídia",
			"error", err,
			"chat_id", message.Chat.ID,
		)

		b.reply(
			message,
			"❌ A mídia foi baixada, mas não consegui enviá-la pelo Telegram.",
		)
		return
	}

	slog.Info(
		"mídia enviada com sucesso",
		"platform", result.Platform,
		"items", len(result.Items),
		"chat_id", message.Chat.ID,
	)
}

func (b *Bot) sendStatus(
	message *tgbotapi.Message,
	text string,
) (tgbotapi.Message, error) {
	response := tgbotapi.NewMessage(
		message.Chat.ID,
		text,
	)
	response.ReplyToMessageID = message.MessageID

	sent, err := b.api.Send(response)
	if err != nil {
		return tgbotapi.Message{}, err
	}

	return sent, nil
}

func (b *Bot) deleteMessage(
	chatID int64,
	messageID int,
) {
	config := tgbotapi.NewDeleteMessage(
		chatID,
		messageID,
	)

	if _, err := b.api.Request(config); err != nil {
		slog.Warn(
			"não foi possível remover mensagem de status",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
	}
}

func downloadErrorMessage(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "⏱️ O download demorou mais do que o esperado e foi cancelado."

	case errors.Is(err, downloader.ErrAuthentication):
		return "🔒 Essa publicação exige autenticação ou não está acessível publicamente."

	case errors.Is(err, downloader.ErrUnavailable):
		return "⚠️ Essa publicação não está mais disponível."

	case errors.Is(err, downloader.ErrUnsupportedMedia):
		return "⚠️ Esse tipo de mídia ainda não é suportado."

	case errors.Is(err, downloader.ErrNoMedia):
		return "⚠️ Não encontrei nenhuma mídia válida nessa publicação."

	case errors.Is(err, downloader.ErrUnsupportedPlatform):
		return "⚠️ A plataforma foi reconhecida, mas o downloader ainda não foi implementado."

	default:
		return "❌ Não foi possível baixar essa publicação."
	}
}

func (b *Bot) reply(message *tgbotapi.Message, text string) {
	response := tgbotapi.NewMessage(
		message.Chat.ID,
		text,
	)
	response.ReplyToMessageID = message.MessageID

	if _, err := b.api.Send(response); err != nil {
		slog.Error(
			"falha ao enviar mensagem",
			"error", err,
			"chat_id", message.Chat.ID,
		)
	}
}
