//go:build windows

package internal

import (
	"os/exec"
	"syscall"
)

// HideWindow 在 Windows 图形界面下调用 MP4Box 等控制台程序时不弹出控制台窗口
func HideWindow(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}
