package integrity

import (
	"context"

	"asset-manager/core/reconcile"
	"asset-manager/core/storage"
	furnitureAdp "asset-manager/feature/furniture/reconcile"

	"gorm.io/gorm"
)

// ReconcileFurniture performs furniture reconciliation and returns raw results.
// This is exported for CLI use to get detailed reconcile results.
func ReconcileFurniture(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string) ([]reconcile.ReconcileResult, error) {
	// Create furniture adapter and spec
	adapter := furnitureAdp.NewAdapter()
	spec := &reconcile.Spec{
		Adapter:            adapter,
		CacheTTL:           0, // No caching for full scan
		StoragePrefix:      "bundled/furniture",
		StorageExtension:   ".nitro",
		GamedataPaths:      []string{"roomitemtypes.furnitype", "wallitemtypes.furnitype"},
		GamedataObjectName: "gamedata/FurnitureData.json",
		ServerProfile:      emulator,
	}

	// Run reconciliation and return raw results
	return reconcile.ReconcileAll(ctx, spec, db, client, bucket)
}

// ReconcileFurnitureWithPlan performs reconciliation and returns a plan with summary for accurate counting.
func ReconcileFurnitureWithPlan(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string) (*reconcile.ReconcilePlan, error) {
	// Create adapter and spec
	adapter := furnitureAdp.NewAdapter()
	spec := &reconcile.Spec{
		Adapter:            adapter,
		CacheTTL:           0, // No caching for full scan
		StoragePrefix:      "bundled/furniture",
		StorageExtension:   ".nitro",
		GamedataPaths:      []string{"roomitemtypes.furnitype", "wallitemtypes.furnitype"},
		GamedataObjectName: "gamedata/FurnitureData.json",
		ServerProfile:      emulator,
	}

	// Build plan with proper counting
	opts := reconcile.ReconcileOptions{
		DoPurge: false,
		DoSync:  false,
		DryRun:  true,
	}

	return reconcile.ReconcileWithPlan(ctx, spec, db, client, bucket, opts)
}
