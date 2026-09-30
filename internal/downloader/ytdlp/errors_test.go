package ytdlp

import (
	"errors"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
)

func TestClassifyUnsupportedError(t *testing.T) {
	err := classifyError(
		errors.New("exit status 1"),
		"ERROR: Unsupported URL",
	)

	if !errors.Is(err, downloader.ErrUnsupported) {
		t.Fatalf(
			"esperava ErrUnsupported; recebido %v",
			err,
		)
	}
}

func TestClassifyAuthenticationError(t *testing.T) {
	err := classifyError(
		errors.New("exit status 1"),
		"ERROR: Login required",
	)

	if !errors.Is(err, downloader.ErrAuthentication) {
		t.Fatalf(
			"esperava ErrAuthentication; recebido %v",
			err,
		)
	}
}

func TestClassifyUnavailableError(t *testing.T) {
	err := classifyError(
		errors.New("exit status 1"),
		"ERROR: Video unavailable",
	)

	if !errors.Is(err, downloader.ErrUnavailable) {
		t.Fatalf(
			"esperava ErrUnavailable; recebido %v",
			err,
		)
	}
}

func TestClassifyNoMediaError(t *testing.T) {
	err := classifyError(
		errors.New("exit status 1"),
		"ERROR: No video formats found",
	)

	if !errors.Is(err, downloader.ErrNoMedia) {
		t.Fatalf(
			"esperava ErrNoMedia; recebido %v",
			err,
		)
	}
}

func TestTwitterTextPostDoesNotBecomeGenericFailure(t *testing.T) {
	err := classifyError(errors.New("exit status 1"),
		"ERROR: [twitter] 2104948145985835437: No video could be found in this tweet")
	if !errors.Is(err, downloader.ErrNoMedia) {
		t.Fatalf("expected no media, got %v", err)
	}
	// gallery-dl completed without media; yt-dlp confirms it has no video.
	combined := errors.Join(downloader.ErrNoMedia, err)
	if !downloader.IsNoMediaOnly(combined) {
		t.Fatalf("text-only post would still produce a reply: %v", combined)
	}
	// No video alone does not establish that a photo download did not fail.
	combined = errors.Join(errors.New("gallery-dl: HTTP 403 Forbidden"), err)
	if downloader.IsNoMediaOnly(combined) {
		t.Fatal("real access error must not be hidden")
	}
}
