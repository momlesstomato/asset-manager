package furniture

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetDBFurnitureItem(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	// Test by ID (Arcturus)
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "allow_stack", "allow_sit", "allow_lay", "allow_walk", "type", "interaction_type"})
	rows.AddRow(1, 1, "chair", "Chair", 1, 1, 1.0, 0, 1, 0, 0, "s", "default")
	// GORM First adds ORDER BY id LIMIT 1
	sqlMock.ExpectQuery("SELECT \\* FROM `items_base` WHERE `items_base`.`id` = .+ ORDER BY `items_base`.`id` LIMIT .+").
		WithArgs(1, 1).
		WillReturnRows(rows)

	item, err := GetDBFurnitureItem(db, "arcturus", "1")
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "Chair", item.PublicName)

	// Test by Name (Arcturus)
	rows2 := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "allow_stack", "allow_sit", "allow_lay", "allow_walk", "type", "interaction_type"})
	rows2.AddRow(2, 2, "table", "Table", 2, 2, 1.0, 1, 0, 0, 0, "s", "default")
	// GORM Where(...).First adds ORDER BY id LIMIT 1
	sqlMock.ExpectQuery("SELECT \\* FROM `items_base` WHERE item_name = .+ ORDER BY `items_base`.`id` LIMIT .+").
		WithArgs("table", 1).
		WillReturnRows(rows2)

	item2, err := GetDBFurnitureItem(db, "arcturus", "table")
	require.NoError(t, err)
	require.NotNil(t, item2)
	assert.Equal(t, 2, item2.ID)
}

func TestCheckFurnitureItem_Found(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	// Mock FurniData
	mockJSON := `{
		"roomitemtypes": {
			"furnitype": [
				{"id": 10, "classname": "lamp", "name": "Lamp", "category": "furniture", "xdim": 1, "ydim": 1}
			]
		},
		"wallitemtypes": { "furnitype": [] }
	}`

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)
	mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
		Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

	// Mock Storage ListObjects (File found)
	// CheckFurnitureItem Logic: checks "l/lamp.nitro" then "L/lamp.nitro"
	// We simulate finding it immediately on lowercase 'l'.

	createCh := func(key string) <-chan minio.ObjectInfo {
		ch := make(chan minio.ObjectInfo, 1)
		ch <- minio.ObjectInfo{Key: key}
		close(ch)
		return ch
	}

	// Mock Storage ListObjects
	// 1. Flat Check (not found)
	// Return empty channel
	emptyCh := func() <-chan minio.ObjectInfo {
		ch := make(chan minio.ObjectInfo)
		close(ch)
		return ch
	}
	mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
		return opts.Prefix == "bundled/furniture/lamp.nitro"
	})).Return(emptyCh())

	// 2. Subdir Check (Found)
	// Expect ListObjects call for bundled/furniture/l/lamp.nitro
	mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
		return opts.Prefix == "bundled/furniture/l/lamp.nitro" && opts.MaxKeys == 1
	})).Return(createCh("bundled/furniture/l/lamp.nitro"))

	// Mock DB
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "allow_stack", "allow_sit", "allow_lay", "allow_walk", "type", "interaction_type"})
	rows.AddRow(10, 10, "lamp", "Lamp", 1, 1, 1.0, 0, 0, 0, 0, "s", "default")
	sqlMock.ExpectQuery("SELECT \\* FROM `items_base` WHERE `items_base`.`id` = .+ ORDER BY `items_base`.`id` LIMIT .+").
		WithArgs(10, 1).
		WillReturnRows(rows)

	report, err := CheckFurnitureItem(context.Background(), mockClient, "assets", db, "arcturus", "10")
	require.NoError(t, err)
	assert.Equal(t, "PASS", report.IntegrityStatus)
	assert.True(t, report.InFurniData)
	assert.True(t, report.InDB)
	assert.True(t, report.FileExists)
	assert.Empty(t, report.Mismatches)
}

func TestCheckFurnitureItem_MissingDB(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	mockJSON := `{ "roomitemtypes": { "furnitype": [] }, "wallitemtypes": { "furnitype": [] } }`

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)
	mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
		Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

	// DB Lookups fail
	sqlMock.ExpectQuery("SELECT \\* FROM `items_base` WHERE `items_base`.`id` = .+ ORDER BY `items_base`.`id` LIMIT .+").
		WithArgs(999, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	sqlMock.ExpectQuery("SELECT \\* FROM `items_base` WHERE item_name = .+ ORDER BY `items_base`.`id` LIMIT .+").
		WithArgs("999", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// Mock Storage Check (Flat & Subdirs - Not Found)
	emptyCh := func() <-chan minio.ObjectInfo {
		ch := make(chan minio.ObjectInfo)
		close(ch)
		return ch
	}
	mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
		// Matches any prefix for storage check
		return strings.HasPrefix(opts.Prefix, "bundled/furniture/")
	})).Return(emptyCh())

	report, err := CheckFurnitureItem(context.Background(), mockClient, "assets", db, "arcturus", "999")
	require.NoError(t, err)
	assert.Equal(t, "FAIL", report.IntegrityStatus)
	assert.False(t, report.InFurniData)
	assert.False(t, report.InDB)
	assert.Contains(t, report.Mismatches, "Missing in Database")
}
