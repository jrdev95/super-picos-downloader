package downloader

import (
	"context"
	"errors"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type fakeDownloader struct {
	supportedPlatform platform.Platform
	result            *media.Result
	err               error
	called            bool
}

func (f *fakeDownloader) CanHandle(
	detectedPlatform platform.Platform,
) bool {
	return detectedPlatform == f.supportedPlatform
}

func (f *fakeDownloader) Download(
	ctx context.Context,
	url string,
	outputDir string,
) (*media.Result, error) {
	f.called = true

	return f.result, f.err
}

func TestManagerUsesCorrectDownloader(t *testing.T) {
	instagramDownloader := &fakeDownloader{
		supportedPlatform: platform.Instagram,
		result: &media.Result{
			Platform: platform.Instagram,
			URL:      "https://www.instagram.com/p/123/",
			Items: []media.Item{
				{
					Type: media.Photo,
					Path: "/tmp/photo.jpg",
				},
			},
		},
	}

	tiktokDownloader := &fakeDownloader{
		supportedPlatform: platform.TikTok,
	}

	manager := NewManager(
		instagramDownloader,
		tiktokDownloader,
	)

	result, err := manager.Download(
		context.Background(),
		platform.Instagram,
		"https://www.instagram.com/p/123/",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if result.Platform != platform.Instagram {
		t.Fatalf(
			"plataforma retornada = %q; esperado %q",
			result.Platform,
			platform.Instagram,
		)
	}

	if !instagramDownloader.called {
		t.Fatal("downloader do Instagram deveria ter sido chamado")
	}

	if tiktokDownloader.called {
		t.Fatal("downloader do TikTok não deveria ter sido chamado")
	}
}

func TestManagerUnsupportedPlatform(t *testing.T) {
	manager := NewManager()

	_, err := manager.Download(
		context.Background(),
		platform.Erome,
		"https://www.erome.com/a/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro para plataforma sem downloader")
	}

	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf(
			"erro = %v; esperado ErrUnsupportedPlatform",
			err,
		)
	}
}

func TestManagerPropagatesDownloadError(t *testing.T) {
	expectedErr := errors.New("download falhou")

	fake := &fakeDownloader{
		supportedPlatform: platform.TikTok,
		err:               expectedErr,
	}

	manager := NewManager(fake)

	_, err := manager.Download(
		context.Background(),
		platform.TikTok,
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro do downloader")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"erro = %v; esperado %v",
			err,
			expectedErr,
		)
	}
}

func TestManagerRejectsInvalidResult(t *testing.T) {
	fake := &fakeDownloader{
		supportedPlatform: platform.Reddit,
		result: &media.Result{
			Platform: platform.Reddit,
			URL:      "https://reddit.com/r/test/comments/123",
		},
	}

	manager := NewManager(fake)

	_, err := manager.Download(
		context.Background(),
		platform.Reddit,
		"https://reddit.com/r/test/comments/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("resultado sem mídias deveria ser rejeitado")
	}
}
