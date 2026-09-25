package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

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
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"github.com/jrdev95/super-picos-downloader/internal/tools"
	"github.com/jrdev95/super-picos-downloader/internal/urlutil"
	"github.com/jrdev95/super-picos-downloader/internal/workspace"
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, nil),
	)
	slog.SetDefault(logger)

	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  go run ./cmd/test-download <URL>")
		os.Exit(1)
	}

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.Warn(
			"não foi possível carregar .env",
			"error", err,
		)
	}

	rawURL, found := urlutil.ExtractFirst(os.Args[1])
	if !found {
		slog.Error("nenhuma URL válida encontrada")
		os.Exit(1)
	}

	detectedPlatform := platform.Detect(rawURL)
	if detectedPlatform == platform.Unknown {
		slog.Error(
			"plataforma não suportada",
			"url", rawURL,
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

	ws, err := workspace.New()
	if err != nil {
		slog.Error(
			"falha ao criar workspace",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info(
		"workspace criado",
		"path", ws.Path(),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Minute,
	)
	defer cancel()

	ytdlpStrategy := ytdlp.New(
		externalTools.YTDLP,
		externalTools.FFmpeg,
	)

	tiktokGalleryDL := gallerydl.New(
		externalTools.GalleryDL,
		gallerydl.DefaultBrowserProfile(),
	)

	shortsDownloader := shorts.New(ytdlpStrategy)
	tiktokDownloader := tiktok.New(ytdlpStrategy, tiktokGalleryDL)

	instagramGalleryDL := instagram.NewGalleryDLStrategy(
		externalTools.GalleryDL,
		os.Getenv("INSTAGRAM_COOKIES_FILE"),
	)
	instagramDownloader := instagram.New(
		instagramGalleryDL,
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

	redditDownloader := reddit.New(
		ytdlpStrategy,
		reddit.NewRSSStrategy(),
	)

	eromeDownloader := erome.New()

	manager := downloader.NewManager(
		shortsDownloader,
		tiktokDownloader,
		instagramDownloader,
		threadsDownloader,
		twitterDownloader,
		redditDownloader,
		eromeDownloader,
	)

	slog.Info(
		"iniciando download",
		"platform", detectedPlatform.DisplayName(),
		"url", rawURL,
	)

	result, err := manager.Download(
		ctx,
		detectedPlatform,
		rawURL,
		ws.Path(),
	)
	if err != nil {
		_ = ws.Cleanup()

		slog.Error(
			"download falhou",
			"error", err,
		)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Download concluído")
	fmt.Println("------------------")
	fmt.Printf("Plataforma: %s\n", result.Platform.DisplayName())
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("Mídias: %d\n", len(result.Items))
	fmt.Printf("Álbum: %t\n", result.IsAlbum())
	fmt.Println()

	for index, item := range result.Items {
		fmt.Printf(
			"%d. tipo=%s arquivo=%s\n",
			index+1,
			item.Type,
			item.Path,
		)
	}

	fmt.Println()
	fmt.Printf("Arquivos mantidos em: %s\n", ws.Path())
	fmt.Println("Este comando de teste não remove o workspace automaticamente.")
}
