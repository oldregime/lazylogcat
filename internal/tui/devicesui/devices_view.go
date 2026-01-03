package devicesui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/model"
)

type DeviceSelectionModel struct {
	devices []model.Device
	cursor  int
	err     error
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

func (m DeviceSelectionModel) Update(msg tea.Msg) (DeviceSelectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.cursor = 0
			return m, GetDevices
		case "enter":
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				selectedDevice := m.devices[m.cursor]
				return m, func() tea.Msg {
					return DeviceSelectedMsg{Device: selectedDevice}
				}
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.devices)-1 {
				m.cursor++
			}
		}
	case getDevicesMsg:
		m.devices = msg.Devices
	case getDevicesErrorMsg:
		m.err = msg.Err
	}
	return m, nil
}

func (m DeviceSelectionModel) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	if len(m.devices) == 0 {
		return "No devices connected. Press 'r' to refresh."
	}

	var s strings.Builder
	s.WriteString("Select a device:\n\n")
	for i, device := range m.devices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s.WriteString(fmt.Sprintf("%s %s - %s\n", cursor, device.Id, device.Name))
	}

	return s.String()
}

func GetDevices() tea.Msg {
	cmd := exec.Command("adb", "devices", "-l")

	output, err := cmd.Output()
	if err != nil {
		return getDevicesErrorMsg{Err: err}
	}

	devices := make([]model.Device, 0)

	strOutput := string(output)
	lines := strings.Split(strOutput, "\n")
	for _, l := range lines[1:] {
		if strings.Contains(l, "device") {
			parts := strings.Split(l, " ")
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
