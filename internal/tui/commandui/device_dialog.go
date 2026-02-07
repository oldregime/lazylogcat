package commandui

import (
	"fmt"

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

func deviceSingleSelectItems(devices []model.Device) []SingleSelectItem {
	var items []SingleSelectItem
	for _, device := range devices {
		name := device.Name
		maxNameLen := 18
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}
		items = append(items, SingleSelectItem{
			Key:     device.Id,
			Columns: []string{name, device.Id},
		})
	}
	return items
}

func newDeviceSingleSelect(devices []model.Device, selectedDevice *model.Device) SingleSelectModel {
	currentKey := ""
	if selectedDevice != nil {
		currentKey = selectedDevice.Id
	}
	return NewSingleSelect(SingleSelectConfig{
		Title:  "Select Device",
		Footer: "esc to close \u2022 r refresh",
		Columns: []table.Column{
			{Title: "", Width: 20},
			{Title: "", Width: 18},
		},
		Items:      deviceSingleSelectItems(devices),
		CurrentKey: currentKey,
	})
}

func loadDevices(selectedDevice *model.Device) (SingleSelectModel, []model.Device, error) {
	devices, err := util.GetConnectedDevices()
	if err != nil {
		return SingleSelectModel{}, nil, fmt.Errorf("failed to get devices: %w", err)
	}
	ss := newDeviceSingleSelect(devices, selectedDevice)
	return ss, devices, nil
}

func (m CommandDialogModel) updateDevices(msg tea.KeyMsg, key string) (CommandDialogModel, tea.Cmd) {
	if key == "r" {
		ss, allDevices, err := loadDevices(m.selectedDevice)
		m.singleSelect = ss
		m.allDevices = allDevices
		m.deviceErr = err
		return m, nil
	}

	var cmd tea.Cmd
	m.singleSelect, cmd = m.singleSelect.Update(msg, key)
	if m.singleSelect.Selected() {
		deviceId := m.singleSelect.SelectedKey()
		// Find the device by ID from allDevices
		for _, device := range m.allDevices {
			if device.Id == deviceId {
				d := device
				return m, func() tea.Msg { return CommandDialogDeviceSelectedMsg{Device: d} }
			}
		}
	}
	return m, cmd
}

func (m CommandDialogModel) viewDevices() string {
	// Devices has special error/empty states, so we render manually instead of using singleSelect.View()
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
	} else {
		// Delegate to singleSelect.View() when we have devices
		return m.singleSelect.View()
	}

	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close \u2022 r refresh")
	content := title + "\n\n" + body + "\n\n" + footer
	return dialogStyle().Render(content)
}
