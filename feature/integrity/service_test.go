package integrity

import (
	"bytes"
	"context"
	"io"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestService_Structure(t *testing.T) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, nil)

	t.Run("CheckStructure", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		// Mock ListObjects for required folders
		// Using a simplified approach: just return empty for keys to simulate missing
		ch := make(chan minio.ObjectInfo)
		close(ch)
		// checks.CheckFolders calls ListObjects for each folder
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		missing, err := svc.CheckStructure(context.Background())
		assert.NoError(t, err)
		assert.NotEmpty(t, missing)
	})

	t.Run("FixStructure", func(t *testing.T) {
		mockClient.On("PutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, int64(0), mock.Anything).Return(minio.UploadInfo{}, nil)
		err := svc.FixStructure(context.Background(), []string{"bundled"})
		assert.NoError(t, err)
	})
}

func TestService_GameData(t *testing.T) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, nil)

	mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
	ch := make(chan minio.ObjectInfo)
	close(ch)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

	missing, err := svc.CheckGameData(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, missing)
}

func TestService_Bundled(t *testing.T) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, nil)

	t.Run("CheckBundled", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		missing, err := svc.CheckBundled(context.Background())
		assert.NoError(t, err)
		assert.NotEmpty(t, missing)
	})

	t.Run("FixBundled", func(t *testing.T) {
		mockClient.On("PutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, int64(0), mock.Anything).Return(minio.UploadInfo{}, nil)
		err := svc.FixBundled(context.Background(), []string{"bundled/furni"})
		assert.NoError(t, err)
	})
}

func TestService_Furniture(t *testing.T) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	svc := NewService(mockClient, "test-bucket", logger, nil)

	// Mock furniture integrity check dependency
	// Since furniture.CheckIntegrity uses the client, we mock the client behaviour for furniture check.
	// This mirrors logic in furniture/integrity_test.go

	// Mock valid JSON
	mockJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`

	t.Run("Failure", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil).Once()
		report, err := svc.CheckFurniture(context.Background())
		assert.Error(t, err)
		assert.Nil(t, report)
	})

	t.Run("Success", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		report, err := svc.CheckFurniture(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, report)
	})
}
