package reconcile

import (
	"context"
	"fmt"

	"asset-manager/core/storage"

	"gorm.io/gorm"
)

// ReconcileWithPlan performs reconciliation and returns a plan with results and actions.
// It does NOT execute actions; use ApplyPlan for that.
// This function builds indices once and generates both reconciliation results and planned actions.
func ReconcileWithPlan(
	ctx context.Context,
	spec *Spec,
	db *gorm.DB,
	client storage.Client,
	bucket string,
	opts ReconcileOptions,
) (*ReconcilePlan, error) {
	// Build cache (which loads all indices concurrently)
	cache, err := GetOrBuildCache(ctx, spec, db, client, bucket)
	if err != nil {
		return nil, err
	}

	// Build results using existing reconcile logic
	results, err := reconcileFromCache(cache, spec.Adapter)
	if err != nil {
		return nil, err
	}

	// Build summary and actions
	summary, actions := buildPlanFromResults(results, cache, spec.Adapter, opts)

	return &ReconcilePlan{
		Results: results,
		Actions: actions,
		Summary: summary,
	}, nil
}

// ApplyPlan executes the actions in a reconcile plan.
// Returns the number of actions executed and any error encountered.
// Requires opts.Confirmed=true and opts.DryRun=false to actually execute.
func ApplyPlan(
	ctx context.Context,
	spec *Spec,
	db *gorm.DB,
	client storage.Client,
	bucket string,
	plan *ReconcilePlan,
	opts ReconcileOptions,
) (executed int, err error) {
	// Safety check: do not execute if not confirmed or dry-run
	if !opts.Confirmed || opts.DryRun {
		return 0, nil
	}

	// Check if adapter implements Mutator
	mutator, ok := spec.Adapter.(Mutator)
	if !ok {
		return 0, fmt.Errorf("adapter %s does not implement Mutator interface", spec.Adapter.Name())
	}

	// Group actions by type for efficient execution
	var (
		deleteDBKeys       []string
		deleteGamedataKeys []string
		deleteStorageKeys  []string
		syncActions        []Action
	)

	for _, action := range plan.Actions {
		switch action.Type {
		case ActionDeleteDB:
			deleteDBKeys = append(deleteDBKeys, action.Key)
		case ActionDeleteGamedata:
			deleteGamedataKeys = append(deleteGamedataKeys, action.Key)
		case ActionDeleteStorage:
			deleteStorageKeys = append(deleteStorageKeys, action.Key)
		case ActionSyncDB:
			syncActions = append(syncActions, action)
		}
	}

	// Execute deletions (purge actions) using batch methods if available

	// DB deletions
	if len(deleteDBKeys) > 0 {
		// Try batch delete first
		type DBBatchDeleter interface {
			DeleteDBBatch(ctx context.Context, keys []string) error
		}
		if batchDeleter, ok := mutator.(DBBatchDeleter); ok {
			if err := batchDeleter.DeleteDBBatch(ctx, deleteDBKeys); err != nil {
				return executed, fmt.Errorf("failed to batch delete DB keys: %w", err)
			}
			executed += len(deleteDBKeys)
		} else {
			// Fallback to one-at-a-time
			for _, key := range deleteDBKeys {
				if err := mutator.DeleteDB(ctx, key); err != nil {
					return executed, fmt.Errorf("failed to delete DB key %s: %w", key, err)
				}
				executed++
			}
		}
	}

	// Gamedata deletions
	if len(deleteGamedataKeys) > 0 {
		// Try batch delete first
		type GDBatchDeleter interface {
			DeleteGamedataBatch(ctx context.Context, keys []string) error
		}
		if batchDeleter, ok := mutator.(GDBatchDeleter); ok {
			if err := batchDeleter.DeleteGamedataBatch(ctx, deleteGamedataKeys); err != nil {
				return executed, fmt.Errorf("failed to batch delete gamedata keys: %w", err)
			}
			executed += len(deleteGamedataKeys)
		} else {
			// Fallback to one-at-a-time
			for _, key := range deleteGamedataKeys {
				if err := mutator.DeleteGamedata(ctx, key); err != nil {
					return executed, fmt.Errorf("failed to delete gamedata key %s: %w", key, err)
				}
				executed++
			}
		}
	}

	// Storage deletions
	if len(deleteStorageKeys) > 0 {
		// Try batch delete first
		type StorageBatchDeleter interface {
			DeleteStorageBatch(ctx context.Context, keys []string) error
		}
		if batchDeleter, ok := mutator.(StorageBatchDeleter); ok {
			if err := batchDeleter.DeleteStorageBatch(ctx, deleteStorageKeys); err != nil {
				return executed, fmt.Errorf("failed to batch delete storage keys: %w", err)
			}
			executed += len(deleteStorageKeys)
		} else {
			// Fallback to one-at-a-time
			for _, key := range deleteStorageKeys {
				if err := mutator.DeleteStorage(ctx, key); err != nil {
					return executed, fmt.Errorf("failed to delete storage key %s: %w", key, err)
				}
				executed++
			}
		}
	}

	// Execute syncs
	if len(syncActions) > 0 {
		// Try batch sync first
		type SyncBatcher interface {
			SyncDBBatch(ctx context.Context, actions []Action) error
		}
		if batchSyncer, ok := mutator.(SyncBatcher); ok {
			if err := batchSyncer.SyncDBBatch(ctx, syncActions); err != nil {
				return executed, fmt.Errorf("failed to batch sync DB: %w", err)
			}
			executed += len(syncActions)
		} else {
			// Fallback to one-at-a-time
			for _, action := range syncActions {
				if err := mutator.SyncDBFromGamedata(ctx, action.Key, action.GDItem); err != nil {
					return executed, fmt.Errorf("failed to sync key %s: %w", action.Key, err)
				}
				executed++
			}
		}
	}

	return executed, nil
}

// ReconcileAndApply is a convenience wrapper that plans and optionally applies actions.
// It returns the plan, number of actions executed, and any error.
func ReconcileAndApply(
	ctx context.Context,
	spec *Spec,
	db *gorm.DB,
	client storage.Client,
	bucket string,
	opts ReconcileOptions,
) (*ReconcilePlan, int, error) {
	plan, err := ReconcileWithPlan(ctx, spec, db, client, bucket, opts)
	if err != nil {
		return nil, 0, err
	}

	executed, err := ApplyPlan(ctx, spec, db, client, bucket, plan, opts)
	return plan, executed, err
}

// reconcileFromCache builds results from a cache (extracted from ReconcileAll logic).
func reconcileFromCache(cache *ReconcileCache, adapter Adapter) ([]ReconcileResult, error) {
	// Build union of all keys
	unionKeys := buildUnion(cache.DBIndex, cache.GDIndex, cache.StorageSet, adapter)

	// Build results for each key
	results := make([]ReconcileResult, 0, len(unionKeys))
	for key := range unionKeys {
		result := buildResult(key, cache.DBIndex, cache.GDIndex, cache.StorageSet, adapter)
		results = append(results, result)
	}

	return results, nil
}

// buildPlanFromResults generates a summary and action plan from reconciliation results.
func buildPlanFromResults(results []ReconcileResult, cache *ReconcileCache, adapter Adapter, opts ReconcileOptions) (PlanSummary, []Action) {
	var summary PlanSummary
	var actions []Action

	summary.TotalItems = len(results)

	for _, result := range results {
		// Count incomplete items using correct OR semantics:
		// - storage_missing: Items in (DB OR gamedata) that don't have .nitro file
		// - gamedata_missing: Items in (DB OR storage) that don't have gamedata
		// - db_missing: Items in (gamedata OR storage) that don't have DB

		// storage_missing: in (DB OR gamedata) but NOT in storage
		if (result.DBPresent || result.GamedataPresent) && !result.StoragePresent {
			summary.MissingStorage++
		}

		// gamedata_missing: in (DB OR storage) but NOT in gamedata
		if (result.DBPresent || result.StoragePresent) && !result.GamedataPresent {
			summary.MissingGamedata++
		}

		// db_missing: in (gamedata OR storage) but NOT in DB
		if (result.GamedataPresent || result.StoragePresent) && !result.DBPresent {
			summary.MissingDB++
		}

		// Count mismatches
		if len(result.Mismatch) > 0 {
			summary.Mismatches++
		}

		// Plan purge actions: delete if missing in ANY store
		if opts.DoPurge {
			missingInAny := !result.GamedataPresent || !result.StoragePresent || !result.DBPresent
			if missingInAny {
				// Delete from all stores
				if result.DBPresent {
					actions = append(actions, Action{
						Type:   ActionDeleteDB,
						Key:    result.ID,
						Reason: getMissingReason(result),
					})
					summary.PurgeActions++
				}
				if result.GamedataPresent {
					actions = append(actions, Action{
						Type:   ActionDeleteGamedata,
						Key:    result.ID,
						Reason: getMissingReason(result),
					})
					summary.PurgeActions++
				}
				if result.StoragePresent {
					actions = append(actions, Action{
						Type:   ActionDeleteStorage,
						Key:    result.ID,
						Reason: getMissingReason(result),
					})
					summary.PurgeActions++
				}
				// Purge takes precedence: skip sync for this item
				continue
			}
		}

		// Plan sync actions: update DB from gamedata if mismatches exist
		if opts.DoSync && len(result.Mismatch) > 0 {
			if result.DBPresent && result.GamedataPresent {
				gdItem := cache.GDIndex[result.ID]
				actions = append(actions, Action{
					Type:   ActionSyncDB,
					Key:    result.ID,
					Reason: fmt.Sprintf("mismatch: %v", result.Mismatch),
					GDItem: gdItem,
				})
				summary.SyncActions++
			}
		}
	}

	return summary, actions
}

// getMissingReason builds a reason string for why an entity should be purged.
func getMissingReason(result ReconcileResult) string {
	var missing []string
	if !result.GamedataPresent {
		missing = append(missing, "gamedata")
	}
	if !result.StoragePresent {
		missing = append(missing, "storage")
	}
	if !result.DBPresent {
		missing = append(missing, "database")
	}

	if len(missing) == 0 {
		return "complete"
	}
	return fmt.Sprintf("missing in: %v", missing)
}
