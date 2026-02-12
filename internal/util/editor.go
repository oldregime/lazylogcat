package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrEditorNotSet = fmt.Errorf("$EDITOR environment variable is not set")

// OpenInEditor writes content to a temporary file in the OS temp directory
// and returns an *exec.Cmd that opens the file in the user's $EDITOR.
// Returns ErrEditorNotSet if the EDITOR environment variable is not set.
// The temporary file is not cleaned up automatically.
func OpenInEditor(lines ...string) (*exec.Cmd, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil, ErrEditorNotSet
	}

	cleanLines := make([]string, len(lines))
	for i, line := range lines {
		cleanLines[i] = stripANSI(line)
	}
	content := strings.Join(cleanLines, "\n")

	f, err := os.CreateTemp("", "lazylogcat-*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	cmd := exec.Command(editor, f.Name())
	return cmd, nil
}
