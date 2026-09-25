package media

import (
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

func TestResultEmpty(t *testing.T) {
	result := Result{}

	if !result.Empty() {
		t.Fatal("resultado sem itens deveria estar vazio")
	}
}

func TestResultSingleMediaIsNotAlbum(t *testing.T) {
	result := Result{
		Items: []Item{
			{
				Type: Photo,
				Path: "/tmp/photo.jpg",
			},
		},
	}

	if result.IsAlbum() {
		t.Fatal("resultado com uma mídia não deveria ser álbum")
	}
}

func TestResultMultipleMediaIsAlbum(t *testing.T) {
	result := Result{
		Items: []Item{
			{
				Type: Photo,
				Path: "/tmp/photo1.jpg",
			},
			{
				Type: Photo,
				Path: "/tmp/photo2.jpg",
			},
		},
	}

	if !result.IsAlbum() {
		t.Fatal("resultado com múltiplas mídias deveria ser álbum")
	}
}

func TestValidResult(t *testing.T) {
	result := Result{
		Platform: platform.Instagram,
		URL:      "https://www.instagram.com/p/123/",
		Items: []Item{
			{
				Type: Photo,
				Path: "/tmp/photo.jpg",
			},
		},
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("resultado deveria ser válido: %v", err)
	}
}

func TestResultWithoutMedia(t *testing.T) {
	result := Result{
		Platform: platform.Instagram,
		URL:      "https://www.instagram.com/p/123/",
	}

	if err := result.Validate(); err == nil {
		t.Fatal("resultado sem mídia deveria ser inválido")
	}
}

func TestItemWithoutPath(t *testing.T) {
	item := Item{
		Type: Photo,
	}

	if err := item.Validate(); err == nil {
		t.Fatal("mídia sem caminho deveria ser inválida")
	}
}

func TestInvalidMediaType(t *testing.T) {
	item := Item{
		Type: Type("banana"),
		Path: "/tmp/coisa.jpg",
	}

	if err := item.Validate(); err == nil {
		t.Fatal("tipo de mídia desconhecido deveria ser inválido")
	}
}
