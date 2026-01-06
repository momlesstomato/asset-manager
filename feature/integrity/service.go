package integrity

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"asset-manager/core/storage"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// RequiredFolders lists the folders that must exist in the bucket.
var RequiredFolders = []string{
	"bundled", "c_images", "dcr", "gamedata", "images", "logos", "sounds",
}

// Service handles integrity checks.
type Service struct {
	client storage.Client
	bucket string
	logger *zap.Logger
}

// NewService creates a new integrity service.
func NewService(client storage.Client, bucket string, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		bucket: bucket,
		logger: logger,
	}
}

// CheckStructure returns a list of missing folders.
func (s *Service) CheckStructure(ctx context.Context) ([]string, error) {
	var missing []string

	// Ensure bucket exists first
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", s.bucket)
	}

	// For each required folder, check if it exists.
	// We check for the explicit "folder/" object which is standard for empty folders in S3.
	// Alternatively, we could list objects with prefix, but creating "folder/" is the "fix" logic, so we expect it.
	for _, folder := range RequiredFolders {
		folderPath := folder
		if !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
		}

		// Use ListObjects with prefix and maxKeys 1 to see if anything exists?
		// Or try to StatObject? My storage.Client interface only has ListObjects from my previous write.
		// Let's use ListObjects.

		opts := minio.ListObjectsOptions{
			Prefix:    folderPath,
			Recursive: false,
			MaxKeys:   1,
		}

		// If we find the folder entry itself or content inside it, we consider it present?
		// User said: "If not present, create it". This implies we want the folder placeholder.
		// But if there are files inside "images/foo.png", "images/" exists conceptually.
		// However, standard "Create Folder" in S3 creates a 0-byte object.
		// Let's check if there is ANY object with that prefix.

		found := false
		for range s.client.ListObjects(ctx, s.bucket, opts) {
			found = true
			break // found at least one thing
		}

		if !found {
			missing = append(missing, folder)
		}
	}

	return missing, nil
}

// FixStructure creates the missing folders.
func (s *Service) FixStructure(ctx context.Context, missing []string) error {
	for _, folder := range missing {
		folderPath := folder
		if !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
		}

		// Create 0-byte object
		_, err := s.client.PutObject(ctx, s.bucket, folderPath, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
		if err != nil {
			s.logger.Error("Failed to create folder", zap.String("folder", folder), zap.Error(err))
			return err
		}
		s.logger.Info("Created missing folder", zap.String("folder", folder))
	}
	return nil
}
