package downloader

import (
	"context"
	"errors"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type fakeStrategy struct {
	name   string
	result *media.Result
	err    error
	called bool
}

func (f *fakeStrategy) Name() string {
	return f.name
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

func validResult() *media.Result {
	return &media.Result{
		Platform: platform.TikTok,
		URL:      "https://www.tiktok.com/@usuario/video/123",
		Items: []media.Item{
			{
				Type: media.Video,
				Path: "/tmp/video.mp4",
			},
		},
	}
}

func TestChainStopsOnSuccess(t *testing.T) {
	first := &fakeStrategy{
		name:   "yt-dlp",
		result: validResult(),
	}

	second := &fakeStrategy{
		name: "fallback",
	}

	chain := NewChain(first, second)

	result, err := chain.Download(
		context.Background(),
		platform.TikTok,
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if result == nil {
		t.Fatal("esperava resultado")
	}

	if !first.called {
		t.Fatal("primeira estratégia deveria ter sido chamada")
	}

	if second.called {
		t.Fatal("fallback não deveria ter sido chamado")
	}
}

func TestChainUsesFallback(t *testing.T) {
	first := &fakeStrategy{
		name: "yt-dlp",
		err:  ErrNoMedia,
	}

	second := &fakeStrategy{
		name:   "fallback",
		result: validResult(),
	}

	chain := NewChain(first, second)

	result, err := chain.Download(
		context.Background(),
		platform.TikTok,
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if result == nil {
		t.Fatal("esperava resultado do fallback")
	}

	if !first.called {
		t.Fatal("primeira estratégia deveria ter sido chamada")
	}

	if !second.called {
		t.Fatal("fallback deveria ter sido chamado")
	}
}

func TestChainStopsOnAuthenticationError(t *testing.T) {
	first := &fakeStrategy{
		name: "yt-dlp",
		err:  ErrAuthentication,
	}

	second := &fakeStrategy{
		name: "fallback",
	}

	chain := NewChain(first, second)

	_, err := chain.Download(
		context.Background(),
		platform.Instagram,
		"https://www.instagram.com/p/123/",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro")
	}

	if !errors.Is(err, ErrAuthentication) {
		t.Fatalf(
			"esperava ErrAuthentication; recebido: %v",
			err,
		)
	}

	if second.called {
		t.Fatal(
			"fallback não deveria ser chamado após erro de autenticação",
		)
	}
}

func TestChainAllStrategiesFail(t *testing.T) {
	first := &fakeStrategy{
		name: "yt-dlp",
		err:  ErrNoMedia,
	}

	second := &fakeStrategy{
		name: "fallback",
		err:  ErrUnsupported,
	}

	chain := NewChain(first, second)

	_, err := chain.Download(
		context.Background(),
		platform.Threads,
		"https://www.threads.com/@usuario/post/123",
		"/tmp",
	)

	if err == nil {
		t.Fatal("esperava erro")
	}

	if !first.called || !second.called {
		t.Fatal("todas as estratégias deveriam ter sido executadas")
	}
}

func TestChainFallsBackOnInvalidResult(t *testing.T) {
	first := &fakeStrategy{
		name: "yt-dlp",
		result: &media.Result{
			Platform: platform.TikTok,
			URL:      "https://www.tiktok.com/@usuario/video/123",
		},
	}

	second := &fakeStrategy{
		name:   "fallback",
		result: validResult(),
	}

	chain := NewChain(first, second)

	result, err := chain.Download(
		context.Background(),
		platform.TikTok,
		"https://www.tiktok.com/@usuario/video/123",
		"/tmp",
	)

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}

	if result == nil {
		t.Fatal("fallback deveria retornar resultado válido")
	}

	if !second.called {
		t.Fatal("fallback deveria ter sido utilizado")
	}
}
