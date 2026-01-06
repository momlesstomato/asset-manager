package cmd

import (
	"context"
	"fmt"
	"os"

	"asset-manager/core/config"
	"asset-manager/core/logger"
	"asset-manager/core/storage"
	"asset-manager/feature/integrity"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var fixFlag bool

// integrityCmd represents the integrity command
var integrityCmd = &cobra.Command{
	Use:   "integrity",
	Short: "Perform integrity checks on the asset storage",
	Long:  `Checks if the storage bucket has the required folder structure and other integrity requirements.`,
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), false)
	},
}

// structureCmd represents the integrity structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Check and fix folder structure",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), true)
	},
}

func init() {
	RootCmd.AddCommand(integrityCmd)
	integrityCmd.AddCommand(structureCmd)

	integrityCmd.PersistentFlags().BoolVar(&fixFlag, "fix", false, "Fix missing folders")
}

func runIntegrityChecks(ctx context.Context, onlyStructure bool) {
	// 1. Load Config
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Logger (Console format for CLI usually better, but respect config)
	// Override format to console for CLI if nice output desired?
	// User said "integrity structure will check it and log warning".
	// Let's use configured logger.
	logg, err := logger.New(&cfg.Log)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 3. Initialize Storage
	store, err := storage.NewClient(cfg.Storage)
	if err != nil {
		logg.Fatal("Failed to create storage client", zap.Error(err))
	}

	// 4. Initialize Integrity Service
	svc := integrity.NewService(store, cfg.Storage.Bucket, logg)

	// Run Checks
	// Currently only Structure check is implemented.

	if onlyStructure || true { // "integrity" triggers all, currently all = structure
		logg.Info("Checking folder structure...")
		missing, err := svc.CheckStructure(ctx)
		if err != nil {
			logg.Fatal("Structure check failed", zap.Error(err))
		}

		if len(missing) == 0 {
			logg.Info("Structure is intact.")
		} else {
			logg.Warn("Missing folders detected", zap.Strings("missing", missing))

			if fixFlag {
				logg.Info("Fixing missing folders...")
				if err := svc.FixStructure(ctx, missing); err != nil {
					logg.Fatal("Failed to fix structure", zap.Error(err))
				}
				logg.Info("Structure fixed successfully.")
			} else {
				logg.Info("Run with --fix to create missing folders.")
			}
		}
	}
}
