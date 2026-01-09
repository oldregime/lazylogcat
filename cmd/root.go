package cmd

import (
	"log/slog"
	"os"

	"github.com/parfenovvs/lazylogcat/internal/app"
	"github.com/parfenovvs/lazylogcat/internal/config"
	"github.com/spf13/cobra"
)

var (
	debugFlag  bool
	configFile string
)

var rootCmd = &cobra.Command{
	Use:          "lazylogcat",
	Short:        "Interactive Android logcat viewer",
	Long:         `lazylogcat is an interactive TUI application for viewing Android device logs.`,
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			return app.SetupLogging()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := app.PreLaunchChecks(); err != nil {
			return err
		}

		var c config.Config
		if configFile != "" {
			f, err := os.Open(configFile)
			if err != nil {
				slog.Error("Failed to open config file", "error", err)
				return err
			}
			c, err = config.Load(f)
			if err != nil {
				slog.Error("Failed to load config file, using default config", "error", err)
			}
			slog.Debug("Configuration loaded", "Config", c.String())
		}
		return app.LaunchTUI(c)
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if debugFlag {
			return app.CloseLog()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging to .lazylogcat.log file")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to configuration file")
}

func Execute() error {
	return rootCmd.Execute()
}
