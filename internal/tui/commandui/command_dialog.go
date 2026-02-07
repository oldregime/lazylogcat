package commandui

import (
	"fmt"
	"maps"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

// DialogConfig holds the parameters for creating a new command dialog.
type DialogConfig struct {
	Filter         model.Filter
	Format         model.Format
	SoftWrap       bool
	SelectedDevice *model.Device
	DeviceId       string
}

type CommandDialogCloseMsg struct{}

type CommandDialogSelectMsg struct {
	Command model.Command
}

type dialogState int

const (
	stateCommands dialogState = iota
	stateLogLevel
	stateFormat
	stateDevices
	stateModifiers
	stateTextInput
)

var dialogStyle = func() lipgloss.Style {
	return theme.ActivePanel().
		Padding(1, 2).
		Width(tui.DialogWidth).
		MaxHeight(tui.DialogMaxHeight)
}

type CommandDialogModel struct {
	state      dialogState
	table      table.Model
	skipRows   map[int]bool
	commandMap map[int]model.CommandData

	// Unfiltered originals for the main command table
	allCommandRows []table.Row
	allSkipRows    map[int]bool
	allCommandMap  map[int]model.CommandData

	levelTable   table.Model
	levelMap     map[int]model.Level
	currentLevel model.Level

	formatTable   table.Model
	formatMap     map[int]string
	currentFormat string

	deviceTable    table.Model
	deviceMap      map[int]model.Device
	allDevices     []model.Device
	selectedDevice *model.Device
	deviceErr      error

	modifiersTable  table.Model
	modifierMap     map[int]string
	tempModifiers   map[string]bool
	activeModifiers map[string]bool

	textInput        textinput.Model
	textInputCommand model.Command
	textInputTitle   string
	textInputError   string

	searchInput textinput.Model

	filter   model.Filter
	deviceId string
}

func NewDialog(cfg DialogConfig) CommandDialogModel {
	resolveValue := func(cmd model.Command) string {
		switch cmd {
		case model.CommandPackage:
			return cfg.Filter.PackageName
		case model.CommandTag:
			return cfg.Filter.Tag
		case model.CommandLevel:
			lvl := string(cfg.Filter.Level)
			if lvl == "" {
				lvl = string(model.LvlV)
			}
			return lvl
		case model.CommandContent:
			return cfg.Filter.Text
		case model.CommandFormat:
			return cfg.Format.Value()
		case model.CommandModifiers:
			mods := cfg.Format.Modifiers()
			switch len(mods) {
			case 0:
				return ""
			case 1:
				return mods[0]
			default:
				return fmt.Sprintf("[%d]", len(mods))
			}
		case model.CommandToggleWrap:
			if cfg.SoftWrap {
				return "on"
			}
			return "off"
		case model.CommandDevices:
			if cfg.SelectedDevice != nil {
				return cfg.SelectedDevice.Name
			}
			return ""
		default:
			return ""
		}
	}

	columns := []table.Column{
		{Title: "", Width: 16},
		{Title: "", Width: 10},
		{Title: "", Width: 10},
	}

	skipRows := make(map[int]bool)
	commandMap := make(map[int]model.CommandData)
	var rows []table.Row
	for i, group := range model.Commands() {
		if i > 0 {
			skipRows[len(rows)] = true
			rows = append(rows, table.Row{"", "", ""})
		}
		skipRows[len(rows)] = true
		groupName := lipgloss.NewStyle().Bold(true).Render(group.Name)
		rows = append(rows, table.Row{groupName, "", ""})
		for _, cmd := range group.Commands {
			commandMap[len(rows)] = cmd
			value := truncateMiddle(resolveValue(cmd.Command), 10)
			rows = append(rows, table.Row{cmd.Name, value, cmd.Shortcut})
		}
	}

	t := newTable(columns, rows, len(rows)+1)
	t.SetCursor(1) // Skip the first group header

	currentLevel := cfg.Filter.Level
	if currentLevel == "" {
		currentLevel = model.LvlV
	}
	levelTable, levelMap := newLevelTable(currentLevel)
	formatTable, formatMap := newFormatTable(cfg.Format.Value())
	modifiersTable, modifierMap := newModifiersTable(cfg.Format.ActiveModifiers)

	// Store unfiltered originals for the main command table
	allSkipRows := make(map[int]bool)
	maps.Copy(allSkipRows, skipRows)
	allCommandMap := make(map[int]model.CommandData)
	maps.Copy(allCommandMap, commandMap)
	allCommandRows := make([]table.Row, len(rows))
	copy(allCommandRows, rows)

	return CommandDialogModel{
		state:           stateCommands,
		table:           t,
		skipRows:        skipRows,
		commandMap:      commandMap,
		allCommandRows:  allCommandRows,
		allSkipRows:     allSkipRows,
		allCommandMap:   allCommandMap,
		levelTable:      levelTable,
		levelMap:        levelMap,
		currentLevel:    currentLevel,
		formatTable:     formatTable,
		formatMap:       formatMap,
		currentFormat:   cfg.Format.Value(),
		selectedDevice:  cfg.SelectedDevice,
		modifiersTable:  modifiersTable,
		modifierMap:     modifierMap,
		activeModifiers: cfg.Format.ActiveModifiers,
		searchInput:     newSearchInput(),
		filter:          cfg.Filter,
		deviceId:        cfg.DeviceId,
	}
}

// NewDialogForCommand creates a command dialog that opens directly in the sub-dialog
// for the given command, skipping the main command list. Returns the model and an optional
// tea.Cmd (needed for text input cursor blink).
func NewDialogForCommand(cfg DialogConfig, cmd model.Command) (CommandDialogModel, tea.Cmd) {
	m := NewDialog(cfg)

	switch cmd {
	case model.CommandLevel:
		m.state = stateLogLevel
	case model.CommandFormat:
		m.state = stateFormat
	case model.CommandModifiers:
		m.tempModifiers = make(map[string]bool)
		maps.Copy(m.tempModifiers, m.activeModifiers)
		m.modifiersTable = m.refreshModifierRows()
		m.state = stateModifiers
	case model.CommandDevices:
		deviceTable, deviceMap, allDevices, err := loadDevices(cfg.SelectedDevice)
		m.deviceTable = deviceTable
		m.deviceMap = deviceMap
		m.allDevices = allDevices
		m.deviceErr = err
		m.state = stateDevices
	case model.CommandPackage:
		m.textInputCommand = cmd
		m.textInputTitle = textInputTitle(cmd)
		m.textInput = newDialogTextInput(textInputPlaceholder(cmd), cfg.Filter.PackageName)
		m.state = stateTextInput
		return m, textinput.Blink
	case model.CommandTag:
		m.textInputCommand = cmd
		m.textInputTitle = textInputTitle(cmd)
		m.textInput = newDialogTextInput(textInputPlaceholder(cmd), cfg.Filter.Tag)
		m.state = stateTextInput
		return m, textinput.Blink
	case model.CommandContent:
		m.textInputCommand = cmd
		m.textInputTitle = textInputTitle(cmd)
		m.textInput = newDialogTextInput(textInputPlaceholder(cmd), cfg.Filter.Text)
		m.state = stateTextInput
		return m, textinput.Blink
	}

	return m, nil
}

// NewDeviceDialog creates a command dialog that opens directly in the device selection state.
func NewDeviceDialog(selectedDevice *model.Device) CommandDialogModel {
	deviceTable, deviceMap, allDevices, err := loadDevices(selectedDevice)
	return CommandDialogModel{
		state:          stateDevices,
		deviceTable:    deviceTable,
		deviceMap:      deviceMap,
		allDevices:     allDevices,
		deviceErr:      err,
		selectedDevice: selectedDevice,
		searchInput:    newSearchInput(),
	}
}

func (m CommandDialogModel) Update(msg tea.Msg) (CommandDialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+p" || key == "esc" {
			if m.state == stateModifiers {
				return m, func() tea.Msg {
					return CommandDialogModifiersSelectedMsg{Modifiers: m.tempModifiers}
				}
			}
			return m, func() tea.Msg { return CommandDialogCloseMsg{} }
		}

		switch m.state {
		case stateCommands:
			return m.updateCommands(msg, key)
		case stateLogLevel:
			return m.updateLogLevel(msg, key)
		case stateFormat:
			return m.updateFormat(msg, key)
		case stateDevices:
			return m.updateDevices(msg, key)
		case stateModifiers:
			return m.updateModifiers(msg, key)
		case stateTextInput:
			return m.updateTextInput(msg, key)
		}
	default:
		// Forward non-key messages (e.g. cursor blink) to the text input when active.
		if m.state == stateTextInput {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// Forward non-key messages to search input for cursor blink in table-based states.
		if m.state != stateTextInput {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m CommandDialogModel) updateCommands(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "enter" {
		if cmdData, ok := m.commandMap[m.table.Cursor()]; ok {
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandLevel {
				resetSearchInput(&m.searchInput)
				m.state = stateLogLevel
				return m, nil
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandFormat {
				resetSearchInput(&m.searchInput)
				m.state = stateFormat
				return m, nil
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandModifiers {
				resetSearchInput(&m.searchInput)
				m.tempModifiers = make(map[string]bool)
				for k, v := range m.activeModifiers {
					m.tempModifiers[k] = v
				}
				m.modifiersTable = m.refreshModifierRows()
				m.state = stateModifiers
				return m, nil
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandDevices {
				resetSearchInput(&m.searchInput)
				deviceTable, deviceMap, allDevices, err := loadDevices(m.selectedDevice)
				m.deviceTable = deviceTable
				m.deviceMap = deviceMap
				m.allDevices = allDevices
				m.deviceErr = err
				m.state = stateDevices
				return m, nil
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandPackage {
				resetSearchInput(&m.searchInput)
				m.textInputCommand = cmdData.Command
				m.textInputTitle = textInputTitle(cmdData.Command)
				m.textInput = newDialogTextInput(textInputPlaceholder(cmdData.Command), m.filter.PackageName)
				m.state = stateTextInput
				return m, textinput.Blink
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandTag {
				resetSearchInput(&m.searchInput)
				m.textInputCommand = cmdData.Command
				m.textInputTitle = textInputTitle(cmdData.Command)
				m.textInput = newDialogTextInput(textInputPlaceholder(cmdData.Command), m.filter.Tag)
				m.state = stateTextInput
				return m, textinput.Blink
			}
			if cmdData.Type == model.CommandTypeNavigation && cmdData.Command == model.CommandContent {
				resetSearchInput(&m.searchInput)
				m.textInputCommand = cmdData.Command
				m.textInputTitle = textInputTitle(cmdData.Command)
				m.textInput = newDialogTextInput(textInputPlaceholder(cmdData.Command), m.filter.Text)
				m.state = stateTextInput
				return m, textinput.Blink
			}
			return m, func() tea.Msg { return CommandDialogSelectMsg{Command: cmdData.Command} }
		}
		return m, nil
	}

	// Arrow keys go to table navigation
	if key == "up" || key == "down" {
		prevCursor := m.table.Cursor()
		m.table, _ = m.table.Update(msg)
		newCursor := m.table.Cursor()

		if m.skipRows[newCursor] && newCursor != prevCursor {
			dir := 1
			if newCursor < prevCursor {
				dir = -1
			}
			rowCount := len(m.table.Rows())
			target := newCursor + dir
			for target >= 0 && target < rowCount && m.skipRows[target] {
				target += dir
			}
			if target >= 0 && target < rowCount {
				m.table.SetCursor(target)
			} else {
				m.table.SetCursor(prevCursor)
			}
		}
		return m, nil
	}

	// All other keys go to the search input
	prevValue := m.searchInput.Value()
	m.searchInput, _ = m.searchInput.Update(msg)
	if m.searchInput.Value() != prevValue {
		m.filterCommandRows()
	}

	return m, nil
}

// filterCommandRows rebuilds the table rows from the unfiltered originals,
// keeping only rows whose command name matches the current search query.
func (m *CommandDialogModel) filterCommandRows() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	if query == "" {
		// No filter: restore all original rows
		rows := make([]table.Row, len(m.allCommandRows))
		copy(rows, m.allCommandRows)
		m.skipRows = make(map[int]bool)
		maps.Copy(m.skipRows, m.allSkipRows)
		m.commandMap = make(map[int]model.CommandData)
		maps.Copy(m.commandMap, m.allCommandMap)
		m.table.SetRows(rows)
		m.table.SetHeight(len(rows) + 1)
		// Set cursor to first non-skip row
		for i := 0; i < len(rows); i++ {
			if !m.skipRows[i] {
				m.table.SetCursor(i)
				break
			}
		}
		return
	}

	// Build filtered rows: include group headers only if at least one command in the group matches
	var rows []table.Row
	skipRows := make(map[int]bool)
	commandMap := make(map[int]model.CommandData)

	// Walk through the original rows, tracking group boundaries
	i := 0
	for i < len(m.allCommandRows) {
		// Check if this is a group header (skip row that is not a blank separator)
		if m.allSkipRows[i] {
			// Could be a blank separator or a group header
			// Blank separator: all columns empty
			row := m.allCommandRows[i]
			isSeparator := row[0] == "" && row[1] == "" && row[2] == ""

			if isSeparator {
				// Skip blank separator, advance to group header
				i++
				continue
			}

			// This is a group header. Collect commands in this group.
			headerIdx := i
			i++
			var groupCommands []struct {
				origIdx int
				row     table.Row
				cmdData model.CommandData
			}
			for i < len(m.allCommandRows) && !m.allSkipRows[i] {
				if cmdData, ok := m.allCommandMap[i]; ok {
					if strings.Contains(strings.ToLower(cmdData.Name), query) {
						groupCommands = append(groupCommands, struct {
							origIdx int
							row     table.Row
							cmdData model.CommandData
						}{i, m.allCommandRows[i], cmdData})
					}
				}
				i++
			}

			if len(groupCommands) > 0 {
				// Add separator before group (if not the first group in filtered results)
				if len(rows) > 0 {
					skipRows[len(rows)] = true
					rows = append(rows, table.Row{"", "", ""})
				}
				// Add group header
				skipRows[len(rows)] = true
				rows = append(rows, m.allCommandRows[headerIdx])
				// Add matching commands
				for _, gc := range groupCommands {
					commandMap[len(rows)] = gc.cmdData
					rows = append(rows, gc.row)
				}
			}
			continue
		}
		i++
	}

	m.skipRows = skipRows
	m.commandMap = commandMap
	m.table.SetRows(rows)
	m.table.SetHeight(len(rows) + 1)

	// Set cursor to first non-skip row
	cursorSet := false
	for idx := 0; idx < len(rows); idx++ {
		if !skipRows[idx] {
			m.table.SetCursor(idx)
			cursorSet = true
			break
		}
	}
	if !cursorSet {
		m.table.SetCursor(0)
	}
}

func (m CommandDialogModel) View() string {
	switch m.state {
	case stateLogLevel:
		return m.viewLogLevel()
	case stateFormat:
		return m.viewFormat()
	case stateDevices:
		return m.viewDevices()
	case stateModifiers:
		return m.viewModifiers()
	case stateTextInput:
		return m.viewTextInput()
	default:
		return m.viewCommands()
	}
}

func (m CommandDialogModel) viewCommands() string {
	title := lipgloss.NewStyle().Bold(true).Render("Command List")
	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close")

	var body string
	if len(m.table.Rows()) == 0 && m.searchInput.Value() != "" {
		body = lipgloss.NewStyle().Foreground(theme.FGHelp).Render("No results found")
	} else {
		body = m.table.View()
	}

	content := title + "\n\n" + m.searchInput.View() + "\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
