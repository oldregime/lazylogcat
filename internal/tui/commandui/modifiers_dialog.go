package commandui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

type CommandDialogModifiersSelectedMsg struct {
	Modifiers map[string]bool
}

func newModifiersTable(activeModifiers map[string]bool) (table.Model, map[int]string) {
	columns := []table.Column{
		{Title: "", Width: 14},
		{Title: "", Width: 3},
	}

	modifierMap := make(map[int]string)
	var rows []table.Row
	for i, name := range model.AllModifiers {
		modifierMap[i] = name
		marker := ""
		if activeModifiers[name] {
			marker = "✓"
		}
		rows = append(rows, table.Row{name, marker})
	}

	t := newTable(columns, rows, len(rows)+1)

	return t, modifierMap
}

func (m CommandDialogModel) refreshModifierRows() table.Model {
	var rows []table.Row
	for _, name := range model.AllModifiers {
		marker := ""
		if m.tempModifiers[name] {
			marker = "✓"
		}
		rows = append(rows, table.Row{name, marker})
	}
	m.modifiersTable.SetRows(rows)
	return m.modifiersTable
}

func (m CommandDialogModel) updateModifiers(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "enter" || key == " " {
		if name, ok := m.modifierMap[m.modifiersTable.Cursor()]; ok {
			if m.tempModifiers[name] {
				delete(m.tempModifiers, name)
			} else {
				m.tempModifiers[name] = true
			}
			m.filterModifierRows()
		}
		return m, nil
	}

	// Arrow keys go to table navigation
	if key == "up" || key == "down" {
		m.modifiersTable, _ = m.modifiersTable.Update(msg)
		return m, nil
	}

	// All other keys go to the search input
	prevValue := m.searchInput.Value()
	m.searchInput, _ = m.searchInput.Update(msg)
	if m.searchInput.Value() != prevValue {
		m.filterModifierRows()
	}

	return m, nil
}

func (m *CommandDialogModel) filterModifierRows() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	modifierMap := make(map[int]string)
	var rows []table.Row
	cursor := m.modifiersTable.Cursor()
	// Track the name at the old cursor so we can preserve position
	oldCursorName := ""
	if name, ok := m.modifierMap[cursor]; ok {
		oldCursorName = name
	}

	newCursor := 0
	for _, name := range model.AllModifiers {
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}
		if name == oldCursorName {
			newCursor = len(rows)
		}
		modifierMap[len(rows)] = name
		marker := ""
		if m.tempModifiers[name] {
			marker = "✓"
		}
		rows = append(rows, table.Row{name, marker})
	}

	m.modifierMap = modifierMap
	m.modifiersTable.SetRows(rows)
	m.modifiersTable.SetHeight(len(rows) + 1)
	if len(rows) > 0 {
		if newCursor < len(rows) {
			m.modifiersTable.SetCursor(newCursor)
		} else {
			m.modifiersTable.SetCursor(0)
		}
	}
}

func (m CommandDialogModel) viewModifiers() string {
	title := lipgloss.NewStyle().Bold(true).Render("Modifiers")
	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("space/enter toggle, esc to apply")

	var body string
	if len(m.modifiersTable.Rows()) == 0 && m.searchInput.Value() != "" {
		body = lipgloss.NewStyle().Foreground(theme.FGHelp).Render("No results found")
	} else {
		body = m.modifiersTable.View()
	}

	content := title + "\n\n" + m.searchInput.View() + "\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
