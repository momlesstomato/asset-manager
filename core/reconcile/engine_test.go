package reconcile

import (
	"context"
	"fmt"
	"testing"
	"time"

	"asset-manager/core/storage"
	"asset-manager/core/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// mockAdapter is a simple test adapter
type mockAdapter struct {
	dbIndex         map[string]DBItem
	gdIndex         map[string]GDItem
	storageSet      map[string]struct{}
	mismatches      map[string][]string
	nameResolver    func(DBItem, GDItem) string
	dbLoadFunc      func(context.Context, *gorm.DB, string) (map[string]DBItem, error)
	gdLoadFunc      func(context.Context, storage.Client, string, string, []string) (map[string]GDItem, error)
	storageLoadFunc func(context.Context, storage.Client, string, string, string) (map[string]struct{}, error)
}

func (m *mockAdapter) Name() string {
	return "mock"
}

// TestBuildCache_ErrorHandling tests that BuildCache correctly handles errors from adapter load functions.
func TestBuildCache_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		dbErr      error
		gdErr      error
		storageErr error
		expectErr  string
	}{
		{
			name:      "DB load error",
			dbErr:     fmt.Errorf("db error"),
			expectErr: "db error",
		},
		{
			name:      "Gamedata load error",
			gdErr:     fmt.Errorf("gamedata error"),
			expectErr: "gamedata error",
		},
		{
			name:       "Storage load error",
			storageErr: fmt.Errorf("storage error"),
			expectErr:  "storage error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &mockAdapter{
				dbLoadFunc: func(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]DBItem, error) {
					if tt.dbErr != nil {
						return nil, tt.dbErr
					}
					return map[string]DBItem{}, nil
				},
				gdLoadFunc: func(ctx context.Context, client storage.Client, bucket, objectName string, paths []string) (map[string]GDItem, error) {
					if tt.gdErr != nil {
						return nil, tt.gdErr
					}
					return map[string]GDItem{}, nil
				},
				storageLoadFunc: func(ctx context.Context, client storage.Client, bucket, prefix, extension string) (map[string]struct{}, error) {
					if tt.storageErr != nil {
						return nil, tt.storageErr
					}
					return map[string]struct{}{}, nil
				},
			}

			spec := &Spec{
				Adapter:  adapter,
				CacheTTL: 5 * time.Minute,
			}

			// Mock client for bucket check
			mockClient := new(mocks.Client)
			// Expect BucketExists check (called by BuildCache for liveness)
			// Return true to allow proceeding to adapter load
			mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

			_, err := BuildCache(context.Background(), spec, nil, mockClient, "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

func (m *mockAdapter) LoadDBIndex(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]DBItem, error) {
	if m.dbLoadFunc != nil {
		return m.dbLoadFunc(ctx, db, serverProfile)
	}
	return m.dbIndex, nil
}

func (m *mockAdapter) LoadGamedataIndex(ctx context.Context, client storage.Client, bucket, objectName string, paths []string) (map[string]GDItem, error) {
	if m.gdLoadFunc != nil {
		return m.gdLoadFunc(ctx, client, bucket, objectName, paths)
	}
	return m.gdIndex, nil
}

func (m *mockAdapter) LoadStorageSet(ctx context.Context, client storage.Client, bucket, prefix, extension string) (map[string]struct{}, error) {
	if m.storageLoadFunc != nil {
		return m.storageLoadFunc(ctx, client, bucket, prefix, extension)
	}
	return m.storageSet, nil
}

func (m *mockAdapter) ExtractDBKey(item DBItem) string {
	return item.(string)
}

func (m *mockAdapter) ExtractGDKey(item GDItem) string {
	return item.(string)
}

func (m *mockAdapter) ExtractStorageKey(objectKey, prefix, extension string) (string, bool) {
	return objectKey, true
}

func (m *mockAdapter) Prepare(ctx context.Context, db *gorm.DB) error {
	return nil
}

func (m *mockAdapter) ResolveName(dbItem DBItem, gdItem GDItem) string {
	if m.nameResolver != nil {
		return m.nameResolver(dbItem, gdItem)
	}
	if dbItem != nil {
		return "db-name"
	}
	if gdItem != nil {
		return "gd-name"
	}
	return ""
}

func (m *mockAdapter) CompareFields(dbItem DBItem, gdItem GDItem) []string {
	key := m.ExtractDBKey(dbItem)
	if mismatches, ok := m.mismatches[key]; ok {
		return mismatches
	}
	return []string{}
}

func (m *mockAdapter) GetMetadata(dbItem DBItem, gdItem GDItem) map[string]string {
	return map[string]string{}
}

func (m *mockAdapter) QueryDB(ctx context.Context, db *gorm.DB, serverProfile string, query Query) (DBItem, error) {
	if item, ok := m.dbIndex[query.ID]; ok {
		return item, nil
	}
	return nil, nil
}

func (m *mockAdapter) QueryGamedata(ctx context.Context, client storage.Client, bucket, objectName string, paths []string, query Query) (GDItem, error) {
	if item, ok := m.gdIndex[query.ID]; ok {
		return item, nil
	}
	return nil, nil
}

func (m *mockAdapter) CheckStorage(ctx context.Context, client storage.Client, bucket, prefix, extension string, key string) (bool, error) {
	_, exists := m.storageSet[key]
	return exists, nil
}

// TestReconcileAll_UnionKeys tests that the union of all keys is built correctly.
func TestReconcileAll_UnionKeys(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"A": "A",
			"B": "B",
		},
		gdIndex: map[string]GDItem{
			"B": "B",
			"C": "C",
		},
		storageSet: map[string]struct{}{
			"C": {},
			"D": {},
		},
		mismatches: map[string][]string{},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0, // No caching for this test
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	results, err := ReconcileAll(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.Len(t, results, 4)

	// Check that all keys are present
	keys := make(map[string]bool)
	for _, r := range results {
		keys[r.ID] = true
	}
	assert.True(t, keys["A"])
	assert.True(t, keys["B"])
	assert.True(t, keys["C"])
	assert.True(t, keys["D"])
}

// TestReconcileAll_PresenceFlags tests that presence flags are set correctly.
func TestReconcileAll_PresenceFlags(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"A": "A",
			"B": "B",
		},
		gdIndex: map[string]GDItem{
			"B": "B",
			"C": "C",
		},
		storageSet: map[string]struct{}{
			"C": {},
			"D": {},
		},
		mismatches: map[string][]string{},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	results, err := ReconcileAll(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)

	// Build a map for easy lookup
	resultMap := make(map[string]ReconcileResult)
	for _, r := range results {
		resultMap[r.ID] = r
	}

	// A: DB only
	assert.True(t, resultMap["A"].DBPresent)
	assert.False(t, resultMap["A"].GamedataPresent)
	assert.False(t, resultMap["A"].StoragePresent)

	// B: DB + Gamedata
	assert.True(t, resultMap["B"].DBPresent)
	assert.True(t, resultMap["B"].GamedataPresent)
	assert.False(t, resultMap["B"].StoragePresent)

	// C: Gamedata + Storage
	assert.False(t, resultMap["C"].DBPresent)
	assert.True(t, resultMap["C"].GamedataPresent)
	assert.True(t, resultMap["C"].StoragePresent)

	// D: Storage only
	assert.False(t, resultMap["D"].DBPresent)
	assert.False(t, resultMap["D"].GamedataPresent)
	assert.True(t, resultMap["D"].StoragePresent)
}

// TestReconcileAll_OrphanDetection tests orphan detection across all sources.
func TestReconcileAll_OrphanDetection(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"db-only": "db-only",
		},
		gdIndex: map[string]GDItem{
			"gd-only": "gd-only",
		},
		storageSet: map[string]struct{}{
			"storage-only": {},
		},
		mismatches: map[string][]string{},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	results, err := ReconcileAll(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	// Build a map for easy lookup
	resultMap := make(map[string]ReconcileResult)
	for _, r := range results {
		resultMap[r.ID] = r
	}

	// Storage-only orphan
	storageOrphan := resultMap["storage-only"]
	assert.True(t, storageOrphan.StoragePresent)
	assert.False(t, storageOrphan.DBPresent)
	assert.False(t, storageOrphan.GamedataPresent)

	// Gamedata-only orphan
	gdOrphan := resultMap["gd-only"]
	assert.True(t, gdOrphan.GamedataPresent)
	assert.False(t, gdOrphan.DBPresent)
	assert.False(t, gdOrphan.StoragePresent)

	// DB-only orphan
	dbOrphan := resultMap["db-only"]
	assert.True(t, dbOrphan.DBPresent)
	assert.False(t, dbOrphan.GamedataPresent)
	assert.False(t, dbOrphan.StoragePresent)
}

// TestReconcileAll_MismatchDetection tests field mismatch detection.
func TestReconcileAll_MismatchDetection(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"item1": "item1",
			"item2": "item2",
		},
		gdIndex: map[string]GDItem{
			"item1": "item1",
			"item2": "item2",
		},
		storageSet: map[string]struct{}{},
		mismatches: map[string][]string{
			"item1": {"sprite_id: gd=0 db=1", "width: gd=1 db=2"},
			"item2": {}, // No mismatches
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	results, err := ReconcileAll(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)

	resultMap := make(map[string]ReconcileResult)
	for _, r := range results {
		resultMap[r.ID] = r
	}

	// item1 should have mismatches
	assert.Len(t, resultMap["item1"].Mismatch, 2)
	assert.Contains(t, resultMap["item1"].Mismatch, "sprite_id: gd=0 db=1")
	assert.Contains(t, resultMap["item1"].Mismatch, "width: gd=1 db=2")

	// item2 should have no mismatches
	assert.Empty(t, resultMap["item2"].Mismatch)
}

// TestCache_Hit tests that cache is reused on second call.
func TestCache_Hit(t *testing.T) {
	loadCount := 0

	adapter := &mockAdapter{
		dbIndex:    map[string]DBItem{"A": "A"},
		gdIndex:    map[string]GDItem{},
		storageSet: map[string]struct{}{},
		mismatches: map[string][]string{},
		dbLoadFunc: func(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]DBItem, error) {
			loadCount++
			return map[string]DBItem{"A": "A"}, nil
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 5 * time.Minute,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	// First call - should build cache
	cache1, err := GetOrBuildCache(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.NotNil(t, cache1)
	assert.Equal(t, 1, loadCount)

	// Second call - should use cached
	cache2, err := GetOrBuildCache(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.NotNil(t, cache2)
	assert.Equal(t, 1, loadCount) // Still 1, not called again

	// Cleanup
	InvalidateCache(spec)
}

// TestCache_Expiration tests that expired cache is rebuilt.
func TestCache_Expiration(t *testing.T) {
	loadCount := 0

	adapter := &mockAdapter{
		dbIndex:    map[string]DBItem{},
		gdIndex:    map[string]GDItem{},
		storageSet: map[string]struct{}{},
		mismatches: map[string][]string{},
		dbLoadFunc: func(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]DBItem, error) {
			loadCount++
			return map[string]DBItem{"A": "A"}, nil
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 10 * time.Millisecond, // Very short TTL
	}

	mockClient := new(mocks.Client)
	// Expect bucket check twice because cache expires and rebuilds
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	// First call
	_, err := GetOrBuildCache(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, loadCount)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Second call - should rebuild
	_, err = GetOrBuildCache(context.Background(), spec, nil, mockClient, "")
	assert.NoError(t, err)
	assert.Equal(t, 2, loadCount) // Called again

	// Cleanup
	InvalidateCache(spec)
}

// TestReconcileOne_WithCache tests targeted reconcile using cache.
func TestReconcileOne_WithCache(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"item1": "item1",
		},
		gdIndex: map[string]GDItem{
			"item1": "item1",
		},
		storageSet: map[string]struct{}{
			"item1": {},
		},
		mismatches: map[string][]string{
			"item1": {"field: gd=a db=b"},
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 5 * time.Minute,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	query := Query{ID: "item1"}
	result, err := ReconcileOne(context.Background(), spec, nil, mockClient, "", query)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "item1", result.ID)
	assert.True(t, result.DBPresent)
	assert.True(t, result.GamedataPresent)
	assert.True(t, result.StoragePresent)
	assert.Len(t, result.Mismatch, 1)

	// Cleanup
	InvalidateCache(spec)
}

// TestReconcileOne_NotFound tests targeted reconcile for missing item.
func TestReconcileOne_NotFound(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex:    map[string]DBItem{},
		gdIndex:    map[string]GDItem{},
		storageSet: map[string]struct{}{},
		mismatches: map[string][]string{},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 5 * time.Minute,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	query := Query{ID: "nonexistent"}
	result, err := ReconcileOne(context.Background(), spec, nil, mockClient, "", query)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.DBPresent)
	assert.False(t, result.GamedataPresent)
	assert.False(t, result.StoragePresent)

	// Cleanup
	InvalidateCache(spec)
}
