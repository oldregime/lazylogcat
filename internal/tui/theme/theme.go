package theme

import "github.com/charmbracelet/lipgloss"

// Package-private ANSI-16 base colors
const (
	// Standard colors (0-7)
	black   string = "0"
	red     string = "1"
	green   string = "2"
	yellow  string = "3"
	blue    string = "4"
	magenta string = "5"
	cyan    string = "6"
	white   string = "7"

	// Bright/Bold colors (8-15)
	brightBlack   string = "8"
	brightRed     string = "9"
	brightGreen   string = "10"
	brightYellow  string = "11"
	brightBlue    string = "12"
	brightMagenta string = "13"
	brightCyan    string = "14"
	brightWhite   string = "15"
)

var (
	// Foreground Colors - Borders
	// FGBorder is used for inactive panel borders, dividers, and outlines
	FGBorder = lipgloss.AdaptiveColor{Light: black, Dark: black}

	// FGActiveBorder is used for active panel border highlighting
	FGActiveBorder = lipgloss.AdaptiveColor{Light: brightGreen, Dark: brightGreen}

	// Foreground Colors - Text
	// FGTitle is used for section titles, headers, and panel names (inactive state)
	FGTitle = lipgloss.AdaptiveColor{Light: brightBlack, Dark: brightBlack}

	// FGActiveTitle is used for active panel title text
	FGActiveTitle = lipgloss.AdaptiveColor{Light: brightGreen, Dark: brightGreen}

	// FGSelected is used for selected item text
	FGSelected = lipgloss.AdaptiveColor{Light: brightWhite, Dark: brightWhite}

	// FGHelp is used for help text and secondary information
	FGHelp = lipgloss.AdaptiveColor{Light: black, Dark: black}

	// FGError is used for error messages and validation warnings
	FGError = lipgloss.AdaptiveColor{Light: brightRed, Dark: brightRed}

	// Background Colors
	// BGCursor is used for cursor/selection highlight backgrounds
	BGCursor = lipgloss.AdaptiveColor{Light: black, Dark: black}
)

func GetLogColor(level string) lipgloss.AdaptiveColor {
	switch level {
	case "V":
		return lipgloss.AdaptiveColor{Light: brightBlack, Dark: brightBlack}
	case "D":
		return lipgloss.AdaptiveColor{Light: blue, Dark: blue}
	case "I":
		return lipgloss.AdaptiveColor{Light: green, Dark: green}
	case "W":
		return lipgloss.AdaptiveColor{Light: yellow, Dark: yellow}
	case "E":
		return lipgloss.AdaptiveColor{Light: red, Dark: red}
	case "F":
		return lipgloss.AdaptiveColor{Light: magenta, Dark: magenta}
	default:
		return lipgloss.AdaptiveColor{Light: black, Dark: black}
	}
}
