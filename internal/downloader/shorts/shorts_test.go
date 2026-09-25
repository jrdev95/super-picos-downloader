package shorts

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

func TestCanHandleYouTube(t *testing.T) {
	d := New()

	if !d.CanHandle(platform.YouTube) {
		t.Fatal("downloader deveria aceitar YouTube Shorts")
	}
}

func TestDoesNotHandleOtherPlatforms(t *testing.T) {
	d := New()

	platforms := []platform.Platform{
		platform.Instagram,
		platform.TikTok,
		platform.Threads,
		platform.Twitter,
		platform.Reddit,
		platform.Erome,
	}

	for _, p := range platforms {
		if d.CanHandle(p) {
			t.Fatalf(
				"downloader de Shorts não deveria aceitar %q",
				p,
			)
		}
	}
}

func TestDownloadUsesStrategy(t *testing.T) {
	strategy := &fakeStrategy{
		result: &media.Result{
			Platform: platform.YouTube,
			URL:      "https://youtube.com/shorts/123",
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
		"https://youtube.com/shorts/123",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if !strategy.called {
		t.Fatal("estratégia deveria ter sido chamada")
	}

	if result.Platform != platform.YouTube {
		t.Fatalf(
			"plataforma = %q; esperado %q",
			result.Platform,
			platform.YouTube,
		)
	}
}

func TestDownloadPropagatesError(t *testing.T) {
	expectedErr := downloader.ErrNoMedia

	strategy := &fakeStrategy{
		err: expectedErr,
	}

	d := New(strategy)

	_, err := d.Download(
		context.Background(),
		"https://youtube.com/shorts/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"erro = %v; esperado %v",
			err,
			expectedErr,
		)
	}
}
