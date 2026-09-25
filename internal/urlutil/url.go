package urlutil

import (
	"net/url"
	"strings"
	"unicode"
)

func ExtractFirst(text string) (string, bool) {
	text = strings.TrimSpace(text)

	start := findURLStart(text)
	if start == -1 {
		return "", false
	}

	candidate := text[start:]

	if end := strings.IndexFunc(candidate, isURLDelimiter); end >= 0 {
		candidate = candidate[:end]
	}

	candidate = strings.TrimRight(
		candidate,
		".,;!?",
	)

	return validate(candidate)
}

func findURLStart(text string) int {
	httpsIndex := strings.Index(text, "https://")
	httpIndex := strings.Index(text, "http://")

	switch {
	case httpsIndex >= 0 && httpIndex >= 0:
		if httpsIndex < httpIndex {
			return httpsIndex
		}

		return httpIndex

	case httpsIndex >= 0:
		return httpsIndex

	case httpIndex >= 0:
		return httpIndex

	default:
		return -1
	}
}

func isURLDelimiter(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	switch r {
	case '[', ']', '(', ')', '<', '>', '\\', '"', '\'':
		return true

	default:
		return false
	}
}

func validate(rawURL string) (string, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		return "", false
	}

	if parsedURL.Host == "" {
		return "", false
	}

	return rawURL, true
}
