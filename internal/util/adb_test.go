package util

import (
	"testing"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

func TestParseDeviceList(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []model.Device
	}{
		{
			name: "NormalOutputWithModel",
			output: "List of devices attached\n" +
				"emulator-5554          device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 transport_id:1\n\n",
			want: []model.Device{
				{Id: "emulator-5554", Name: "sdk_gphone64_x86_64"},
			},
		},
		{
			name: "MultipleDevices",
			output: "List of devices attached\n" +
				"emulator-5554          device product:sdk model:Pixel_4 transport_id:1\n" +
				"192.168.1.100:5555     device product:raven model:Pixel_6_Pro transport_id:2\n\n",
			want: []model.Device{
				{Id: "emulator-5554", Name: "Pixel_4"},
				{Id: "192.168.1.100:5555", Name: "Pixel_6_Pro"},
			},
		},
		{
			name: "NoModelField_Undefined",
			output: "List of devices attached\n" +
				"abc123    device usb:1-1 transport_id:3\n",
			want: []model.Device{
				{Id: "abc123", Name: "Undefined"},
			},
		},
		{
			name: "OfflineDeviceExcluded",
			output: "List of devices attached\n" +
				"abc123    offline usb:1-1\n",
			want: []model.Device{},
		},
		{
			name:   "HeaderOnly",
			output: "List of devices attached\n\n",
			want:   []model.Device{},
		},
		{
			name:   "EmptyOutput",
			output: "",
			want:   []model.Device{},
		},
		{
			name: "TrailingNewlinesAndCR",
			output: "List of devices attached\r\n" +
				"emulator-5554          device model:Pixel_4\r\n\r\n",
			want: []model.Device{
				{Id: "emulator-5554", Name: "Pixel_4"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDeviceList(tt.output)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseDeviceList() returned %d devices, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Id != tt.want[i].Id {
					t.Errorf("device[%d].Id = %q, want %q", i, got[i].Id, tt.want[i].Id)
				}
				if got[i].Name != tt.want[i].Name {
					t.Errorf("device[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
			}
		})
	}
}
