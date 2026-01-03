package logcatui

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

const maxLogLines = 10000

type LogcatModel struct {
	viewportSize model.Size
	viewport     viewport.Model
	cmd          *exec.Cmd
	scanner      *bufio.Scanner
	log          []message
	Device       model.Device
	err          error
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

type logcatConnectedMsg struct {
	cmd     *exec.Cmd
	scanner *bufio.Scanner
}

type BackMsg struct{}

func New(viewportSize model.Size, device model.Device) LogcatModel {
	vp := viewport.New(viewportSize.Width, viewportSize.Height)
	return LogcatModel{
		viewportSize: viewportSize,
		viewport:     vp,
		Device:       device,
	}
}

func (m LogcatModel) Update(msg tea.Msg) (LogcatModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "alt+d":
			return m, func() tea.Msg {
				return BackMsg{}
			}
		case "b":
			m.viewport.GotoBottom()
			return m, nil
		}

	case logcatConnectedMsg:
		m.cmd = msg.cmd
		m.scanner = msg.scanner
		cmds = append(cmds, m.WaitForNextLine)

	case logcatLineMsg:
		wasAtBottom := m.viewport.AtBottom()

		if len(m.log) >= maxLogLines {
			m.log = m.log[1:]
		}
		m.log = append(m.log, message{
			text:   msg.Line + "\n",
			source: logcatMessage,
		})

		var b strings.Builder
		for _, msg := range m.log {
			b.WriteString(msg.text)
		}
		m.viewport.SetContent(b.String())

		if wasAtBottom {
			m.viewport.GotoBottom()
		}

		cmds = append(cmds, m.WaitForNextLine)

	case logcatErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	// Update viewport to handle scrolling
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m LogcatModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return m.viewport.View()
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

	scanner := bufio.NewScanner(stdout)

	return logcatConnectedMsg{
		cmd:     cmd,
		scanner: scanner,
	}
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

func (m *LogcatModel) Close() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}
}
