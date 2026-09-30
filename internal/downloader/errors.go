package downloader

import (
	"context"
	"errors"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

var (
	ErrNoMedia          = media.ErrNoMedia
	ErrUnsupported      = errors.New("estratégia não suporta o conteúdo")
	ErrAuthentication   = errors.New("autenticação necessária")
	ErrUnavailable      = errors.New("conteúdo indisponível")
	ErrInvalidURL       = errors.New("URL inválida")
	ErrUnsupportedMedia = errors.New("tipo de mídia ainda não suportado")
)

func ShouldFallback(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, context.Canceled):
		return false

	case errors.Is(err, context.DeadlineExceeded):
		return false

	case errors.Is(err, ErrAuthentication):
		return false

	case errors.Is(err, ErrUnavailable):
		return false

	case errors.Is(err, ErrInvalidURL):
		return false

	default:
		return true
	}
}

// IsNoMediaOnly reports absence of media without hiding other failures in a
// fallback chain. Unsupported strategies are neutral, but at least one strategy
// must explicitly report no media. Authentication, network and format errors
// must remain visible even when another strategy returns ErrNoMedia.
func IsNoMediaOnly(err error) bool {
	found := false
	var inspect func(error) bool
	inspect = func(current error) bool {
		if current == nil {
			return false
		}
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			children := wrapped.Unwrap()
			if len(children) == 0 {
				return false
			}
			for _, child := range children {
				if !inspect(child) {
					return false
				}
			}
			return true
		case interface{ Unwrap() error }:
			return inspect(wrapped.Unwrap())
		default:
			if current == ErrNoMedia {
				found = true
				return true
			}
			return current == ErrUnsupported
		}
	}
	safe := inspect(err)
	return safe && found
}
