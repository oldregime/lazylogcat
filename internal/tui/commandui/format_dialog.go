package commandui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

type CommandDialogFormatSelectedMsg struct {
	Format string
}

func newFormatSingleSelect(currentFormat string) SingleSelectModel {
	var items []SingleSelectItem
	for _, name := range model.AllFormats {
		items = append(items, SingleSelectItem{
			Key:     name,
			Columns: []string{name},
		})
	}
	return NewSingleSelect(SingleSelectConfig{
		Title:  "Format",
		Footer: "esc to close",
		Columns: []table.Column{
			{Title: "", Width: 12},
		},
		Items:      items,
		CurrentKey: currentFormat,
	})
}

func (m CommandDialogModel) updateFormat(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.singleSelect, cmd = m.singleSelect.Update(msg, key)
	if m.singleSelect.Selected() {
		fmt := m.singleSelect.SelectedKey()
		return m, func() tea.Msg { return CommandDialogFormatSelectedMsg{Format: fmt} }
	}
	return m, cmd
}
