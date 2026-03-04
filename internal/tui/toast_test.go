package tui

import (
	"testing"
)

func TestToastModel_IsVisible_ZeroValue(t *testing.T) {
	var tm ToastModel
	if tm.IsVisible() {
		t.Error("zero-value ToastModel.IsVisible() = true, want false")
	}
}

func TestToastModel_Show(t *testing.T) {
	var tm ToastModel
	cmd := tm.Show("hello", ToastInfo)

	if !tm.visible {
		t.Error("after Show(), visible = false, want true")
	}
	if tm.message != "hello" {
		t.Errorf("after Show(), message = %q, want %q", tm.message, "hello")
	}
	if tm.level != ToastInfo {
		t.Errorf("after Show(), level = %d, want %d", tm.level, ToastInfo)
	}
	if tm.id != 1 {
		t.Errorf("after Show(), id = %d, want 1", tm.id)
	}
	if cmd == nil {
		t.Error("Show() returned nil cmd, want non-nil")
	}

	// Second Show increments ID
	_ = tm.Show("world", ToastWarning)
	if tm.id != 2 {
		t.Errorf("after second Show(), id = %d, want 2", tm.id)
	}
}

func TestToastModel_Update_MatchingID(t *testing.T) {
	var tm ToastModel
	_ = tm.Show("hello", ToastInfo)

	tm.Update(ToastExpiredMsg{ID: tm.id})

	if tm.visible {
		t.Error("after Update with matching ID, visible = true, want false")
	}
}

func TestToastModel_Update_StaleID(t *testing.T) {
	var tm ToastModel
	_ = tm.Show("first", ToastInfo)
	_ = tm.Show("second", ToastWarning) // ID is now 2

	tm.Update(ToastExpiredMsg{ID: 1}) // stale ID

	if !tm.visible {
		t.Error("after Update with stale ID, visible = false, want true")
	}
}

func TestToastModel_View_NotVisible(t *testing.T) {
	var tm ToastModel
	got := tm.View()
	if got != "" {
		t.Errorf("View() on not-visible toast = %q, want empty string", got)
	}
}

func TestToastModel_View_Visible(t *testing.T) {
	var tm ToastModel
	_ = tm.Show("test message", ToastError)

	got := tm.View()
	if got == "" {
		t.Error("View() on visible toast = empty string, want non-empty")
	}
	// The rendered output should contain the message text
	if len(got) < len("test message") {
		t.Errorf("View() = %q, expected it to contain %q", got, "test message")
	}
}
