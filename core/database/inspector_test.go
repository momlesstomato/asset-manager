package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTableColumns(t *testing.T) {
	// Setup In-Memory DB
	cfg := Config{
		Driver: "sqlite",
		Name:   ":memory:",
	}
	db, err := Connect(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Create a test table
	// SQLite specific types: INTEGER, TEXT.
	err = db.Exec("CREATE TABLE test_items (id INTEGER PRIMARY KEY, name TEXT, description TEXT)").Error
	assert.NoError(t, err)

	// Test GetTableColumns
	columns, err := GetTableColumns(db, "test_items")
	assert.NoError(t, err)
	assert.Len(t, columns, 3)

	// Map columns to map for easy assertion
	colMap := make(map[string]string)
	for _, col := range columns {
		colMap[col.Field] = col.Type
	}

	assert.Equal(t, "integer", colMap["id"])
	assert.Equal(t, "text", colMap["name"])
	assert.Equal(t, "text", colMap["description"])

	// Test non-existent table
	cols, err := GetTableColumns(db, "non_existent")
	// PRAGMA table_info returns empty result for non-existent table in SQLite, implies no error but empty columns
	assert.NoError(t, err)
	assert.Empty(t, cols)
}
