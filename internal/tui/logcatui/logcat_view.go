package logcatui

import (
	"bufio"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

const maxLogLines = 1000

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type LogcatModel struct {
	viewportSize  model.Size
	viewport      viewport.Model
	cmd           *exec.Cmd
	scanner       *bufio.Scanner
	device        model.Device
	filter        filter
	format        format
	log           []message
	visualMode    bool
	currentLine   int
	startSelected int
	pkgInputMode  bool
	packageInput  textinput.Model
	softWrap      bool
	err           error
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

type filter struct {
	packageName   string
	tagPriorities map[string]priority
}

type format struct {
	color bool
	tag   bool
}

type priority string

const (
	priorityVerbose priority = "V"
	priorityDebug   priority = "D"
	priorityInfo    priority = "I"
	priorityWarn    priority = "W"
	priorityError   priority = "E"
	priorityFatal   priority = "F"
)

const priorities = "VDIWEF"

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
	m := LogcatModel{
		viewportSize:  viewportSize,
		device:        device,
		softWrap:      true,
		startSelected: -1,
		format: format{
			color: true,
		},
		filter: filter{
			tagPriorities: map[string]priority{
				"*": priorityVerbose,
			},
		},
	}

	// Initialize text input for package filtering
	ti := textinput.New()
	ti.Placeholder = "Enter package name..."
	ti.Prompt = "Package: "
	ti.CharLimit = 100
	ti.Width = viewportSize.Width - 20
	m.packageInput = ti

	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	vp := viewport.New(viewportSize.Width, viewportSize.Height-footerHeight-headerHeight)
	m.viewport = vp

	return m
}

func (m LogcatModel) Update(msg tea.Msg) (LogcatModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case model.Size:
		m.viewportSize = msg
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		m.viewport.Width = m.viewportSize.Width
		m.viewport.Height = m.viewportSize.Height - footerHeight - headerHeight
		m.Render()
		return m, nil

	case tea.KeyMsg:
		if m.pkgInputMode {
			switch msg.String() {
			case "enter":
				m.pkgInputMode = false
				m.packageInput.Blur()
				packageName := strings.TrimSpace(m.packageInput.Value())
				if packageName != m.filter.packageName {
					m.filter.packageName = packageName
					m.Close()
					return m, m.ConnectToLogcat
				}
				m.packageInput.SetValue(m.filter.packageName)
				return m, nil

			case "esc":
				m.pkgInputMode = false
				m.packageInput.Blur()
				m.packageInput.SetValue(m.filter.packageName)
				return m, nil

			default:
				var cmd tea.Cmd
				m.packageInput, cmd = m.packageInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+r":
			m.Close()
			return m, m.ConnectToLogcat

		case "ctrl+d":
			return m, func() tea.Msg {
				return BackMsg{}
			}

		case "alt+p":
			m.pkgInputMode = true
			m.packageInput.SetValue(m.filter.packageName)
			m.packageInput.Focus()
			return m, textinput.Blink

		case "alt+w":
			m.softWrap = !m.softWrap

		case "alt+c":
			m.format.color = !m.format.color
			m.Close()
			return m, m.ConnectToLogcat

		case "alt+t":
			m.format.tag = !m.format.tag
			m.Close()
			return m, m.ConnectToLogcat

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "alt+l":
			if !m.visualMode && !m.pkgInputMode {
				m.filter.tagPriorities = nextTagPriority(m.filter.tagPriorities)
				m.Close()
				return m, m.ConnectToLogcat
			}

		case "v":
			m.visualMode = !m.visualMode
			if m.visualMode {
				m.viewport.GotoBottom()
				m.currentLine = len(m.log) - 1
				m.Render()
				return m, nil
			}
			m.Close()
			return m, m.ConnectToLogcat

		case "V":
			if m.visualMode {
				if m.startSelected >= 0 {
					m.startSelected = -1
				} else {
					m.startSelected = m.currentLine
				}
				m.Render()
				return m, nil
			}

		case "esc":
			if m.visualMode {
				m.visualMode = false
				m.startSelected = -1
				m.Close()
				return m, m.ConnectToLogcat
			}

		case "y":
			if m.visualMode && m.currentLine >= 0 && m.currentLine < len(m.log) {
				var err error
				if m.startSelected >= 0 {
					min := min(m.currentLine, m.startSelected)
					max := max(m.currentLine, m.startSelected)
					var lines []string
					for i := min; i <= max; i++ {
						lines = append(lines, strings.TrimSpace(m.log[i].text))
					}
					err = util.CopyToClipboard(lines...)
					m.startSelected = -1
					m.Render()
				} else {
					lineText := strings.TrimSpace(m.log[m.currentLine].text)
					err = util.CopyToClipboard(lineText)
				}
				if err != nil {
					slog.Error("Failed to copy to clipboard", "error", err)
				}
			}
			return m, nil

		case "j", "down":
			if m.visualMode && m.currentLine < len(m.log)-1 {
				m.currentLine++
				m.Render()
				m.ensureLineVisible()
			}

		case "k", "up":
			if m.visualMode && m.currentLine > 0 {
				m.currentLine--
				m.Render()
				m.ensureLineVisible()
			}
		}

	case logcatConnectedMsg:
		m.cmd = msg.cmd
		m.scanner = msg.scanner
		cmds = append(cmds, m.WaitForNextLine)

	case logcatLineMsg:
		if m.visualMode {
			return m, nil
		}

		wasAtBottom := m.viewport.AtBottom()

		if len(m.log) >= maxLogLines {
			m.log = m.log[1:]
		}
		m.log = append(m.log, message{
			text:   msg.Line + "\n",
			source: logcatMessage,
		})

		m.Render()

		if wasAtBottom {
			m.viewport.GotoBottom()
		}

		cmds = append(cmds, m.WaitForNextLine)

	case logcatErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	if !m.pkgInputMode && !m.visualMode {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *LogcatModel) Render() {
	var b strings.Builder
	for i, msg := range m.log {
		if m.visualMode {
			selected := false
			if m.startSelected >= 0 {
				min := min(m.currentLine, m.startSelected)
				max := max(m.currentLine, m.startSelected)
				selected = i >= min && i <= max
			} else if i == m.currentLine {
				selected = true
			}
			if selected {
				line := strings.TrimSuffix(msg.text, "\n")
				styled := lipgloss.NewStyle().
					Background(lipgloss.Color("240")).
					Width(m.viewport.Width).
					Render(line)
				b.WriteString(styled)
				b.WriteString("\n")
				continue
			}
		}
		b.WriteString(msg.text)
	}
	wrapped := b.String()
	if m.softWrap {
		wrapped = lipgloss.NewStyle().Width(m.viewport.Width).Render(wrapped)
	}
	m.viewport.SetContent(wrapped)
}

func (m *LogcatModel) ensureLineVisible() {
	if !m.visualMode || m.currentLine < 0 || m.currentLine >= len(m.log) {
		return
	}

	min := min(m.currentLine, m.startSelected)
	max := max(m.currentLine, m.startSelected)

	linesUpToCurrent := 0
	for i := 0; i <= m.currentLine; i++ {
		line := strings.TrimSuffix(m.log[i].text, "\n")
		if m.softWrap || (m.startSelected != -1 && i >= min && i <= max) {
			linesUpToCurrent += lipgloss.Height(lipgloss.NewStyle().Width(m.viewport.Width).Render(line))
		} else {
			linesUpToCurrent++
		}
	}

	if linesUpToCurrent < m.viewport.YOffset+2 {
		m.viewport.HalfPageUp()
	} else if linesUpToCurrent > m.viewport.YOffset+m.viewport.Height-1 {
		m.viewport.HalfPageDown()
	}
}

func (m LogcatModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m LogcatModel) headerView() string {
	var filters []string
	if !m.filter.isEmpty() {
		filters = append(filters, " | Filters:")
		if m.filter.packageName != "" {
			filters = append(filters, fmt.Sprintf("[pkg: %s]", m.filter.packageName))
		}
		if len(m.filter.tagPriorities) > 0 && m.filter.tagPriorities["*"] != priorityVerbose {
			filters = append(filters, fmt.Sprintf("[%s]", m.filter.tagPriorities["*"]))
		}
	}

	var formats []string
	if m.format != (format{}) {
		formats = append(formats, " | Formats:")
		if m.format.color {
			formats = append(formats, "[color]")
		}
		if m.format.tag {
			formats = append(formats, "[tag]")
		}
	}

	title := titleStyle.Render(fmt.Sprintf("Device: %s%s%s", m.device.Name, strings.Join(filters, " "), strings.Join(formats, " ")))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m LogcatModel) footerView() string {
	if m.pkgInputMode {
		return m.packageInput.View()
	}

	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m LogcatModel) ConnectToLogcat() tea.Msg {
	args := []string{"-s", m.device.Id, "logcat", "-T", "60"}

	if m.filter.packageName != "" {
		pidCmd := exec.Command("adb", "-s", m.device.Id, "shell", "pidof", m.filter.packageName)
		pid, err := pidCmd.Output()
		if err != nil {
			return logcatErrorMsg{Err: fmt.Errorf("failed to get pid: %w", err)}
		}
		pidStr := strings.Trim(string(pid), "\n\r ")
		if len(pidStr) > 0 {
			args = append(args, fmt.Sprintf("--pid=%s", pidStr))
		}
	}

	if m.format != (format{}) {
		args = append(args, "-v")
		var formatArgs []string
		if m.format.color {
			formatArgs = append(formatArgs, "color")
		}
		if m.format.tag {
			formatArgs = append(formatArgs, "tag")
		}
		args = append(args, strings.Join(formatArgs, ","))
	}

	args = append(args, fmt.Sprintf("*:%s", m.filter.tagPriorities["*"]))

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

func (f *filter) isEmpty() bool {
	return f.packageName == "" && (len(f.tagPriorities) == 0 || f.tagPriorities["*"] == priorityVerbose)
}

func nextTagPriority(tp map[string]priority) map[string]priority {
	if tp == nil {
		tp = make(map[string]priority)
	}

	if len(tp) == 0 {
		tp["*"] = priorityDebug
		return tp
	}

	if tp["*"] == priorityFatal {
		tp["*"] = priorityVerbose
		return tp
	}

	i := strings.Index(priorities, string(tp["*"]))
	tp["*"] = priority(priorities[i+1])
	return tp
}
