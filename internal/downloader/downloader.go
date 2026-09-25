package downloader

import (
	"context"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Downloader interface {
	CanHandle(platform.Platform) bool

	Download(
		ctx context.Context,
		url string,
		outputDir string,
	) (*media.Result, error)
}
