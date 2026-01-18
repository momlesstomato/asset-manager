package integrity

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"asset-manager/core/storage/mocks"
	"asset-manager/feature/furniture/models"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandler() (*Handler, *mocks.Client) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	// Create an in-memory SQLite DB using GORM for integrity checks that require a DB connection
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Auto-migrate the schema for the in-memory DB because CheckIntegrity queries it.
	// Since we pass "arcturus" as emulator, we migrate ArcturusItemsBase.
	if err := db.AutoMigrate(&models.ArcturusItemsBase{}); err != nil {
		panic(err)
	}

	// No DB query expectations needed for SQLite in-memory DB
	svc := NewService(mockClient, "test-bucket", logger, db, "arcturus")
	return NewHandler(svc), mockClient
}

func TestHandler_HandleIntegrityCheck(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity", h.HandleIntegrityCheck)

		// Mock Structure Check
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		chStructure := make(chan minio.ObjectInfo)
		close(chStructure)
		// It might be called multiple times for different prefixes
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(chStructure))

		// Mock Furniture Check (requires GetObject)
		validJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(validJSON))), nil)

		req := httptest.NewRequest("GET", "/integrity", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Error Partial", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity", h.HandleIntegrityCheck)

		// Mock BucketExists failure (affects multiple checks)
		// We use .Maybe() or let it return error.
		// Since checks are sequential, if Structure fails (first), it might continue or not?
		// Handler calls them all sequentially.
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil) // Bucket missing

		// Furniture might try GetObject and fail
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(nil, minio.ErrorResponse{Code: "NoSuchKey"})

		// ListObjects might be called
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/integrity", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// It should still return 200 OK because it returns a report with error statuses
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestHandler_HandleStructureCheck(t *testing.T) {
	t.Run("Check Only", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity/structure", h.HandleStructureCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/integrity/structure", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Fix Logic", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity/structure", h.HandleStructureCheck)

		// Simulate missing
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		// Return empty list objects -> all missing
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		// If fix=true, it calls PutObject
		mockClient.On("PutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, int64(0), mock.Anything).Return(minio.UploadInfo{}, nil)

		req := httptest.NewRequest("GET", "/integrity/structure?fix=true", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // Returns "status": "fixed"
	})
}

func TestHandler_HandleFurnitureCheck(t *testing.T) {
	t.Run("Check Error", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity/furniture", h.HandleFurnitureCheck)

		// Mock failure
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, io.EOF)

		req := httptest.NewRequest("GET", "/integrity/furniture", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	// Success path
	t.Run("Success", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity/furniture", h.HandleFurnitureCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)

		validJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(validJSON))), nil)

		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/integrity/furniture", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Response body: %s", body)
		}
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestHandler_RegisterRoutes(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		h.RegisterRoutes(app)

		// We need to mock calls for /integrity call that happens in this test?
		// No, we just check if route exists. But app.Test() executes the handler if it matches.
		// h.HandleIntegrityCheck triggers all checks.
		// This makes testing "RegisterRoutes" hard without extensive mocking.
		// We can just verify the route logic separately or skip execution.
		// But let's mock headers to 404 on Root path?
		// The test checks "/integrity/" which maps to HandleIntegrityCheck.

		// Mock Dependencies for HandleIntegrityCheck
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))
		validJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(validJSON))), nil)

		req := httptest.NewRequest("GET", "/integrity/", nil)
		resp, _ := app.Test(req)

		// Should receive 200 OK from HandleIntegrityCheck
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}
