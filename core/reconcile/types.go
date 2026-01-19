package reconcile

import "time"

// ReconcileResult represents the reconciliation output for a single entity.
// It contains presence flags for each source and any detected mismatches.
type ReconcileResult struct {
	// ID is the unique identifier for the entity.
	ID string `json:"id"`

	// Name is the display name of the entity.
	Name string `json:"name"`

	// DBPresent indicates whether the entity exists in the database.
	DBPresent bool `json:"db_present"`

	// StoragePresent indicates whether the entity exists in storage.
	StoragePresent bool `json:"storage_present"`

	// GamedataPresent indicates whether the entity exists in gamedata JSON.
	GamedataPresent bool `json:"gamedata_present"`

	// Mismatch contains descriptions of field mismatches between DB and gamedata.
	// Each string describes a specific mismatch, e.g., "sprite_id: gd=0 db=1".
	Mismatch []string `json:"mismatch"`

	// Metadata contains model-specific arbitrary data (e.g., classname, category).
	Metadata map[string]string `json:"metadata"`
}

// Query represents a search query for targeted reconciliation.
// The adapter decides how to translate query fields into lookups.
type Query struct {
	// ID is the entity ID to search for.
	ID string

	// Name is the entity name to search for.
	Name string

	// Classname is the entity classname to search for.
	Classname string
}

// Spec defines the configuration for a reconciliation operation.
// It bundles the adapter, cache settings, and data source parameters.
type Spec struct {
	// Adapter provides model-specific reconciliation logic.
	Adapter Adapter

	// CacheTTL is the time-to-live for cached indices.
	// If zero, caching is disabled.
	CacheTTL time.Duration

	// StoragePrefix is the prefix under which to list storage objects.
	StoragePrefix string

	// StorageExtension is the file extension to filter storage objects.
	StorageExtension string

	// GamedataPaths is the list of JSON paths to search for gamedata items.
	// Example: ["roomitemtypes.furnitype", "roomitemtypes.wallitemtypes"]
	GamedataPaths []string

	// GamedataObjectName is the name of the gamedata JSON object in storage.
	// Example: "gamedata/FurnitureData.json"
	GamedataObjectName string

	// ServerProfile is the emulator-specific configuration (e.g., "arcturus", "comet").
	ServerProfile string
}

// CacheKey returns a unique key for caching based on spec parameters.
// This ensures different models/configs don't share the same cache.
func (s *Spec) CacheKey() string {
	// Simple concatenation for now; could use a hash for efficiency
	key := s.Adapter.Name() + "|" + s.ServerProfile + "|" + s.StoragePrefix + "|" + s.StorageExtension
	for _, path := range s.GamedataPaths {
		key += "|" + path
	}
	return key
}

// DBItem represents a database entity with arbitrary fields.
// Adapters define the concrete type and provide a way to extract this.
type DBItem any

// GDItem represents a gamedata JSON entity with arbitrary fields.
// Adapters define the concrete type and provide a way to extract this.
type GDItem any
