//go:build !windows

package internal

import "os/exec"

func HideWindow(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
