package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// dimView applies a dimming effect to the view content
func DimView(view string) string {
	// Apply faint style to each line to dim the content
	dimStyle := lipgloss.NewStyle().Faint(true)

	lines := strings.Split(view, "\n")
	dimmedLines := make([]string, len(lines))

	for i, line := range lines {
		dimmedLines[i] = dimStyle.Render(line)
	}

	return strings.Join(dimmedLines, "\n")
}
