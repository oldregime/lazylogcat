package util

import (
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
)

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func CopyToClipboard(lines ...string) error {
	cleanLines := make([]string, len(lines))
	for i, line := range lines {
		cleanLines[i] = stripANSI(line)
	}
	return clipboard.WriteAll(strings.Join(cleanLines, "\n"))
}

func stripANSI(text string) string {
	return ansiEscapeRegex.ReplaceAllString(text, "")
}
