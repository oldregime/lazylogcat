package main

import (
	"os"

	"github.com/parfenovvs/lazylogcat/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
