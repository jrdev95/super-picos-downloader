package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jrdev95/super-picos-downloader/internal/bot"
	"github.com/jrdev95/super-picos-downloader/internal/config"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/erome"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/gallerydl"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/instagram"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/reddit"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/shorts"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/threads"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/tiktok"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/twitter"
	"github.com/jrdev95/super-picos-downloader/internal/downloader/ytdlp"
	"github.com/jrdev95/super-picos-downloader/internal/tools"
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, nil),
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

	redditRSSStrategy := reddit.NewRSSStrategy()

	redditDownloader := reddit.New(
		ytdlpStrategy,
		redditRSSStrategy,
	)

	eromeDownloader := erome.New()

	downloadManager := downloader.NewManager(
		shortsDownloader,
		tiktokDownloader,
		instagramDownloader,
		threadsDownloader,
		twitterDownloader,
		redditDownloader,
		eromeDownloader,
	)

	telegramBot, err := bot.New(
		cfg.BotToken,
		downloadManager,
		cfg.MaxWorkers,
	)
	if err != nil {
		slog.Error(
			"falha ao iniciar bot",
			"error", err,
		)

		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

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
