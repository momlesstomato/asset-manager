package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"asset-manager/core/config"
	"asset-manager/core/storage"
	"asset-manager/feature/furniture/reconcile"

	"github.com/minio/minio-go/v7"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal(err)
	}

	client, err := storage.NewClient(cfg.Storage)
	if err != nil {
		log.Fatal(err)
	}

	adapter := reconcile.NewAdapter()
	ctx := context.Background()

	// First load gamedata to build the mapping
	fmt.Println("Loading gamedata...")
	gdIndex, err := adapter.LoadGamedataIndex(ctx, client, cfg.Storage.Bucket, "gamedata/FurnitureData.json", []string{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded %d gamedata items\n", len(gdIndex))

	// Now check storage files for 02_caterbody
	fmt.Println("\n=== Checking storage for files containing '02_caterbody' ===")

	opts := minio.ListObjectsOptions{
		Prefix:    "bundled/furniture",
		Recursive: true,
	}

	count := 0
	for obj := range client.ListObjects(ctx, cfg.Storage.Bucket, opts) {
		if obj.Err != nil {
			log.Fatal(obj.Err)
		}

		if strings.Contains(obj.Key, "02_caterbody") {
			count++
			// Try to extract key
			key, ok := adapter.ExtractStorageKey(obj.Key, ".nitro")
			fmt.Printf("File: %s\n  -> Extracted key: %s, success: %v\n", obj.Key, key, ok)

			// Check if key is numeric (mapped) or string (unmapped)
			if key == "02_caterbody" {
				fmt.Println("  ⚠️  Fallback to classname - mapping failed!")
			} else {
				fmt.Println("  ✅ Successfully mapped to ID")
			}
		}
	}

	fmt.Printf("\nTotal files containing '02_caterbody': %d\n", count)
}
