package executil

import (
	"context"
	"os/exec"
)

// CommandContext cria um processo externo e aplica as opções
// específicas da plataforma antes da execução.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	configure(cmd)

	return cmd
}
