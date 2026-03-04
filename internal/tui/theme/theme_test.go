package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGetLogColor(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  lipgloss.Color
	}{
		{name: "Verbose", level: "V", want: ColorRegular},
		{name: "Debug", level: "D", want: lipgloss.Color(blue)},
		{name: "Info", level: "I", want: lipgloss.Color(green)},
		{name: "Warn", level: "W", want: lipgloss.Color(yellow)},
		{name: "Error", level: "E", want: lipgloss.Color(red)},
		{name: "Fatal", level: "F", want: lipgloss.Color(magenta)},
		{name: "Unknown", level: "X", want: ""},
		{name: "Empty", level: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLogColor(tt.level)
			if got != tt.want {
				t.Errorf("GetLogColor(%q) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
