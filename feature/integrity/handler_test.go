package integrity

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func setupHandler() (*Handler, *mocks.Client) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, nil, "")
	return NewHandler(svc), mockClient
}

func TestHandler_HandleIntegrityCheck(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity", h.HandleIntegrityCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/integrity/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Error", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/integrity", h.HandleIntegrityCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, errors.New("fail"))

		req := httptest.NewRequest("GET", "/integrity/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHandler_HandleStructureCheck(t *testing.T) {
	t.Run("Check Only", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/structure", h.HandleStructureCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/structure", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Fix Logic", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/structure", h.HandleStructureCheck)

		// Simulate missing
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		// Return empty list objects -> all missing
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		// If fix=true, it calls PutObject
		mockClient.On("PutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, int64(0), mock.Anything).Return(minio.UploadInfo{}, nil)

		req := httptest.NewRequest("GET", "/structure?fix=true", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // Returns "status": "fixed"
	})
}

func TestHandler_HandleFurnitureCheck(t *testing.T) {
	t.Run("Check Error", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/furniture", h.HandleFurnitureCheck)

		// Mock failure
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, errors.New("fail"))

		req := httptest.NewRequest("GET", "/furniture", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	// Success path
	t.Run("Success", func(t *testing.T) {
		h, mockClient := setupHandler()
		app := fiber.New()
		app.Get("/furniture", h.HandleFurnitureCheck)

		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)

		validJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(validJSON))), nil)

		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		req := httptest.NewRequest("GET", "/furniture", nil)
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

		// Prevent panic by mocking expected call
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil)

		// Verify route matches
		req := httptest.NewRequest("GET", "/integrity/", nil)
		resp, _ := app.Test(req)

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
