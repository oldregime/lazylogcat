package util

import "strings"

func GetLogLevel(line string) string {
	i := strings.Index(line, "/")
	if i <= 0 {
		return ""
	}
	return line[i-1 : i]
}
