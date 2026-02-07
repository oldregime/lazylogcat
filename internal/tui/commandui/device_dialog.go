package commandui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

type CommandDialogDeviceSelectedMsg struct {
	Device model.Device
}

func loadDevices(selectedDevice *model.Device) (table.Model, map[int]model.Device, []model.Device, error) {
	devices, err := util.GetConnectedDevices()
	if err != nil {
		return table.Model{}, nil, nil, fmt.Errorf("failed to get devices: %w", err)
	}

	t, dm := newDeviceTable(devices, selectedDevice)
	return t, dm, devices, nil
}

func newDeviceTable(devices []model.Device, selectedDevice *model.Device) (table.Model, map[int]model.Device) {
	columns := []table.Column{
		{Title: "", Width: 20},
		{Title: "", Width: 18},
		{Title: "", Width: 3},
	}

	deviceMap := make(map[int]model.Device)
	var rows []table.Row
	initialCursor := 0
	for i, device := range devices {
		deviceMap[i] = device
		marker := ""
		if selectedDevice != nil && selectedDevice.Id == device.Id {
			marker = "●"
			initialCursor = i
		}

		name := device.Name
		maxNameLen := 18
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		rows = append(rows, table.Row{name, device.Id, marker})
	}

	height := len(rows) + 1
	if height < 2 {
		height = 2
	}

	t := newTable(columns, rows, height)
	if len(rows) > 0 {
		t.SetCursor(initialCursor)
	}

	return t, deviceMap
}

func (m CommandDialogModel) updateDevices(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "r" {
		deviceTable, deviceMap, allDevices, err := loadDevices(m.selectedDevice)
		m.deviceTable = deviceTable
		m.deviceMap = deviceMap
		m.allDevices = allDevices
		m.deviceErr = err
		// Reapply current filter after refresh
		if m.searchInput.Value() != "" {
			m.filterDeviceRows()
		}
		return m, nil
	}

	if key == "enter" {
		if device, ok := m.deviceMap[m.deviceTable.Cursor()]; ok {
			return m, func() tea.Msg { return CommandDialogDeviceSelectedMsg{Device: device} }
		}
		return m, nil
	}

	// Arrow keys go to table navigation
	if key == "up" || key == "down" {
		if len(m.deviceMap) > 0 {
			m.deviceTable, _ = m.deviceTable.Update(msg)
		}
		return m, nil
	}

	// All other keys go to the search input
	prevValue := m.searchInput.Value()
	m.searchInput, _ = m.searchInput.Update(msg)
	if m.searchInput.Value() != prevValue {
		m.filterDeviceRows()
	}

	return m, nil
}

func (m *CommandDialogModel) filterDeviceRows() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	var filtered []model.Device
	for _, device := range m.allDevices {
		if query != "" &&
			!strings.Contains(strings.ToLower(device.Name), query) &&
			!strings.Contains(strings.ToLower(device.Id), query) {
			continue
		}
		filtered = append(filtered, device)
	}

	t, dm := newDeviceTable(filtered, m.selectedDevice)
	m.deviceTable = t
	m.deviceMap = dm
}

func (m CommandDialogModel) viewDevices() string {
	title := lipgloss.NewStyle().Bold(true).Render("Select Device")

	var body string
	if m.deviceErr != nil {
		errorMsg := lipgloss.NewStyle().
			Foreground(theme.FGHelp).
			Render(fmt.Sprintf("Error: %s", m.deviceErr.Error()))
		hint := lipgloss.NewStyle().
			Foreground(theme.FGHelp).
			Render("Press 'r' to retry")
		body = "\n" + errorMsg + "\n\n" + hint
	} else if len(m.allDevices) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(theme.FGHelp).
			Render("No devices connected.")
		hint := lipgloss.NewStyle().
			Foreground(theme.FGHelp).
			Render("Press 'r' to refresh")
		body = "\n" + emptyMsg + "\n\n" + hint
	} else if len(m.deviceMap) == 0 && m.searchInput.Value() != "" {
		body = lipgloss.NewStyle().Foreground(theme.FGHelp).Render("No results found")
	} else {
		body = m.deviceTable.View()
	}

	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close • r refresh")

	// Show search input only when there are devices to search through
	if m.deviceErr == nil && len(m.allDevices) > 0 {
		content := title + "\n\n" + m.searchInput.View() + "\n" + body + "\n\n" + footer
		return dialogStyle().Render(content)
	}

	content := title + "\n\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
