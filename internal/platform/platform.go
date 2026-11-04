package platform

import (
	"net/url"
	"strings"
)

type Platform string

const (
	Unknown   Platform = ""
	Instagram Platform = "instagram"
	TikTok    Platform = "tiktok"
	Threads   Platform = "threads"
	Twitter   Platform = "twitter"
	YouTube   Platform = "youtube"
	Erome     Platform = "erome"
)

func Detect(rawURL string) Platform {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return Unknown
	}

	host := strings.ToLower(parsedURL.Hostname())
	path := strings.ToLower(parsedURL.Path)

	switch {
	case matchesDomain(host, "instagram.com"):
		return Instagram

	case matchesDomain(host, "tiktok.com"):
		return TikTok

	case matchesDomain(host, "threads.net"),
		matchesDomain(host, "threads.com"):
		return Threads

	case matchesDomain(host, "x.com"),
		matchesDomain(host, "twitter.com"):
		return Twitter

	case matchesDomain(host, "youtube.com") &&
		strings.HasPrefix(path, "/shorts/"):
		return YouTube

	case matchesDomain(host, "erome.com"):
		return Erome

	default:
		return Unknown
	}
}

// IsInstagramProfileURL identifies profile pages and their profile tabs,
// which do not represent a single media post for the downloader.
func IsInstagramProfileURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || !matchesDomain(strings.ToLower(parsedURL.Hostname()), "instagram.com") {
		return false
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) == 1 && parts[0] != "" {
		switch strings.ToLower(parts[0]) {
		case "accounts", "api", "direct", "explore", "p", "reel", "reels", "share", "stories", "tv":
			return false
		default:
			return true
		}
	}

	if len(parts) == 2 {
		switch strings.ToLower(parts[1]) {
		case "reels", "tagged", "saved":
			return true
		}
	}

	return false
}

func matchesDomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func (p Platform) DisplayName() string {
	switch p {
	case Instagram:
		return "Instagram"

	case TikTok:
		return "TikTok"

	case Threads:
		return "Threads"

	case Twitter:
		return "X / Twitter"

	case YouTube:
		return "YouTube Shorts"

	case Erome:
		return "Erome"

	default:
		return "Desconhecida"
	}
}
