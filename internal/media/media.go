package media

import (
	"errors"
	"fmt"

	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type Type string

const (
	Unknown Type = ""
	Photo   Type = "photo"
	Video   Type = "video"
)

type Item struct {
	Type     Type
	Path     string
	Filename string
	Caption  string
}

type Result struct {
	Platform platform.Platform
	URL      string
	Items    []Item
}

func (r Result) Empty() bool {
	return len(r.Items) == 0
}

func (r Result) IsAlbum() bool {
	return len(r.Items) > 1
}

func (r Result) Validate() error {
	if r.Platform == platform.Unknown {
		return errors.New("plataforma não definida")
	}

	if r.URL == "" {
		return errors.New("URL de origem não definida")
	}

	if len(r.Items) == 0 {
		return errors.New("nenhuma mídia encontrada")
	}

	for i, item := range r.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("mídia %d inválida: %w", i+1, err)
		}
	}

	return nil
}

func (i Item) Validate() error {
	switch i.Type {
	case Photo, Video:
	default:
		return errors.New("tipo de mídia inválido")
	}

	if i.Path == "" {
		return errors.New("caminho do arquivo não definido")
	}

	return nil
}
