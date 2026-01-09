package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/parfenovvs/lazylogcat/internal/tui"
)

var logFile *os.File

var (
	ErrAdbNotFound = fmt.Errorf("adb not found in PATH")
)

func PreLaunchChecks() error {
	_, err := exec.LookPath("adb")
	if err != nil {
		return ErrAdbNotFound
	}
	return nil
}

func SetupLogging() error {
	f, err := tea.LogToFile(".lazylogcat.log", "debug")
	if err != nil {
		return fmt.Errorf("could not open log file: %w", err)
	}
	logFile = f

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	return nil
}

func LaunchTUI() error {
	p := tea.NewProgram(
		tui.InitMainModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func CloseLog() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
