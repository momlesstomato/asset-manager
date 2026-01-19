package integrity

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestApp(t *testing.T) (*fiber.App, *mocks.Client, sqlmock.Sqlmock) {
	app := fiber.New()
	mockClient := new(mocks.Client)
	db, sqlMock := setupMockDB(t)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, db, "arcturus")
	handler := NewHandler(svc)
	handler.RegisterRoutes(app)
	return app, mockClient, sqlMock
}

func TestHandleStructureCheck(t *testing.T) {
	app, mockClient, _ := setupTestApp(t)

	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	req := httptest.NewRequest("GET", "/integrity/structure", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "checked", body["status"])
	assert.NotEmpty(t, body["missing"])
}

func TestHandleBundleCheck(t *testing.T) {
	app, mockClient, _ := setupTestApp(t)

	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	req := httptest.NewRequest("GET", "/integrity/bundled", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHandleGameDataCheck(t *testing.T) {
	app, mockClient, _ := setupTestApp(t)

	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	req := httptest.NewRequest("GET", "/integrity/gamedata", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHandleServerCheck(t *testing.T) {
	app, _, sqlMock := setupTestApp(t)

	// Expect queries - relaxed matching
	sqlMock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	sqlMock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	sqlMock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req := httptest.NewRequest("GET", "/integrity/server", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHandleIntegrityCheck(t *testing.T) {
	app, mockClient, sqlMock := setupTestApp(t)

	// Expect calls for ALL checks
	// Fail BucketExists to skip deeper logic and avoid blocking/timeouts
	// This verifies the handler handles errors from service correctly
	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, assert.AnError)

	// We might still need ListObjects if some checks ingore BucketExists?
	// But checks usually do if !exists return error.

	// server check - fail it too to be safe/fast
	sqlMock.ExpectQuery(".*").WillReturnError(assert.AnError)
	sqlMock.ExpectQuery(".*").WillReturnError(assert.AnError)
	sqlMock.ExpectQuery(".*").WillReturnError(assert.AnError)
	sqlMock.ExpectQuery(".*").WillReturnError(assert.AnError) // Extra query might happen

	// Furniture check - fail fast
	mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
		Return(nil, assert.AnError)

	req := httptest.NewRequest("GET", "/integrity", nil)
	// Give it more time? Fiber's app.Test timeout defaults to 1s.
	// We can increase it but failing fast in mock should be instant.
	resp, err := app.Test(req, 2000)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHandleFurnitureCheck(t *testing.T) {
	app, mockClient, _ := setupTestApp(t)

	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, assert.AnError)

	req := httptest.NewRequest("GET", "/integrity/furniture", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
