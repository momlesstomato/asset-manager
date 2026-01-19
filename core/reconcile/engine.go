package reconcile

import (
	"context"
	"sort"

	"asset-manager/core/storage"

	"gorm.io/gorm"
)

// ReconcileAll performs a full reconciliation across all entities.
// It builds indices from all three sources, computes the union of keys,
// and returns a result for each key indicating presence and mismatches.
func ReconcileAll(ctx context.Context, spec *Spec, db *gorm.DB, client storage.Client, bucket string) ([]ReconcileResult, error) {
	// Build cache (which loads all indices concurrently)
	cache, err := BuildCache(ctx, spec, db, client, bucket)
	if err != nil {
		return nil, err
	}

	// Build union of all keys
	unionKeys := buildUnion(cache.DBIndex, cache.GDIndex, cache.StorageSet, spec.Adapter)

	// Build results for each key
	results := make([]ReconcileResult, 0, len(unionKeys))
	for key := range unionKeys {
		result := buildResult(key, cache.DBIndex, cache.GDIndex, cache.StorageSet, spec.Adapter)
		results = append(results, result)
	}

	// Sort results by key for deterministic output
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

// ReconcileOne performs a targeted reconciliation for a single entity.
// It uses cached indices if available, or performs targeted queries.
func ReconcileOne(ctx context.Context, spec *Spec, db *gorm.DB, client storage.Client, bucket string, query Query) (*ReconcileResult, error) {
	// Try to use cache if enabled
	if spec.CacheTTL > 0 {
		cache, err := GetOrBuildCache(ctx, spec, db, client, bucket)
		if err != nil {
			return nil, err
		}

		// Find the key from the query
		key := findKeyFromQuery(query, cache.DBIndex, cache.GDIndex, spec.Adapter)
		if key == "" {
			// Not found in cache
			return &ReconcileResult{
				ID:              query.ID,
				DBPresent:       false,
				StoragePresent:  false,
				GamedataPresent: false,
			}, nil
		}

		result := buildResult(key, cache.DBIndex, cache.GDIndex, cache.StorageSet, spec.Adapter)
		return &result, nil
	}

	// Fast path without cache: use targeted queries
	dbItem, err := spec.Adapter.QueryDB(ctx, db, spec.ServerProfile, query)
	if err != nil {
		return nil, err
	}

	gdItem, err := spec.Adapter.QueryGamedata(ctx, client, bucket, spec.GamedataObjectName, spec.GamedataPaths, query)
	if err != nil {
		return nil, err
	}

	// For storage, we need a key to check
	var key string
	if dbItem != nil {
		key = spec.Adapter.ExtractDBKey(dbItem)
	} else if gdItem != nil {
		key = spec.Adapter.ExtractGDKey(gdItem)
	} else {
		// No DB or GD item, use query ID as key
		key = query.ID
	}

	storagePresent := false
	if key != "" {
		storagePresent, err = spec.Adapter.CheckStorage(ctx, client, bucket, spec.StoragePrefix, spec.StorageExtension, key)
		if err != nil {
			return nil, err
		}
	}

	result := ReconcileResult{
		ID:              key,
		Name:            spec.Adapter.ResolveName(dbItem, gdItem),
		Metadata:        spec.Adapter.GetMetadata(dbItem, gdItem),
		DBPresent:       dbItem != nil,
		GamedataPresent: gdItem != nil,
		StoragePresent:  storagePresent,
		Mismatch:        []string{},
	}

	if dbItem != nil && gdItem != nil {
		result.Mismatch = spec.Adapter.CompareFields(dbItem, gdItem)
	}

	return &result, nil
}

// buildUnion creates a union of all keys from DB, gamedata, and storage.
func buildUnion(dbIndex map[string]DBItem, gdIndex map[string]GDItem, storageSet map[string]struct{}, adapter Adapter) map[string]struct{} {
	union := make(map[string]struct{})

	// Add DB keys
	for key := range dbIndex {
		union[key] = struct{}{}
	}

	// Add gamedata keys
	for key := range gdIndex {
		union[key] = struct{}{}
	}

	// Add storage keys
	for key := range storageSet {
		union[key] = struct{}{}
	}

	return union
}

// buildResult creates a ReconcileResult for a single key.
func buildResult(key string, dbIndex map[string]DBItem, gdIndex map[string]GDItem, storageSet map[string]struct{}, adapter Adapter) ReconcileResult {
	dbItem, dbPresent := dbIndex[key]
	gdItem, gdPresent := gdIndex[key]
	_, storagePresent := storageSet[key]

	result := ReconcileResult{
		ID:              key,
		DBPresent:       dbPresent,
		GamedataPresent: gdPresent,
		StoragePresent:  storagePresent,
		Mismatch:        []string{},
	}

	// Resolve name and metadata
	if dbPresent || gdPresent {
		var dbItemPtr DBItem
		var gdItemPtr GDItem
		if dbPresent {
			dbItemPtr = dbItem
		}
		if gdPresent {
			gdItemPtr = gdItem
		}
		result.Name = adapter.ResolveName(dbItemPtr, gdItemPtr)
		result.Metadata = adapter.GetMetadata(dbItemPtr, gdItemPtr)
	}

	// Compare fields if both present
	if dbPresent && gdPresent {
		result.Mismatch = adapter.CompareFields(dbItem, gdItem)
	}

	return result
}

// findKeyFromQuery attempts to find the entity key from a query using cached indices.
func findKeyFromQuery(query Query, dbIndex map[string]DBItem, gdIndex map[string]GDItem, adapter Adapter) string {
	// Try direct key match first
	if query.ID != "" {
		if _, exists := dbIndex[query.ID]; exists {
			return query.ID
		}
		if _, exists := gdIndex[query.ID]; exists {
			return query.ID
		}
	}

	// Search by name or classname in DB index
	if query.Name != "" || query.Classname != "" {
		for key, item := range dbIndex {
			// This is adapter-specific; we can't generically compare here
			// For now, just return the first match by key if ID matches
			_ = item
			if key == query.Name || key == query.Classname {
				return key
			}
		}
	}

	// Search in gamedata index
	if query.Name != "" || query.Classname != "" {
		for key, item := range gdIndex {
			_ = item
			if key == query.Name || key == query.Classname {
				return key
			}
		}
	}

	return ""
}
