package logcatui

import (
	"testing"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

func TestBuildShortcutMap(t *testing.T) {
	m := buildShortcutMap()

	expectedKeys := map[string]model.Command{
		"p": model.CommandPackage,
		"t": model.CommandTag,
		"l": model.CommandLevel,
		"c": model.CommandContent,
		"o": model.CommandOutput,
		"r": model.CommandReconnect,
		"d": model.CommandDevices,
	}

	for key, wantCmd := range expectedKeys {
		t.Run("Key_"+key, func(t *testing.T) {
			data, ok := m[key]
			if !ok {
				t.Fatalf("shortcutMap missing key %q", key)
			}
			if data.Command != wantCmd {
				t.Errorf("shortcutMap[%q].Command = %d, want %d", key, data.Command, wantCmd)
			}
		})
	}

	// Verify no unexpected keys (ctrl+c is "ctrl+c", not "ctrl+x c")
	for key := range m {
		if _, ok := expectedKeys[key]; !ok {
			t.Errorf("unexpected key %q in shortcutMap", key)
		}
	}
}
