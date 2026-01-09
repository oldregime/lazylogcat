package filterui

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

type FilterViewModel struct {
	viewportSize model.Size
	packageInput textinput.Model
	tagInput     textinput.Model
	textInput    textinput.Model

	deviceId   string
	filter     model.Filter
	format     model.Format
	tempFilter model.Filter
	tempFormat model.Format

	activePanel    int
	formatCursor   int
	modifierCursor int
	validationErr  string
}

func New(viewportSize model.Size, deviceId string, filter model.Filter, format model.Format) FilterViewModel {
	pi := textinput.New()
	pi.Placeholder = "Enter package name..."
	pi.CharLimit = 100
	pi.Width = viewportSize.Width - 20
	pi.SetValue(filter.PackageName)
	pi.Blur()

	tagInput := textinput.New()
	tagInput.Placeholder = "Enter tag value..."
	tagInput.CharLimit = 100
	tagInput.Width = viewportSize.Width - 20
	tagInput.SetValue(filter.Tag)
	tagInput.Blur()

	txtInput := textinput.New()
	txtInput.Placeholder = "Enter text to search..."
	txtInput.CharLimit = 100
	txtInput.Width = viewportSize.Width - 20
	txtInput.SetValue(filter.Text)
	txtInput.Blur()

	return FilterViewModel{
		viewportSize:   viewportSize,
		deviceId:       deviceId,
		filter:         filter,
		format:         format,
		tempFilter:     filter,
		tempFormat:     format,
		packageInput:   pi,
		tagInput:       tagInput,
		textInput:      txtInput,
		activePanel:    0,
		formatCursor:   getCurrentFormatIndex(&format),
		modifierCursor: 0,
	}
}

func (m *FilterViewModel) Exit(apply bool) (bool, error) {
	if apply {
		newPackage := strings.TrimSpace(m.packageInput.Value())

		if newPackage != m.filter.PackageName {
			if newPackage != "" {
				_, err := util.GetPidByPackageName(m.deviceId, newPackage)
				if err != nil {
					m.validationErr = "Package not found."
					return false, err
				}
			}

			m.tempFilter.PackageName = newPackage
		}

		m.tempFilter.Tag = strings.TrimSpace(m.tagInput.Value())
		m.tempFilter.Text = strings.TrimSpace(m.textInput.Value())

		filterChanged := m.filter != m.tempFilter
		formatChanged := m.format != m.tempFormat

		m.filter = m.tempFilter
		m.format = m.tempFormat

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
	m.packageInput.SetValue(m.filter.PackageName)
	m.tagInput.SetValue(m.filter.Tag)
	m.textInput.SetValue(m.filter.Text)
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

func (m FilterViewModel) Update(msg tea.Msg) (FilterViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.Size:
		m.viewportSize = msg
		m.packageInput.Width = msg.Width - 20
		m.tagInput.Width = msg.Width - 20
		m.textInput.Width = msg.Width - 20

	case tea.KeyMsg:
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
			changed, err := m.Exit(true)
			if err != nil {
				slog.Error("Error saving filter", "err", err)
				return m, nil
			}
			return m, func() tea.Msg {
				if changed {
					return tui.UpdateFilterCmd{
						Filter: m.filter,
						Format: m.format,
					}
				}
				return tui.NavigateToLogcatCmd{}
			}

		case "esc":
			m.Exit(false)
			return m, func() tea.Msg {
				return tui.NavigateToLogcatCmd{}
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

func (m FilterViewModel) View() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
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

func (m FilterViewModel) renderFormatPanel(height int) string {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true)

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
		selected := isFormatSelected(&m.tempFormat, fmt.field)
		cursor := m.activePanel == 0 && m.formatCursor == i

		line := m.renderRadioButton(fmt.name, selected, cursor)
		b.WriteString(line + "\n")
	}

	// Create bordered panel
	panelStyle := theme.Panel().
		Width(m.viewportSize.Width/2 - 4).
		Height(height)

	if m.activePanel == 0 {
		// Active panel - highlight border
		panelStyle = theme.ActivePanel().
			Width(m.viewportSize.Width/2 - 4).
			Height(height)
	}

	return panelStyle.Render(b.String())
}

func (m FilterViewModel) renderModifierPanel() (string, int) {
	var b strings.Builder

	// Panel title
	panelTitleStyle := lipgloss.NewStyle().
		Bold(true)

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
		selected := isModifierSelected(&m.tempFormat, mod.field)
		cursor := m.activePanel == 1 && m.modifierCursor == i

		line := m.renderCheckbox(mod.name, selected, cursor)
		b.WriteString(line + "\n")
	}

	// Create bordered panel
	panelStyle := theme.Panel().
		Width(m.viewportSize.Width/2 - 4)

	if m.activePanel == 1 {
		panelStyle = theme.ActivePanel().
			Width(m.viewportSize.Width/2 - 4)
	}

	result := b.String()
	return panelStyle.Render(result), lipgloss.Height(result) + 2
}

func (m FilterViewModel) renderPackagePanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true)

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

	panelStyle := theme.Panel().
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 2 {
		panelStyle = theme.ActivePanel().
			Width(m.viewportSize.Width - 8)
	}

	return panelStyle.Render(b.String())
}

func (m FilterViewModel) renderTagPanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true)

	if m.activePanel == 3 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Tag Filter (exact match)") + "\n\n")

	b.WriteString(m.tagInput.View())

	panelStyle := theme.Panel().
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 3 {
		panelStyle = theme.ActivePanel().
			Width(m.viewportSize.Width - 8)
	}

	return panelStyle.Render(b.String())
}

func (m FilterViewModel) renderTextPanel() string {
	var b strings.Builder

	panelTitleStyle := lipgloss.NewStyle().
		Bold(true)

	if m.activePanel == 4 {
		panelTitleStyle = panelTitleStyle.Foreground(theme.FGActiveTitle)
	}

	b.WriteString(panelTitleStyle.Render("Text Filter (substring match)") + "\n\n")

	b.WriteString(m.textInput.View())

	panelStyle := theme.Panel().
		Width(m.viewportSize.Width - 8)

	if m.activePanel == 4 {
		panelStyle = theme.ActivePanel().
			Width(m.viewportSize.Width - 8)
	}

	return panelStyle.Render(b.String())
}

func (m FilterViewModel) renderRadioButton(label string, selected bool, cursor bool) string {
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

func (m FilterViewModel) renderCheckbox(label string, checked bool, cursor bool) string {
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

func getCurrentFormatIndex(f *model.Format) int {
	if f.Brief {
		return 0
	}
	if f.Long {
		return 1
	}
	if f.Process {
		return 2
	}
	if f.Raw {
		return 3
	}
	if f.Tag {
		return 4
	}
	if f.Thread {
		return 5
	}
	if f.Threadtime {
		return 6
	}
	if f.Time {
		return 7
	}
	return 0
}

func clearAllFormats(f *model.Format) {
	f.Brief = false
	f.Long = false
	f.Process = false
	f.Raw = false
	f.Tag = false
	f.Thread = false
	f.Threadtime = false
	f.Time = false
}

func setFormatByIndex(f *model.Format, i int) {
	switch i {
	case 0:
		f.Brief = true
	case 1:
		f.Long = true
	case 2:
		f.Process = true
	case 3:
		f.Raw = true
	case 4:
		f.Tag = true
	case 5:
		f.Thread = true
	case 6:
		f.Threadtime = true
	case 7:
		f.Time = true
	}
}

func isFormatSelected(f *model.Format, field string) bool {
	switch field {
	case "brief":
		return f.Brief
	case "long":
		return f.Long
	case "process":
		return f.Process
	case "raw":
		return f.Raw
	case "tag":
		return f.Tag
	case "thread":
		return f.Thread
	case "threadtime":
		return f.Threadtime
	case "time":
		return f.Time
	}
	return false
}

func toggleModifierByIndex(f *model.Format, i int) {
	switch i {
	case 0:
		f.Color = !f.Color
	case 1:
		f.Descriptive = !f.Descriptive
	case 2:
		f.Epoch = !f.Epoch
	case 3:
		f.Monotonic = !f.Monotonic
	case 4:
		f.Printable = !f.Printable
	case 5:
		f.Uid = !f.Uid
	case 6:
		f.Usec = !f.Usec
	case 7:
		f.UTC = !f.UTC
	case 8:
		f.Year = !f.Year
	case 9:
		f.Zone = !f.Zone
	}
}

func isModifierSelected(f *model.Format, field string) bool {
	switch field {
	case "color":
		return f.Color
	case "descriptive":
		return f.Descriptive
	case "epoch":
		return f.Epoch
	case "monotonic":
		return f.Monotonic
	case "printable":
		return f.Printable
	case "uid":
		return f.Uid
	case "usec":
		return f.Usec
	case "UTC":
		return f.UTC
	case "year":
		return f.Year
	case "zone":
		return f.Zone
	}
	return false
}
