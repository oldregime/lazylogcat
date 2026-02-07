package commandui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

// TextInputConfig holds the parameters for creating a TextInputModel.
type TextInputConfig struct {
	Title       string
	Footer      string
	Placeholder string
	Value       string
	ValidateFn  func(value string) error // Optional validation run on Enter; nil means no validation
}

// TextInputModel is a reusable text input dialog widget with optional validation.
type TextInputModel struct {
	title      string
	footer     string
	input      textinput.Model
	validateFn func(value string) error
	errorMsg   string
	submitted  bool
}

// NewTextInput creates a new TextInputModel from the given config.
func NewTextInput(cfg TextInputConfig) TextInputModel {
	ti := textinput.New()
	ti.Placeholder = cfg.Placeholder
	ti.CharLimit = 100
	ti.Width = 30
	ti.SetValue(cfg.Value)
	ti.Focus()

	footer := cfg.Footer
	if footer == "" {
		footer = "enter to apply, esc to cancel"
	}

	return TextInputModel{
		title:      cfg.Title,
		footer:     footer,
		input:      ti,
		validateFn: cfg.ValidateFn,
	}
}

// Update handles messages. Returns the updated model and a tea.Cmd.
// After calling Update, check Submitted() to see if the user pressed Enter successfully.
func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key == "enter" {
			value := strings.TrimSpace(m.input.Value())
			if m.validateFn != nil && value != "" {
				if err := m.validateFn(value); err != nil {
					m.errorMsg = err.Error()
					return m, nil
				}
			}
			m.submitted = true
			return m, nil
		}

		// Forward other keys to the text input, clear error on change
		prevValue := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prevValue {
			m.errorMsg = ""
		}
		return m, cmd

	default:
		// Forward non-key messages (e.g. cursor blink)
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// Submitted returns true if the user pressed Enter and validation passed.
func (m TextInputModel) Submitted() bool {
	return m.submitted
}

// Value returns the current trimmed text input value.
func (m TextInputModel) Value() string {
	return strings.TrimSpace(m.input.Value())
}

// View renders the text input dialog.
func (m TextInputModel) View() string {
	title := theme.DialogTitle().Render(m.title)
	footer := theme.DialogHelp().Render(m.footer)

	var errorLine string
	if m.errorMsg != "" {
		errorLine = "\n" + theme.DialogError().Render(m.errorMsg)
	}

	input := theme.DialogSearch().Render(m.input.View())
	content := title + "\n\n" + input + errorLine + "\n\n" + footer
	return dialogStyle().Render(content)
}
