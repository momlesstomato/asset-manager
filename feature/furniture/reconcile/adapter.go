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

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// FurnitureAdapter implements the reconcile.Adapter interface for furniture assets.
type FurnitureAdapter struct {
	// classnameToID maps classnames to IDs for storage key resolution
	classnameToID map[string]string
	mu            sync.RWMutex
	// mappingReady signals when the classnameToID map is fully populated
	mappingReady chan struct{}
}

// NewAdapter creates a new furniture adapter.
func NewAdapter() *FurnitureAdapter {
	return &FurnitureAdapter{
		classnameToID: make(map[string]string),
		mappingReady:  make(chan struct{}),
	}
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
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := dbRows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		item := DBItem{}

		// Extract values using profile column mappings
		if id, ok := row[profile.Columns[ColID]]; ok {
			item.ID = toInt(id)
		}
		if spriteID, ok := row[profile.Columns[ColSpriteID]]; ok {
			item.SpriteID = toInt(spriteID)
		}
		if itemName, ok := row[profile.Columns[ColItemName]]; ok {
			item.ItemName = toString(itemName)
		}
		if publicName, ok := row[profile.Columns[ColPublicName]]; ok {
			item.PublicName = toString(publicName)
		}
		if width, ok := row[profile.Columns[ColWidth]]; ok {
			item.Width = toInt(width)
		}
		if length, ok := row[profile.Columns[ColLength]]; ok {
			item.Length = toInt(length)
		}

		// Boolean fields (handle different types)
		if canSitCol, ok := profile.Columns[ColCanSit]; ok {
			if val, exists := row[canSitCol]; exists {
				item.CanSit = toBool(val)
			}
		}
		if canWalkCol, ok := profile.Columns[ColCanWalk]; ok {
			if val, exists := row[canWalkCol]; exists {
				item.CanWalk = toBool(val)
			}
		}
		if canLayCol, ok := profile.Columns[ColCanLay]; ok {
			if val, exists := row[canLayCol]; exists {
				item.CanLay = toBool(val)
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
	a.classnameToID = make(map[string]string)

	// Process room items
	for _, item := range furniData.RoomItemTypes.FurniType {
		if item.ID > 0 && item.ClassName != "" {
			key := strconv.Itoa(item.ID)
			index[key] = item
			// Map classname directly to ID (classname in gamedata matches filename)
			a.classnameToID[item.ClassName] = key
		}
	}

	// Process wall items
	for _, item := range furniData.WallItemTypes.FurniType {
		if item.ID > 0 && item.ClassName != "" {
			key := strconv.Itoa(item.ID)
			index[key] = item
			// Map classname directly to ID (classname in gamedata matches filename)
			a.classnameToID[item.ClassName] = key
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
		if key, ok := a.ExtractStorageKey(obj.Key, extension); ok {
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
func (a *FurnitureAdapter) ExtractStorageKey(objectKey, extension string) (key string, ok bool) {
	// Check if it has the right extension
	if !strings.HasSuffix(objectKey, extension) {
		return "", false
	}

	// Extract filename from path
	parts := strings.Split(objectKey, "/")
	if len(parts) == 0 {
		return "", false
	}
	filename := parts[len(parts)-1]

	// Remove extension - filename now matches classname from gamedata
	filename = strings.TrimSuffix(filename, extension)

	// Look up ID from classname mapping (filename IS the classname)
	a.mu.RLock()
	id, found := a.classnameToID[filename]
	a.mu.RUnlock()

	if found {
		return id, true
	}

	// If not found in mapping, the item exists in storage but not in gamedata
	// Return the classname as the key so it shows up as "unregistered"
	return filename, true
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

// CompareFields compares DB and gamedata items and returns mismatch descriptions.
func (a *FurnitureAdapter) CompareFields(dbItem reconcile.DBItem, gdItem reconcile.GDItem) []string {
	db := dbItem.(DBItem)
	gd := gdItem.(GDItem)

	var mismatches []string

	// Compare name
	if db.PublicName != gd.Name {
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

	return mismatches
}

// QueryDB performs a targeted database lookup.
func (a *FurnitureAdapter) QueryDB(ctx context.Context, db *gorm.DB, serverProfile string, query reconcile.Query) (reconcile.DBItem, error) {
	profile := GetProfileByName(serverProfile)

	var row map[string]interface{}
	tableName := profile.TableName

	// Try by ID first
	if query.ID != "" {
		if id, err := strconv.Atoi(query.ID); err == nil {
			err := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColID]+" = ?", id).First(&row).Error
			if err == nil {
				return a.parseDBRow(row, profile), nil
			}
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
	}

	// Try by classname
	if query.Classname != "" {
		err := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColItemName]+" = ?", query.Classname).First(&row).Error
		if err == nil {
			return a.parseDBRow(row, profile), nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	// Try by name
	if query.Name != "" {
		err := db.WithContext(ctx).Table(tableName).Where(profile.Columns[ColPublicName]+" = ?", query.Name).First(&row).Error
		if err == nil {
			return a.parseDBRow(row, profile), nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
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
	// We need to derive the filename from the key
	// This is problematic - we'd need the classname mapping
	// For now, we'll do a simple check

	// Try to find object with this key as classname
	objectKey := fmt.Sprintf("%s/%s%s", prefix, key, extension)

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

// Helper functions for type conversion

func toInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toBool(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case int64:
		return v == 1
	case string:
		return v == "1" || v == "true"
	default:
		return false
	}
}

// parseDBRow converts a raw DB row to a DBItem.
func (a *FurnitureAdapter) parseDBRow(row map[string]interface{}, profile ServerProfile) DBItem {
	item := DBItem{}

	if id, ok := row[profile.Columns[ColID]]; ok {
		item.ID = toInt(id)
	}
	if spriteID, ok := row[profile.Columns[ColSpriteID]]; ok {
		item.SpriteID = toInt(spriteID)
	}
	if itemName, ok := row[profile.Columns[ColItemName]]; ok {
		item.ItemName = toString(itemName)
	}
	if publicName, ok := row[profile.Columns[ColPublicName]]; ok {
		item.PublicName = toString(publicName)
	}
	if width, ok := row[profile.Columns[ColWidth]]; ok {
		item.Width = toInt(width)
	}
	if length, ok := row[profile.Columns[ColLength]]; ok {
		item.Length = toInt(length)
	}

	if canSitCol, ok := profile.Columns[ColCanSit]; ok {
		if val, exists := row[canSitCol]; exists {
			item.CanSit = toBool(val)
		}
	}
	if canWalkCol, ok := profile.Columns[ColCanWalk]; ok {
		if val, exists := row[canWalkCol]; exists {
			item.CanWalk = toBool(val)
		}
	}
	if canLayCol, ok := profile.Columns[ColCanLay]; ok {
		if val, exists := row[canLayCol]; exists {
			item.CanLay = toBool(val)
		}
	}

	return item
}
