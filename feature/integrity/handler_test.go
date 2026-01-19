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

	// Mock Service call
	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	// Structure check logic calls checking specific folders.
	// Since we are integration testing the handler with mocked service dependencies,
	// we need to expect what Checks.CheckStructure calls.
	// CheckStructure calls ListObjects for each folder.
	// We'll mock ListObjects to return empty channel (missing folders)
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	req := httptest.NewRequest("GET", "/integrity/structure", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHandleServerCheck(t *testing.T) {
	app, _, sqlMock := setupTestApp(t)

	// Expect DB queries
	permissions := sqlmock.NewRows([]string{"id", "permission", "description"})
	sqlMock.ExpectQuery("SELECT \\* FROM permissions").WillReturnRows(permissions)

	pages := sqlmock.NewRows([]string{"id", "parent_id", "caption"})
	sqlMock.ExpectQuery("SELECT \\* FROM catalog_pages").WillReturnRows(pages)

	items := sqlmock.NewRows([]string{"id", "sprite_id", "public_name"})
	sqlMock.ExpectQuery("SELECT \\* FROM items_base").WillReturnRows(items)

	req := httptest.NewRequest("GET", "/integrity/server", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
