package checks

import (
	"context"
	"io"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockClient is a mock implementation of storage.Client
// Duplicated here to avoid cycle if imported from somewhere that imports checks?
// Or maybe we can move MockClient to storage package or a test package.
// For now, duplicate it.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

func (m *MockClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, opts)
	if ch, ok := args.Get(0).(<-chan minio.ObjectInfo); ok {
		return ch
	}
	ch := make(chan minio.ObjectInfo)
	close(ch)
	return ch
}

func TestCheckStructure(t *testing.T) {
	t.Run("Bucket Missing", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(false, nil)

		_, err := CheckStructure(context.Background(), mockClient, "assets")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("All Missing", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)
		ch := make(chan minio.ObjectInfo)
		close(ch)
		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		missing, err := CheckStructure(context.Background(), mockClient, "assets")
		assert.NoError(t, err)
		assert.Len(t, missing, len(RequiredFolders))
	})

	t.Run("All Present", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		for _, folder := range RequiredFolders {
			ch := make(chan minio.ObjectInfo, 1)
			ch <- minio.ObjectInfo{Key: folder + "/"}
			close(ch)
			mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
				return opts.Prefix == folder+"/"
			})).Return((<-chan minio.ObjectInfo)(ch))
		}

		missing, err := CheckStructure(context.Background(), mockClient, "assets")
		assert.NoError(t, err)
		assert.Len(t, missing, 0)
	})
}

func TestFixStructure(t *testing.T) {
	logger := zap.NewNop()
	mockClient := new(MockClient)

	mockClient.On("PutObject", mock.Anything, "assets", mock.Anything, mock.Anything, int64(0), mock.Anything).Return(minio.UploadInfo{}, nil)

	err := FixStructure(context.Background(), mockClient, "assets", logger, []string{"bundled"})
	assert.NoError(t, err)
	mockClient.AssertNumberOfCalls(t, "PutObject", 1)
}
