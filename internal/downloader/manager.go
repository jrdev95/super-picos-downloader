package downloader

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

var ErrUnsupportedPlatform = errors.New("plataforma sem downloader disponível")

type Manager struct {
	downloaders []Downloader
}

func NewManager(downloaders ...Downloader) *Manager {
	return &Manager{
		downloaders: downloaders,
	}
}

func (m *Manager) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	url string,
	outputDir string,
) (*media.Result, error) {
	for _, d := range m.downloaders {
		if !d.CanHandle(detectedPlatform) {
			continue
		}

		result, err := d.Download(
			ctx,
			url,
			outputDir,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"falha ao baixar mídia de %s: %w",
				detectedPlatform.DisplayName(),
				err,
			)
		}

		if result == nil {
			return nil, errors.New(
				"downloader retornou resultado nulo",
			)
		}

		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf(
				"resultado inválido: %w",
				err,
			)
		}

		return result, nil
	}

	return nil, fmt.Errorf(
		"%w: %s",
		ErrUnsupportedPlatform,
		detectedPlatform.DisplayName(),
	)
}
