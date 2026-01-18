package sync_test

import (
	"context"
	"testing"

	"asset-manager/core/database"
	"asset-manager/core/storage/mocks"
	"asset-manager/feature/furniture/models"
	furnituresync "asset-manager/feature/furniture/sync"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSyncOperations(t *testing.T) {
	logger := zap.NewNop()

	// Setup In-Memory DB
	dbCfg := database.Config{Driver: "sqlite", Name: ":memory:"}
	db, err := database.Connect(dbCfg)
	assert.NoError(t, err)

	// Migrate Init Table (Simulate existing state)
	err = db.AutoMigrate(&models.ArcturusItemsBase{})
	assert.NoError(t, err)

	mockClient := new(mocks.Client)
	svc := furnituresync.NewSyncService(mockClient, "assets", db, "arcturus", logger)
	ops := furnituresync.NewSyncOperations(svc)

	t.Run("GetParameterMappings", func(t *testing.T) {
		mappings, err := svc.GetParameterMappings()
		assert.NoError(t, err)
		assert.NotEmpty(t, mappings)
	})

	t.Run("SyncSchema", func(t *testing.T) {
		// New columns (like description) should be added.
		// Initially description might not be in the strict struct used for migration if it was added only in Sync logic?
		// Actually ArcturusItemsBase struct defined in models likely matches CURRENT schema code expectations?
		// If the struct already has it, AutoMigrate added it.
		// But let's check. Struct in models has tags.
		// If SyncSchema logic is "IsNewColumn: true", it tries to add it.
		// If struct has it, GORM AutoMigrate added it. hasColumn returns true, SyncSchema skips.
		// To test SyncSchema actually ADDING, we need a table WITHOUT that column.

		// Let's drop a column to test SyncSchema adding it back?
		// SQLite doesn't support DROP COLUMN easily in old versions, but GORM might.
		// Or we can just trust the "No error" flow if it exists.

		changes, err := ops.SyncSchema(context.Background())
		assert.NoError(t, err)
		// SyncSchema should add new columns that aren't in the base struct
		assert.NotEmpty(t, changes)

		// To force a change, maybe we need to pretend a column is missing?
		// Difficult with SQLite limitations and single shared DB.
		// But basic execution flow validation is good for coverage.
	})

	t.Run("CleanupDuplicates", func(t *testing.T) {
		// Insert duplicates manually
		// 1. Valid item
		item1 := models.ArcturusItemsBase{ID: 10, SpriteID: 555, ItemName: "dup"}
		// 2. Duplicate item (lower ID)
		item2 := models.ArcturusItemsBase{ID: 5, SpriteID: 555, ItemName: "dup"}

		db.Create(&item1)
		db.Create(&item2)

		var count int64
		db.Model(&models.ArcturusItemsBase{}).Where("sprite_id = 555").Count(&count)
		assert.Equal(t, int64(2), count)

		err := ops.CleanupDuplicates(context.Background())
		assert.NoError(t, err)

		db.Model(&models.ArcturusItemsBase{}).Where("sprite_id = 555").Count(&count)
		assert.Equal(t, int64(1), count)

		// Verify ID 10 remains (Highest ID)
		var remaining models.ArcturusItemsBase
		db.First(&remaining, "sprite_id = 555")
		assert.Equal(t, 10, remaining.ID)
	})
}
