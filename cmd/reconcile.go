package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"asset-manager/core/config"
	"asset-manager/core/database"
	"asset-manager/core/logger"
	"asset-manager/core/reconcile"
	"asset-manager/core/storage"
	furnitureReconcile "asset-manager/feature/furniture/reconcile"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// Flags for reconcile furniture command
	purgeFurniture  bool
	syncFurniture   bool
	dryRunFurniture bool
	yesConfirm      bool
)

// reconcileCmd is the parent command for all reconcile operations.
var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile assets between gamedata, database, and storage",
	Long: `Reconcile assets to detect missing items, orphans, and mismatches.
Supports optional purge (delete missing) and sync (repair mismatches) operations.`,
}

// furnitureReconcileCmd performs furniture reconciliation with optional purge/sync.
var furnitureReconcileCmd = &cobra.Command{
	Use:   "furniture",
	Short: "Reconcile furniture assets (report + optionally purge/sync)",
	Long: `Reconcile furniture assets across gamedata, database, and storage.

Reports missing items, orphans, and field mismatches.
Optionally purge (delete) items missing in any store, or sync (repair) mismatches.

Examples:
  # Report only (dry-run)
  reconcile furniture

  # Purge missing items (with interactive confirmation)
  reconcile furniture --purge

  # Purge with auto-confirm (non-interactive)
  reconcile furniture --purge --yes

  # Sync mismatches with auto-confirm
  reconcile furniture --sync --yes

  # Both purge and sync
  reconcile furniture --purge --sync --yes`,
	RunE: runFurnitureReconcile,
}

func init() {
	// Add furniture command to reconcile
	reconcileCmd.AddCommand(furnitureReconcileCmd)

	// Add flags
	furnitureReconcileCmd.Flags().BoolVar(&purgeFurniture, "purge", false, "Enable purge (delete items missing in any store)")
	furnitureReconcileCmd.Flags().BoolVar(&syncFurniture, "sync", false, "Enable sync (update DB fields from gamedata)")
	furnitureReconcileCmd.Flags().BoolVar(&dryRunFurniture, "dry-run", false, "Force dry-run (no mutations even with --yes)")
	furnitureReconcileCmd.Flags().BoolVar(&yesConfirm, "yes", false, "Auto-confirm destructive actions (non-interactive)")

	// Add reconcile to root
	RootCmd.AddCommand(reconcileCmd)
}

func runFurnitureReconcile(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	//Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	l, err := logger.New(&cfg.Log)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	l.Info("Starting furniture reconciliation")

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connect to storage
	client, err := storage.NewClient(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to connect to storage: %w", err)
	}

	// Create furniture adapter
	adapter := furnitureReconcile.NewAdapter()

	// Set mutation context for purge/sync
	if purgeFurniture || syncFurniture {
		adapter.SetMutationContext(
			db,
			client,
			cfg.Storage.Bucket,
			"bundled/furniture",
			cfg.Server.Emulator,
			"gamedata/FurnitureData.json",
		)
	}

	// Build spec
	spec := &reconcile.Spec{
		Adapter:            adapter,
		CacheTTL:           0, // No caching to prevent stale data after DB changes
		StoragePrefix:      "bundled/furniture",
		StorageExtension:   ".nitro",
		GamedataPaths:      []string{}, // Not used, loads full JSON
		GamedataObjectName: "gamedata/FurnitureData.json",
		ServerProfile:      cfg.Server.Emulator,
	}

	// Build reconcile options
	opts := reconcile.ReconcileOptions{
		DoPurge:   purgeFurniture,
		DoSync:    syncFurniture,
		DryRun:    dryRunFurniture,
		Confirmed: false, // Will be set after confirmation prompt
	}

	// Step 0: Prepare Schema (Auto-fix limits)
	// This ensures database columns are large enough for gamedata values.
	if err := adapter.Prepare(ctx, db); err != nil {
		return fmt.Errorf("failed to prepare schema: %w", err)
	}

	// Step 1: Plan (always runs)
	l.Info("Planning reconciliation...")
	plan, err := reconcile.ReconcileWithPlan(ctx, spec, db, client, cfg.Storage.Bucket, opts)
	if err != nil {
		return fmt.Errorf("failed to plan reconciliation: %w", err)
	}

	// Step 2: Print report
	printReconcileReport(l, plan)

	// Step 3: Check if actions are requested
	if !purgeFurniture && !syncFurniture {
		l.Info("No actions requested. Use --purge to delete incomplete items or --sync to repair mismatches.")
		return nil
	}

	// Step 4: Apply (if confirmed)
	if !dryRunFurniture {
		numberActions := len(plan.Actions)
		if numberActions == 0 {
			l.Info("No actions required based on current flags.")
			return nil
		}

		// Check confirmation
		confirmed := confirmDestructiveAction()
		if !confirmed {
			l.Warn("Operation cancelled by user. No changes were made.")
			return nil
		}

		opts.Confirmed = true

		// Execute actions
		l.Info("Applying actions...")
		executed, err := reconcile.ApplyPlan(ctx, spec, db, client, cfg.Storage.Bucket, plan, opts)
		if err != nil {
			return fmt.Errorf("failed to apply plan: %w", err)
		}

		l.Info("Successfully executed actions", zap.Int("count", executed))
	} else {
		l.Info("Dry-run mode: No changes were made.")
	}

	return nil
}

// printReconcileReport prints a formatted reconciliation report using logger.
func printReconcileReport(l *zap.Logger, plan *reconcile.ReconcilePlan) {
	s := plan.Summary

	l.Info("Reconciliation report",
		zap.Int("total_items", s.TotalItems),
		zap.Int("missing_gamedata", s.MissingGamedata),
		zap.Int("missing_storage", s.MissingStorage),
		zap.Int("missing_db", s.MissingDB),
		zap.Int("mismatches", s.Mismatches),
	)

	if len(plan.Actions) > 0 {
		l.Info("Planned actions",
			zap.Int("purge_actions", s.PurgeActions),
			zap.Int("sync_actions", s.SyncActions),
			zap.Int("total_actions", len(plan.Actions)),
		)

		// Show sample of actions (max 5 for logger)
		if len(plan.Actions) > 0 {
			maxShow := 5
			if len(plan.Actions) < maxShow {
				maxShow = len(plan.Actions)
			}
			for i := 0; i < maxShow; i++ {
				action := plan.Actions[i]
				l.Info("Sample action",
					zap.String("type", string(action.Type)),
					zap.String("key", action.Key),
					zap.String("reason", action.Reason),
				)
			}
			if len(plan.Actions) > maxShow {
				l.Info("Additional actions not shown", zap.Int("count", len(plan.Actions)-maxShow))
			}
		}
	}
}

// confirmDestructiveAction prompts the user for confirmation or uses --yes flag.
func confirmDestructiveAction() bool {
	if yesConfirm {
		fmt.Println("\n✓ Auto-confirmed via --yes flag")
		return true
	}

	fmt.Print("\n⚠️  Type 'yes' to confirm destructive actions: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(response)
	return response == "yes"
}
