package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"asset-manager/feature/furniture/models"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// SyncOperations implements the actual sync logic
type SyncOperations struct {
	service *SyncService
}

// NewSyncOperations creates sync operations handler
func NewSyncOperations(service *SyncService) *SyncOperations {
	return &SyncOperations{service: service}
}

// SyncSchema adds missing columns to database
func (so *SyncOperations) SyncSchema(ctx context.Context) ([]string, error) {
	mappings, err := so.service.GetParameterMappings()
	if err != nil {
		return nil, err
	}

	var changes []string
	tableName := so.service.GetTableName()

	for _, mapping := range mappings {
		if !mapping.IsNewColumn {
			continue
		}

		// Check if column already exists using GORM Migrator (Dialect agnostic)
		if so.service.db.Migrator().HasColumn(tableName, mapping.DBColumn) {
			continue
		}

		// Generate ALTER TABLE statement
		// Note: GORM Migrator doesn't easily support adding columns with specific defaults in a generic way without a struct.
		// Since we are adding columns dynamically based on mappings (without struct updates in some cases?), we still might need Raw SQL for the ALTER.
		// But the check can be generic.

		defaultClause := ""
		if mapping.DefaultValue != "NULL" {
			defaultClause = fmt.Sprintf(" DEFAULT %s", mapping.DefaultValue)
		}

		alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s%s",
			tableName, mapping.DBColumn, mapping.DBType, defaultClause)

		if err := so.service.db.Exec(alterSQL).Error; err != nil {
			return changes, fmt.Errorf("failed to add column %s: %w", mapping.DBColumn, err)
		}

		changes = append(changes, fmt.Sprintf("Added column: %s (%s)", mapping.DBColumn, mapping.DBType))
	}

	return changes, nil
}

// RemoveMissingAssets deletes assets that are missing from one source but present in others.
func (so *SyncOperations) RemoveMissingAssets(ctx context.Context, furniData *models.FurnitureData, report *models.Report) (int, int, int, error) {
	tableName := so.service.GetTableName()
	storageDeleted := 0
	databaseDeleted := 0
	furniDataDeleted := 0

	var dbDeleteIDs []int
	var storageObjects []minio.ObjectInfo
	var ghostIDs []int // IDs missing from BOTH DB and Storage (exist only in FurniData)

	for _, asset := range report.Assets {
		// 1. DELETE FROM DB:
		// If (FurniDataMissing AND InDB) OR (StorageMissing AND InDB - i.e. broken file)
		shouldDeleteFromDB := false
		if asset.FurniDataMissing && !asset.DatabaseMissing {
			shouldDeleteFromDB = true
		}
		if asset.StorageMissing && !asset.DatabaseMissing {
			shouldDeleteFromDB = true
		}

		if shouldDeleteFromDB && asset.ID > 0 {
			dbDeleteIDs = append(dbDeleteIDs, asset.ID)
		}

		// 2. DELETE FROM STORAGE:
		// If (FurniDataMissing AND InStorage) OR (DatabaseMissing AND InStorage - i.e. orphan file)
		shouldDeleteFromStorage := false
		if asset.FurniDataMissing && !asset.StorageMissing {
			shouldDeleteFromStorage = true
		}
		if asset.DatabaseMissing && !asset.StorageMissing {
			shouldDeleteFromStorage = true
		}

		if shouldDeleteFromStorage {
			path := fmt.Sprintf("bundled/furniture/%s", asset.Name)
			storageObjects = append(storageObjects, minio.ObjectInfo{Key: path})
		}

		// 3. REMOVE FROM FURNIDATA (Broken/Ghost/Invalid Items):
		// If missing from ANY source (DB or Storage), it's incomplete and should be purged.
		// "Zero Tolerance" policy to ensure clean integrity report.
		shouldRemoveFromFurniData := false
		if (asset.DatabaseMissing || asset.StorageMissing) && !asset.FurniDataMissing {
			shouldRemoveFromFurniData = true
		}

		// Check for specific validation errors
		for _, m := range asset.Mismatches {
			if strings.HasPrefix(m, "FurniData validation:") {
				shouldRemoveFromFurniData = true
				break
			}
		}

		if shouldRemoveFromFurniData {
			// Allow removing ID 0 if it's invalid
			ghostIDs = append(ghostIDs, asset.ID)
		}
	}

	// DEBUG logging
	so.service.logger.Info("Asset removal collection",
		zap.Int("total_assets", len(report.Assets)),
		zap.Int("db_delete_ids", len(dbDeleteIDs)),
		zap.Int("storage_objects", len(storageObjects)),
		zap.Int("furnidata_deletes", furniDataDeleted))

	if len(dbDeleteIDs) > 0 {
		so.service.logger.Info("Sample DB IDs to delete", zap.Ints("first_10", dbDeleteIDs[:min(10, len(dbDeleteIDs))]))
	}

	// Batch delete from database
	if len(dbDeleteIDs) > 0 {
		// Verify how many IDs actually exist
		var existCount int64
		countResult := so.service.db.Table(tableName).Where("sprite_id IN ?", dbDeleteIDs).Count(&existCount)
		so.service.logger.Info("Pre-deletion verification",
			zap.Int("ids_to_delete", len(dbDeleteIDs)),
			zap.Int64("ids_exist_in_db", existCount),
			zap.Error(countResult.Error))

		// Attempt deletion
		result := so.service.db.Table(tableName).Where("sprite_id IN ?", dbDeleteIDs).Delete(nil)
		if result.Error != nil {
			so.service.logger.Error("Database deletion failed", zap.Error(result.Error))
		} else {
			databaseDeleted = int(result.RowsAffected)
			so.service.logger.Info("Database deletion executed",
				zap.Int("rows_affected", databaseDeleted),
				zap.Int("expected", len(dbDeleteIDs)))
		}
	}

	// Batch delete from storage
	if len(storageObjects) > 0 {
		objectsCh := make(chan minio.ObjectInfo, len(storageObjects))
		for _, obj := range storageObjects {
			objectsCh <- obj
		}
		close(objectsCh)

		errorsCh := so.service.client.RemoveObjects(ctx, so.service.bucket, objectsCh, minio.RemoveObjectsOptions{})

		errorCount := 0
		for range errorsCh {
			errorCount++
		}

		storageDeleted = len(storageObjects) - errorCount
	}

	// 4. Executing FurniData Updates (Removal of Ghost Items)
	furniDataDeleted = len(ghostIDs)
	if furniDataDeleted > 0 {
		so.service.logger.Info("Removing ghost items from FurniData", zap.Int("count", furniDataDeleted))

		// Create a map for faster lookup
		ghostMap := make(map[int]bool)
		for _, id := range ghostIDs {
			ghostMap[id] = true
		}

		// Filter RoomItemTypes
		var newRoomItems []models.FurnitureItem
		for _, item := range furniData.RoomItemTypes.FurniType {
			if !ghostMap[item.ID] {
				newRoomItems = append(newRoomItems, item)
			}
		}
		furniData.RoomItemTypes.FurniType = newRoomItems

		// Filter WallItemTypes
		var newWallItems []models.FurnitureItem
		for _, item := range furniData.WallItemTypes.FurniType {
			if !ghostMap[item.ID] {
				newWallItems = append(newWallItems, item)
			}
		}
		furniData.WallItemTypes.FurniType = newWallItems

		// Save updated FurniData
		if err := so.saveFurnitureData(ctx, furniData); err != nil {
			return storageDeleted, databaseDeleted, furniDataDeleted, fmt.Errorf("failed to save FurniData: %w", err)
		}
		so.service.logger.Info("FurniData updated successfully")
	}

	return storageDeleted, databaseDeleted, furniDataDeleted, nil
}

func (so *SyncOperations) saveFurnitureData(ctx context.Context, data *models.FurnitureData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal FurniData: %w", err)
	}

	objName := "gamedata/FurnitureData.json"
	reader := bytes.NewReader(jsonData)

	_, err = so.service.client.PutObject(ctx, so.service.bucket, objName, reader, int64(len(jsonData)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to upload FurniData.json: %w", err)
	}
	return nil
}

// PerformFullSync executes complete sync operation
func (so *SyncOperations) PerformFullSync(ctx context.Context, skipDataSync bool) (*SyncReport, error) {
	startTime := time.Now()
	report := &SyncReport{
		SchemaChanges: []string{},
		Errors:        []string{},
	}

	// 1. Load FurniData
	stepStart := time.Now()
	so.service.logger.Info("Loading FurniData")
	furniData, err := loadFurnitureData(ctx, so.service.client, so.service.bucket)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Failed to load FurniData: %v", err))
		report.ExecutionTime = time.Since(startTime).String()
		return report, nil
	}
	so.service.logger.Info("FurniData loaded", zap.Duration("duration", time.Since(stepStart)))

	// 2. Run integrity check
	stepStart = time.Now()
	so.service.logger.Info("Running integrity check")
	integrityReport, err := CheckIntegrity(ctx, so.service.client, so.service.bucket, so.service.db, so.service.emulator)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Integrity check failed: %v", err))
		report.ExecutionTime = time.Since(startTime).String()
		return report, nil
	}
	so.service.logger.Info("Integrity check completed", zap.Duration("duration", time.Since(stepStart)))

	// 3. Schema sync
	stepStart = time.Now()
	so.service.logger.Info("Syncing database schema")
	schemaChanges, err := so.SyncSchema(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Schema sync failed: %v", err))
		report.ExecutionTime = time.Since(startTime).String()
		return report, nil
	}
	report.SchemaChanges = schemaChanges
	so.service.logger.Info("Schema sync completed", zap.Duration("duration", time.Since(stepStart)), zap.Int("changes", len(schemaChanges)))

	// 3.5 Cleanup Duplicates (Before Data Sync to avoid updating deleted rows)
	stepStart = time.Now()
	if err := so.CleanupDuplicates(ctx); err != nil {
		so.service.logger.Error("Duplicate cleanup failed", zap.Error(err))
		// Don't fail the sync, just log
	}

	// 4. Data sync
	if !skipDataSync {
		stepStart = time.Now()
		so.service.logger.Info("Syncing data (batch mode)")
		rowsUpdated, err := so.SyncDataBatch(ctx, furniData, integrityReport, so.service.logger)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Data sync failed: %v", err))
			report.ExecutionTime = time.Since(startTime).String()
			return report, nil
		}
		report.RowsUpdated = rowsUpdated
		so.service.logger.Info("Data sync completed", zap.Duration("duration", time.Since(stepStart)), zap.Int("rows_updated", rowsUpdated))
	} else {
		report.RowsUpdated = 0
		so.service.logger.Info("Data sync skipped")
	}

	// 5. Remove missing assets
	stepStart = time.Now()
	so.service.logger.Info("Removing missing assets")
	so.DebugDBCount()
	storageDeleted, databaseDeleted, furniDataDeleted, err := so.RemoveMissingAssets(ctx, furniData, integrityReport)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Asset removal failed: %v", err))
		report.ExecutionTime = time.Since(startTime).String()
		return report, nil
	}
	report.StorageDeleted = storageDeleted
	report.DatabaseDeleted = databaseDeleted
	report.FurniDataDeleted = furniDataDeleted
	report.AssetsDeleted = storageDeleted + databaseDeleted + furniDataDeleted
	so.service.logger.Info("Asset removal completed",
		zap.Duration("duration", time.Since(stepStart)),
		zap.Int("database_deleted", databaseDeleted),
		zap.Int("storage_deleted", storageDeleted))

	report.ExecutionTime = time.Since(startTime).String()
	so.service.logger.Info("Sync completed", zap.String("total_time", report.ExecutionTime))
	return report, nil
}

func (so *SyncOperations) DebugDBCount() {
	var totalCount int64
	so.service.db.Table(so.service.GetTableName()).Count(&totalCount)
	so.service.logger.Info("Database total row count", zap.Int64("count", totalCount))
}

// CleanupDuplicates removes duplicate rows sharing the same sprite_id
func (so *SyncOperations) CleanupDuplicates(ctx context.Context) error {
	tableName := so.service.GetTableName()
	if tableName == "" {
		return nil
	}

	// SQL Agnostic approach: Find duplicates then delete
	// 1. Find sprite_ids with duplicates
	type DuplicateResult struct {
		SpriteID int
		Count    int
	}
	var duplicates []DuplicateResult

	err := so.service.db.Table(tableName).
		Select("sprite_id, count(*) as count").
		Group("sprite_id").
		Having("count(*) > 1").
		Scan(&duplicates).Error

	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		so.service.logger.Info("No duplicates found")
		return nil
	}

	var deletedCount int64

	// 2. For each duplicate group, keep the max ID, delete others
	for _, dup := range duplicates {
		var ids []int
		// Fetch all IDs for this sprite_id
		if err := so.service.db.Table(tableName).Select("id").Where("sprite_id = ?", dup.SpriteID).Order("id DESC").Scan(&ids).Error; err != nil {
			continue
		}

		if len(ids) > 1 {
			// Keep first (max ID because Ordered DESC), delete rest
			toDelete := ids[1:]
			res := so.service.db.Table(tableName).Where("id IN ?", toDelete).Delete(nil)
			if res.Error == nil {
				deletedCount += res.RowsAffected
			}
		}
	}

	if deletedCount > 0 {
		so.service.logger.Info("Removed duplicate assets", zap.Int64("count", deletedCount))
	} else {
		so.service.logger.Info("No duplicates actually removed (check logic)")
	}
	return nil
}
