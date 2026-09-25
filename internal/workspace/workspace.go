package workspace

import (
	"fmt"
	"os"
)

type Workspace struct {
	path string
}

func New() (*Workspace, error) {
	path, err := os.MkdirTemp("", "super-picos-*")
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório temporário: %w",
			err,
		)
	}

	return &Workspace{
		path: path,
	}, nil
}

func (w *Workspace) Path() string {
	return w.path
}

func (w *Workspace) Cleanup() error {
	if w == nil || w.path == "" {
		return nil
	}

	if err := os.RemoveAll(w.path); err != nil {
		return fmt.Errorf(
			"falha ao remover diretório temporário: %w",
			err,
		)
	}

	return nil
}
