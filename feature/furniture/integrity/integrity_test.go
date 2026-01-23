package integrity

import (
	"context"
	"io"
	"strings"
	"testing"

	"asset-manager/core/storage/mocks"
	"asset-manager/feature/furniture/models"

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

func TestCheckIntegrity(t *testing.T) {
	// Mock Data
	furniDataJSON := `{
		"roomitemtypes": {
			"furnitype": [
				{"id": 100, "classname": "chair", "name": "Chair"}
			]
		},
		"wallitemtypes": {
			"furnitype": []
		}
	}`

	t.Run("Success", func(t *testing.T) {
		mockClient := new(mocks.Client)
		db, sqlMock := setupMockDB(t)

		// 1. Bucket Exists
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)

		// 2. Gamedata Loading
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(strings.NewReader(furniDataJSON)), nil)

		// 3. Storage Listing
		objCh := make(chan minio.ObjectInfo, 1)
		objCh <- minio.ObjectInfo{Key: "bundled/furniture/chair.nitro"}
		close(objCh)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).
			Return((<-chan minio.ObjectInfo)(objCh)).Once()

		// 4. DB Query (Arcturus profile by default)
		rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
		rows.AddRow(1, 100, "chair", "Chair", 1, 1, 1, "s")
		sqlMock.ExpectQuery("SELECT \\* FROM items_base").WillReturnRows(rows)

		report, err := CheckIntegrity(context.Background(), mockClient, "test-bucket", db, "arcturus")
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, 1, report.TotalExpected)
		assert.Equal(t, 1, report.TotalFound)
		assert.Empty(t, report.MissingAssets)
	})

	t.Run("BucketMissing", func(t *testing.T) {
		mockClient := new(mocks.Client)
		db, sqlMock := setupMockDB(t)

		// Expect bucket check
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil).Once()

		// Since CheckIntegrity calls ReconcileAll which starts LoadDBIndex in parallel,
		// we MUST expect the DB query even if bucket check fails fast.
		// Return empty rows to satisfy the query.
		sqlMock.ExpectQuery("SELECT \\* FROM items_base").WillReturnRows(sqlmock.NewRows([]string{}))

		// LoadStorageSet might run concurrently and call ListObjects.
		// Use Maybe() because raciness determines if it gets called before context cancel.
		emptyCh := make(chan minio.ObjectInfo)
		close(emptyCh)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).
			Return((<-chan minio.ObjectInfo)(emptyCh)).Maybe()

		report, err := CheckIntegrity(context.Background(), mockClient, "test-bucket", db, "arcturus")
		assert.Error(t, err)
		assert.Nil(t, report)
		assert.Contains(t, err.Error(), "bucket test-bucket not found")
	})
}

func TestCheckFurnitureItem(t *testing.T) {
	mockClient := new(mocks.Client)
	db, sqlMock := setupMockDB(t)

	// Mock Data
	furniDataJSON := `{
		"roomitemtypes": {
			"furnitype": [
				{"id": 100, "classname": "chair", "name": "Chair", "cansiton": true, "xdim": 1, "ydim": 1}
			]
		},
		"wallitemtypes": {
			"furnitype": []
		}
	}`

	t.Run("Found", func(t *testing.T) {
		// 1. Gamedata Loading (LoadGamedataIndex called by QueryGamedata)
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(strings.NewReader(furniDataJSON)), nil).Maybe()

		// 2. Storage Check (CheckStorage called by ReconcileOne)
		// Usually ListObjects with prefix or similar. The adapter's CheckStorage uses ListObjects with MaxKeys 1
		objCh := make(chan minio.ObjectInfo, 1)
		objCh <- minio.ObjectInfo{Key: "bundled/furniture/chair.nitro"}
		close(objCh)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).
			Return((<-chan minio.ObjectInfo)(objCh)).Maybe()

		// 3. DB Query (QueryDB called by ReconcileOne)
		rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
		rows.AddRow(1, 100, "chair", "Chair", 1, 1, 1, "s")

		// QueryDB tries ID, then Classname, then Name.
		// If we search by "chair", it tries partials.
		// Adapter logic:
		// Try by ID (fail) -> parsing "chair" as int fails
		// Try by Classname -> Success

		// Adapter QueryDB uses specific queries with LIMIT
		sqlMock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE item_name = \\?.*").
			WithArgs("chair", 1).
			WillReturnRows(rows)

		report, err := CheckFurnitureItem(context.Background(), mockClient, "test-bucket", db, "arcturus", "chair")
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, "PASS", report.IntegrityStatus)
		assert.True(t, report.InFurniData)
		assert.True(t, report.InDB)
		assert.True(t, report.FileExists)
	})
}

func TestCheckIntegrityWithDB(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	furniData := &models.FurnitureData{}
	furniData.RoomItemTypes.FurniType = []models.FurnitureItem{
		{ID: 100, ClassName: "chair", Name: "Chair", XDim: 1, YDim: 1},
	}

	t.Run("Match", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
		rows.AddRow(1, 100, "chair", "Chair", 1, 1, 0, "s") // allow_sit=0
		sqlMock.ExpectQuery("SELECT \\* FROM items_base").WillReturnRows(rows)

		mismatches, err := CheckIntegrityWithDB(context.Background(), furniData, db, "arcturus")
		assert.NoError(t, err)
		assert.Empty(t, mismatches)
	})
}

func TestGetDBFurnitureItem(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	t.Run("Found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "allow_sit", "type"})
		rows.AddRow(1, 100, "chair", "Chair", 1, 1, 1, "s")

		sqlMock.ExpectQuery("SELECT \\* FROM [`]?items_base[`]? WHERE item_name = \\?.*").
			WithArgs("chair", 1).
			WillReturnRows(rows)

		item, err := GetDBFurnitureItem(db, "arcturus", "chair")
		assert.NoError(t, err)
		assert.NotNil(t, item)
		// Note: GetDBFurnitureItem maps ID to SpriteID if adapter parses it that way?
		// Actually adapter.DBItem has ID and SpriteID.
		// GetDBFurnitureItem returns models.DBFurnitureItem.
		// Let's check mapping in integrity.go:
		// models.DBFurnitureItem.ID = item.ID (which is DB ID, not SpriteID?)
		// Wait, adapter loads SpriteID into Key, but DBItem struct has both.
		// GetDBFurnitureItem mapping: ID: item.ID.
		// So it returns the Database ID (1), NOT the SpriteID (100).

		// Wait, looking at code:
		// item.ID = toInt(id) -> DB ID.
		// item.SpriteID = toInt(sprite_id).
		// integrity.go:284: ID: item.ID
		// So assert should match DB ID.
		assert.Equal(t, 1, item.ID) // Matches DB ID
		assert.Equal(t, "chair", item.ItemName)
	})
}
