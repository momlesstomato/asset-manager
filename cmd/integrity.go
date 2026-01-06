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

	structureCmd.Flags().BoolVar(&fixFlag, "fix", false, "Fix missing folders")
}

func runIntegrityChecks(ctx context.Context, onlyStructure bool) {
	// 1. Load Config
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Logger
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

	// 1. Structure Check
	logg.Info("Checking folder structure...")
	missingStructure, err := svc.CheckStructure(ctx)
	if err != nil {
		logg.Fatal("Structure check failed", zap.Error(err))
	}

	if len(missingStructure) == 0 {
		logg.Info("Structure is intact.")
	} else {
		logg.Warn("Missing folders detected", zap.Strings("missing", missingStructure))

		if onlyStructure && fixFlag {
			logg.Info("Fixing missing folders...")
			if err := svc.FixStructure(ctx, missingStructure); err != nil {
				logg.Fatal("Failed to fix structure", zap.Error(err))
			}
			logg.Info("Structure fixed successfully.")
		} else if onlyStructure {
			logg.Info("Run with --fix to create missing folders.")
		}
	}

	// 2. GameData Check (Only if running full integrity check)
	if !onlyStructure {
		logg.Info("Checking gamedata files...")
		missingGameData, err := svc.CheckGameData(ctx)
		if err != nil {
			logg.Fatal("GameData check failed", zap.Error(err))
		}

		if len(missingGameData) == 0 {
			logg.Info("GameData files are present.")
		} else {
			logg.Warn("Missing gamedata files detected", zap.Strings("missing", missingGameData))
		}
	}
}
