package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/devicesui"
	"github.com/parfenovvs/lazylogcat/internal/tui/logcatui"
)

var style = lipgloss.NewStyle()

type sessionState int

const (
	devicesView sessionState = iota
	logcatView
)

type MainModel struct {
	viewportSize model.Size
	state        sessionState
	devicesView  devicesui.DeviceSelectionModel
	logcatView   logcatui.LogcatModel
}

func InitMainModel() MainModel {
	return MainModel{
		state:       devicesView,
		devicesView: devicesui.New(),
	}
}

func (m MainModel) Init() tea.Cmd {
	return devicesui.GetDevices
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportSize = model.Size{
			Width:  msg.Width,
			Height: msg.Height,
		}
		// Forward resize with allocated viewport size to child views
		switch m.state {
		case logcatView:
			resizeMsg := model.Size{
				Width:  m.viewportSize.Width,
				Height: m.viewportSize.Height,
			}
			newLogcatViewing, newCmd := m.logcatView.Update(resizeMsg)
			m.logcatView = newLogcatViewing
			return m, newCmd
		case devicesView:
			// Forward resize to DevicesView for centering
			newDevicesView, newCmd := m.devicesView.Update(msg)
			m.devicesView = newDevicesView
			return m, newCmd
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.logcatView.Close()
			return m, tea.Quit
		}
	case devicesui.DeviceSelectedMsg:
		m.state = logcatView
		m.logcatView = logcatui.New(m.viewportSize, msg.Device)
		return m, m.logcatView.ConnectToLogcat

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
	style = style.Width(m.viewportSize.Width).
		Height(m.viewportSize.Height)

	var content string

	switch m.state {
	case devicesView:
		content = m.devicesView.View()
	case logcatView:
		content = m.logcatView.View()
	}

	return style.Render(content)
}
