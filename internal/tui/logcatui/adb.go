package logcatui

import (
	"bufio"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

const initialLogHistorySeconds = 180

var firstConnectionTime *time.Time
var once sync.Once

func getFirstConnectionTime() *time.Time {
	once.Do(func() {
		t := time.Now()
		firstConnectionTime = &t
	})
	return firstConnectionTime
}

func timeDiffInSeconds(start *time.Time, end *time.Time) int {
	if start == nil || end == nil {
		return -1
	}
	return int(end.Sub(*start).Seconds())
}

func ConnectToLogcat(m LogcatViewModel) tea.Msg {
	now := time.Now()
	diff := timeDiffInSeconds(getFirstConnectionTime(), &now)
	t := initialLogHistorySeconds
	if diff > 180 {
		t = diff
	}

	args := []string{"-s", m.device.Id, "logcat", "-T", strconv.Itoa(t)}

	if m.filter.PackageName != "" {
		pidStr, err := util.GetPidByPackageName(m.device.Id, m.filter.PackageName)
		if err != nil {
			return logcatErrorMsg{Err: fmt.Errorf("failed to get pid by package name: %w", err)}
		}
		if len(pidStr) > 0 {
			args = append(args, fmt.Sprintf("--pid=%s", pidStr))
		}
	}

	if m.format != (model.Format{}) {
		args = append(args, "-v")
		var formats []string

		// Add single-choice format (only one should be true)
		format := m.format.Value()
		if format != "" {
			formats = append(formats, format)
		}

		// Add multi-choice modifiers

		formats = append(formats, m.format.Modifiers()...)

		{ // Color modifier is not provided to logcat. Instead, the program handles coloring itself.
			colorless := make([]string, 0, len(formats))
			for _, f := range formats {
				if f != "color" {
					colorless = append(colorless, f)
				}
			}
			formats = colorless
		}

		if len(formats) > 0 {
			args = append(args, strings.Join(formats, ","))
		}
	}

	tag := "*"
	if m.filter.Tag != "" {
		tag = m.filter.Tag
		args = append(args, "-s")
	}
	args = append(args, fmt.Sprintf("%s:%s", tag, m.filter.Level))

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

func WaitForNextLine(m LogcatViewModel) tea.Msg {
	if m.scanner == nil {
		return nil
	}

	if m.scanner.Scan() {
		s := m.scanner.Text()
		if strings.Trim(s, "\n\r ") == "" {
			return WaitForNextLine(m)
		}
		f := m.filter.Text
		if f != "" && !strings.Contains(s, f) {
			return WaitForNextLine(m)
		}
		return logcatLineMsg{Line: m.scanner.Text()}
	}

	if err := m.scanner.Err(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("error reading logcat: %w", err)}
	}

	return nil
}

func Close(m *LogcatViewModel) {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}
	m.scanner = nil
	m.log = nil
}
