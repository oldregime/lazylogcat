package tui

import (
	"strings"
	"testing"
)

func TestOverlayLine(t *testing.T) {
	tests := []struct {
		name       string
		background string
		foreground string
		x          int
		want       string
	}{
		{
			name:       "ForegroundAtX0",
			background: "AAAAAAAAAA",
			foreground: "BBB",
			x:          0,
			want:       "BBB       ",
		},
		{
			name:       "ForegroundAtMiddle",
			background: "AAAAAAAAAA",
			foreground: "BBB",
			x:          3,
			want:       "AAABBB    ",
		},
		{
			name:       "ForegroundPastEnd",
			background: "AAAAA",
			foreground: "BBB",
			x:          10,
			want:       "AAAAA",
		},
		{
			name:       "ForegroundWiderThanRemaining",
			background: "AAAAAAAAAA",
			foreground: "BBBBBBBBB",
			x:          5,
			want:       "AAAAABBBBBBBBB",
		},
		{
			name:       "EmptyForeground",
			background: "AAAAAAAAAA",
			foreground: "",
			x:          3,
			want:       "AAA       ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overlayLine(tt.background, tt.foreground, tt.x)
			if got != tt.want {
				t.Errorf("overlayLine(%q, %q, %d) = %q, want %q",
					tt.background, tt.foreground, tt.x, got, tt.want)
			}
		})
	}
}

func TestOverlayDialog(t *testing.T) {
	t.Run("SmallDialog", func(t *testing.T) {
		baseView := strings.Repeat(strings.Repeat(".", 20)+"\n", 19) + strings.Repeat(".", 20)
		dialog := "XXXX\nXXXX"

		result := OverlayDialog(
			struct{ Width, Height int }{Width: 20, Height: 20},
			baseView,
			dialog,
		)

		if !strings.Contains(result, "XXXX") {
			t.Error("OverlayDialog result does not contain dialog content")
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 20 {
			t.Errorf("OverlayDialog result has %d lines, want 20", len(lines))
		}
	})
}
