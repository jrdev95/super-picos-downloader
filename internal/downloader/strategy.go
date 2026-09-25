package downloader

import (
	"context"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Strategy interface {
	Name() string

	Download(
		ctx context.Context,
		detectedPlatform platform.Platform,
		url string,
		outputDir string,
	) (*media.Result, error)
}
