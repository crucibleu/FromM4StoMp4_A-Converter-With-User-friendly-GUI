//go:build !windows

package common

import "os/exec"

func hideWindow(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
