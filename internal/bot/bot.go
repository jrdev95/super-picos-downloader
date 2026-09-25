package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	manager *downloader.Manager

	jobs    chan downloadJob
	workers int

	wg sync.WaitGroup
}

func New(
	token string,
	manager *downloader.Manager,
	workers int,
) (*Bot, error) {
	if manager == nil {
		return nil, fmt.Errorf(
			"downloader manager não foi definido",
		)
	}

	if workers < 1 {
		return nil, fmt.Errorf(
			"quantidade de workers inválida: %d",
			workers,
		)
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao conectar ao Telegram: %w",
			err,
		)
	}

	slog.Info(
		"conectado ao Telegram",
		"bot", "@"+api.Self.UserName,
	)

	return &Bot{
		api:     api,
		manager: manager,

		jobs: make(
			chan downloadJob,
			downloadQueueSize,
		),

		workers: workers,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	workerCtx, cancelWorkers := context.WithCancel(ctx)

	b.startWorkers(workerCtx)

	defer func() {
		cancelWorkers()
		b.wg.Wait()
	}()

	updateConfig, err := b.prepareUpdates()
	if err != nil {
		return err
	}

	slog.Info("aguardando mensagens")

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
		}

		updates, err := b.api.GetUpdates(updateConfig)
		if err != nil {
			slog.Error(
				"erro ao buscar atualizações do Telegram",
				"error", err,
			)

			select {
			case <-ctx.Done():
				return nil

			case <-time.After(3 * time.Second):
				continue
			}
		}

		for _, update := range updates {
			updateConfig.Offset = update.UpdateID + 1

			slog.Info(
				"update recebido",
				"update_id", update.UpdateID,
				"has_message", update.Message != nil,
			)

			if update.Message == nil {
				continue
			}

			slog.Info(
				"mensagem recebida",
				"chat_id", update.Message.Chat.ID,
				"text", update.Message.Text,
				"caption", update.Message.Caption,
			)

			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) prepareUpdates() (
	tgbotapi.UpdateConfig,
	error,
) {
	// Busca somente o update pendente mais recente.
	// Depois usamos o ID dele para ignorar tudo que chegou
	// enquanto o bot estava desligado.
	pendingConfig := tgbotapi.NewUpdate(-1)
	pendingConfig.Limit = 1
	pendingConfig.Timeout = 0

	pending, err := b.api.GetUpdates(pendingConfig)
	if err != nil {
		return tgbotapi.UpdateConfig{}, fmt.Errorf(
			"falha ao descartar updates pendentes: %w",
			err,
		)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	if len(pending) > 0 {
		lastUpdate := pending[len(pending)-1]

		updateConfig.Offset = lastUpdate.UpdateID + 1

		slog.Info(
			"updates anteriores descartados",
			"last_update_id", lastUpdate.UpdateID,
		)
	}

	return updateConfig, nil
}
