package logcatui

import (
	"bufio"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

const maxLogLines = 1000

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	helpTextNormal = "ctrl+f filters • ctrl+r reconnect • ctrl+d back • alt+w toggle wrap • alt+l toggle level • G jump to recent • v visual"
	helpTextVisual = "j/↓ down • k/↑ up • V select multiple • y copy • esc exit visual"
)

type logcatState int

const (
	logcatViewing logcatState = iota
	filterManagement
)

type LogcatModel struct {
	state         logcatState
	viewportSize  model.Size
	viewport      viewport.Model
	cmd           *exec.Cmd
	scanner       *bufio.Scanner
	device        model.Device
	filterMgmt    FilterManagementModel
	log           []message
	visualMode    bool
	currentLine   int
	startSelected int
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
		state:         logcatViewing,
		viewportSize:  viewportSize,
		device:        device,
		softWrap:      true,
		startSelected: -1,
		filterMgmt:    NewFilterManagementModel(viewportSize, device.Id),
	}

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
		m.filterMgmt.viewportSize = msg
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		m.viewport.Width = m.viewportSize.Width
		m.viewport.Height = m.viewportSize.Height - footerHeight - headerHeight
		m.Render()
		return m, nil

	case filterExitMsg:
		m.state = logcatViewing
		m.Close()
		return m, m.ConnectToLogcat
	}

	if m.state == filterManagement {
		m.filterMgmt, cmd = m.filterMgmt.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {
		case "ctrl+r":
			m.Close()
			return m, m.ConnectToLogcat

		case "ctrl+d":
			return m, func() tea.Msg {
				return BackMsg{}
			}

		case "ctrl+f":
			m.state = filterManagement
			m.filterMgmt.EnterEditMode()
			return m, nil

		case "alt+w":
			m.softWrap = !m.softWrap

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "alt+l":
			if !m.visualMode {
				m.filterMgmt.filter.level = nextPriority(m.filterMgmt.filter.level)
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

	if !m.visualMode {
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
					Background(theme.BGCursor).
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

	switch m.state {
	case logcatViewing:
		return fmt.Sprintf(
			"%s\n%s\n%s",
			m.headerView(),
			m.viewport.View(),
			m.footerView(),
		)
	case filterManagement:
		return m.filterMgmt.View()
	}

	return ""
}

func (m LogcatModel) headerView() string {
	var filters []string
	if !m.filterMgmt.filter.isEmpty() {
		filters = append(filters, " | Filters:")
		if m.filterMgmt.filter.packageName != "" {
			filters = append(filters, fmt.Sprintf("[pkg:%s]", m.filterMgmt.filter.packageName))
		}
		if m.filterMgmt.filter.level != "" && m.filterMgmt.filter.level != priorityVerbose {
			filters = append(filters, fmt.Sprintf("[level:%s]", m.filterMgmt.filter.level))
		}
		if m.filterMgmt.filter.tag != "" {
			filters = append(filters, fmt.Sprintf("[tag:%s]", m.filterMgmt.filter.tag))
		}
		if m.filterMgmt.filter.text != "" {
			filters = append(filters, fmt.Sprintf("[text:%s]", m.filterMgmt.filter.text))
		}
	}

	var formats []string
	var formatsStr string

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
	if m.filterMgmt.format.color {
		formats = append(formats, "color")
	}
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
		formatsStr = fmt.Sprintf(" | Formats: %s", strings.Join(formats, ","))
	}

	title := titleStyle.Render(fmt.Sprintf("Device: %s%s%s", m.device.Name, strings.Join(filters, ""), formatsStr))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m LogcatModel) footerView() string {
	var helpText string
	if m.visualMode {
		helpText = helpTextVisual
	} else {
		helpText = helpTextNormal
	}

	help := lipgloss.NewStyle().
		Foreground(theme.FGHelp).
		Render(helpText)

	return help
}
