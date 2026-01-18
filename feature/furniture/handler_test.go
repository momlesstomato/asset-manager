package furniture_test

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"asset-manager/core/database"
	"asset-manager/core/storage/mocks"
	"asset-manager/feature/furniture"
	"asset-manager/feature/furniture/models"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestHandleGetFurnitureDetail(t *testing.T) {
	// Setup Logger
	logger := zap.NewNop()

	// Setup In-Memory DB
	dbCfg := database.Config{
		Driver: "sqlite",
		Name:   ":memory:",
	}
	db, err := database.Connect(dbCfg)
	assert.NoError(t, err)

	// Migrate & Seed
	err = db.AutoMigrate(&models.ArcturusItemsBase{})
	assert.NoError(t, err)
	dbItem := models.ArcturusItemsBase{
		ID:         1,
		SpriteID:   1,
		ItemName:   "handler_test",
		PublicName: "Handler Test",
		Width:      1,
		Length:     1,
		AllowSit:   1,
		Type:       "s",
	}
	err = db.Create(&dbItem).Error
	assert.NoError(t, err)

	// Setup Mock Storage
	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

	furniData := `{
	  "roomitemtypes": {
	    "furnitype": [
	      {
	        "id": 1,
	        "classname": "handler_test",
	        "revision": 1,
	        "category": "test",
	        "name": "Handler Test",
	        "description": "Desc",
	        "xdim": 1,
	        "ydim": 1,
	        "cansiton": true
	      }
	    ]
	  },
	  "wallitemtypes": { "furnitype": [] }
	}`
	mockClient.On("GetObject", mock.Anything, "assets", "gamedata/FurnitureData.json", mock.Anything).
		Return(io.NopCloser(bytes.NewReader([]byte(furniData))), nil)

	mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).
		Return(func(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo, 1)
			if strings.Contains(opts.Prefix, "handler_test.nitro") {
				ch <- minio.ObjectInfo{
					Key: "bundled/furniture/handler_test.nitro",
					Err: nil,
				}
			}
			close(ch)
			return ch
		})

	// Setup Service & Handler
	svc := furniture.NewService(mockClient, "assets", logger, db, "arcturus")
	h := furniture.NewHandler(svc)

	// Setup Fiber
	app := fiber.New()
	h.RegisterRoutes(app)

	// Make Request
	req := httptest.NewRequest("GET", "/furniture/handler_test.nitro", nil)
	resp, err := app.Test(req, 2000) // 2s timeout
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify Body (optional, just status is good for now)
}
