package downloader

import (
	"context"
	"errors"
)

var (
	ErrNoMedia          = errors.New("nenhuma mídia encontrada")
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
