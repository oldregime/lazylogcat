package util

import (
	"regexp"

	"github.com/atotto/clipboard"
)

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func CopyToClipboard(text string) error {
	cleanText := stripANSI(text)
	return clipboard.WriteAll(cleanText)
}

func stripANSI(text string) string {
	return ansiEscapeRegex.ReplaceAllString(text, "")
}
