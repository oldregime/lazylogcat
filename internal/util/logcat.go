package util

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

// GetLogLevel extracts the log level from a logcat line.
//
// Formats:
//
//	brief, long, tag, time: "...I/..."
//	process, thread: "I(..."
//	raw: no log level
//	threadtime: "<date> <time> <pid> <tid> I <tag>..."
func GetLogLevel(line string, format model.Format) string {
	i := -1
	switch {
	case format.Brief, format.Long, format.Tag, format.Time:
		i = strings.Index(line, "/")
		if i <= 0 {
			return ""
		}
		return line[i-1 : i]
	case format.Process, format.Thread:
		i = strings.Index(line, "(")

		if i <= 0 {
			return ""
		}
		return line[i-1 : i]
	case format.Threadtime:
		parts := strings.Fields(line)
		if len(parts) < 5 {
			return ""
		}
		l := parts[4]
		if len(l) != 1 {
			return ""
		}
		return parts[4]
	case format.Time:
		parts := strings.Fields(line)
		if len(parts) < 3 {
			return ""
		}
		l := strings.Split(parts[2], "/")
		if len(l) == 0 || len(l[0]) != 1 {
			return ""
		}
		return l[0]
	}
	return ""
}

func GetPidByPackageName(deviceId string, pkg string) (string, error) {
	pidCmd := exec.Command("adb", "-s", deviceId, "shell", "pidof", pkg)
	pid, err := pidCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pid: %w", err)
	}
	return strings.Trim(string(pid), "\n\r "), nil
}
