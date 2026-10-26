package tiktok

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Downloader struct {
	chain  *downloader.Chain
	client *http.Client
}

func New(strategies ...downloader.Strategy) *Downloader {
	return &Downloader{
		chain:  downloader.NewChain(strategies...),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *Downloader) CanHandle(
	detectedPlatform platform.Platform,
) bool {
	return detectedPlatform == platform.TikTok
}

func (d *Downloader) Download(
	ctx context.Context,
	url string,
	outputDir string,
) (*media.Result, error) {
	resolvedURL := d.resolveShortURL(ctx, url)
	result, err := d.chain.Download(ctx, platform.TikTok, resolvedURL, outputDir)
	if err != nil {
		return nil, err
	}
	// Preserve the link that the user originally sent in the result/caption metadata.
	result.URL = url
	return result, nil
}

// resolveShortURL follows TikTok share links before strategy selection. Some
// extractors can resolve the short URL internally but don't pass the canonical
// photo URL to the next strategy in a fallback chain.
func (d *Downloader) resolveShortURL(ctx context.Context, rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || !matchesShortTikTokHost(strings.ToLower(parsed.Hostname())) {
		return rawURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return rawURL
	}
	response, err := d.client.Do(req)
	if err != nil {
		return rawURL
	}
	defer response.Body.Close()
	if response.Request == nil || response.Request.URL == nil {
		return rawURL
	}
	finalURL := response.Request.URL.String()
	final, err := url.Parse(finalURL)
	if err != nil || !strings.EqualFold(final.Hostname(), "www.tiktok.com") && !strings.EqualFold(final.Hostname(), "tiktok.com") {
		return rawURL
	}
	return finalURL
}
func matchesShortTikTokHost(host string) bool {
	return host == "vt.tiktok.com" || host == "vm.tiktok.com"
}
