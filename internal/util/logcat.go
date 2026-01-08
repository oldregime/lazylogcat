package util

import (
	"fmt"
	"os/exec"
	"strings"
)

func GetLogLevel(line string) string {
	i := strings.Index(line, "/") // TODO more robust way to get log level (this one doesn't work in some formtats)
	if i <= 0 {
		return ""
	}
	return line[i-1 : i]
}

func GetPidByPackageName(deviceId string, pkg string) (string, error) {
	pidCmd := exec.Command("adb", "-s", deviceId, "shell", "pidof", pkg)
	pid, err := pidCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pid: %w", err)
	}
	return strings.Trim(string(pid), "\n\r "), nil
}
