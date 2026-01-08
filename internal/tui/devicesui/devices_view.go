package devicesui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
)

type DeviceSelectionModel struct {
	devices      []model.Device
	selected     *model.Device
	cursor       int
	viewportSize model.Size
	err          error
}

type getDevicesMsg struct {
	devices  []model.Device
	selected *model.Device
}

type getDevicesErrorMsg struct {
	Err error
}

type DeviceSelectedMsg struct {
	Device model.Device
}

func New() DeviceSelectionModel {
	return DeviceSelectionModel{
		cursor: 0,
	}
}

func (m DeviceSelectionModel) Update(msg tea.Msg) (DeviceSelectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportSize = model.Size{
			Width:  msg.Width,
			Height: msg.Height,
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, func() tea.Msg {
				return GetDevices(m.selected)
			}

		case "j", "down":
			if len(m.devices) > 0 && m.cursor < len(m.devices)-1 {
				m.cursor++
			}
			return m, nil

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "enter":
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				selectedDevice := m.devices[m.cursor]
				return m, func() tea.Msg {
					return DeviceSelectedMsg{Device: selectedDevice}
				}
			}
			return m, nil
		}

	case getDevicesMsg:
		m.devices = msg.devices
		if msg.selected != nil {
			for i, device := range m.devices {
				if device.Id == msg.selected.Id {
					m.cursor = i
					m.selected = msg.selected
					break
				}
			}
		} else {
			m.cursor = 0
		}
		return m, nil

	case getDevicesErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

func (m DeviceSelectionModel) View() string {
	content := m.renderDevicePanelWithHelp()
	return m.centerContent(content)
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

func (m DeviceSelectionModel) renderRadioButton(label string, selected bool, cursor bool) string {
	indicator := "( )"
	if selected {
		indicator = "(●)"
	}

	line := fmt.Sprintf("%s %s", indicator, label)

	if cursor {
		return lipgloss.NewStyle().
			Background(theme.BGCursor).
			Foreground(theme.FGSelected).
			Bold(true).
			Width(60).
			Render(line)
	}

	return line
}

func (m DeviceSelectionModel) renderDevicePanel() string {
	var b strings.Builder

	panelTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FGActiveTitle).
		Render("Select a Device")

	b.WriteString(panelTitle + "\n\n")

	// Handle error state
	if m.err != nil {
		errorMsg := lipgloss.NewStyle().
			Render(fmt.Sprintf("Error: %s", m.err.Error()))
		b.WriteString(errorMsg + "\n\n")

		hint := lipgloss.NewStyle().
			Render("Press 'r' to retry")
		b.WriteString(hint)
	} else if len(m.devices) == 0 {
		// Handle empty state
		emptyMsg := lipgloss.NewStyle().
			Render("No devices connected.")
		b.WriteString(emptyMsg + "\n\n")

		hint := lipgloss.NewStyle().
			Render("Press 'r' to refresh")
		b.WriteString(hint)
	} else {
		// Render device list
		for i, device := range m.devices {
			label := fmt.Sprintf("%s (%s)", device.Name, device.Id)

			// Truncate if too long
			maxLabelLen := 50
			if len(label) > maxLabelLen {
				label = label[:maxLabelLen-3] + "..."
			}

			selected := false
			if m.selected != nil && m.selected.Id == device.Id {
				selected = true
			}
			line := m.renderRadioButton(label, selected, i == m.cursor)
			b.WriteString(line + "\n")
		}
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.FGActiveBorder).
		Padding(1, 2).
		Width(m.viewportSize.Width/2 - 4)

	return panelStyle.Render(b.String())
}

func (m DeviceSelectionModel) renderDevicePanelWithHelp() string {
	var b strings.Builder

	devicePanel := m.renderDevicePanel()
	b.WriteString(devicePanel + "\n\n")

	help := lipgloss.NewStyle().
		Foreground(theme.FGHelp).
		Width(m.viewportSize.Width/2 - 4).
		AlignHorizontal(lipgloss.Center).
		Render("↑/k up • ↓/j down • enter select • r refresh • ctrl+c quit")

	b.WriteString(help)

	return b.String()
}

func GetDevices(selected *model.Device) tea.Msg {
	cmd := exec.Command("adb", "devices", "-l")

	output, err := cmd.Output()
	if err != nil {
		return getDevicesErrorMsg{Err: fmt.Errorf("failed to get devices: %w", err)}
	}

	devices := make([]model.Device, 0)

	strOutput := string(output)
	lines := strings.Split(strOutput, "\n")
	var preSelected *model.Device
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
			if selected != nil && selected.Id == id {
				preSelected = selected
			}
		}
	}

	return getDevicesMsg{devices: devices, selected: preSelected}
}
