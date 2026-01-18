package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"asset-manager/core/config"
	"asset-manager/core/database"
	"asset-manager/core/logger"
	"asset-manager/core/storage"
	"asset-manager/feature/integrity"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

var fixFlag bool
var dbFlag bool

// integrityCmd represents the integrity command
var integrityCmd = &cobra.Command{
	Use:   "integrity",
	Short: "Perform integrity checks on the asset storage",
	Long:  `Checks if the storage bucket has the required folder structure and other integrity requirements.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			cmd.Help()
			return
		}
		runIntegrityChecks(cmd.Context(), false, false, false, false, false)
	},
}

// structureCmd represents the integrity structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Check and fix folder structure",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), true, false, false, false, false)
	},
}

// bundleCmd represents the integrity bundle command
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Check and fix bundled asset folders",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), false, true, false, false, false)
	},
}

// gamedataCmd represents the integrity gamedata command
var gamedataCmd = &cobra.Command{
	Use:   "gamedata",
	Short: "Check gamedata files",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), false, false, true, false, false)
	},
}

// furnitureCmd represents the integrity furniture command
var furnitureCmd = &cobra.Command{
	Use:   "furniture",
	Short: "Check integrity of bundled furniture",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), false, false, false, true, false)
	},
}

// serverCmd represents the integrity server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Check integrity of the emulator database schema",
	Run: func(cmd *cobra.Command, args []string) {
		runIntegrityChecks(cmd.Context(), false, false, false, false, true)
	},
}

func init() {
	RootCmd.AddCommand(integrityCmd)
	integrityCmd.AddCommand(structureCmd)
	integrityCmd.AddCommand(bundleCmd)
	integrityCmd.AddCommand(gamedataCmd)
	integrityCmd.AddCommand(furnitureCmd)
	integrityCmd.AddCommand(serverCmd)

	structureCmd.Flags().BoolVar(&fixFlag, "fix", false, "Fix missing folders")
	bundleCmd.Flags().BoolVar(&fixFlag, "fix", false, "Fix missing folders")
	furnitureCmd.Flags().BoolVar(&dbFlag, "db", false, "Check database integrity")
}

func runIntegrityChecks(ctx context.Context, onlyStructure, onlyBundle, onlyGameData, onlyFurniture, onlyServer bool) {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logg, err := logger.New(&cfg.Log)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	store, err := storage.NewClient(cfg.Storage)
	if err != nil {
		logg.Fatal("Failed to create storage client", zap.Error(err))
	}

	// Connect to Database (Optional)
	var db *gorm.DB
	if conn, err := database.Connect(cfg.Database); err != nil {
		logg.Warn("Optional database connection failed", zap.Error(err))
	} else {
		db = conn
		logg = logg.With(zap.String("server", cfg.Server.Emulator))
	}

	svc := integrity.NewService(store, cfg.Storage.Bucket, logg, db, cfg.Server.Emulator)

	runAll := !onlyStructure && !onlyBundle && !onlyGameData && !onlyFurniture && !onlyServer

	runStructure := onlyStructure || runAll
	runGameData := onlyGameData || runAll
	runBundle := onlyBundle || runAll
	runFurniture := onlyFurniture || runAll // Should furniture be in All? It's slow. Assuming yes based on logic.
	runServer := onlyServer || runAll

	// Run Checks

	if runStructure {
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
	}

	if runGameData {
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

	if runBundle {
		logg.Info("Checking bundled folders...")
		missingBundled, err := svc.CheckBundled(ctx)
		if err != nil {
			logg.Fatal("Bundle check failed", zap.Error(err))
		}

		if len(missingBundled) == 0 {
			logg.Info("Bundled folders are intact.")
		} else {
			logg.Warn("Missing bundled folders detected", zap.Strings("missing", missingBundled))

			if onlyBundle && fixFlag {
				logg.Info("Fixing missing bundled folders...")
				if err := svc.FixBundled(ctx, missingBundled); err != nil {
					logg.Fatal("Failed to fix bundled folders", zap.Error(err))
				}
				logg.Info("Bundled folders fixed successfully.")
			} else if onlyBundle {
				logg.Info("Run with --fix to create missing bundled folders.")
			}
		}
	}

	if runFurniture {
		// Check write access by attempting to write? No, just proceed. os.WriteFile will handle errors.

		logg.Info("Checking furniture assets (this might take a while)...")
		report, err := svc.CheckFurniture(ctx, dbFlag)
		if err != nil {
			logg.Fatal("Furniture check failed", zap.Error(err))
		}

		// Save Report
		filename := fmt.Sprintf("integrity_furniture_%d.json", time.Now().Unix())
		data, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filename, data, 0644); err != nil {
			logg.Error("Failed to save integrity report", zap.Error(err))
		} else {
			logg.Info("Integrity report saved", zap.String("file", filename))
		}

		logg.Info("Furniture Integrity Report",
			zap.Int("Expected", report.TotalExpected),
			zap.Int("Found", report.TotalFound),
			zap.Int("MissingAssets", len(report.MissingAssets)),
			zap.Int("UnregisteredAssets", len(report.UnregisteredAssets)),
			zap.Int("MalformedAssets", len(report.MalformedAssets)),
			zap.Int("ParameterMismatches", len(report.ParameterMismatches)),
			zap.String("ExecutionTime", report.ExecutionTime),
		)
	}

	if runServer {
		logg.Info("Checking server schema integrity...", zap.String("emulator", cfg.Server.Emulator))
		report, err := svc.CheckServer()
		if err != nil {
			logg.Error("Server schema check failed", zap.Error(err))
		} else {
			if report.Matched {
				logg.Info("Server schema matches expected definition.", zap.String("emulator", report.Emulator))
			} else {
				logg.Warn("Server schema mismatches found", zap.String("emulator", report.Emulator))
				for table, tblReport := range report.Tables {
					if tblReport.Status != "ok" {
						if len(tblReport.MissingColumns) > 0 {
							logg.Warn("Missing Columns", zap.String("table", table), zap.Strings("columns", tblReport.MissingColumns))
						}
						if len(tblReport.TypeMismatches) > 0 {
							logg.Warn("Type Mismatches", zap.String("table", table), zap.Strings("mismatches", tblReport.TypeMismatches))
						}
					}
				}
				for _, e := range report.Errors {
					logg.Error("Inspection Error", zap.String("error", e))
				}
			}
		}
	}
}
