package reconcile

// Mutation methods implementing reconcile.Mutator interface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"asset-manager/core/reconcile"

	"github.com/minio/minio-go/v7"
)

// DeleteDB removes a furniture item from the database using server-aware mapping.
func (a *FurnitureAdapter) DeleteDB(ctx context.Context, key string) error {
	if a.db == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	profile := GetProfileByName(a.serverProfile)
	tableName := profile.TableName

	// Convert key (sprite_id) to int
	spriteID, err := strconv.Atoi(key)
	if err != nil {
		return fmt.Errorf("invalid key %s: %w", key, err)
	}

	// Delete WHERE sprite_id = key
	result := a.db.WithContext(ctx).
		Table(tableName).
		Where(profile.Columns[ColSpriteID]+" = ?", spriteID).
		Delete(nil)

	if result.Error != nil {
		return fmt.Errorf("failed to delete from DB: %w", result.Error)
	}

	return nil
}

// DeleteGamedata removes a furniture item from FurnitureData.json.
// This implementation batches deletions and writes once per call.
func (a *FurnitureAdapter) DeleteGamedata(ctx context.Context, key string) error {
	if a.client == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Read the entire FurnitureData.json
	reader, err := a.client.GetObject(ctx, a.bucket, a.gamedataObj, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get gamedata: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read gamedata: %w", err)
	}

	// Parse JSON
	var furniData FurnitureData
	if err := json.Unmarshal(data, &furniData); err != nil {
		return fmt.Errorf("failed to parse gamedata: %w", err)
	}

	// Convert key to int
	id, err := strconv.Atoi(key)
	if err != nil {
		return fmt.Errorf("invalid key %s: %w", key, err)
	}

	// Remove from room items
	newRoomItems := make([]GDItem, 0)
	for _, item := range furniData.RoomItemTypes.FurniType {
		if item.ID != id {
			newRoomItems = append(newRoomItems, item)
		}
	}
	furniData.RoomItemTypes.FurniType = newRoomItems

	// Remove from wall items
	newWallItems := make([]GDItem, 0)
	for _, item := range furniData.WallItemTypes.FurniType {
		if item.ID != id {
			newWallItems = append(newWallItems, item)
		}
	}
	furniData.WallItemTypes.FurniType = newWallItems

	// Marshal back to JSON
	newData, err := json.MarshalIndent(furniData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gamedata: %w", err)
	}

	// Write back to storage
	_, err = a.client.PutObject(
		ctx,
		a.bucket,
		a.gamedataObj,
		io.NopCloser(bytes.NewReader(newData)),
		int64(len(newData)),
		minio.PutObjectOptions{ContentType: "application/json"},
	)
	if err != nil {
		return fmt.Errorf("failed to write gamedata: %w", err)
	}

	return nil
}

// DeleteStorage removes the .nitro file from storage using classname mapping.
func (a *FurnitureAdapter) DeleteStorage(ctx context.Context, key string) error {
	if a.client == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	// Get classname from key
	a.mu.RLock()
	classname, ok := a.idToClassname[key]
	a.mu.RUnlock()

	if !ok {
		// Classname not found - best effort: log and skip
		// This can happen if gamedata is already deleted
		return fmt.Errorf("classname not found for key %s (may already be deleted from gamedata)", key)
	}

	// Build object key
	objectKey := fmt.Sprintf("%s/%s.nitro", a.storagePrefix, classname)

	// Delete object
	err := a.client.RemoveObject(ctx, a.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete storage object %s: %w", objectKey, err)
	}

	return nil
}

// SyncDBFromGamedata updates DB fields to match gamedata using server-aware mapping.
func (a *FurnitureAdapter) SyncDBFromGamedata(ctx context.Context, key string, gdItem reconcile.GDItem) error {
	if a.db == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	profile := GetProfileByName(a.serverProfile)
	gd := gdItem.(GDItem)

	// Build update map based on field mappings
	// Truncate strings to match updated DB schema limits (varchar(120))
	// We use 110 as a safe buffer.
	const maxNameLen = 110

	updates := map[string]any{
		profile.Columns[ColItemName]:    truncateStr(gd.ClassName, maxNameLen),
		profile.Columns[ColPublicName]:  truncateStr(gd.Name, maxNameLen),
		profile.Columns[ColWidth]:       gd.XDim,
		profile.Columns[ColLength]:      gd.YDim,
		profile.Columns[ColStackHeight]: 1, // Default, gamedata doesn't always have this
	}

	// Add boolean fields if mapped
	if col, ok := profile.Columns[ColCanSit]; ok {
		updates[col] = gd.CanSitOn
	}
	if col, ok := profile.Columns[ColCanWalk]; ok {
		updates[col] = gd.CanStandOn
	}
	if col, ok := profile.Columns[ColCanLay]; ok {
		updates[col] = gd.CanLayOn
	}
	if col, ok := profile.Columns[ColType]; ok {
		updates[col] = gd.Type
	}

	// Convert key to sprite_id
	spriteID, err := strconv.Atoi(key)
	if err != nil {
		return fmt.Errorf("invalid key %s: %w", key, err)
	}

	// Execute update
	result := a.db.WithContext(ctx).
		Table(profile.TableName).
		Where(profile.Columns[ColSpriteID]+" = ?", spriteID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to sync DB from gamedata: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no rows updated for key %s (sprite_id %d)", key, spriteID)
	}

	return nil
}

// truncateStr truncates a string to the specified length.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
