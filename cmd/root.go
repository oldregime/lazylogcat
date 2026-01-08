package cmd

import (
	"github.com/parfenovvs/lazylogcat/internal/app"
	"github.com/spf13/cobra"
)

var (
	debugFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "lazylogcat",
	Short: "Interactive Android logcat viewer",
	Long:  `lazylogcat is an interactive TUI application for viewing Android device logs.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			return app.SetupLogging()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.LaunchTUI()
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			return app.CloseLog()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging to debug.log file")
}

func Execute() error {
	return rootCmd.Execute()
}
