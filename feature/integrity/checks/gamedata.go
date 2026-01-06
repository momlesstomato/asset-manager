package checks

import (
	"context"
	"fmt"

	"asset-manager/core/storage"

	"github.com/minio/minio-go/v7"
)

// RequiredGameDataFiles lists the files that must exist in the gamedata folder.
var RequiredGameDataFiles = []string{
	"EffectMap.json",
	"FigureData.json",
	"ProductData.json",
	"HabboAvatarActions.json",
	"ExternalTexts.json",
	"UITexts.json",
	"FurnitureData.json",
}

// CheckGameData returns a list of missing files in the gamedata folder.
func CheckGameData(ctx context.Context, client storage.Client, bucket string) ([]string, error) {
	var missing []string

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", bucket)
	}

	for _, filename := range RequiredGameDataFiles {
		filePath := "gamedata/" + filename
		opts := minio.ListObjectsOptions{
			Prefix:    filePath,
			Recursive: false,
			MaxKeys:   1,
		}

		found := false
		for obj := range client.ListObjects(ctx, bucket, opts) {
			if obj.Err == nil && obj.Key == filePath {
				found = true
			}
			break
		}

		if !found {
			missing = append(missing, filename)
		}
	}

	return missing, nil
}
