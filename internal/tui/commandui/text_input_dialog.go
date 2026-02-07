package commandui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

// CommandDialogTextInputAppliedMsg is sent when the user confirms the text input value with enter.
type CommandDialogTextInputAppliedMsg struct {
	Command model.Command
	Value   string
}

func textInputTitle(cmd model.Command) string {
	switch cmd {
	case model.CommandPackage:
		return "Package"
	case model.CommandTag:
		return "Tag"
	case model.CommandContent:
		return "Content"
	default:
		return ""
	}
}

func textInputPlaceholder(cmd model.Command) string {
	switch cmd {
	case model.CommandPackage:
		return "Package name..."
	case model.CommandTag:
		return "Tag value..."
	case model.CommandContent:
		return "Search text..."
	default:
		return ""
	}
}

func newCommandTextInput(cmd model.Command, currentValue string, deviceId string) TextInputModel {
	var validateFn func(string) error
	if cmd == model.CommandPackage {
		validateFn = func(value string) error {
			_, err := util.GetPidByPackageName(deviceId, value)
			if err != nil {
				return fmt.Errorf("Package not found on device")
			}
			return nil
		}
	}

	return NewTextInput(TextInputConfig{
		Title:       textInputTitle(cmd),
		Placeholder: textInputPlaceholder(cmd),
		Value:       currentValue,
		ValidateFn:  validateFn,
	})
}

func (m CommandDialogModel) updateTextInput(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInputDlg, cmd = m.textInputDlg.Update(msg)
	if m.textInputDlg.Submitted() {
		activeCmd := m.activeCommand
		value := m.textInputDlg.Value()
		return m, func() tea.Msg {
			return CommandDialogTextInputAppliedMsg{Command: activeCmd, Value: value}
		}
	}
	return m, cmd
}

// initTextInputCmd returns the tea.Cmd needed to start the text input cursor blink.
func initTextInputCmd() tea.Cmd {
	return textinput.Blink
}
