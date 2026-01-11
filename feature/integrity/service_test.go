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
	svc := NewService(mockClient, "test-bucket", logger, nil, "")

	t.Run("CheckStructure", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)

		ch := make(chan minio.ObjectInfo)
		close(ch)
		// checks.CheckStructure calls ListObjects for each required folder
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
	svc := NewService(mockClient, "test-bucket", logger, nil, "")

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
	svc := NewService(mockClient, "test-bucket", logger, nil, "")

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
	svc := NewService(mockClient, "test-bucket", logger, nil, "")

	// Mock valid JSON
	mockJSON := `{"roomitemtypes":{"furnitype":[]},"wallitemtypes":{"furnitype":[]}}`

	t.Run("Failure", func(t *testing.T) {
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(false, nil).Once()
		report, err := svc.CheckFurniture(context.Background(), false)
		assert.Error(t, err)
		assert.Nil(t, report)
	})

	t.Run("Success", func(t *testing.T) {
		// Mock BucketExists
		mockClient.On("BucketExists", mock.Anything, "test-bucket").Return(true, nil)

		// Mock GetObject
		mockClient.On("GetObject", mock.Anything, "test-bucket", "gamedata/FurnitureData.json", mock.Anything).
			Return(io.NopCloser(bytes.NewReader([]byte(mockJSON))), nil)

		// Mock ListObjects for concurrent scanner
		// We use a catch-all mock that returns an empty channel.
		emptyCh := func() <-chan minio.ObjectInfo {
			ch := make(chan minio.ObjectInfo)
			close(ch)
			return ch
		}

		// The scanner will make multiple calls concurrently.
		// testify/mock handles concurrent calls to the same mock method if the return values are safe.
		// Returns a channel which is read-only, so it should be fine.
		mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).Return(emptyCh())

		report, err := svc.CheckFurniture(context.Background(), false)
		assert.NoError(t, err)
		assert.NotNil(t, report)
	})
}
