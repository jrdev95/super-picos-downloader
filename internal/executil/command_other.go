//go:build !windows

package executil

import "os/exec"

func configure(_ *exec.Cmd) {}
