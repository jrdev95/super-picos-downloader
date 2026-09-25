package downloader

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Chain struct {
	strategies []Strategy
}

func NewChain(strategies ...Strategy) *Chain {
	return &Chain{
		strategies: strategies,
	}
}

func (c *Chain) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	url string,
	outputDir string,
) (*media.Result, error) {
	if len(c.strategies) == 0 {
		return nil, errors.New("nenhuma estratégia de download configurada")
	}

	var attemptErrors []error

	for _, strategy := range c.strategies {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, err := strategy.Download(
			ctx,
			detectedPlatform,
			url,
			outputDir,
		)

		if err == nil && result != nil {
			if validationErr := result.Validate(); validationErr == nil {
				return result, nil
			} else {
				err = fmt.Errorf(
					"resultado inválido: %w",
					validationErr,
				)
			}
		}

		if err == nil {
			err = ErrNoMedia
		}

		attemptErrors = append(
			attemptErrors,
			fmt.Errorf(
				"%s: %w",
				strategy.Name(),
				err,
			),
		)

		if !ShouldFallback(err) {
			break
		}
	}

	return nil, fmt.Errorf(
		"todas as estratégias de download falharam: %w",
		errors.Join(attemptErrors...),
	)
}
