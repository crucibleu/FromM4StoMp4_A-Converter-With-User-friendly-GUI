package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// getCliPath 获取命令绝对路径
func getCliPath(name string) (string, error) {
	execCmd := exec.Command("which", name)

	var stdout bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stdout

	if err := execCmd.Run(); err != nil {
		return "", fmt.Errorf("找不到MP4Box命令,安装GPAC后重试: %v, %v", err, stdout.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
