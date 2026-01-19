package reconcile

import (
	"context"
	"io"
	"strings"
	"testing"

	"asset-manager/core/reconcile"
	"asset-manager/core/storage/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupMockDB creates a mock GORM DB for testing.
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open mock sql db: %v", err)
	}

	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open gorm db: %v", err)
	}

	return gormDB, mock
}

func TestFurnitureAdapter_ExtractKeys(t *testing.T) {
	adapter := NewAdapter()

	// Test ExtractDBKey
	t.Run("ExtractDBKey", func(t *testing.T) {
		item := DBItem{ID: 1, SpriteID: 500}
		key := adapter.ExtractDBKey(item)
		assert.Equal(t, "500", key, "Should use SpriteID as key")
	})

	// Test ExtractGDKey
	t.Run("ExtractGDKey", func(t *testing.T) {
		item := GDItem{ID: 100}
		key := adapter.ExtractGDKey(item)
		assert.Equal(t, "100", key, "Should use ID as key")
	})

	// Test ExtractStorageKey
	t.Run("ExtractStorageKey", func(t *testing.T) {
		// Populate map first
		adapter.mu.Lock()
		adapter.classnameToID["chair"] = "200"
		adapter.mu.Unlock()

		// Valid storage key
		key, ok := adapter.ExtractStorageKey("bundled/furniture/chair.nitro", ".nitro")
		assert.True(t, ok)
		assert.Equal(t, "200", key, "Should map classname to ID")

		// Unknown classname -> use classname as fallback
		key, ok = adapter.ExtractStorageKey("bundled/furniture/unknown.nitro", ".nitro")
		assert.True(t, ok)
		assert.Equal(t, "unknown", key, "Should fallback to classname if not found")

		// Invalid extension
		_, ok = adapter.ExtractStorageKey("bundled/furniture/chair.png", ".nitro")
		assert.False(t, ok, "Should reject wrong extension")

		// Nested path
		key, ok = adapter.ExtractStorageKey("bundled/furniture/nested/chair.nitro", ".nitro")
		assert.True(t, ok)
		assert.Equal(t, "200", key, "Should handle nested paths")
	})
}

func TestFurnitureAdapter_LoadGamedataIndex(t *testing.T) {
	adapter := NewAdapter()
	mockClient := new(mocks.Client)

	// Mock JSON data with Room and Wall items
	mockJSON := `{
		"roomitemtypes": {
			"furnitype": [
				{"id": 10, "classname": "floor_item", "name": "Floor Item"}
			]
		},
		"wallitemtypes": {
			"furnitype": [
				{"id": 20, "classname": "wall_item", "name": "Wall Item"}
			]
		}
	}`

	mockClient.On("GetObject", mock.Anything, "bucket", "gamedata.json", mock.Anything).
		Return(io.NopCloser(strings.NewReader(mockJSON)), nil)

	ctx := context.Background()
	index, err := adapter.LoadGamedataIndex(ctx, mockClient, "bucket", "gamedata.json", []string{})
	assert.NoError(t, err)

	// Verify items loaded
	assert.Len(t, index, 2)

	// Verify Floor Item (RoomItemTypes -> Type "s")
	floorItem := index["10"].(GDItem)
	assert.Equal(t, "floor_item", floorItem.ClassName)
	assert.Equal(t, "s", floorItem.Type, "Room items should be type 's'")

	// Verify Wall Item (WallItemTypes -> Type "i")
	wallItem := index["20"].(GDItem)
	assert.Equal(t, "wall_item", wallItem.ClassName)
	assert.Equal(t, "i", wallItem.Type, "Wall items should be type 'i'")

	// Verify Mapping populated
	adapter.mu.RLock()
	assert.Equal(t, "10", adapter.classnameToID["floor_item"])
	assert.Equal(t, "20", adapter.classnameToID["wall_item"])
	adapter.mu.RUnlock()

	// Verify channel closed
	select {
	case <-adapter.mappingReady:
		// success
	default:
		t.Error("mappingReady channel should be closed")
	}
}

func TestFurnitureAdapter_LoadDBIndex_Profiles(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		tableName string
		mockRun   func(sqlmock.Sqlmock)
	}{
		{
			name:      "Arcturus",
			profile:   "arcturus",
			tableName: "items_base",
			mockRun: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
				rows.AddRow(1, 100, "chair", "Public Chair", 1, 1, 1, "s")
				mock.ExpectQuery("SELECT \\* FROM items_base").WillReturnRows(rows)
			},
		},
		{
			name:      "Comet",
			profile:   "comet",
			tableName: "furniture",
			mockRun: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "can_sit", "type"})
				rows.AddRow(1, 100, "chair", "Public Chair", 1, 1, "1", "s") // Comet uses strings for some bools
				mock.ExpectQuery("SELECT \\* FROM furniture").WillReturnRows(rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			db, mock := setupMockDB(t)

			tt.mockRun(mock)

			index, err := adapter.LoadDBIndex(context.Background(), db, tt.profile)
			assert.NoError(t, err)
			assert.Len(t, index, 1)

			item := index["100"].(DBItem)
			assert.Equal(t, 100, item.SpriteID)
			assert.Equal(t, "chair", item.ItemName)
			assert.Equal(t, "s", item.Type)
			assert.True(t, item.CanSit) // Should handle 1/"1" conversion
		})
	}
}

func TestFurnitureAdapter_CompareFields(t *testing.T) {
	adapter := NewAdapter()

	t.Run("Match", func(t *testing.T) {
		db := DBItem{PublicName: "Chair", ItemName: "chair", Width: 1, Type: "s"}
		gd := GDItem{Name: "Chair", ClassName: "chair", XDim: 1, Type: "s"}
		mismatches := adapter.CompareFields(db, gd)
		assert.Empty(t, mismatches)
	})

	t.Run("Name Fallback", func(t *testing.T) {
		// DB PublicName matches GD ClassName -> Should be ALLOWED
		db := DBItem{PublicName: "chair_classname", ItemName: "chair_classname", Type: "s"}
		gd := GDItem{Name: "Real Name", ClassName: "chair_classname", Type: "s"}
		mismatches := adapter.CompareFields(db, gd)
		assert.Empty(t, mismatches)
	})

	t.Run("Type Mismatch Wall vs Room", func(t *testing.T) {
		// DB says Wall (i), GD says Room (s)
		db := DBItem{Type: "i", PublicName: "N", ItemName: "C"}
		gd := GDItem{Type: "s", Name: "N", ClassName: "C"}
		mismatches := adapter.CompareFields(db, gd)
		assert.Len(t, mismatches, 1)
		assert.Contains(t, mismatches[0], "type: gd='room'")
	})

	t.Run("Type Mismatch Room vs Wall", func(t *testing.T) {
		// DB says Room (s), GD says Wall (i)
		db := DBItem{Type: "s", PublicName: "N", ItemName: "C"}
		gd := GDItem{Type: "i", Name: "N", ClassName: "C"}
		mismatches := adapter.CompareFields(db, gd)
		assert.Len(t, mismatches, 1)
		assert.Contains(t, mismatches[0], "type: gd='wall'")
	})
}

func TestFurnitureAdapter_LoadStorageSet(t *testing.T) {
	adapter := NewAdapter()
	mockClient := new(mocks.Client)

	// Setup ListObjects mock
	objCh := make(chan minio.ObjectInfo, 1)
	objCh <- minio.ObjectInfo{Key: "bundled/furniture/chair.nitro"}
	close(objCh)

	mockClient.On("ListObjects", mock.Anything, "bucket", mock.Anything).
		Return((<-chan minio.ObjectInfo)(objCh))

	// Populate mapping
	adapter.mu.Lock()
	adapter.classnameToID["chair"] = "100"
	adapter.mu.Unlock()

	// Signal readiness immediately
	close(adapter.mappingReady)

	set, err := adapter.LoadStorageSet(context.Background(), mockClient, "bucket", "bundled/furniture", ".nitro")
	assert.NoError(t, err)

	if _, ok := set["100"]; !ok {
		t.Error("expected key '100' not found")
	}
}

func TestFurnitureAdapter_QueryDB(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		query     reconcile.Query
		mockRun   func(sqlmock.Sqlmock)
		expectErr bool
		expectNil bool
	}{
		{
			name:    "Found by ID",
			profile: "arcturus",
			query:   reconcile.Query{ID: "100"},
			mockRun: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
				rows.AddRow(1, 100, "chair", "Public Chair", 1, 1, 1, "s")
				// QueryDB uses ColID ("id") for ByID lookup, not sprite_id
				// Use .* to match optional LIMIT/ORDER BY clauses added by GORM
				// Also handle optional backticks for table/column names
				mock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE id = \\?.*").
					WithArgs(100, 1). // Add LIMIT arg (1)
					WillReturnRows(rows)
			},
		},
		{
			name:    "Found by Name",
			profile: "arcturus",
			query:   reconcile.Query{Name: "chair"},
			mockRun: func(mock sqlmock.Sqlmock) {
				// ID is empty, so ByID check is skipped.
				// Classname is empty, so ByClassname check is skipped.

				// Name lookup succeeds
				rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
				rows.AddRow(1, 100, "chair", "Public Chair", 1, 1, 1, "s")
				mock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE public_name = \\?.*").
					WithArgs("chair", 1). // Add LIMIT arg (1)
					WillReturnRows(rows)
			},
		},
		{
			name:    "Not Found",
			profile: "arcturus",
			query:   reconcile.Query{ID: "999", Name: "unknown"},
			mockRun: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE id = \\?.*").WithArgs(999, 1).WillReturnRows(sqlmock.NewRows([]string{}))
				// If ID not found, it proceeds to Classname (empty) then Name
				mock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE public_name = \\?.*").WithArgs("unknown", 1).WillReturnRows(sqlmock.NewRows([]string{}))
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			db, mock := setupMockDB(t)

			tt.mockRun(mock)

			item, err := adapter.QueryDB(context.Background(), db, tt.profile, tt.query)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, item)
			} else {
				assert.NotNil(t, item)
			}
		})
	}
}
