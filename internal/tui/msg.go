package tui

import "github.com/parfenovvs/lazylogcat/internal/model"

type NavigateToDevicesCmd struct{}

type NavigateToLogcatCmd struct{}

type NavigateToFilterCmd struct{}

type LoadDevicesCmd struct{}

type DevicesLoadedMsg struct {
	Devices  []model.Device
	Selected *model.Device
}

type UpdateFilterCmd struct {
	Filter model.Filter
	Format model.Format
}

type ReconnectLogcatCmd struct{}
