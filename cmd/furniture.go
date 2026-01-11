package cmd

import (
	"context"
	"fmt"
	"os"

	"asset-manager/core/config"
	"asset-manager/core/database"
	"asset-manager/core/logger"
	"asset-manager/core/storage"
	"asset-manager/feature/furniture"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// furnitureDetailCmd represents the top-level furniture command
var furnitureDetailCmd = &cobra.Command{
	Use:   "furniture [identifier]",
	Short: "View details and validity of a furniture item",
	Long:  `Checks the presence and matching parameters of a furniture item across FurniData, Database, and Storage.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runFurnitureDetailCheck(cmd.Context(), args[0])
	},
}

func init() {
	RootCmd.AddCommand(furnitureDetailCmd)
}

func runFurnitureDetailCheck(ctx context.Context, identifier string) {
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

	svc := furniture.NewService(store, cfg.Storage.Bucket, logg, db, cfg.Server.Emulator)

	logg.Info("Checking furniture item...", zap.String("identifier", identifier))
	report, err := svc.GetFurnitureDetail(ctx, identifier)
	if err != nil {
		logg.Fatal("Furniture detail check failed", zap.Error(err))
	}

	// Pretty Console Output
	fmt.Println("\n--- Furniture Detail View ---")
	fmt.Printf("Query:          %s\n", identifier)
	fmt.Printf("ID:             %d\n", report.ID)
	fmt.Printf("Name:           %s\n", report.Name)
	fmt.Printf("ClassName:      %s\n", report.ClassName)
	fmt.Printf("Nitro File:     %s\n", report.NitroFile)
	fmt.Println("-----------------------------")
	fmt.Printf("In FurniData:   %v\n", report.InFurniData)
	fmt.Printf("In Database:    %v\n", report.InDB)
	fmt.Printf("File Exists:    %v\n", report.FileExists)

	statusColor := "\033[32m" // Green
	if report.IntegrityStatus == "FAIL" {
		statusColor = "\033[31m" // Red
	} else if report.IntegrityStatus == "WARNING" {
		statusColor = "\033[33m" // Yellow
	}
	resetColor := "\033[0m"

	fmt.Printf("Integrity:      %s%s%s\n", statusColor, report.IntegrityStatus, resetColor)

	if len(report.Mismatches) > 0 {
		fmt.Println("\nMismatches/Errors:")
		for _, m := range report.Mismatches {
			fmt.Printf("- %s\n", m)
		}
	}
	fmt.Println("-----------------------------")
}
