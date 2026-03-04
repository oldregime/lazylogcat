package commandui

import (
	"testing"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

func TestTextInputTitle(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.Command
		want string
	}{
		{name: "Package", cmd: model.CommandPackage, want: "Package"},
		{name: "Tag", cmd: model.CommandTag, want: "Tag"},
		{name: "Content", cmd: model.CommandContent, want: "Content"},
		{name: "Unknown", cmd: model.CommandLevel, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textInputTitle(tt.cmd)
			if got != tt.want {
				t.Errorf("textInputTitle(%d) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestTextInputPlaceholder(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.Command
		want string
	}{
		{name: "Package", cmd: model.CommandPackage, want: "Package name..."},
		{name: "Tag", cmd: model.CommandTag, want: "Tag value..."},
		{name: "Content", cmd: model.CommandContent, want: "Search text..."},
		{name: "Unknown", cmd: model.CommandLevel, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textInputPlaceholder(tt.cmd)
			if got != tt.want {
				t.Errorf("textInputPlaceholder(%d) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
