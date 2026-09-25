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
