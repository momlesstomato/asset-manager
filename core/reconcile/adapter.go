package reconcile

import (
	"context"

	"asset-manager/core/storage"

	"gorm.io/gorm"
)

// Adapter defines the interface for model-specific reconciliation logic.
// Each adapter implements how to load, index, and compare data for a specific model
// (e.g., furniture, effects, clothing).
type Adapter interface {
	// Name returns the unique name of this adapter (e.g., "furniture", "effects").
	Name() string

	// LoadDBIndex loads all relevant DB items and returns them indexed by entity key.
	// The serverProfile parameter specifies the emulator type (e.g., "arcturus", "comet").
	// Implementations should use batch queries to load minimal columns efficiently.
	LoadDBIndex(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]DBItem, error)

	// LoadGamedataIndex loads gamedata items from the specified JSON paths and returns
	// them indexed by entity key. The paths parameter contains JSON dot-notation paths
	// to arrays (e.g., "roomitemtypes.furnitype").
	// Implementations should parse the JSON once and merge all arrays into a single index.
	LoadGamedataIndex(ctx context.Context, client storage.Client, bucket, objectName string, paths []string) (map[string]GDItem, error)

	// LoadStorageSet lists all storage objects under the given prefix, filtered by extension,
	// and returns a set of entity keys. Implementations should use paginated listing
	// and avoid per-item HEAD calls.
	LoadStorageSet(ctx context.Context, client storage.Client, bucket, prefix, extension string) (map[string]struct{}, error)

	// ExtractDBKey returns the entity key from a DB item.
	// The key is used to build the union and match items across sources.
	ExtractDBKey(item DBItem) string

	// ExtractGDKey returns the entity key from a gamedata item.
	ExtractGDKey(item GDItem) string

	// ExtractStorageKey parses a storage object key and returns the entity key.
	// If the object key doesn't match the expected pattern, ok should be false.
	// Example: "bundled/furniture/chair.nitro" -> ("1", true) if ID is 1 for chair.
	ExtractStorageKey(objectKey, extension string) (key string, ok bool)

	// ResolveName returns the display name for an entity given available DB and/or gamedata items.
	// Either item may be nil if not present in that source.
	ResolveName(dbItem DBItem, gdItem GDItem) string

	// CompareFields compares mapped fields between DB and gamedata items and returns
	// a list of mismatch descriptions. Each string should include the field label and
	// both values (e.g., "sprite_id: gd=0 db=1").
	// Both items are guaranteed to be non-nil when this is called.
	CompareFields(dbItem DBItem, gdItem GDItem) []string

	// QueryDB performs a targeted database lookup based on the query parameters.
	// This is used for fast targeted reconciliation without building the full index.
	// Returns nil if no match is found.
	QueryDB(ctx context.Context, db *gorm.DB, serverProfile string, query Query) (DBItem, error)

	// QueryGamedata performs a targeted gamedata lookup based on the query parameters.
	// This is used for fast targeted reconciliation without parsing the full JSON.
	// Returns nil if no match is found.
	// Note: For performance, this may still require parsing the full JSON file,
	// so using cached indices is preferred for repeated queries.
	QueryGamedata(ctx context.Context, client storage.Client, bucket, objectName string, paths []string, query Query) (GDItem, error)

	// CheckStorage checks if a specific entity exists in storage.
	// This is used for fast targeted reconciliation without listing all objects.
	// Returns true if the entity's storage object exists.
	CheckStorage(ctx context.Context, client storage.Client, bucket, prefix, extension string, key string) (bool, error)

	// GetMetadata returns model-specific metadata (e.g., classname, category) for the entity.
	// This data is included in the ReconcileResult.
	GetMetadata(dbItem DBItem, gdItem GDItem) map[string]string
}
