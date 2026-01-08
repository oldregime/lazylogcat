package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui/devicesui"
	"github.com/parfenovvs/lazylogcat/internal/tui/filterui"
	"github.com/parfenovvs/lazylogcat/internal/tui/logcatui"
)

var style = lipgloss.NewStyle()

type sessionState int

const (
	devicesView sessionState = iota
	logcatView
	filterView
)

type MainModel struct {
	viewportSize model.Size
	state        sessionState
	devicesView  devicesui.DevicesViewModel
	logcatView   logcatui.LogcatViewModel
	filterView   filterui.FilterViewModel
}

func InitMainModel() MainModel {
	return MainModel{
		state:       devicesView,
		devicesView: devicesui.New(),
	}
}

func (m MainModel) Init() tea.Cmd {
	return func() tea.Msg {
		return devicesui.GetDevices(nil)
	}
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			logcatui.Close(&m.logcatView)
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.viewportSize = model.Size{
			Width:  msg.Width,
			Height: msg.Height,
		}
		resizeMsg := m.viewportSize
		switch m.state {
		case logcatView:
			newLogcatViewing, newCmd := m.logcatView.Update(resizeMsg)
			m.logcatView = newLogcatViewing
			return m, newCmd
		case filterView:
			newFilterView, newCmd := m.filterView.Update(resizeMsg)
			m.filterView = newFilterView
			return m, newCmd
		case devicesView:
			newDevicesView, newCmd := m.devicesView.Update(resizeMsg)
			m.devicesView = newDevicesView
			return m, newCmd
		}
		return m, nil

	case devicesui.DeviceSelectedMsg:
		m.state = logcatView
		m.logcatView = logcatui.New(m.viewportSize, msg.Device)
		return m, func() tea.Msg {
			return logcatui.ConnectToLogcat(m.logcatView)
		}

	case logcatui.GoToFilterMsg:
		m.state = filterView
		m.filterView = filterui.New(
			m.viewportSize,
			msg.Device.Id,
			msg.Filter,
			msg.Format,
		)
		return m, nil

	case filterui.FilterExitMsg:
		if m.state == filterView {
			m.state = logcatView
			return m, func() tea.Msg {
				return logcatui.UpdateFiltersMsg{
					Filter: msg.Filter,
					Format: msg.Format,
				}
			}
		}

	case logcatui.GoToDevicesMsg:
		logcatui.Close(&m.logcatView)
		m.state = devicesView
		return m, func() tea.Msg {
			return devicesui.GetDevices(msg.Selected)
		}
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

	case filterView:
		newFilterView, newCmd := m.filterView.Update(msg)
		m.filterView = newFilterView
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
	case filterView:
		content = m.filterView.View()
	}

	return style.Render(content)
}
