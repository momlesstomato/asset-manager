package reconcile

import (
	"context"
	"fmt"
	"testing"
	"time"

	"asset-manager/core/reconcile"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite DB for testing mutations
func setupTestDB(t *testing.T, dbName string) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create table for Arcturus
	err = db.Exec(`CREATE TABLE items_base (
		id INTEGER PRIMARY KEY,
		sprite_id INTEGER,
		item_name VARCHAR(60),
		public_name VARCHAR(60),
		width INTEGER,
		length INTEGER,
		stack_height INTEGER,
		allow_stack INTEGER,
		allow_sit INTEGER,
		allow_walk INTEGER,
		allow_lay INTEGER,
		type VARCHAR(1),
		interaction_type VARCHAR(100)
	)`).Error
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func TestSyncDBFromGamedata_Truncation(t *testing.T) {
	db := setupTestDB(t, "db_truncation")
	adapter := NewAdapter()
	adapter.SetMutationContext(db, nil, "", "", "arcturus", "")

	// Insert initial row
	initialRow := `INSERT INTO items_base (id, sprite_id, item_name, public_name) VALUES (1, 100, 'old_name', 'Old Name')`
	db.Exec(initialRow)

	// Create a long name that needs truncation (over 110 chars)
	longName := "This is a very long name that should be truncated because it exceeds the limit of one hundred and ten characters safely for the database schema."
	expectedName := longName[:110]

	gdItem := GDItem{
		ID:        100,
		ClassName: longName, // Also test classname truncation
		Name:      longName,
		XDim:      1,
		YDim:      1,
	}

	// Perform Sync
	err := adapter.SyncDBFromGamedata(context.Background(), "100", gdItem)
	assert.NoError(t, err)

	// Verify DB state
	var result map[string]interface{}
	db.Table("items_base").Where("sprite_id = ?", 100).Take(&result)

	assert.Equal(t, expectedName, result["public_name"], "Public Name should be truncated")
	assert.Equal(t, expectedName, result["item_name"], "Item Name should be truncated")
}

func TestSyncDBBatch_Concurrency(t *testing.T) {
	db := setupTestDB(t, "db_concurrency")
	adapter := NewAdapter()
	adapter.SetMutationContext(db, nil, "", "", "arcturus", "")

	count := 100
	actions := make([]reconcile.Action, count)

	// Insert 100 rows
	for i := 0; i < count; i++ {
		db.Exec("INSERT INTO items_base (sprite_id, public_name) VALUES (?, ?)", i, "Original")

		actions[i] = reconcile.Action{
			Key:  fmt.Sprintf("%d", i),
			Type: reconcile.ActionSyncDB,
			GDItem: GDItem{
				ID:   i,
				Name: fmt.Sprintf("Updated_%d", i),
			},
		}
	}

	// Run Batch Sync
	start := time.Now()
	err := adapter.SyncDBBatch(context.Background(), actions)
	assert.NoError(t, err)
	duration := time.Since(start)

	t.Logf("Synced %d items in %v", count, duration)

	// Verify all updated
	var results []struct {
		SpriteID   int
		PublicName string
	}
	db.Table("items_base").Order("sprite_id").Find(&results)

	assert.Len(t, results, count)
	for i, res := range results {
		expected := fmt.Sprintf("Updated_%d", i)
		assert.Equal(t, expected, res.PublicName, "Row %d should be updated", i)
	}
}
