package util

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

var (
	ErrFailedToGetDevices    = fmt.Errorf("failed to get connected devices")
	ErrFailedToStartLogcat   = fmt.Errorf("failed to start logcat process")
	ErrFailedToGetStdoutPipe = fmt.Errorf("failed to get stdout pipe")
)

func GetConnectedDevices() ([]model.Device, error) {
	cmd := exec.Command("adb", "devices", "-l")

	output, err := cmd.Output()
	if err != nil {
		return nil, ErrFailedToGetDevices
	}

	return ParseDeviceList(string(output)), nil
}

// ParseDeviceList parses the output of `adb devices -l` into a slice of
// Device structs. Lines that do not contain the keyword "device" (excluding
// the header) are skipped. If no model: field is present the device name
// defaults to "Undefined".
func ParseDeviceList(output string) []model.Device {
	devices := make([]model.Device, 0)

	lines := strings.Split(output, "\n")
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

	return devices
}
