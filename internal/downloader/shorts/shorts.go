package shorts

import (
	"context"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Downloader struct {
	chain *downloader.Chain
}

func New(strategies ...downloader.Strategy) *Downloader {
	return &Downloader{
		chain: downloader.NewChain(strategies...),
	}
}

func (d *Downloader) CanHandle(
	detectedPlatform platform.Platform,
) bool {
	return detectedPlatform == platform.YouTube
}

func (d *Downloader) Download(
	ctx context.Context,
	url string,
	outputDir string,
) (*media.Result, error) {
	return d.chain.Download(
		ctx,
		platform.YouTube,
		url,
		outputDir,
	)
}
