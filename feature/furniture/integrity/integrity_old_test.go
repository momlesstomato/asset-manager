package integrity

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"asset-manager/core/storage/mocks"
	"asset-manager/feature/furniture/models"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckIntegrity(t *testing.T) {
	// Mock JSON data
	mockJSON := `{
		"roomitemtypes": {
			"furnitype": [
				{"id": 1, "classname": "chair", "name": "Chair", "category": "furniture"},
				{"id": 2, "classname": "table*1", "name": "Table", "category": "furniture"}
			]
		},
		"wallitemtypes": {
			"furnitype": [
				{"id": 3, "classname": "picture_frame", "name": "Frame", "category": "wall"},
				{"id": 4, "classname": "poster*123", "name": "Poster", "category": "wall"}
			]
		}
	}`

	// Expected files: chair.nitro, table.nitro, picture_frame.nitro, poster.nitro

	t.Run("All Files Present", func(t *testing.T) {
		mockClient := new(mocks.Client)

		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		// Mock GetObject
		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

		// Mock ListObjects
		createCh := func(key string) <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo, 1)
			ch <- minio.ObjectInfo{Key: key}
			close(ch)
			return ch
		}

		emptyCh := func() <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo)
			close(ch)
			return ch
		}

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/c"
		})).Return(createCh("bundled/furniture/chair.nitro"))

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/t"
		})).Return(createCh("bundled/furniture/table.nitro"))

		chP := make(chan minio.ObjectInfo, 2)
		chP <- minio.ObjectInfo{Key: "bundled/furniture/picture_frame.nitro"}
		chP <- minio.ObjectInfo{Key: "bundled/furniture/poster.nitro"}
		close(chP)

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/p"
		})).Return((<-chan minio.ObjectInfo)(chP))

		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return(emptyCh())

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.NoError(t, err)
		assert.IsType(t, &models.Report{}, report)
		assert.Equal(t, 4, report.TotalExpected)
		assert.Equal(t, 4, report.TotalFound)
		assert.Empty(t, report.MissingAssets)
		assert.Empty(t, report.UnregisteredAssets)
		assert.Empty(t, report.MalformedAssets)
	})

	t.Run("Missing and Extra Files", func(t *testing.T) {
		mockClient := new(mocks.Client)

		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

		emptyCh := func() <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo)
			close(ch)
			return ch
		}

		createCh := func(keys ...string) <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo, len(keys))
			for _, k := range keys {
				ch <- minio.ObjectInfo{Key: k}
			}
			close(ch)
			return ch
		}

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/c"
		})).Return(createCh("bundled/furniture/chair.nitro"))

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/t"
		})).Return(createCh("bundled/furniture/table.nitro"))

		mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
			return opts.Prefix == "bundled/furniture/e"
		})).Return(createCh("bundled/furniture/extra.nitro"))

		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return(emptyCh())

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.NoError(t, err)
		assert.Equal(t, 4, report.TotalExpected)
		assert.Equal(t, 3, report.TotalFound)
		assert.Len(t, report.MissingAssets, 2)
		assert.Contains(t, report.MissingAssets, "picture_frame.nitro")
		assert.Contains(t, report.MissingAssets, "poster.nitro")
		assert.Len(t, report.UnregisteredAssets, 1)
		assert.Contains(t, report.UnregisteredAssets, "extra.nitro")
		assert.Empty(t, report.MalformedAssets)
	})

	t.Run("Malformed JSON Data", func(t *testing.T) {
		malformedJSON := `{
			"roomitemtypes": {
				"furnitype": [
					{"id": 1, "classname": "chair"},
					{"id": 0, "classname": "broken"}
				]
			},
			"wallitemtypes": { "furnitype": [] }
		}`

		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)
		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(malformedJSON))), nil)
		// Return empty for listing to simplify
		channel := make(chan minio.ObjectInfo)
		close(channel)
		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return((<-chan minio.ObjectInfo)(channel))

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.NoError(t, err)
		// chair missing name -> malformed
		// broken missing name & ID=0 -> malformed
		assert.NotEmpty(t, report.MalformedAssets)
		assert.Len(t, report.MalformedAssets, 2)
	})

	t.Run("Bucket Does Not Exist", func(t *testing.T) {
		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(false, nil)

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket assets not found")
		assert.Nil(t, report)
	})

	t.Run("Bucket Check Error", func(t *testing.T) {
		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(false, errors.New("connection failed"))

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
		assert.Nil(t, report)
	})

	t.Run("Load Data Error - GetObject Failed", func(t *testing.T) {
		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)
		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(nil, errors.New("read failed"))

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read failed")
		assert.Nil(t, report)
	})

	t.Run("Load Data Error - Unmarshal Failed", func(t *testing.T) {
		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		invalidJSON := "{ invalid json"
		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(invalidJSON))), nil)

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse FurnitureData.json")
		assert.Nil(t, report)
	})

	t.Run("Scan Error - ListObjects Failed", func(t *testing.T) {
		mockClient := new(mocks.Client)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		validJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`
		mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(validJSON))), nil)

		// Return an error channel from ListObjects with enough items for all goroutines
		errCh := func() <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo, 100)
			for i := 0; i < 100; i++ {
				ch <- minio.ObjectInfo{Err: errors.New("list failed")}
			}
			close(ch)
			return ch
		}

		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return(errCh())

		report, err := CheckIntegrity(context.Background(), mockClient, "assets", nil, "")
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "list failed")
		}
		assert.Nil(t, report)
	})
}
