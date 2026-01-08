package cmd

import (
	"github.com/parfenovvs/lazylogcat/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lazylogcat",
	Short: "Interactive Android logcat viewer",
	Long:  `lazylogcat is an interactive TUI application for viewing Android device logs.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return app.SetupLogging()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.LaunchTUI()
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return app.CloseLog()
	},
}

func Execute() error {
	return rootCmd.Execute()
}
