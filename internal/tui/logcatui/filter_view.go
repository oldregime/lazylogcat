package logcatui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

type filterExitMsg struct {
	changed bool
}

type filter struct {
	packageName string
	level       priority
	tag         string
	text        string
}

type format struct {
	//Single choice
	brief      bool
	long       bool
	process    bool
	raw        bool
	tag        bool
	thread     bool
	threadtime bool
	time       bool

	//Multiple choice (conflicts are silently ignored)
	color       bool //Shows each priority level with a different color.
	descriptive bool //Shows log buffer event descriptions. This modifier affects event log buffer messages only and has no effect on the other non-binary buffers. The event descriptions come from the event-log-tags database.
	epoch       bool //Displays time in seconds starting from Jan 1, 1970.
	monotonic   bool //Displays time in CPU seconds starting from the last boot.
	printable   bool //Ensures that any binary logging content is escaped.
	uid         bool //If permitted by access controls, displays the UID or Android ID of the logged process.
	usec        bool //Displays the time, with precision in microseconds.
	UTC         bool //Displays the time as UTC.
	year        bool //Adds the year to the displayed time.
	zone        bool //Adds the local time zone to the displayed time.
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

type FilterManagementModel struct {
	viewportSize model.Size
	deviceId     string // Current device ID
	filter       filter // Current active filter state
	format       format // Current active format state

	// UI state (only used when in filter management mode)
	isEditing      bool            // true when in filter management UI
	activePanel    int             // 0=format, 1=modifier, 2=package
	formatCursor   int             // 0-7
	modifierCursor int             // 0-9
	packageInput   textinput.Model // Package filter input
	tagInput       textinput.Model // Tag filter input
	textInput      textinput.Model // Text filter input
	tempFilter     filter          // Working copy during editing
	tempFormat     format          // Working copy during editing
	validationErr  string          // Validation error message
}

func NewFilterManagementModel(viewportSize model.Size, deviceId string) FilterManagementModel {
	pi := textinput.New()
	pi.Placeholder = "Enter package name..."
	pi.CharLimit = 100
	pi.Width = viewportSize.Width - 20

	ti := textinput.New()
	ti.Placeholder = "Enter tag value..."
	ti.CharLimit = 100
	ti.Width = viewportSize.Width - 20

	txtInput := textinput.New()
	txtInput.Placeholder = "Enter text to search..."
	txtInput.CharLimit = 100
	txtInput.Width = viewportSize.Width - 20

	return FilterManagementModel{
		viewportSize: viewportSize,
		deviceId:     deviceId,
		filter: filter{
			level: priorityVerbose,
		},
		format: format{
			brief: true,
			color: true,
		},
		packageInput:   pi,
		tagInput:       ti,
		textInput:      txtInput,
		isEditing:      false,
		activePanel:    0,
		formatCursor:   0,
		modifierCursor: 0,
	}
}

func (f *filter) isEmpty() bool {
	return f.packageName == "" &&
		(f.level == "" || f.level == priorityVerbose) &&
		f.tag == "" &&
		f.text == ""
}

func nextPriority(p priority) priority {
	if p == "" || p == priorityFatal {
		return priorityVerbose
	}

	i := strings.Index(priorities, string(p))
	if i == -1 {
		return priorityVerbose
	}
	return priority(priorities[i+1])
}

func (m *FilterManagementModel) EnterEditMode() {
	m.isEditing = true
	m.activePanel = 0
	m.formatCursor = getCurrentFormatIndex(m.format)
	m.modifierCursor = 0
	m.tempFilter = m.filter
	m.tempFormat = m.format

	m.packageInput.SetValue(m.filter.packageName)
	m.packageInput.Blur()

	m.tagInput.SetValue(m.filter.tag)
	m.tagInput.Blur()

	m.textInput.SetValue(m.filter.text)
	m.textInput.Blur()
}

func (m *FilterManagementModel) ExitEditMode(apply bool) (bool, error) {
	if apply {
		newPackage := strings.TrimSpace(m.packageInput.Value())

		if newPackage != m.filter.packageName {
			if newPackage != "" {
				_, err := getPidByPackageName(m.deviceId, newPackage)
				if err != nil {
					m.validationErr = "Package not found."
					m.isEditing = true
					return false, err
				}
			}

			m.tempFilter.packageName = newPackage
		}

		m.tempFilter.tag = strings.TrimSpace(m.tagInput.Value())
		m.tempFilter.text = strings.TrimSpace(m.textInput.Value())

		filterChanged := m.filter.packageName != m.tempFilter.packageName ||
			m.filter.level != m.tempFilter.level ||
			m.filter.tag != m.tempFilter.tag ||
			m.filter.text != m.tempFilter.text
		formatChanged := m.format != m.tempFormat

		m.filter = m.tempFilter
		m.format = m.tempFormat

		m.isEditing = false
		if m.packageInput.Focused() {
			m.packageInput.Blur()
		}
		if m.tagInput.Focused() {
			m.tagInput.Blur()
		}
		if m.textInput.Focused() {
			m.textInput.Blur()
		}

		return filterChanged || formatChanged, nil
	}

	m.validationErr = ""
	m.packageInput.SetValue(m.filter.packageName)
	m.tagInput.SetValue(m.filter.tag)
	m.textInput.SetValue(m.filter.text)
	m.isEditing = false
	if m.packageInput.Focused() {
		m.packageInput.Blur()
	}
	if m.tagInput.Focused() {
		m.tagInput.Blur()
	}
	if m.textInput.Focused() {
		m.textInput.Blur()
	}
	return false, nil
}

func (m FilterManagementModel) Update(msg tea.Msg) (FilterManagementModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.Size:
		m.viewportSize = msg
		m.packageInput.Width = msg.Width - 20
		m.tagInput.Width = msg.Width - 20
		m.textInput.Width = msg.Width - 20
	case tea.KeyMsg:
		if !m.isEditing {
			return m, nil
		}

		if m.activePanel == 0 || m.activePanel == 1 {
			switch msg.String() {
			case "j", "down":
				switch m.activePanel {
				case 0:
					if m.formatCursor < 7 {
						m.formatCursor++
					}
				case 1:
					if m.modifierCursor < 9 {
						m.modifierCursor++
					}
				}
			case "k", "up":
				switch m.activePanel {
				case 0:
					if m.formatCursor > 0 {
						m.formatCursor--
					}
				case 1:
					if m.modifierCursor > 0 {
						m.modifierCursor--
					}
				}
			case " ", "enter":
				switch m.activePanel {
				case 0:
					clearAllFormats(&m.tempFormat)
					setFormatByIndex(&m.tempFormat, m.formatCursor)
				case 1:
					toggleModifierByIndex(&m.tempFormat, m.modifierCursor)
				}
			}
		}

		switch msg.String() {
		case "ctrl+s":
			changed, err := m.ExitEditMode(true)
			if err != nil {
				slog.Error("error exiting filter edit mode", "err", err)
				return m, nil
			}
			return m, func() tea.Msg {
				return filterExitMsg{changed: changed}
			}

		case "esc":
			m.ExitEditMode(false)
			return m, func() tea.Msg {
				return filterExitMsg{changed: false}
			}
		case "tab", "shift+tab":
			switch m.activePanel {
			case 2:
				m.packageInput.Blur()
			case 3:
				m.tagInput.Blur()
			case 4:
				m.textInput.Blur()
			}
			if msg.String() == "shift+tab" {
				m.activePanel = (m.activePanel - 1 + 5) % 5
			} else {
				m.activePanel = (m.activePanel + 1) % 5
			}
			if m.activePanel == 2 {
				m.packageInput.Focus()
				return m, textinput.Blink
			}
			if m.activePanel == 3 {
				m.tagInput.Focus()
				return m, textinput.Blink
			}
			if m.activePanel == 4 {
				m.textInput.Focus()
				return m, textinput.Blink
			}
		default:
			if m.activePanel == 2 {
				var cmd tea.Cmd
				m.packageInput, cmd = m.packageInput.Update(msg)
				if m.validationErr != "" {
					m.validationErr = ""
				}
				return m, cmd
			}
			if m.activePanel == 3 {
				var cmd tea.Cmd
				m.tagInput, cmd = m.tagInput.Update(msg)
				return m, cmd
			}
			if m.activePanel == 4 {
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m FilterManagementModel) View() string {
	if !m.isEditing {
		return "" // Not in editing mode
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle).
		Render("Format Management")
	b.WriteString(title + "\n\n")

	// Two panels side-by-side
	modifierPanel, modifierPanelHeight := m.renderModifierPanel()
	formatPanel := m.renderFormatPanel(modifierPanelHeight)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, formatPanel, "  ", modifierPanel)
	b.WriteString(panels + "\n\n")

	// Package panel below (full width)
	packagePanel := m.renderPackagePanel()
	b.WriteString(packagePanel + "\n\n")

	// Tag panel below (full width)
	tagPanel := m.renderTagPanel()
	b.WriteString(tagPanel + "\n\n")

	// Text panel below (full width)
	textPanel := m.renderTextPanel()
	b.WriteString(textPanel + "\n\n")

	// Help text
	help := lipgloss.NewStyle().
		Foreground(theme.FGHelp).
		AlignHorizontal(lipgloss.Center).
		Render("tab switch panels • ↑/k up • ↓/j down • space/enter select\nesc back • ctrl+s apply")
	b.WriteString(help)

	// Center everything
	return lipgloss.Place(
		m.viewportSize.Width,
		m.viewportSize.Height,
		lipgloss.Center,
		lipgloss.Center,
		b.String(),
	)
}

func (m FilterManagementModel) renderFormatPanel(height int) string {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle)

	if m.activePanel == 0 {
		// Active panel - use purple highlight
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Format") + "\n\n")

	// Format options
	formats := []struct {
		name  string
		field string
	}{
		{"brief", "brief"},
		{"long", "long"},
		{"process", "process"},
		{"raw", "raw"},
		{"tag", "tag"},
		{"thread", "thread"},
		{"threadtime", "threadtime"},
		{"time", "time"},
	}

	for i, fmt := range formats {
		selected := isFormatSelected(m.tempFormat, fmt.field)
		cursor := m.activePanel == 0 && m.formatCursor == i

		line := m.renderRadioButton(fmt.name, selected, cursor)
		b.WriteString(line + "\n")
	}

	// Create bordered panel
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width/2 - 4).
		Height(height)

	if m.activePanel == 0 {
		// Active panel - highlight border
		panelStyle = panelStyle.BorderForeground(theme.FGActiveBorder)
	}

	return panelStyle.Render(b.String())
}

func (m FilterManagementModel) renderModifierPanel() (string, int) {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle)

	if m.activePanel == 1 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Modifiers") + "\n\n")

	// Modifier options
	modifiers := []struct {
		name  string
		field string
	}{
		{"color", "color"},
		{"descriptive", "descriptive"},
		{"epoch", "epoch"},
		{"monotonic", "monotonic"},
		{"printable", "printable"},
		{"uid", "uid"},
		{"usec", "usec"},
		{"UTC", "UTC"},
		{"year", "year"},
		{"zone", "zone"},
	}

	for i, mod := range modifiers {
		selected := isModifierSelected(m.tempFormat, mod.field)
		cursor := m.activePanel == 1 && m.modifierCursor == i

		line := m.renderCheckbox(mod.name, selected, cursor)
		b.WriteString(line + "\n")
	}

	// Create bordered panel
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width/2 - 4)

	if m.activePanel == 1 {
		panelStyle = panelStyle.BorderForeground(theme.FGActiveBorder)
	}

	result := b.String()
	return panelStyle.Render(result), lipgloss.Height(result) + 2
}

func (m FilterManagementModel) renderPackagePanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle)

	if m.activePanel == 2 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Package Filter") + "\n\n")

	b.WriteString(m.packageInput.View())

	if m.validationErr != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.FGError)
		b.WriteString("\n\n" + errorStyle.Render(m.validationErr))
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 2 {
		panelStyle = panelStyle.BorderForeground(theme.FGActiveBorder)
	}

	return panelStyle.Render(b.String())
}

func (m FilterManagementModel) renderTagPanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle)

	if m.activePanel == 3 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Tag Filter (exact match)") + "\n\n")

	b.WriteString(m.tagInput.View())

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 3 {
		panelStyle = panelStyle.BorderForeground(theme.FGActiveBorder)
	}

	return panelStyle.Render(b.String())
}

func (m FilterManagementModel) renderTextPanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGTitle)

	if m.activePanel == 4 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Text Filter (substring match)") + "\n\n")

	b.WriteString(m.textInput.View())

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 4 {
		panelStyle = panelStyle.BorderForeground(theme.FGActiveBorder)
	}

	return panelStyle.Render(b.String())
}

func (m FilterManagementModel) renderRadioButton(label string, selected bool, cursor bool) string {
	indicator := "( )"
	if selected {
		indicator = "(●)"
	}

	line := fmt.Sprintf("%s %s", indicator, label)

	if cursor {
		// Highlight current cursor position
		return lipgloss.NewStyle().
			Background(theme.BGCursor).
			Foreground(theme.FGSelected).
			Width(25).
			Render(line)
	}

	return line
}

func (m FilterManagementModel) renderCheckbox(label string, checked bool, cursor bool) string {
	indicator := "[ ]"
	if checked {
		indicator = "[✓]"
	}

	line := fmt.Sprintf("%s %s", indicator, label)

	if cursor {
		return lipgloss.NewStyle().
			Background(theme.BGCursor).
			Foreground(theme.FGSelected).
			Width(30).
			Render(line)
	}

	return line
}

// Helper functions for format manipulation
func getCurrentFormatIndex(f format) int {
	if f.brief {
		return 0
	}
	if f.long {
		return 1
	}
	if f.process {
		return 2
	}
	if f.raw {
		return 3
	}
	if f.tag {
		return 4
	}
	if f.thread {
		return 5
	}
	if f.threadtime {
		return 6
	}
	if f.time {
		return 7
	}
	return 0 // Default to brief
}

func clearAllFormats(f *format) {
	f.brief = false
	f.long = false
	f.process = false
	f.raw = false
	f.tag = false
	f.thread = false
	f.threadtime = false
	f.time = false
}

func setFormatByIndex(f *format, i int) {
	switch i {
	case 0:
		f.brief = true
	case 1:
		f.long = true
	case 2:
		f.process = true
	case 3:
		f.raw = true
	case 4:
		f.tag = true
	case 5:
		f.thread = true
	case 6:
		f.threadtime = true
	case 7:
		f.time = true
	}
}

func isFormatSelected(f format, field string) bool {
	switch field {
	case "brief":
		return f.brief
	case "long":
		return f.long
	case "process":
		return f.process
	case "raw":
		return f.raw
	case "tag":
		return f.tag
	case "thread":
		return f.thread
	case "threadtime":
		return f.threadtime
	case "time":
		return f.time
	}
	return false
}

func toggleModifierByIndex(f *format, i int) {
	switch i {
	case 0:
		f.color = !f.color
	case 1:
		f.descriptive = !f.descriptive
	case 2:
		f.epoch = !f.epoch
	case 3:
		f.monotonic = !f.monotonic
	case 4:
		f.printable = !f.printable
	case 5:
		f.uid = !f.uid
	case 6:
		f.usec = !f.usec
	case 7:
		f.UTC = !f.UTC
	case 8:
		f.year = !f.year
	case 9:
		f.zone = !f.zone
	}
}

func isModifierSelected(f format, field string) bool {
	switch field {
	case "color":
		return f.color
	case "descriptive":
		return f.descriptive
	case "epoch":
		return f.epoch
	case "monotonic":
		return f.monotonic
	case "printable":
		return f.printable
	case "uid":
		return f.uid
	case "usec":
		return f.usec
	case "UTC":
		return f.UTC
	case "year":
		return f.year
	case "zone":
		return f.zone
	}
	return false
}
