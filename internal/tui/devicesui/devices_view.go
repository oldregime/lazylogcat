package devicesui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

type DeviceSelectionModel struct {
	devices      []model.Device
	table        table.Model
	viewportSize model.Size
	err          error
}

type getDevicesMsg struct {
	Devices []model.Device
}

type getDevicesErrorMsg struct {
	Err error
}

type DeviceSelectedMsg struct {
	Device model.Device
}

func New() DeviceSelectionModel {
	columns := []table.Column{
		{Title: "ID", Width: 20},
		{Title: "Model", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return DeviceSelectionModel{
		table: t,
	}
}

func (m DeviceSelectionModel) Update(msg tea.Msg) (DeviceSelectionModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportSize = model.Size{
			Width:  msg.Width,
			Height: msg.Height,
		}

		availableHeight := max(msg.Height-8, 5)
		m.table.SetHeight(availableHeight)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, GetDevices

		case "enter":
			if len(m.devices) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.devices) {
					selectedDevice := m.devices[selectedIdx]
					return m, func() tea.Msg {
						return DeviceSelectedMsg{Device: selectedDevice}
					}
				}
			}
			return m, nil
		}

	case getDevicesMsg:
		m.devices = msg.Devices

		rows := make([]table.Row, len(m.devices))
		for i, device := range m.devices {
			rows[i] = table.Row{device.Id, device.Name}
		}
		m.table.SetRows(rows)
		m.table.SetCursor(0)

		return m, nil

	case getDevicesErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m DeviceSelectionModel) View() string {
	if m.err != nil {
		errorMsg := fmt.Sprintf("Error: %s\n\nPress 'r' to retry", m.err.Error())
		return m.centerContent(errorMsg)
	}

	if len(m.devices) == 0 {
		emptyMsg := "No devices connected.\n\nPress 'r' to refresh"
		return m.centerContent(emptyMsg)
	}

	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("Select a Device")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("↑/k up • ↓/j down • enter select • r refresh • ctrl+c quit")

	b.WriteString(title + "\n\n")
	b.WriteString(m.table.View())
	b.WriteString("\n\n" + help)

	return m.centerContent(b.String())
}

func (m DeviceSelectionModel) centerContent(content string) string {
	if m.viewportSize.Width == 0 || m.viewportSize.Height == 0 {
		return content
	}

	return lipgloss.Place(
		m.viewportSize.Width,
		m.viewportSize.Height,
		lipgloss.Center, // Horizontal position
		lipgloss.Center, // Vertical position
		content,
	)
}

func GetDevices() tea.Msg {
	cmd := exec.Command("adb", "devices", "-l")

	output, err := cmd.Output()
	if err != nil {
		return getDevicesErrorMsg{Err: fmt.Errorf("failed to get devices: %w", err)}
	}

	devices := make([]model.Device, 0)

	strOutput := string(output)
	lines := strings.Split(strOutput, "\n")
	for _, l := range lines[1:] {
		if strings.Contains(l, "device") {
			parts := strings.Fields(l)
			if len(parts) == 0 {
				continue
			}
			id := parts[0]
			name := "Undefined"
			for _, p := range parts {
				if strings.HasPrefix(p, "model:") {
					product := strings.Split(p, ":")
					name = product[len(product)-1]
					break
				}
			}
			devices = append(devices, model.Device{Id: id, Name: name})
		}
	}

	return getDevicesMsg{Devices: devices}
}
