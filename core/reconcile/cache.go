package reconcile

import (
	"context"
	"sync"
	"time"

	"asset-manager/core/storage"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// ReconcileCache holds pre-built indices for fast targeted reconciliation.
type ReconcileCache struct {
	// DBIndex is the indexed map of database items by entity key.
	DBIndex map[string]DBItem

	// GDIndex is the indexed map of gamedata items by entity key.
	GDIndex map[string]GDItem

	// StorageSet is the set of entity keys present in storage.
	StorageSet map[string]struct{}

	// Built is the timestamp when this cache was built.
	Built time.Time

	// TTL is the time-to-live for this cache.
	TTL time.Duration
}

// IsExpired returns true if this cache has expired based on its TTL.
func (c *ReconcileCache) IsExpired() bool {
	if c.TTL == 0 {
		return true // No caching
	}
	return time.Since(c.Built) > c.TTL
}

// cacheStore holds all reconcile caches keyed by spec cache key.
type cacheStore struct {
	mu     sync.RWMutex
	caches map[string]*ReconcileCache
	sf     singleflight.Group
}

// globalCacheStore is the singleton cache store for all reconcile operations.
var globalCacheStore = &cacheStore{
	caches: make(map[string]*ReconcileCache),
}

// BuildCache builds a new cache for the given spec by loading all indices.
// This function does NOT store the cache; use GetOrBuildCache for that.
func BuildCache(ctx context.Context, spec *Spec, db *gorm.DB, client storage.Client, bucket string) (*ReconcileCache, error) {
	var (
		dbIndex    map[string]DBItem
		gdIndex    map[string]GDItem
		storageSet map[string]struct{}
		dbErr      error
		gdErr      error
		storageErr error
		wg         sync.WaitGroup
	)

	// Build indices concurrently
	wg.Add(3)

	// Build DB index
	go func() {
		defer wg.Done()
		dbIndex, dbErr = spec.Adapter.LoadDBIndex(ctx, db, spec.ServerProfile)
	}()

	// Build gamedata index
	go func() {
		defer wg.Done()
		gdIndex, gdErr = spec.Adapter.LoadGamedataIndex(ctx, client, bucket, spec.GamedataObjectName, spec.GamedataPaths)
	}()

	// Build storage set
	go func() {
		defer wg.Done()
		storageSet, storageErr = spec.Adapter.LoadStorageSet(ctx, client, bucket, spec.StoragePrefix, spec.StorageExtension)
	}()

	wg.Wait()

	// Check for errors
	if dbErr != nil {
		return nil, dbErr
	}
	if gdErr != nil {
		return nil, gdErr
	}
	if storageErr != nil {
		return nil, storageErr
	}

	return &ReconcileCache{
		DBIndex:    dbIndex,
		GDIndex:    gdIndex,
		StorageSet: storageSet,
		Built:      time.Now(),
		TTL:        spec.CacheTTL,
	}, nil
}

// GetOrBuildCache retrieves a cache for the given spec from the store,
// or builds a new one if it doesn't exist or has expired.
// Uses singleflight to prevent cache stampedes.
func GetOrBuildCache(ctx context.Context, spec *Spec, db *gorm.DB, client storage.Client, bucket string) (*ReconcileCache, error) {
	cacheKey := spec.CacheKey()

	// Fast path: check if cache exists and is fresh
	globalCacheStore.mu.RLock()
	cache, exists := globalCacheStore.caches[cacheKey]
	globalCacheStore.mu.RUnlock()

	if exists && !cache.IsExpired() {
		return cache, nil
	}

	// Slow path: build cache using singleflight to prevent stampedes
	result, err, _ := globalCacheStore.sf.Do(cacheKey, func() (interface{}, error) {
		// Double-check after acquiring singleflight lock
		globalCacheStore.mu.RLock()
		cache, exists := globalCacheStore.caches[cacheKey]
		globalCacheStore.mu.RUnlock()

		if exists && !cache.IsExpired() {
			return cache, nil
		}

		// Build new cache
		newCache, err := BuildCache(ctx, spec, db, client, bucket)
		if err != nil {
			return nil, err
		}

		// Store in cache
		globalCacheStore.mu.Lock()
		globalCacheStore.caches[cacheKey] = newCache
		globalCacheStore.mu.Unlock()

		return newCache, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*ReconcileCache), nil
}

// InvalidateCache removes the cache for the given spec from the store.
// This is useful for testing or forcing a rebuild.
func InvalidateCache(spec *Spec) {
	cacheKey := spec.CacheKey()
	globalCacheStore.mu.Lock()
	delete(globalCacheStore.caches, cacheKey)
	globalCacheStore.mu.Unlock()
}
