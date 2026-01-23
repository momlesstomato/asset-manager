package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"asset-manager/core/reconcile"
	"asset-manager/core/storage"
	"asset-manager/core/utils"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// FurnitureAdapter implements the reconcile.Adapter interface for furniture assets.
type FurnitureAdapter struct {
	// classnameToID maps classnames to IDs for storage key resolution
	classnameToID map[string]string
	// idToClassname maps IDs to classnames for checking storage by ID
	idToClassname map[string]string
	mu            sync.RWMutex
	// mappingReady signals when the classnameToID map is fully populated
	mappingReady chan struct{}

	// Mutation context (stored for purge/sync operations)
	db            *gorm.DB
	client        storage.Client
	bucket        string
	storagePrefix string
	serverProfile string
	gamedataObj   string

	// batchConcurrency allows overriding worker count (default 50)
	batchConcurrency int
}

// NewAdapter creates a new furniture adapter.
func NewAdapter() *FurnitureAdapter {
	return &FurnitureAdapter{
		classnameToID: make(map[string]string),
		idToClassname: make(map[string]string),
		mappingReady:  make(chan struct{}),
	}
}

// SetMutationContext stores database, storage client, and configuration for mutation operations.
// This must be called before using DeleteDB, DeleteStorage, DeleteGamedata, or SyncDBFromGamedata.
func (a *FurnitureAdapter) SetMutationContext(db *gorm.DB, client storage.Client, bucket, prefix, serverProfile, gamedataObj string) {
	a.db = db
	a.client = client
	a.bucket = bucket
	a.storagePrefix = prefix
	a.serverProfile = serverProfile
	a.gamedataObj = gamedataObj

	// Auto-detect SQLite and force sequential execution to avoid locking/deadlocks
	if db != nil && db.Dialector.Name() == "sqlite" {
		a.batchConcurrency = 1
	}
}

// SetBatchConcurrency sets the number of concurrent workers for batch operations.
// Set to 1 for sequential execution (useful for SQLite tests).
func (a *FurnitureAdapter) SetBatchConcurrency(n int) {
	a.batchConcurrency = n
}

// Name returns the unique name of this adapter.
func (a *FurnitureAdapter) Name() string {
	return "furniture"
}

// DBItem represents a normalized database furniture item.
type DBItem struct {
	ID         int
	SpriteID   int
	ItemName   string
	PublicName string
	Width      int
	Length     int
	CanSit     bool
	CanWalk    bool
	CanLay     bool
	Type       string
}

// GDItem represents a gamedata furniture item.
type GDItem struct {
	ID         int    `json:"id"`
	ClassName  string `json:"classname"`
	Name       string `json:"name"`
	XDim       int    `json:"xdim"`
	YDim       int    `json:"ydim"`
	CanSitOn   bool   `json:"cansiton"`
	CanStandOn bool   `json:"canstandon"`
	CanLayOn   bool   `json:"canlayon"`
	Type       string `json:"-"` // "s" for room items, "i" for wall items
}

// FurnitureData represents the structure of FurnitureData.json.
type FurnitureData struct {
	RoomItemTypes struct {
		FurniType []GDItem `json:"furnitype"`
	} `json:"roomitemtypes"`
	WallItemTypes struct {
		FurniType []GDItem `json:"furnitype"`
	} `json:"wallitemtypes"`
}

// LoadDBIndex loads all furniture items from the database.
func (a *FurnitureAdapter) LoadDBIndex(ctx context.Context, db *gorm.DB, serverProfile string) (map[string]reconcile.DBItem, error) {
	index := make(map[string]reconcile.DBItem)

	// Handle nil DB
	if db == nil {
		return index, nil
	}

	profile := GetProfileByName(serverProfile)

	// Build query based on server profile
	tableName := profile.TableName

	// Query all items using raw SQL (GORM's Find doesn't populate map slices properly)
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	dbRows, err := db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", tableName, err)
	}
	defer dbRows.Close()

	// Get column names
	columns, err := dbRows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse rows into DBItem
	for dbRows.Next() {
		// Create a map to scan into
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := dbRows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		item := DBItem{}

		// Extract values using profile column mappings
		if id, ok := row[profile.Columns[ColID]]; ok {
			item.ID = utils.ToInt(id)
		}
		if spriteID, ok := row[profile.Columns[ColSpriteID]]; ok {
			item.SpriteID = utils.ToInt(spriteID)
		}
		if itemName, ok := row[profile.Columns[ColItemName]]; ok {
			item.ItemName = utils.ToString(itemName)
		}
		if publicName, ok := row[profile.Columns[ColPublicName]]; ok {
			item.PublicName = utils.ToString(publicName)
		}
		if width, ok := row[profile.Columns[ColWidth]]; ok {
			item.Width = utils.ToInt(width)
		}
		if length, ok := row[profile.Columns[ColLength]]; ok {
			item.Length = utils.ToInt(length)
		}

		// Boolean fields (handle different types)
		if canSitCol, ok := profile.Columns[ColCanSit]; ok {
			if val, exists := row[canSitCol]; exists {
				item.CanSit = utils.ToBool(val)
			}
		}
		if canWalkCol, ok := profile.Columns[ColCanWalk]; ok {
			if val, exists := row[canWalkCol]; exists {
				item.CanWalk = utils.ToBool(val)
			}
		}
		if canLayCol, ok := profile.Columns[ColCanLay]; ok {
			if val, exists := row[canLayCol]; exists {
				item.CanLay = utils.ToBool(val)
			}
		}

		// Load Type
		if typeCol, ok := profile.Columns[ColType]; ok {
			if val, exists := row[typeCol]; exists {
				item.Type = utils.ToString(val)
			}
		}

		// Use sprite_id as key (sprite_id matches gamedata id, not database id)
		key := strconv.Itoa(item.SpriteID)
		index[key] = item
	}

	return index, nil
}

// LoadGamedataIndex loads furniture items from gamedata JSON.
func (a *FurnitureAdapter) LoadGamedataIndex(ctx context.Context, client storage.Client, bucket, objectName string, paths []string) (map[string]reconcile.GDItem, error) {
	// Download gamedata JSON
	reader, err := client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get gamedata object: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read gamedata: %w", err)
	}

	// Parse JSON
	var furniData FurnitureData
	if err := json.Unmarshal(data, &furniData); err != nil {
		return nil, fmt.Errorf("failed to parse gamedata JSON: %w", err)
	}

	// Build index from both arrays
	index := make(map[string]reconcile.GDItem)

	// Build classname mapping concurrently
	a.mu.Lock()
	defer a.mu.Unlock()

	// Process room items
	for _, item := range furniData.RoomItemTypes.FurniType {
		if item.ID > 0 && item.ClassName != "" {
			item.Type = "s" // Floor item
			key := strconv.Itoa(item.ID)
			index[key] = item
			// Map classname directly to ID (classname in gamedata matches filename)
			a.classnameToID[item.ClassName] = key
			a.idToClassname[key] = item.ClassName
		}
	}

	// Process wall items
	for _, item := range furniData.WallItemTypes.FurniType {
		if item.ID > 0 && item.ClassName != "" {
			item.Type = "i" // Wall item
			key := strconv.Itoa(item.ID)
			index[key] = item
			// Map classname directly to ID (classname in gamedata matches filename)
			a.classnameToID[item.ClassName] = key
			a.idToClassname[key] = item.ClassName
		}
	}

	// Signal that mapping is ready
	// Use a select to ensure we don't close an already closed channel if this runs multiple times
	select {
	case <-a.mappingReady:
		// already closed
	default:
		close(a.mappingReady)
	}

	return index, nil
}

// LoadStorageSet lists all furniture objects in storage.
func (a *FurnitureAdapter) LoadStorageSet(ctx context.Context, client storage.Client, bucket, prefix, extension string) (map[string]struct{}, error) {
	// Wait for mapping to be ready before processing storage
	// This ensures we can resolve classnames to IDs
	select {
	case <-a.mappingReady:
		// Mapping is ready
	case <-ctx.Done():
		return nil, ctx.Err()
	// Add a timeout just in case gamedata loading fails silently or hangs, though ctx.Done should cover it
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timeout waiting for gamedata mapping")
	}

	set := make(map[string]struct{})
	var mu sync.Mutex

	// List all objects under prefix
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for obj := range client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", obj.Err)
		}

		// Extract key from object
		if key, ok := a.ExtractStorageKey(obj.Key, prefix, extension); ok {
			mu.Lock()
			set[key] = struct{}{}
			mu.Unlock()
		}
	}

	return set, nil
}

// ExtractDBKey returns the entity key from a DB item.
func (a *FurnitureAdapter) ExtractDBKey(item reconcile.DBItem) string {
	dbItem := item.(DBItem)
	// Use sprite_id as key (matches gamedata id)
	return strconv.Itoa(dbItem.SpriteID)
}

// ExtractGDKey returns the entity key from a gamedata item.
func (a *FurnitureAdapter) ExtractGDKey(item reconcile.GDItem) string {
	gdItem := item.(GDItem)
	return strconv.Itoa(gdItem.ID)
}

// ExtractStorageKey parses a storage object key to extract the entity key.
// For furniture, we map the filename (classname) to an ID using the gamedata mapping.
func (a *FurnitureAdapter) ExtractStorageKey(objectKey, prefix, extension string) (key string, ok bool) {
	// Check if it has the right extension
	if !strings.HasSuffix(objectKey, extension) {
		return "", false
	}

	// Check if it matches prefix
	if !strings.HasPrefix(objectKey, prefix) {
		return "", false
	}

	// Extract relative path (strip prefix and leading slash)
	// objectKey: bundled/furniture/subdir/item.nitro
	// prefix: bundled/furniture
	// relPath: /subdir/item.nitro -> subdir/item.nitro
	relPath := objectKey[len(prefix):]
	relPath = strings.TrimPrefix(relPath, "/")

	// Remove extension
	// relPathNoExt: subdir/item
	relPathNoExt := strings.TrimSuffix(relPath, extension)

	// Get filename (basename) for mapping lookup
	// filename: item
	parts := strings.Split(relPathNoExt, "/")
	filename := parts[len(parts)-1]

	// Look up ID from classname mapping (filename IS the classname)
	a.mu.RLock()
	id, found := a.classnameToID[filename]
	canonicalCN, hasReverse := a.idToClassname[id]
	a.mu.RUnlock()

	if found {
		// If we are in a subdirectory, strictly speaking it's an orphan/misplaced file,
		// but we still return the ID so the engine knows we HAVE the item (just in wrong place).
		// This enables the engine to plan a Move action instead of Delete+Download.

		if hasReverse && canonicalCN == filename {
			return id, true
		}
	}

	// If not found in mapping, or is nested/shadowed, return the relative path as key.
	// This ensures DeleteStorage can reconstruct the correct path: prefix + "/" + relPathNoExt + extension
	return relPathNoExt, true
}

// ResolveName returns the display name for an entity.
func (a *FurnitureAdapter) ResolveName(dbItem reconcile.DBItem, gdItem reconcile.GDItem) string {
	if dbItem != nil {
		return dbItem.(DBItem).PublicName
	}
	if gdItem != nil {
		return gdItem.(GDItem).Name
	}
	return ""
}

// GetMetadata returns classname for furniture.
func (a *FurnitureAdapter) GetMetadata(dbItem reconcile.DBItem, gdItem reconcile.GDItem) map[string]string {
	meta := make(map[string]string)

	val := ""
	if gdItem != nil {
		val = gdItem.(GDItem).ClassName
	}
	if val == "" && dbItem != nil {
		val = dbItem.(DBItem).ItemName
	}

	if val != "" {
		meta["classname"] = val
	}
	return meta
}

// CompareFields compares DB and gamedata items and returns mismatch descriptions.
func (a *FurnitureAdapter) CompareFields(dbItem reconcile.DBItem, gdItem reconcile.GDItem) []string {
	db := dbItem.(DBItem)
	gd := gdItem.(GDItem)

	var mismatches []string

	// Compare name
	// Relaxed check: Accept if DB PublicName matches GD Name OR GD ClassName
	// (Common in emulators to use classname as public_name default)
	if db.PublicName != gd.Name && db.PublicName != gd.ClassName {
		mismatches = append(mismatches, fmt.Sprintf("name: gd='%s' db='%s'", gd.Name, db.PublicName))
	}

	// Compare classname
	if db.ItemName != gd.ClassName {
		mismatches = append(mismatches, fmt.Sprintf("classname: gd='%s' db='%s'", gd.ClassName, db.ItemName))
	}

	// Compare dimensions
	if db.Width != gd.XDim {
		mismatches = append(mismatches, fmt.Sprintf("width: gd=%d db=%d", gd.XDim, db.Width))
	}
	if db.Length != gd.YDim {
		mismatches = append(mismatches, fmt.Sprintf("length: gd=%d db=%d", gd.YDim, db.Length))
	}

	// Compare boolean flags
	if db.CanSit != gd.CanSitOn {
		mismatches = append(mismatches, fmt.Sprintf("can_sit: gd=%v db=%v", gd.CanSitOn, db.CanSit))
	}
	if db.CanWalk != gd.CanStandOn {
		mismatches = append(mismatches, fmt.Sprintf("can_walk: gd=%v db=%v", gd.CanStandOn, db.CanWalk))
	}
	if db.CanLay != gd.CanLayOn {
		mismatches = append(mismatches, fmt.Sprintf("can_lay: gd=%v db=%v", gd.CanLayOn, db.CanLay))
	}

	// Compare Type (Wall vs Floor)
	// If DB type is "i", it MUST be a wall item in gamedata (Type="i").
	// If DB type is NOT "i", it is generally a room item.
	// Note: DB might use other letters for floor items (s, e, r, etc.), but 'i' is exclusively wall.
	if db.Type == "i" {
		if gd.Type != "i" {
			mismatches = append(mismatches, "type: gd='room' (WallItemTypes=false) db='i' (wall)")
		}
	} else {
		// If DB is not wall, GD should not be wall (unless we discover specific exceptions)
		if gd.Type == "i" {
			mismatches = append(mismatches, fmt.Sprintf("type: gd='wall' (WallItemTypes=true) db='%s' (not wall)", db.Type))
		}
	}

	return mismatches
}

// QueryDB performs a targeted database lookup.
func (a *FurnitureAdapter) QueryDB(ctx context.Context, db *gorm.DB, serverProfile string, query reconcile.Query) (reconcile.DBItem, error) {
	profile := GetProfileByName(serverProfile)

	var row map[string]any
	tableName := profile.TableName

	// Try by ID first
	if query.ID != "" {
		if id, err := strconv.Atoi(query.ID); err == nil {
			row = make(map[string]any) // Ensure map is initialized
			result := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColID]+" = ?", id).Take(&row)
			if result.Error != nil {
				if result.Error == gorm.ErrRecordNotFound {
					// Fall through to other checks might be unsafe if ID is numeric but not found?
					// But we should continue just in case.
				} else {
					return nil, result.Error
				}
			} else if result.RowsAffected > 0 {
				return a.parseDBRow(row, profile), nil
			}
		}
	}

	// Try by classname
	if query.Classname != "" {
		row = make(map[string]any)
		result := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColItemName]+" = ?", query.Classname).Take(&row)
		if result.Error == nil && result.RowsAffected > 0 {
			return a.parseDBRow(row, profile), nil
		} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return nil, result.Error
		}
	}

	// Try by name
	if query.Name != "" {
		row = make(map[string]any)
		result := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColPublicName]+" = ?", query.Name).Take(&row)
		if result.Error == nil && result.RowsAffected > 0 {
			return a.parseDBRow(row, profile), nil
		} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return nil, result.Error
		}
	}

	return nil, nil
}

// QueryGamedata performs a targeted gamedata lookup.
func (a *FurnitureAdapter) QueryGamedata(ctx context.Context, client storage.Client, bucket, objectName string, paths []string, query reconcile.Query) (reconcile.GDItem, error) {
	// Load full index (no way to avoid parsing the entire JSON)
	index, err := a.LoadGamedataIndex(ctx, client, bucket, objectName, paths)
	if err != nil {
		return nil, err
	}

	// Try by ID
	if query.ID != "" {
		if item, ok := index[query.ID]; ok {
			return item, nil
		}
	}

	// Try by classname or name
	for _, item := range index {
		gdItem := item.(GDItem)
		if query.Classname != "" && gdItem.ClassName == query.Classname {
			return gdItem, nil
		}
		if query.Name != "" && gdItem.Name == query.Name {
			return gdItem, nil
		}
	}

	return nil, nil
}

// CheckStorage checks if a specific entity exists in storage.
func (a *FurnitureAdapter) CheckStorage(ctx context.Context, client storage.Client, bucket, prefix, extension string, key string) (bool, error) {
	// For furniture, the key is the ID, but storage uses classname
	// We check the idToClassname mapping.

	a.mu.RLock()
	classname, ok := a.idToClassname[key]
	a.mu.RUnlock()

	filename := key
	if ok {
		filename = classname
	}

	// Try to find object with this key as classname
	objectKey := fmt.Sprintf("%s/%s%s", prefix, filename, extension)

	opts := minio.ListObjectsOptions{
		Prefix:  objectKey,
		MaxKeys: 1,
	}

	for obj := range client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return false, obj.Err
		}
		if obj.Key == objectKey {
			return true, nil
		}
	}

	return false, nil
}

// parseDBRow converts a raw DB row to a DBItem.
func (a *FurnitureAdapter) parseDBRow(row map[string]any, profile ServerProfile) DBItem {
	item := DBItem{}

	if id, ok := row[profile.Columns[ColID]]; ok {
		item.ID = utils.ToInt(id)
	}
	if spriteID, ok := row[profile.Columns[ColSpriteID]]; ok {
		item.SpriteID = utils.ToInt(spriteID)
	}
	if itemName, ok := row[profile.Columns[ColItemName]]; ok {
		item.ItemName = utils.ToString(itemName)
	}
	if publicName, ok := row[profile.Columns[ColPublicName]]; ok {
		item.PublicName = utils.ToString(publicName)
	}
	if width, ok := row[profile.Columns[ColWidth]]; ok {
		item.Width = utils.ToInt(width)
	}
	if length, ok := row[profile.Columns[ColLength]]; ok {
		item.Length = utils.ToInt(length)
	}

	if canSitCol, ok := profile.Columns[ColCanSit]; ok {
		if val, exists := row[canSitCol]; exists {
			item.CanSit = utils.ToBool(val)
		}
	}
	if canWalkCol, ok := profile.Columns[ColCanWalk]; ok {
		if val, exists := row[canWalkCol]; exists {
			item.CanWalk = utils.ToBool(val)
		}
	}
	if canLayCol, ok := profile.Columns[ColCanLay]; ok {
		if val, exists := row[canLayCol]; exists {
			item.CanLay = utils.ToBool(val)
		}
	}

	return item
}

// Prepare validates and updates the database schema for compatibility.
// It auto-expands name columns to VARCHAR(120) to prevent truncation errors.
func (a *FurnitureAdapter) Prepare(ctx context.Context, db *gorm.DB) error {
	profile := GetProfileByName(a.serverProfile)
	tableName := profile.TableName

	// Columns to ensure size for (public_name and item_name)
	targetCols := []string{ColItemName, ColPublicName}

	for _, colType := range targetCols {
		colName := profile.Columns[colType]
		if colName == "" {
			continue
		}

		// Execute ALTER to resize column to 120 chars
		// This is safe to run multiple times (idempotent for size increase)
		query := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s VARCHAR(120)", tableName, colName)

		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to prepare schema for %s.%s: %w", tableName, colName, err)
		}
	}
	return nil
}
