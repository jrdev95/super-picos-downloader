package workspace

import (
	"os"
	"testing"
)

func TestNewWorkspace(t *testing.T) {
	ws, err := New()
	if err != nil {
		t.Fatalf("não foi possível criar workspace: %v", err)
	}

	defer ws.Cleanup()

	if ws.Path() == "" {
		t.Fatal("workspace deveria possuir um caminho")
	}

	info, err := os.Stat(ws.Path())
	if err != nil {
		t.Fatalf(
			"diretório do workspace não existe: %v",
			err,
		)
	}

	if !info.IsDir() {
		t.Fatal("workspace deveria ser um diretório")
	}
}

func TestCleanupWorkspace(t *testing.T) {
	ws, err := New()
	if err != nil {
		t.Fatalf("não foi possível criar workspace: %v", err)
	}

	path := ws.Path()

	if err := ws.Cleanup(); err != nil {
		t.Fatalf("falha ao remover workspace: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("workspace deveria ter sido removido")
	}
}
