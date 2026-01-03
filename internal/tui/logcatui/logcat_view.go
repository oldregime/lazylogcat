package logcatui

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

type LogcatModel struct {
	cmd     *exec.Cmd
	scanner *bufio.Scanner
	log     []message
	Device  model.Device
	err     error
}

type message struct {
	text   string
	source messageSource
}

type messageSource string

const (
	logcatMessage messageSource = "logcat"
	systemMessage messageSource = "system"
)

type logcatLineMsg struct {
	Line string
}

type logcatErrorMsg struct {
	Err error
}

type BackMsg struct{}

func New(device model.Device) LogcatModel {
	return LogcatModel{
		Device: device,
	}
}

func (m LogcatModel) Update(msg tea.Msg) (LogcatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "alt+d":
			return m, func() tea.Msg {
				return BackMsg{}
			}
		}
	case logcatLineMsg:
		m.log = append(m.log, message{
			text:   msg.Line + "\n",
			source: logcatMessage,
		})
		return m, m.WaitForNextLine

	case logcatErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

func (m LogcatModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	for _, message := range m.log {
		b.WriteString(message.text)
	}

	return b.String()
}

func (m LogcatModel) ConnectToLogcat(device string) tea.Msg {
	cmd := exec.Command("adb", "-s", device, "logcat")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to get stdout pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return logcatErrorMsg{Err: fmt.Errorf("failed to start adb: %w", err)}
	}

	m.scanner = bufio.NewScanner(stdout)

	return m.WaitForNextLine()
}

func (m LogcatModel) WaitForNextLine() tea.Msg {
	if m.scanner == nil {
		return logcatErrorMsg{Err: fmt.Errorf("scanner not initialized")}
	}

	if m.scanner.Scan() {
		return logcatLineMsg{Line: m.scanner.Text()}
	}

	if err := m.scanner.Err(); err != nil {
		return logcatErrorMsg{Err: err}
	}

	return nil
}

func (m LogcatModel) Close() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
}
