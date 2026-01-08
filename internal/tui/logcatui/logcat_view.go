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

	helpTextNormal = "ctrl+f filters • ctrl+r reconnect • ctrl+d devices • alt+w toggle wrap • alt+l toggle level • G jump to recent • v visual"
	helpTextVisual = "j/↓ down • k/↑ up • V select multiple • y copy • esc exit visual"
)

type LogcatViewModel struct {
	viewport      viewport.Model
	cmd           *exec.Cmd
	scanner       *bufio.Scanner
	device        model.Device
	filter        model.Filter
	format        model.Format
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

type GoToDevicesMsg struct {
	Selected *model.Device
}

type GoToFilterMsg struct {
	Device model.Device
	Filter model.Filter
	Format model.Format
}

type UpdateFiltersMsg struct {
	Filter model.Filter
	Format model.Format
}

func New(viewportSize model.Size, device model.Device) LogcatViewModel {
	m := LogcatViewModel{
		device:        device,
		softWrap:      true,
		startSelected: -1,
		filter: model.Filter{
			Level: model.LvlV,
		},
		format: model.Format{
			Brief: true,
			Color: true,
		},
	}

	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	vp := viewport.New(viewportSize.Width, viewportSize.Height-footerHeight-headerHeight)
	m.viewport = vp

	return m
}

func (m LogcatViewModel) Update(msg tea.Msg) (LogcatViewModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case model.Size:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - footerHeight - headerHeight
		m.Render()
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r":
			Close(&m)
			return m, func() tea.Msg {
				return ConnectToLogcat(m)
			}

		case "ctrl+d":
			return m, func() tea.Msg {
				return GoToDevicesMsg{
					Selected: &m.device,
				}
			}

		case "ctrl+f":
			return m, func() tea.Msg {
				return GoToFilterMsg{
					Device: m.device,
					Filter: m.filter,
					Format: m.format,
				}
			}

		case "alt+w":
			if !m.visualMode {
				m.softWrap = !m.softWrap
				return m, func() tea.Msg {
					return ConnectToLogcat(m)
				}
			}
			return m, nil

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "alt+l":
			if !m.visualMode {
				m.filter.Level = m.filter.Level.Next()
				Close(&m)
				return m, func() tea.Msg {
					return ConnectToLogcat(m)
				}
			}

		case "v":
			m.visualMode = !m.visualMode
			if m.visualMode {
				m.viewport.GotoBottom()
				m.currentLine = len(m.log) - 1
				m.Render()
				return m, nil
			}
			Close(&m)
			return m, func() tea.Msg {
				return ConnectToLogcat(m)
			}

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
				if m.startSelected >= 0 {
					m.startSelected = -1
					m.Render()
					return m, nil
				}
				m.visualMode = false
				m.startSelected = -1
				Close(&m)
				return m, func() tea.Msg {
					return ConnectToLogcat(m)
				}
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
		return m, func() tea.Msg {
			return WaitForNextLine(m)
		}

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

		return m, func() tea.Msg {
			return WaitForNextLine(m)
		}

	case logcatErrorMsg:
		m.err = msg.Err
		return m, nil

	case UpdateFiltersMsg:
		m.filter = msg.Filter
		m.format = msg.Format
		Close(&m)
		return m, func() tea.Msg {
			return ConnectToLogcat(m)
		}
	}

	if !m.visualMode {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *LogcatViewModel) Render() {
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
			line := strings.TrimSuffix(msg.text, "\n")
			if selected {
				styled := lipgloss.NewStyle().
					Bold(true).
					Background(theme.BGCursor).
					Foreground(theme.FGSelected).
					Width(m.viewport.Width).
					Render(line)
				b.WriteString(styled)
				b.WriteString("\n")
				continue
			}
		}
		if m.format.Color {
			line := strings.TrimSuffix(msg.text, "\n")
			styled := lipgloss.NewStyle().
				Foreground(theme.GetLogColor(util.GetLogLevel(line))).
				Width(m.viewport.Width).
				Render(line)
			b.WriteString(styled)
		} else {
			b.WriteString(msg.text)
		}
	}
	wrapped := b.String()
	if m.softWrap {
		wrapped = lipgloss.NewStyle().Width(m.viewport.Width).Render(wrapped)
	}
	m.viewport.SetContent(wrapped)
}

func (m *LogcatViewModel) ensureLineVisible() {
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

func (m LogcatViewModel) View() string {
	if m.err != nil {
		slog.Error("Logcat view error", "error", m.err)
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m LogcatViewModel) headerView() string {
	var filters []string
	if !m.filter.IsEmpty() {
		filters = append(filters, " | Filters:")
		if m.filter.PackageName != "" {
			filters = append(filters, fmt.Sprintf("[pkg:%s]", m.filter.PackageName))
		}
		if m.filter.Level != "" && m.filter.Level != model.LvlV {
			filters = append(filters, fmt.Sprintf("[level:%s]", m.filter.Level))
		}
		if m.filter.Tag != "" {
			filters = append(filters, fmt.Sprintf("[tag:%s]", m.filter.Tag))
		}
		if m.filter.Text != "" {
			filters = append(filters, fmt.Sprintf("[text:%s]", m.filter.Text))
		}
	}

	var formats []string
	var formatsStr string

	// Add single-choice format (only one should be true)
	if m.format.Brief {
		formats = append(formats, "brief")
	} else if m.format.Long {
		formats = append(formats, "long")
	} else if m.format.Process {
		formats = append(formats, "process")
	} else if m.format.Raw {
		formats = append(formats, "raw")
	} else if m.format.Tag {
		formats = append(formats, "tag")
	} else if m.format.Thread {
		formats = append(formats, "thread")
	} else if m.format.Threadtime {
		formats = append(formats, "threadtime")
	} else if m.format.Time {
		formats = append(formats, "time")
	}

	// Add multi-choice modifiers
	if m.format.Color {
		formats = append(formats, "color")
	}
	if m.format.Descriptive {
		formats = append(formats, "descriptive")
	}
	if m.format.Epoch {
		formats = append(formats, "epoch")
	}
	if m.format.Monotonic {
		formats = append(formats, "monotonic")
	}
	if m.format.Printable {
		formats = append(formats, "printable")
	}
	if m.format.Uid {
		formats = append(formats, "uid")
	}
	if m.format.Usec {
		formats = append(formats, "usec")
	}
	if m.format.UTC {
		formats = append(formats, "UTC")
	}
	if m.format.Year {
		formats = append(formats, "year")
	}
	if m.format.Zone {
		formats = append(formats, "zone")
	}

	if len(formats) > 0 {
		formatsStr = fmt.Sprintf(" | Formats: %s", strings.Join(formats, ","))
	}

	title := titleStyle.Render(fmt.Sprintf("Device: %s%s%s", m.device.Name, strings.Join(filters, ""), formatsStr))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m LogcatViewModel) footerView() string {
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
