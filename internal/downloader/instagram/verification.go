package instagram

import (
	"context"
	"fmt"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/logging"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"net/url"
	"os"
	"strings"
)

type verifiedCobalt struct {
	source       downloader.Strategy
	alternatives *downloader.Chain
}

func (v *verifiedCobalt) Name() string { return v.source.Name() }
func (v *verifiedCobalt) Download(ctx context.Context, site platform.Platform, rawURL, outputDir string) (*media.Result, error) {
	// Keep the candidate isolated from extractors that scan their output directory.
	candidateDir, err := os.MkdirTemp(outputDir, "cobalt-candidate-")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(candidateDir)
		}
	}()
	result, err := v.source.Download(ctx, site, rawURL, candidateDir)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, downloader.ErrNoMedia
	}
	if err = result.Validate(); err != nil {
		return nil, err
	}
	source, err := url.Parse(rawURL)
	reel := err == nil && (strings.HasPrefix(strings.ToLower(source.Path), "/reel/") || strings.HasPrefix(strings.ToLower(source.Path), "/reels/"))
	photosOnly := true
	for _, item := range result.Items {
		if item.Type != media.Photo {
			photosOnly = false
		}
	}
	if !reel || !photosOnly {
		keep = true
		return result, nil
	}
	logging.DownloadStep(ctx, "Cobalt retornou fotos para Reel; verificando com outro extrator")
	verified, err := v.alternatives.Download(ctx, site, rawURL, outputDir)
	if err != nil {
		// Verification already tried the remaining chain; don't repeat it or send an unverified cover.
		return nil, &verificationError{fmt.Errorf("não foi possível confirmar a mídia do Reel: %w", err)}
	}
	logging.DownloadStep(ctx, "Reel confirmado por extrator alternativo")
	return verified, nil
}

// Stop the outer chain after the inner verification has exhausted its alternatives.
type verificationError struct{ error }

func (e *verificationError) Unwrap() error      { return e.error }
func (e *verificationError) StopFallback() bool { return true }
