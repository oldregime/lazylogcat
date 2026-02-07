package commandui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

type CommandDialogFormatSelectedMsg struct {
	Format string
}

func newFormatTable(currentFormat string) (table.Model, map[int]string) {
	columns := []table.Column{
		{Title: "", Width: 12},
		{Title: "", Width: 3},
	}

	formatMap := make(map[int]string)
	var rows []table.Row
	initialCursor := 0
	for i, name := range model.AllFormats {
		formatMap[i] = name
		marker := ""
		if name == currentFormat {
			marker = "●"
			initialCursor = i
		}
		rows = append(rows, table.Row{name, marker})
	}

	t := newTable(columns, rows, len(rows)+1)
	t.SetCursor(initialCursor)

	return t, formatMap
}

func (m CommandDialogModel) updateFormat(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "enter" {
		if fmt, ok := m.formatMap[m.formatTable.Cursor()]; ok {
			return m, func() tea.Msg { return CommandDialogFormatSelectedMsg{Format: fmt} }
		}
		return m, nil
	}

	// Arrow keys go to table navigation
	if key == "up" || key == "down" {
		m.formatTable, _ = m.formatTable.Update(msg)
		return m, nil
	}

	// All other keys go to the search input
	prevValue := m.searchInput.Value()
	m.searchInput, _ = m.searchInput.Update(msg)
	if m.searchInput.Value() != prevValue {
		m.filterFormatRows()
	}

	return m, nil
}

func (m *CommandDialogModel) filterFormatRows() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	formatMap := make(map[int]string)
	var rows []table.Row

	for _, name := range model.AllFormats {
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}
		formatMap[len(rows)] = name
		marker := ""
		if name == m.currentFormat {
			marker = "●"
		}
		rows = append(rows, table.Row{name, marker})
	}

	m.formatMap = formatMap
	m.formatTable.SetRows(rows)
	m.formatTable.SetHeight(len(rows) + 1)
	if len(rows) > 0 {
		m.formatTable.SetCursor(0)
	}
}

func (m CommandDialogModel) viewFormat() string {
	title := lipgloss.NewStyle().Bold(true).Render("Format")
	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close")

	var body string
	if len(m.formatTable.Rows()) == 0 && m.searchInput.Value() != "" {
		body = lipgloss.NewStyle().Foreground(theme.FGHelp).Render("No results found")
	} else {
		body = m.formatTable.View()
	}

	content := title + "\n\n" + m.searchInput.View() + "\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
