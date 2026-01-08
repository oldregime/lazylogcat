package logcatui

import (
	"bufio"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m LogcatModel) ConnectToLogcat() tea.Msg {
	args := []string{"-s", m.device.Id, "logcat", "-T", "60"}

	if m.filterMgmt.filter.packageName != "" {
		pidStr, err := getPidByPackageName(m.device.Id, m.filterMgmt.filter.packageName)
		if err != nil {
			return logcatErrorMsg{Err: fmt.Errorf("failed to get pid by package name: %w", err)}
		}
		if len(pidStr) > 0 {
			args = append(args, fmt.Sprintf("--pid=%s", pidStr))
		}
	}

	if m.filterMgmt.format != (format{}) {
		args = append(args, "-v")
		var formats []string

		// Add single-choice format (only one should be true)
		if m.filterMgmt.format.brief {
			formats = append(formats, "brief")
		} else if m.filterMgmt.format.long {
			formats = append(formats, "long")
		} else if m.filterMgmt.format.process {
			formats = append(formats, "process")
		} else if m.filterMgmt.format.raw {
			formats = append(formats, "raw")
		} else if m.filterMgmt.format.tag {
			formats = append(formats, "tag")
		} else if m.filterMgmt.format.thread {
			formats = append(formats, "thread")
		} else if m.filterMgmt.format.threadtime {
			formats = append(formats, "threadtime")
		} else if m.filterMgmt.format.time {
			formats = append(formats, "time")
		}

		// Add multi-choice modifiers
		// if m.filterMgmt.format.color {
		// 	formats = append(formats, "color")
		// }
		if m.filterMgmt.format.descriptive {
			formats = append(formats, "descriptive")
		}
		if m.filterMgmt.format.epoch {
			formats = append(formats, "epoch")
		}
		if m.filterMgmt.format.monotonic {
			formats = append(formats, "monotonic")
		}
		if m.filterMgmt.format.printable {
			formats = append(formats, "printable")
		}
		if m.filterMgmt.format.uid {
			formats = append(formats, "uid")
		}
		if m.filterMgmt.format.usec {
			formats = append(formats, "usec")
		}
		if m.filterMgmt.format.UTC {
			formats = append(formats, "UTC")
		}
		if m.filterMgmt.format.year {
			formats = append(formats, "year")
		}
		if m.filterMgmt.format.zone {
			formats = append(formats, "zone")
		}

		if len(formats) > 0 {
			args = append(args, strings.Join(formats, ","))
		}
	}

	tag := "*"
	if m.filterMgmt.filter.tag != "" {
		tag = m.filterMgmt.filter.tag
		args = append(args, "-s")
	}
	args = append(args, fmt.Sprintf("%s:%s", tag, m.filterMgmt.filter.level))

	slog.Debug("Executing adb", "args", args)

	cmd := exec.Command("adb", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to get stdout pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to start adb: %w", err)}
	}

	scanner := bufio.NewScanner(stdout)

	return logcatConnectedMsg{
		cmd:     cmd,
		scanner: scanner,
	}
}

func (m LogcatModel) WaitForNextLine() tea.Msg {
	if m.scanner == nil {
		return nil
	}

	if m.scanner.Scan() {
		s := m.scanner.Text()
		if strings.Trim(s, "\n\r ") == "" {
			return m.WaitForNextLine()
		}
		f := m.filterMgmt.filter.text
		if f != "" && !strings.Contains(s, f) {
			return m.WaitForNextLine()
		}
		return logcatLineMsg{Line: m.scanner.Text()}
	}

	if err := m.scanner.Err(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("error reading logcat: %w", err)}
	}

	return nil
}

func (m *LogcatModel) Close() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}
	m.scanner = nil
	m.log = nil
}

func getPidByPackageName(deviceId string, pkg string) (string, error) {
	pidCmd := exec.Command("adb", "-s", deviceId, "shell", "pidof", pkg)
	pid, err := pidCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pid: %w", err)
	}
	return strings.Trim(string(pid), "\n\r "), nil
}
