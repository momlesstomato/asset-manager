package reconcile

// Batch mutation operations for performance optimization.

import (
	"asset-manager/core/reconcile"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/minio/minio-go/v7"
)

// DeleteStorageBatch deletes multiple storage objects efficiently using batch API.
func (a *FurnitureAdapter) DeleteStorageBatch(ctx context.Context, keys []string) error {
	if a.client == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	if len(keys) == 0 {
		return nil
	}

	// Build object info channel for batch deletion
	objectsCh := make(chan minio.ObjectInfo, len(keys))

	for _, key := range keys {
		var classname string

		// Check if key is numeric (ID) or already a classname/relPath
		if _, err := strconv.Atoi(key); err == nil {
			// Key is an ID, likely a mapped item.
			a.mu.RLock()
			cn, ok := a.idToClassname[key]
			a.mu.RUnlock()

			if ok {
				// It's a mapped item, use the classname
				classname = cn
			} else {
				// Numeric key but not mapped?
				// This happens if ExtractStorageKey returns a numeric filename (e.g. 00011.nitro)
				// which is an orphan. We treat it as the relative path.
				classname = key
			}
		} else {
			// Key is already a classname/relPath (storage orphan)
			classname = key
		}

		objectKey := fmt.Sprintf("%s/%s.nitro", a.storagePrefix, classname)
		objectsCh <- minio.ObjectInfo{Key: objectKey}
	}
	close(objectsCh)

	// Execute batch deletion
	errorCh := a.client.RemoveObjects(ctx, a.bucket, objectsCh, minio.RemoveObjectsOptions{})

	// Collect any errors
	var errors []string
	for err := range errorCh {
		if err.Err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", err.ObjectName, err.Err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete had %d errors: %v", len(errors), errors)
	}

	return nil
}

// DeleteDBBatch deletes multiple DB rows efficiently using IN clause.
func (a *FurnitureAdapter) DeleteDBBatch(ctx context.Context, keys []string) error {
	if a.db == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	if len(keys) == 0 {
		return nil
	}

	profile := GetProfileByName(a.serverProfile)
	tableName := profile.TableName

	// Convert keys to sprite IDs
	spriteIDs := make([]int, 0, len(keys))
	for _, key := range keys {
		spriteID, err := strconv.Atoi(key)
		if err != nil {
			continue // Skip invalid keys
		}
		spriteIDs = append(spriteIDs, spriteID)
	}

	if len(spriteIDs) == 0 {
		return nil
	}

	// Delete using IN clause for batch efficiency
	result := a.db.WithContext(ctx).
		Table(tableName).
		Where(profile.Columns[ColSpriteID]+" IN ?", spriteIDs).
		Delete(nil)

	if result.Error != nil {
		return fmt.Errorf("failed to batch delete from DB: %w", result.Error)
	}

	return nil
}

// SyncDBBatch updates multiple DB rows concurrently using a worker pool.
// Concurrent updates are safe as each action targets a unique SpriteID.
func (a *FurnitureAdapter) SyncDBBatch(ctx context.Context, actions []reconcile.Action) error {
	if a.db == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	if len(actions) == 0 {
		return nil
	}

	// Configuration for worker pool
	// 50 workers allows high throughput without overwhelming DB connection pool (max 100)
	const numWorkers = 50

	// Create channels
	actionsCh := make(chan reconcile.Action, len(actions))
	errorCh := make(chan error, len(actions)) // Buffered to avoid blocking

	// Fill the actions channel
	for _, action := range actions {
		actionsCh <- action
	}
	close(actionsCh)

	// Launch workers
	// Using a WaitGroup to wait for all workers to finish
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for action := range actionsCh {
				// Reuse existing single-item Sync logic
				// It is self-contained and safe for concurrent use (uses local scope vars)
				if err := a.SyncDBFromGamedata(ctx, action.Key, action.GDItem); err != nil {
					errorCh <- fmt.Errorf("sync failed for %s: %w", action.Key, err)
				}
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	close(errorCh)

	// Collect errors
	var errors []string
	for err := range errorCh {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch sync had %d errors: %v", len(errors), errors)
	}

	return nil
}

// DeleteGamedataBatch removes multiple items from FurnitureData.json in one write.
func (a *FurnitureAdapter) DeleteGamedataBatch(ctx context.Context, keys []string) error {
	if a.client == nil {
		return fmt.Errorf("mutation context not set, call SetMutationContext first")
	}

	if len(keys) == 0 {
		return nil
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

	// Convert keys to IDs set for fast lookup
	idsToDelete := make(map[int]struct{})
	for _, key := range keys {
		id, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		idsToDelete[id] = struct{}{}
	}

	// Remove from room items
	newRoomItems := make([]GDItem, 0)
	for _, item := range furniData.RoomItemTypes.FurniType {
		if _, shouldDelete := idsToDelete[item.ID]; !shouldDelete {
			newRoomItems = append(newRoomItems, item)
		}
	}
	furniData.RoomItemTypes.FurniType = newRoomItems

	// Remove from wall items
	newWallItems := make([]GDItem, 0)
	for _, item := range furniData.WallItemTypes.FurniType {
		if _, shouldDelete := idsToDelete[item.ID]; !shouldDelete {
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
