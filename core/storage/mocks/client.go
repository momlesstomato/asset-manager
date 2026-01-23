package mocks

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

// Client is a mock implementation of storage.Client
type Client struct {
	mock.Mock
}

func (m *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *Client) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

func (m *Client) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *Client) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	if obj, ok := args.Get(0).(io.ReadCloser); ok {
		return obj, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *Client) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, opts)
	if ch, ok := args.Get(0).(<-chan minio.ObjectInfo); ok {
		return ch
	}
	ch := make(chan minio.ObjectInfo)
	close(ch)
	return ch
}

func (m *Client) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func (m *Client) RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	args := m.Called(ctx, bucketName, objectsCh, opts)
	if ch, ok := args.Get(0).(<-chan minio.RemoveObjectError); ok {
		return ch
	}
	ch := make(chan minio.RemoveObjectError)
	close(ch)
	return ch
}
