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

const initialLogHistorySeconds = 60

var cmd *exec.Cmd
var scanner *bufio.Scanner

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
	t := max(diff, initialLogHistorySeconds)

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

	lvl := m.filter.Level
	if lvl == "" {
		lvl = model.LvlD
	}
	args = append(args, fmt.Sprintf("%s:%s", tag, lvl))

	slog.Debug("Executing adb", "args", args)

	cmd = exec.Command("adb", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to get stdout pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to start adb: %w", err)}
	}

	scanner = bufio.NewScanner(stdout)

	return logcatConnectedMsg{
		cmd:     cmd,
		scanner: scanner,
	}
}

func WaitForNextLine(m LogcatViewModel) tea.Msg {
	if scanner == nil {
		return nil
	}

	if scanner.Scan() {
		s := scanner.Text()
		if strings.Trim(s, "\n\r ") == "" {
			return WaitForNextLine(m)
		}
		f := m.filter.Text
		if f != "" && !strings.Contains(s, f) {
			return WaitForNextLine(m)
		}
		return logcatLineMsg{Line: scanner.Text()}
	}

	if err := scanner.Err(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("error reading logcat: %w", err)}
	}

	return nil
}

func Close(m *LogcatViewModel) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
		cmd = nil
	}
	scanner = nil
	m.log = nil
}
