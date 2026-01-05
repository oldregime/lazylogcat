package logcatui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

type filter struct {
	packageName string
	level       priority
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
	filter       filter // Current active filter state
	format       format // Current active format state

	// UI state (only used when in filter management mode)
	isEditing      bool   // true when in filter management UI
	activePanel    int    // 0=format, 1=modifier
	formatCursor   int    // 0-7
	modifierCursor int    // 0-9
	tempFilter     filter // Working copy during editing
	tempFormat     format // Working copy during editing
}

func NewFilterManagementModel(viewportSize model.Size) FilterManagementModel {
	return FilterManagementModel{
		viewportSize: viewportSize,
		filter: filter{
			level: priorityVerbose, // Default
		},
		format: format{
			threadtime: true,
			color:      true, // Default
		},
		isEditing:      false,
		activePanel:    0,
		formatCursor:   0,
		modifierCursor: 0,
	}
}

func (m *LogcatModel) filterManagementView() string {
	return m.filterMgmt.View()
}

func (f *filter) isEmpty() bool {
	return f.packageName == "" && (f.level == "" || f.level == priorityVerbose)
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
	m.tempFilter = m.filter // Copy current state
	m.tempFormat = m.format // Copy current state
}

func (m *FilterManagementModel) ExitEditMode(apply bool) bool {
	m.isEditing = false
	if apply {
		// Apply temp changes to actual state
		m.filter = m.tempFilter
		m.format = m.tempFormat
		return true // Signal that reconnection is needed
	}
	// Discard temp changes
	return false
}

func (m FilterManagementModel) Update(msg tea.Msg) (FilterManagementModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.Size:
		m.viewportSize = msg
	case tea.KeyMsg:
		if !m.isEditing {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.activePanel = (m.activePanel + 1) % 2
		case "j", "down":
			if m.activePanel == 0 {
				if m.formatCursor < 7 {
					m.formatCursor++
				}
			} else {
				if m.modifierCursor < 9 {
					m.modifierCursor++
				}
			}
		case "k", "up":
			if m.activePanel == 0 {
				if m.formatCursor > 0 {
					m.formatCursor--
				}
			} else {
				if m.modifierCursor > 0 {
					m.modifierCursor--
				}
			}
		case " ", "enter":
			if m.activePanel == 0 {
				clearAllFormats(&m.tempFormat)
				setFormatByIndex(&m.tempFormat, m.formatCursor)
			} else {
				toggleModifierByIndex(&m.tempFormat, m.modifierCursor)
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
		Foreground(lipgloss.Color("86")).
		Render("Format Management")
	b.WriteString(title + "\n\n")

	// Two panels side-by-side
	formatPanel := m.renderFormatPanel()
	modifierPanel := m.renderModifierPanel()
	panels := lipgloss.JoinHorizontal(lipgloss.Top, formatPanel, "  ", modifierPanel)
	b.WriteString(panels + "\n\n")

	// Help text
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("tab switch • ↑/k up • ↓/j down • space/enter select • esc apply")
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

func (m FilterManagementModel) renderFormatPanel() string {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	if m.activePanel == 0 {
		// Active panel - use purple highlight
		panelTitleStyle = panelTitleStyle.Foreground(lipgloss.Color("57"))
	}

	b.WriteString(panelTitleStyle.Render("Format (single choice)") + "\n\n")

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
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(30)

	if m.activePanel == 0 {
		// Active panel - highlight border
		panelStyle = panelStyle.BorderForeground(lipgloss.Color("57"))
	}

	return panelStyle.Render(b.String())
}

func (m FilterManagementModel) renderModifierPanel() string {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	if m.activePanel == 1 {
		panelTitleStyle = panelTitleStyle.Foreground(lipgloss.Color("57"))
	}

	b.WriteString(panelTitleStyle.Render("Modifiers (multiple choice)") + "\n\n")

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
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(35)

	if m.activePanel == 1 {
		panelStyle = panelStyle.BorderForeground(lipgloss.Color("57"))
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
			Background(lipgloss.Color("240")).
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
			Background(lipgloss.Color("240")).
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
