//go:build windows

package common

import (
	"os/exec"
	"syscall"
)

// hideWindow 图形界面下调用 MP4Box 时不弹出控制台窗口
func hideWindow(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}
