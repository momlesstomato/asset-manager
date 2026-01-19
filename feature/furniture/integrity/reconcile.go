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
