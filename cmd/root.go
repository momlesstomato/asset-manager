package cmd

import (
	"fmt"
	"os"

	"asset-manager/core/logger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "asset-manager",
	Short: "Asset Manager Service",
	Long: `Asset Manager is a robust alternative for serving Nitro client assets.
It supports S3 storage engines and high-performance file serving.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// Use the application's standard logger for error reporting
		// We default to console format to match user expectations (CLI tool)
		// We use "debug" level configuration to get ISO8601 timestamps (DevConfig) instead of Epoch (ProdConfig)
		cfg := &logger.Config{
			Level:  "debug",
			Format: "console",
		}

		l, logErr := logger.New(cfg)
		if logErr == nil {
			// Log the error with structured logger (Console encoding will make it pretty)
			l.Error("command failed", zap.Error(err))
			_ = l.Sync()
		} else {
			// Absolute fallback if logger creation fails (rare)
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func init() {

}
