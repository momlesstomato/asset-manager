// Package storage provides an abstraction layer for object storage services.
//
// It wraps the MinIO Go client to provide a simplified interface for common operations
// like checking bucket existence, uploading files, and listing objects. This abstraction
// supports both AWS S3 and self-hosted MinIO instances.
//
// # Client Interface
//
// The Client interface abstracts the underlying storage provider, making it easier
// to mock storage interactions for unit testing (as seen in core/storage/mocks).
//
// # Operations
//
//   - BucketExists: Verifies access to the target bucket.
//   - MakeBucket: Creates a new bucket if needed.
//   - PutObject: Uploads content (with size and options).
//   - GetObject: Retrieves content as a stream.
//   - ListObjects: Lists objects in a bucket (supports prefix/recursive).
//
// # Usage
//
//	client, err := storage.NewClient(config)
//	exists, err := client.BucketExists(ctx, "assets")
package storage
