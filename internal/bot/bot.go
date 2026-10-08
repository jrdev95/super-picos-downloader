package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/minigame"
)

type Bot struct {
	game    *minigame.Store
	api     *tgbotapi.BotAPI
	manager *downloader.Manager

	jobs    chan downloadJob
	workers int

	reduceMedia bool
	lowMemory   bool
	wg          sync.WaitGroup
}

func New(
	token string,
	manager *downloader.Manager,
	workers int,
	options ...Option,
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

	api, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, newTelegramClient(workers))
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

	b := &Bot{
		api:     api,
		manager: manager,

		jobs: make(
			chan downloadJob,
			downloadQueueSize,
		),

		workers: workers,
	}
	for _, option := range options {
		option(b)
	}
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	// Telegram message dates have second precision. Keep messages from this
	// second, but reject anything sent before this run, even if delivered late.
	startedAt := time.Now().Unix()
	updateConfig, err := b.prepareUpdates()
	if err != nil {
		return err
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)

	b.startWorkers(workerCtx)

	defer func() {
		cancelWorkers()
		b.wg.Wait()
	}()

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

			if update.CallbackQuery != nil {
				b.handleGameCallback(update.CallbackQuery)
				continue
			}
			if update.Message == nil {
				continue
			}
			if int64(update.Message.Date) < startedAt {
				slog.Info("mensagem anterior à inicialização ignorada",
					"update_id", update.UpdateID,
					"message_date", time.Unix(int64(update.Message.Date), 0),
				)
				continue
			}

			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) prepareUpdates() (
	tgbotapi.UpdateConfig,
	error,
) {
	// Explicitly discard Telegram's pending queue, including old callbacks.
	if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}); err != nil {
		return tgbotapi.UpdateConfig{}, fmt.Errorf("falha ao limpar fila pendente do Telegram: %w", err)
	}
	// Busca somente o update pendente mais recente.
	// Depois usamos o ID dele para ignorar tudo que chegou
	// enquanto o bot estava desligado.
	pendingConfig := tgbotapi.NewUpdate(-1)
	pendingConfig.Limit = 1
	pendingConfig.Timeout = 0
	pendingConfig.AllowedUpdates = []string{"message", "callback_query"}

	pending, err := b.api.GetUpdates(pendingConfig)
	if err != nil {
		return tgbotapi.UpdateConfig{}, fmt.Errorf(
			"falha ao descartar updates pendentes: %w",
			err,
		)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updateConfig.AllowedUpdates = []string{"message", "callback_query"}

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

type Option func(*Bot)

// WithMediaReduction enables optional reduction before upload.
func WithMediaReduction(enabled bool) Option {
	return func(b *Bot) { b.reduceMedia = enabled }
}

func WithLowMemory(enabled bool) Option {
	return func(b *Bot) { b.lowMemory = enabled }
}
