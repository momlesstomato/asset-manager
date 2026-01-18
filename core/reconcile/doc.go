// Package reconcile provides a generic, high-performance system for reconciling
// three sources of truth: Database, Gamedata JSON files, and Storage objects.
//
// The reconcile system is designed to handle 60k-100k assets efficiently by:
//   - Building in-memory indices concurrently (DB, Gamedata, Storage)
//   - Using batch operations instead of per-item queries
//   - Providing a caching layer for fast targeted reconciliation
//   - Supporting multiple data models through adapters
//
// # Architecture
//
// The reconcile system consists of three main components:
//
// 1. Engine: Core reconciliation logic that builds a union of keys from all sources,
//    detects presence/absence, and identifies field mismatches.
//
// 2. Adapter: Model-specific implementations that define how to load data from each source,
//    extract keys, and compare fields. Adapters handle variations in DB schemas (server profiles)
//    and gamedata structures (multiple JSON paths).
//
// 3. Cache: TTL-based caching layer with stampede protection for fast targeted queries.
//
// # Performance
//
// Full reconciliation of 60k+ assets completes in a few seconds through:
//   - Concurrent index building (3 goroutines)
//   - Single-pass storage listing (no per-item HEAD calls)
//   - Batch DB queries (no row-by-row iteration)
//   - Efficient union operations over in-memory maps
//
// # Usage Example
//
//	adapter := furniture.NewAdapter(serverProfile)
//	spec := &reconcile.Spec{
//	    Adapter: adapter,
//	    CacheTTL: 5 * time.Minute,
//	}
//
//	// Full reconciliation
//	results, err := reconcile.ReconcileAll(ctx, spec, db, storageClient, bucket)
//
//	// Targeted reconciliation (uses cache)
//	result, err := reconcile.ReconcileOne(ctx, spec, db, storageClient, bucket, query)
//
// # Creating Adapters
//
// To support a new model (e.g., effects, clothing), implement the Adapter interface
// with model-specific logic for loading data, extracting keys, and comparing fields.
// See adapters/furniture for a complete example.
package reconcile
