package util

import (
	"os"
	"strings"
	"testing"
)

func TestOpenInEditor_EditorNotSet(t *testing.T) {
	t.Setenv("EDITOR", "")

	_, err := OpenInEditor("hello")
	if err == nil {
		t.Fatal("expected error when $EDITOR is not set, got nil")
	}
	if err != ErrEditorNotSet {
		t.Errorf("expected ErrEditorNotSet, got %v", err)
	}
}

func TestOpenInEditor_CreatesTempFile(t *testing.T) {
	t.Setenv("EDITOR", "cat")

	lines := []string{"line one", "line two", "line three"}
	cmd, err := OpenInEditor(lines...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// The temp file path is the last argument to the editor command
	if len(cmd.Args) < 2 {
		t.Fatalf("expected at least 2 args, got %v", cmd.Args)
	}
	tempPath := cmd.Args[len(cmd.Args)-1]

	// Verify temp file exists and has expected content
	content, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("failed to read temp file %s: %v", tempPath, err)
	}

	expected := strings.Join(lines, "\n")
	if string(content) != expected {
		t.Errorf("temp file content = %q, want %q", string(content), expected)
	}

	// Verify the file is in the OS temp directory
	if !strings.HasPrefix(tempPath, os.TempDir()) {
		t.Errorf("temp file %s is not in OS temp directory %s", tempPath, os.TempDir())
	}
}

func TestOpenInEditor_StripsANSI(t *testing.T) {
	t.Setenv("EDITOR", "cat")

	ansiLine := "\x1b[31mred text\x1b[0m"
	cmd, err := OpenInEditor(ansiLine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tempPath := cmd.Args[len(cmd.Args)-1]
	content, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	expected := "red text"
	if string(content) != expected {
		t.Errorf("temp file content = %q, want %q", string(content), expected)
	}
}

func TestOpenInEditor_EditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "vim")

	cmd, err := OpenInEditor("test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.Path == "" {
		t.Error("expected non-empty cmd.Path")
	}

	// First arg should be the editor name
	if cmd.Args[0] != "vim" {
		t.Errorf("cmd.Args[0] = %q, want %q", cmd.Args[0], "vim")
	}
}
