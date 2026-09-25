package tiktok

import (
	"context"
	"errors"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type fakeStrategy struct {
	called bool
	result *media.Result
	err    error
}

func (f *fakeStrategy) Name() string {
	return "fake"
}

func (f *fakeStrategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	url string,
	outputDir string,
) (*media.Result, error) {
	f.called = true

	return f.result, f.err
}

func TestCanHandleTikTok(t *testing.T) {
	d := New()

	if !d.CanHandle(platform.TikTok) {
		t.Fatal("downloader deveria aceitar TikTok")
	}
}

func TestDoesNotHandleOtherPlatforms(t *testing.T) {
	d := New()

	platforms := []platform.Platform{
		platform.Instagram,
		platform.Threads,
		platform.Twitter,
		platform.Reddit,
		platform.YouTube,
		platform.Erome,
	}

	for _, p := range platforms {
		if d.CanHandle(p) {
			t.Fatalf(
				"downloader do TikTok não deveria aceitar %q",
				p,
			)
		}
	}
}

func TestDownloadUsesStrategy(t *testing.T) {
	strategy := &fakeStrategy{
		result: &media.Result{
			Platform: platform.TikTok,
			URL:      "https://www.tiktok.com/@usuario/video/123",
			Items: []media.Item{
				{
					Type: media.Video,
					Path: "/tmp/video.mp4",
				},
			},
		},
	}

	d := New(strategy)

	result, err := d.Download(
		context.Background(),
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if !strategy.called {
		t.Fatal("estratégia deveria ter sido chamada")
	}

	if result.Platform != platform.TikTok {
		t.Fatalf(
			"plataforma = %q; esperado %q",
			result.Platform,
			platform.TikTok,
		)
	}
}

func TestDownloadPropagatesError(t *testing.T) {
	strategy := &fakeStrategy{
		err: downloader.ErrNoMedia,
	}

	d := New(strategy)

	_, err := d.Download(
		context.Background(),
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro")
	}

	if !errors.Is(err, downloader.ErrNoMedia) {
		t.Fatalf(
			"erro = %v; esperado ErrNoMedia",
			err,
		)
	}
}
