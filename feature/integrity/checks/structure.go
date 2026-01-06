package checks

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

// CheckStructure returns a list of missing folders.
func CheckStructure(ctx context.Context, client storage.Client, bucket string) ([]string, error) {
	var missing []string

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", bucket)
	}

	for _, folder := range RequiredFolders {
		folderPath := folder
		if !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
		}

		opts := minio.ListObjectsOptions{
			Prefix:    folderPath,
			Recursive: false,
			MaxKeys:   1,
		}

		found := false
		for range client.ListObjects(ctx, bucket, opts) {
			found = true
			break
		}

		if !found {
			missing = append(missing, folder)
		}
	}

	return missing, nil
}

// FixStructure creates the missing folders.
func FixStructure(ctx context.Context, client storage.Client, bucket string, logger *zap.Logger, missing []string) error {
	for _, folder := range missing {
		folderPath := folder
		if !strings.HasSuffix(folderPath, "/") {
			folderPath += "/"
		}

		_, err := client.PutObject(ctx, bucket, folderPath, bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
		if err != nil {
			logger.Error("Failed to create folder", zap.String("folder", folder), zap.Error(err))
			return err
		}
		logger.Info("Created missing folder", zap.String("folder", folder))
	}
	return nil
}
