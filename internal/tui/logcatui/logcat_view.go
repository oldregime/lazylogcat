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
	"github.com/parfenovvs/lazylogcat/internal/tui"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

const maxLogLines = 1000

var (
	titleStyle = func() lipgloss.Style {
		return theme.Panel().
			Padding(0, 1)
	}()

	helpTextNormal = "ctrl+f filters • ctrl+r reconnect • ctrl+d devices • alt+w toggle wrap • alt+l toggle level • G jump to recent • v visual"
	helpTextVisual = "j/↓ down • k/↑ up • V select multiple • y copy • esc exit visual"
)

type LogcatViewModel struct {
	parentSize    model.Size
	viewport      viewport.Model
	device        model.Device
	filter        model.Filter
	format        model.Format
	log           []message
	visualMode    bool
	currentLine   int
	startSelected int
	softWrap      bool // TODO wrap with prefs
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

func New(parentSize model.Size, device model.Device, filter model.Filter, format model.Format) LogcatViewModel {
	m := LogcatViewModel{
		parentSize:    parentSize,
		device:        device,
		softWrap:      true,
		startSelected: -1,
		filter:        filter,
		format:        format,
	}

	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	vp := viewport.New(parentSize.Width, parentSize.Height-footerHeight-headerHeight-1)
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
		m.parentSize = msg
		return m, func() tea.Msg {
			return tui.MeasureCmd{}
		}
	case tui.MeasureCmd:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		m.viewport.Width = m.parentSize.Width
		m.viewport.Height = m.parentSize.Height - footerHeight - headerHeight - 1
		m.Render()
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r":
			return m, func() tea.Msg {
				return tui.ReconnectLogcatCmd{}
			}

		case "ctrl+d":
			return m, func() tea.Msg {
				return tui.NavigateToDevicesCmd{}
			}

		case "ctrl+f":
			return m, func() tea.Msg {
				return tui.NavigateToFilterCmd{}
			}

		case "alt+w":
			if !m.visualMode {
				m.softWrap = !m.softWrap
				return m, func() tea.Msg {
					return tui.ReconnectLogcatCmd{}
				}
			}
			return m, nil

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "alt+l":
			if !m.visualMode {
				m.filter.Level = m.filter.Level.Next()
				return m, func() tea.Msg {
					return tui.ReconnectLogcatCmd{}
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
			return m, func() tea.Msg {
				return tui.ReconnectLogcatCmd{}
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
				return m, func() tea.Msg {
					return tui.ReconnectLogcatCmd{}
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

	case tui.ReconnectLogcatCmd:
		Close(&m)
		return m, tea.Batch(func() tea.Msg {
			return ConnectToLogcat(m)
		}, func() tea.Msg {
			return tui.MeasureCmd{}
		})

	case logcatConnectedMsg:
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
		if m.filter.PackageName != "" {
			filters = append(filters, fmt.Sprintf("pkg:%s", m.filter.PackageName))
		}
		if m.filter.Level != "" && m.filter.Level != model.LvlV {
			filters = append(filters, fmt.Sprintf("level:%s", m.filter.Level))
		}
		if m.filter.Tag != "" {
			filters = append(filters, fmt.Sprintf("tag:%s", m.filter.Tag))
		}
		if m.filter.Text != "" {
			filters = append(filters, fmt.Sprintf("text:%s", m.filter.Text))
		}
	}

	format := m.format.Value()
	var mods []string
	var modsStr string

	if m.format.Color {
		mods = append(mods, "color")
	}
	if m.format.Descriptive {
		mods = append(mods, "descriptive")
	}
	if m.format.Epoch {
		mods = append(mods, "epoch")
	}
	if m.format.Monotonic {
		mods = append(mods, "monotonic")
	}
	if m.format.Printable {
		mods = append(mods, "printable")
	}
	if m.format.Uid {
		mods = append(mods, "uid")
	}
	if m.format.Usec {
		mods = append(mods, "usec")
	}
	if m.format.UTC {
		mods = append(mods, "UTC")
	}
	if m.format.Year {
		mods = append(mods, "year")
	}
	if m.format.Zone {
		mods = append(mods, "zone")
	}

	if len(mods) > 0 {
		modsStr = fmt.Sprintf(" | %s", strings.Join(mods, ","))
	}

	filtersStr := ""
	if len(filters) > 0 {
		filtersStr = fmt.Sprintf("\n%s", strings.Join(filters, " | "))
	}

	deviceName := lipgloss.NewStyle().Bold(true).Render(m.device.Name)
	headerText := titleStyle.Render(fmt.Sprintf("%s | %s%s%s", deviceName, format, modsStr, filtersStr))
	width := lipgloss.Width(headerText)

	if width > m.viewport.Width {
		headerText = titleStyle.Render(fmt.Sprintf("%s | ...", deviceName))
	}

	return headerText
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
		Width(m.viewport.Width).
		AlignHorizontal(lipgloss.Center).
		Padding(1, 2, 0, 2).
		Render(helpText)

	return help
}
