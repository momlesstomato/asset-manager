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

// ActionType represents the type of mutation action.
type ActionType string

const (
	// ActionDeleteDB deletes an entity from the database.
	ActionDeleteDB ActionType = "delete_db"
	// ActionDeleteGamedata deletes an entity from gamedata JSON.
	ActionDeleteGamedata ActionType = "delete_gamedata"
	// ActionDeleteStorage deletes an entity from storage.
	ActionDeleteStorage ActionType = "delete_storage"
	// ActionSyncDB syncs database fields from gamedata.
	ActionSyncDB ActionType = "sync_db"
)

// Action represents a planned mutation operation.
type Action struct {
	// Type specifies the action to perform.
	Type ActionType `json:"type"`

	// Key is the entity identifier.
	Key string `json:"key"`

	// Reason explains why this action is needed.
	Reason string `json:"reason"`

	// GDItem stores the gamedata source for sync actions.
	// Only populated for ActionSyncDB.
	GDItem GDItem `json:"-"`
}

// ReconcilePlan contains reconciliation results and planned actions.
type ReconcilePlan struct {
	// Results contains per-entity reconciliation data.
	Results []ReconcileResult `json:"results"`

	// Actions contains planned mutation operations.
	Actions []Action `json:"actions"`

	// Summary provides aggregate counts.
	Summary PlanSummary `json:"summary"`
}

// PlanSummary provides aggregate statistics for a reconcile plan.
type PlanSummary struct {
	// TotalItems is the total number of unique entities.
	TotalItems int `json:"total_items"`

	// MissingGamedata counts entities missing in gamedata.
	MissingGamedata int `json:"missing_gamedata"`

	// MissingStorage counts entities missing in storage.
	MissingStorage int `json:"missing_storage"`

	// MissingDB counts entities missing in database.
	MissingDB int `json:"missing_db"`

	// Mismatches counts entities with field discrepancies.
	Mismatches int `json:"mismatches"`

	// PurgeActions counts planned purge (delete) actions.
	PurgeActions int `json:"purge_actions"`

	// SyncActions counts planned sync (update) actions.
	SyncActions int `json:"sync_actions"`
}

// ReconcileOptions controls reconcile behavior for purge/sync operations.
type ReconcileOptions struct {
	// DryRun prevents execution of any mutations if true.
	DryRun bool

	// DoPurge enables deletion of entities missing in any store.
	DoPurge bool

	// DoSync enables syncing of mismatched fields from gamedata to DB.
	DoSync bool

	// Confirmed indicates user has confirmed destructive actions.
	// If false, mutations will not execute regardless of DryRun.
	Confirmed bool
}
