package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/tui/devicesui"
	"github.com/parfenovvs/lazylogcat/internal/tui/logcatui"
)

type sessionState int

const (
	devicesView sessionState = iota
	logcatView
)

type MainModel struct {
	state       sessionState
	devicesView devicesui.DeviceSelectionModel
	logcatView  logcatui.LogcatModel
}

func InitMainModel() MainModel {
	return MainModel{
		state: devicesView,
	}
}

func (m MainModel) Init() tea.Cmd {
	return devicesui.GetDevices
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.logcatView.Close()
			return m, tea.Quit
		}
	case devicesui.DeviceSelectedMsg:
		m.state = logcatView
		m.logcatView = logcatui.LogcatModel{
			Device: msg.Device,
		}
		return m, func() tea.Msg {
			return logcatui.ConnectToLogcat(msg.Device.Id)
		}

	case logcatui.BackMsg:
		m.logcatView.Close()
		m.state = devicesView
		return m, devicesui.GetDevices
	}

	switch m.state {
	case devicesView:
		newDeviceSelection, newCmd := m.devicesView.Update(msg)
		m.devicesView = newDeviceSelection
		cmd = newCmd

	case logcatView:
		newLogcatViewing, newCmd := m.logcatView.Update(msg)
		m.logcatView = newLogcatViewing
		cmd = newCmd
	}

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	switch m.state {
	case devicesView:
		return m.devicesView.View()
	case logcatView:
		return m.logcatView.View()
	}

	return ""
}
