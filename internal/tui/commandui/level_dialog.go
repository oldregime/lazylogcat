package commandui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

type CommandDialogLevelSelectedMsg struct {
	Level model.Level
}

var levelEntries = []struct {
	level model.Level
	name  string
}{
	{model.LvlV, "Verbose"},
	{model.LvlD, "Debug"},
	{model.LvlI, "Info"},
	{model.LvlW, "Warning"},
	{model.LvlE, "Error"},
	{model.LvlF, "Fatal"},
}

func newLevelTable(currentLevel model.Level) (table.Model, map[int]model.Level) {
	columns := []table.Column{
		{Title: "", Width: 5},
		{Title: "", Width: 12},
		{Title: "", Width: 3},
	}

	levelMap := make(map[int]model.Level)
	var rows []table.Row
	initialCursor := 0
	for i, entry := range levelEntries {
		levelMap[i] = entry.level
		marker := ""
		if entry.level == currentLevel {
			marker = "●"
			initialCursor = i
		}
		rows = append(rows, table.Row{string(entry.level), entry.name, marker})
	}

	t := newTable(columns, rows, len(rows)+1)
	t.SetCursor(initialCursor)

	return t, levelMap
}

func (m CommandDialogModel) updateLogLevel(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "enter" {
		if lvl, ok := m.levelMap[m.levelTable.Cursor()]; ok {
			return m, func() tea.Msg { return CommandDialogLevelSelectedMsg{Level: lvl} }
		}
		return m, nil
	}

	// Arrow keys go to table navigation
	if key == "up" || key == "down" {
		m.levelTable, _ = m.levelTable.Update(msg)
		return m, nil
	}

	// All other keys go to the search input
	prevValue := m.searchInput.Value()
	m.searchInput, _ = m.searchInput.Update(msg)
	if m.searchInput.Value() != prevValue {
		m.filterLevelRows()
	}

	return m, nil
}

func (m *CommandDialogModel) filterLevelRows() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	levelMap := make(map[int]model.Level)
	var rows []table.Row
	for _, entry := range levelEntries {
		if query != "" && !strings.Contains(strings.ToLower(entry.name), query) &&
			!strings.Contains(strings.ToLower(string(entry.level)), query) {
			continue
		}
		levelMap[len(rows)] = entry.level
		marker := ""
		if entry.level == m.currentLevel {
			marker = "●"
		}
		rows = append(rows, table.Row{string(entry.level), entry.name, marker})
	}

	m.levelMap = levelMap
	m.levelTable.SetRows(rows)
	m.levelTable.SetHeight(len(rows) + 1)
	if len(rows) > 0 {
		m.levelTable.SetCursor(0)
	}
}

func (m CommandDialogModel) viewLogLevel() string {
	title := lipgloss.NewStyle().Bold(true).Render("Log Level")
	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close")

	var body string
	if len(m.levelTable.Rows()) == 0 && m.searchInput.Value() != "" {
		body = lipgloss.NewStyle().Foreground(theme.FGHelp).Render("No results found")
	} else {
		body = m.levelTable.View()
	}

	content := title + "\n\n" + m.searchInput.View() + "\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
