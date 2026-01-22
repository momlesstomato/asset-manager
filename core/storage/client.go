package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client defines the interface for storage operations.
type Client interface {
	// BucketExists checks if a bucket exists.
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	// MakeBucket creates a new bucket.
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	// PutObject uploads an object.
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	// GetObject downloads an object.
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error)
	// ListObjects lists objects in a bucket.
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	// RemoveObject deletes an object from a bucket.
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	// RemoveObjects deletes multiple objects from a bucket efficiently.
	// objectsCh is a channel of object names to delete.
	RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
}

// NewClient creates a new Minio client based on the configuration.
func NewClient(cfg Config) (Client, error) {
	// Minio expects endpoint without scheme
	endpoint := strings.TrimPrefix(cfg.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Ensure timeout defaults if not set
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create custom transport with strict timeouts
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeoutDuration, // Connection setup timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeoutDuration, // TLS Handshake timeout
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: timeoutDuration, // Wait for first response byte timeout
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL,
		Region:    cfg.Region,
		Transport: transport,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	// Note: Minio client performs lazy connection, so we can't ping here easily without a bucket check
	// But ListBuckets or similar would verify. We rely on operation-level timeouts from Context for the rest.
	// The transport timeouts ensure we don't hang on connection setup.

	return &minioClientWrapper{Client: minioClient}, nil
}

type minioClientWrapper struct {
	*minio.Client
}

func (c *minioClientWrapper) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	return c.Client.GetObject(ctx, bucketName, objectName, opts)
}
