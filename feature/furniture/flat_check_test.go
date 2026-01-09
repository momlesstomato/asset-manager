package furniture

import (
	"bytes"
	"context"
	"io"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckFurnitureItem_FoundFlatPath(t *testing.T) {
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

	// Mock Storage ListObjects (Flat file found)
	// Expect ListObjects call for bundled/furniture/lamp.nitro
	createCh := func(key string) <-chan minio.ObjectInfo {
		ch := make(chan minio.ObjectInfo, 1)
		ch <- minio.ObjectInfo{Key: key}
		close(ch)
		return ch
	}

	// We expect checking bundled/furniture/lamp.nitro
	mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
		return opts.Prefix == "bundled/furniture/lamp.nitro" && opts.MaxKeys == 1
	})).Return(createCh("bundled/furniture/lamp.nitro"))

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
