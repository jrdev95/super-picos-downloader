package ytdlp

import (
	"fmt"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
)

func classifyError(commandErr error, stderr string) error {
	message := strings.ToLower(stderr)

	switch {
	case containsAny(
		message,
		"login required",
		"sign in",
		"authentication required",
		"cookies are required",
		"login to confirm",
	):
		return fmt.Errorf(
			"%w: %s",
			downloader.ErrAuthentication,
			cleanError(stderr),
		)

	case containsAny(
		message,
		"video unavailable",
		"content unavailable",
		"content is not available",
		"has been removed",
		"this video has been removed",
		"not found",
	):
		return fmt.Errorf(
			"%w: %s",
			downloader.ErrUnavailable,
			cleanError(stderr),
		)

	case containsAny(
		message,
		"invalid url",
		"malformed url",
	):
		return fmt.Errorf(
			"%w: %s",
			downloader.ErrInvalidURL,
			cleanError(stderr),
		)

	case containsAny(
		message,
		"unsupported url",
	):
		return fmt.Errorf(
			"%w: %s",
			downloader.ErrUnsupported,
			cleanError(stderr),
		)

	case containsAny(
		message,
		"no video formats found",
		"no formats found",
	):
		return fmt.Errorf(
			"%w: %s",
			downloader.ErrNoMedia,
			cleanError(stderr),
		)

	default:
		return fmt.Errorf(
			"yt-dlp falhou: %w: %s",
			commandErr,
			cleanError(stderr),
		)
	}
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}

	return false
}

func cleanError(stderr string) string {
	message := strings.TrimSpace(stderr)

	if message == "" {
		return "erro sem detalhes"
	}

	return message
}
