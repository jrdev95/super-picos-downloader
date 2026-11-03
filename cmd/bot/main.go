package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/bot"
	"github.com/jrdev95/super-picos-downloader/internal/config"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/erome"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/gallerydl"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/instagram"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/shorts"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/threads"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/tiktok"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/twitter"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/ytdlp"
	"github.com/jrdev95/super-picos-downloader/internal/logging"
	"github.com/jrdev95/super-picos-downloader/internal/minigame"
	"github.com/jrdev95/super-picos-downloader/internal/tools"
)

func main() {
	logger := slog.New(
		logging.NewConsoleHandler(os.Stdout),
	)

	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error(
			"falha ao carregar configuração",
			"error", err,
		)

		os.Exit(1)
	}

	if cfg.LowMemory {
		// This is a soft Go runtime limit; subprocesses have their own memory.
		if os.Getenv("GOMEMLIMIT") == "" {
			debug.SetMemoryLimit(256 << 20)
		}
		if os.Getenv("TMPDIR") == "" {
			tmpDir, err := filepath.Abs(filepath.Join("data", "tmp"))
			if err == nil {
				err = os.MkdirAll(tmpDir, 0700)
			}
			if err == nil {
				err = os.Setenv("TMPDIR", tmpDir)
			}
			if err != nil {
				slog.Error("falha ao preparar temporários no disco", "error", err)
				os.Exit(1)
			}
		}
		slog.Info("modo de pouca memória ativado", "workers", cfg.MaxWorkers, "ffmpeg_threads", 1)
	}
	externalTools, err := tools.Detect()
	if err != nil {
		slog.Error(
			"dependências externas não disponíveis",
			"error", err,
		)

		os.Exit(1)
	}

	slog.Info(
		"ferramentas externas detectadas",
		"yt_dlp", externalTools.YTDLP,
		"ffmpeg", externalTools.FFmpeg,
		"gallery_dl", externalTools.GalleryDL,
	)

	ytdlpStrategy := ytdlp.New(
		externalTools.YTDLP,
		externalTools.FFmpeg,
	)
	ytdlpStrategy.SetLowMemory(cfg.LowMemory)
	ytdlpStrategy.SetTwitterCookiesFile(cfg.XCookiesFile)

	galleryDLStrategy := gallerydl.New(
		externalTools.GalleryDL,
		gallerydl.DefaultBrowserProfile(),
	)

	shortsDownloader := shorts.New(
		ytdlpStrategy,
	)

	tiktokDownloader := tiktok.New(
		ytdlpStrategy,
		galleryDLStrategy,
	)

	instagramGalleryDLStrategy := instagram.NewGalleryDLStrategy(
		externalTools.GalleryDL,
		cfg.InstagramCookiesFile,
	)

	instagramDownloader := instagram.New(
		instagram.NewCobaltStrategy(cfg.CobaltAPIURL),
		instagramGalleryDLStrategy,
		ytdlpStrategy,
	)

	threadsDownloader := threads.New()

	twitterGalleryDL := twitter.NewGalleryDLStrategy(
		externalTools.GalleryDL,
	)

	twitterDownloader := twitter.New(
		twitterGalleryDL,
		ytdlpStrategy,
	)

	eromeDownloader := erome.New()

	downloadManager := downloader.NewManager(
		shortsDownloader,
		tiktokDownloader,
		instagramDownloader,
		threadsDownloader,
		twitterDownloader,
		eromeDownloader,
	)

	game, err := minigame.Open("data/minigame.db")
	if err != nil {
		slog.Error("falha ao abrir minigame", "error", err)
		os.Exit(1)
	}
	defer game.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	var telegramBot *bot.Bot
	var lastConnectErrorLog time.Time
	for {
		telegramBot, err = bot.New(
			cfg.BotToken,
			downloadManager,
			cfg.MaxWorkers,
			bot.WithMediaReduction(cfg.ReduceMedia),
			bot.WithLowMemory(cfg.LowMemory),
			bot.WithMinigame(game),
		)
		if err == nil {
			break
		}
		if !retryableTelegramStartupError(err) {
			slog.Error("falha ao iniciar bot", "error", redactBotToken(err, cfg.BotToken))
			os.Exit(1)
		}

		now := time.Now()
		if lastConnectErrorLog.IsZero() || now.Sub(lastConnectErrorLog) >= time.Minute {
			slog.Warn(
				"falha ao conectar ao Telegram; nova tentativa em 5s",
				"error", redactBotToken(err, cfg.BotToken),
			)
			lastConnectErrorLog = now
		}

		select {
		case <-ctx.Done():
			slog.Info("inicialização cancelada")
			return
		case <-time.After(5 * time.Second):
		}
	}

	slog.Info("Super Picos Downloader iniciado")

	if err := telegramBot.Run(ctx); err != nil {
		slog.Error(
			"erro durante execução do bot",
			"error", err,
		)

		os.Exit(1)
	}

	slog.Info("Super Picos Downloader encerrado")
}

func retryableTelegramStartupError(err error) bool {
	var apiErr tgbotapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code >= http.StatusInternalServerError
	}
	var apiErrPointer *tgbotapi.Error
	if errors.As(err, &apiErrPointer) && apiErrPointer != nil {
		return apiErrPointer.Code >= http.StatusInternalServerError
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "bad gateway") ||
		strings.Contains(message, "gateway timeout") ||
		strings.Contains(message, "service unavailable")
}

func redactBotToken(err error, token string) string {
	if err == nil {
		return ""
	}
	if token == "" {
		return err.Error()
	}
	return strings.ReplaceAll(err.Error(), token, "[REDACTED]")
}
