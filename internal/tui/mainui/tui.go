package mainui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parfenovvs/lazylogcat/internal/config"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui"
	"github.com/parfenovvs/lazylogcat/internal/tui/devicesui"
	"github.com/parfenovvs/lazylogcat/internal/tui/filterui"
	"github.com/parfenovvs/lazylogcat/internal/tui/logcatui"
	"github.com/parfenovvs/lazylogcat/internal/util"
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

	currentDevice *model.Device
	filter        model.Filter
	format        model.Format

	devicesView devicesui.DevicesViewModel
	logcatView  logcatui.LogcatViewModel
	filterView  filterui.FilterViewModel
}

func InitMainModel(c config.Config) MainModel {
	var m MainModel

	devices, err := util.GetConnectedDevices()
	if err == nil && len(devices) > 0 {
		for _, d := range devices {
			if d.Id == c.Session.DeviceId {
				m.currentDevice = &d
				break
			}
		}
	}

	if m.currentDevice == nil {
		c.Session.Pkg = ""
	}
	if c.Session.Pkg != "" {
		_, err := util.GetPidByPackageName(m.currentDevice.Id, c.Session.Pkg)
		if err != nil {
			c.Session.Pkg = ""
		}
	}

	m.filter = util.FilterFromConfig(&c)
	m.format = util.FormatFromConfig(&c)

	if m.currentDevice == nil {
		m.state = devicesView
		m.devicesView = devicesui.New()
	} else {
		m.state = logcatView
		m.logcatView = logcatui.New(m.viewportSize, *m.currentDevice, m.filter, m.format)
	}

	return m
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
			cmds = append(cmds, newCmd)
		case filterView:
			newFilterView, newCmd := m.filterView.Update(resizeMsg)
			m.filterView = newFilterView
			cmds = append(cmds, newCmd)
		case devicesView:
			newDevicesView, newCmd := m.devicesView.Update(resizeMsg)
			m.devicesView = newDevicesView
			cmds = append(cmds, newCmd)
		}

	case devicesui.DeviceSelectedMsg:
		m.currentDevice = &msg.Device
		return m, func() tea.Msg {
			return tui.NavigateToLogcatCmd{}
		}

	case tui.NavigateToFilterCmd:
		m.state = filterView
		m.filterView = filterui.New(
			m.viewportSize,
			m.currentDevice.Id,
			m.filter,
			m.format,
		)
		return m, nil

	case tui.UpdateFilterCmd:
		m.filter = msg.Filter
		m.format = msg.Format
		return m, func() tea.Msg {
			return tui.NavigateToLogcatCmd{}
		}

	case tui.NavigateToLogcatCmd:
		m.state = logcatView
		logcatui.Close(&m.logcatView)
		m.logcatView = logcatui.New(m.viewportSize, *m.currentDevice, m.filter, m.format)
		return m, func() tea.Msg {
			return tui.ReconnectLogcatCmd{}
		}

	case tui.NavigateToDevicesCmd:
		logcatui.Close(&m.logcatView)
		m.state = devicesView
		return m, func() tea.Msg {
			return devicesui.GetDevices(m.currentDevice)
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
