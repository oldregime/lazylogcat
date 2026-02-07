package commandui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui"
)

type CommandDialogModifiersSelectedMsg struct {
	Modifiers map[string]bool
}

func newModifiersMultiSelect(activeModifiers map[string]bool) MultiSelectModel {
	return NewMultiSelect(MultiSelectConfig{
		Title:  "Modifiers",
		Footer: "enter toggle, esc to apply",
		Columns: []table.Column{
			{Title: "", Width: tui.DialogWidth - 9},
		},
		Items:  model.AllModifiers,
		Active: activeModifiers,
	})
}

func (m CommandDialogModel) updateModifiers(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.multiSelect, cmd = m.multiSelect.Update(msg, key)
	return m, cmd
}
