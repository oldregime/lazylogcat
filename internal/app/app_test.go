package app

import (
	"errors"
	"testing"
)

func TestPreLaunchChecks_NoAdb(t *testing.T) {
	t.Setenv("PATH", "")

	err := PreLaunchChecks()
	if err == nil {
		t.Fatal("PreLaunchChecks() with empty PATH = nil, want error")
	}
	if !errors.Is(err, ErrAdbNotFound) {
		t.Errorf("PreLaunchChecks() error = %v, want %v", err, ErrAdbNotFound)
	}
}
