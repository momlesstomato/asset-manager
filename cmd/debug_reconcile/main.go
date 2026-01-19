package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"asset-manager/core/config"
	"asset-manager/core/database"
	"asset-manager/core/storage"
	"asset-manager/feature/furniture/reconcile"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal(err)
	}

	// Create storage client
	client, err := storage.NewClient(cfg.Storage)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to DB
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	// Create adapter
	adapter := reconcile.NewAdapter()
	ctx := context.Background()

	// Test 1: Load gamedata and check for 02_caterbody
	fmt.Println("=== TEST 1: Gamedata Loading ===")
	gdIndex, err := adapter.LoadGamedataIndex(ctx, client, cfg.Storage.Bucket, "gamedata/FurnitureData.json", []string{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total gamedata items loaded: %d\n", len(gdIndex))

	// Search for 02_caterbody
	found := false
	for key, item := range gdIndex {
		gdItem := item.(reconcile.GDItem)
		if gdItem.ClassName == "02_caterbody" {
			fmt.Printf("FOUND in gamedata: key=%s, id=%d, classname=%s\n", key, gdItem.ID, gdItem.ClassName)
			found = true
		}
	}
	if !found {
		fmt.Println("NOT FOUND in gamedata by classname search")
		// Try by ID
		if item, ok := gdIndex["44449608"]; ok {
			gdItem := item.(reconcile.GDItem)
			fmt.Printf("FOUND by ID 44449608: classname=%s\n", gdItem.ClassName)
		} else {
			fmt.Println("NOT FOUND by ID 44449608 either")
		}
	}

	// Test 2: Load DB and check for 02_caterbody
	fmt.Println("\n=== TEST 2: Database Loading ===")
	dbIndex, err := adapter.LoadDBIndex(ctx, db, "arcturus")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total DB items loaded: %d\n", len(dbIndex))

	// Search for 02_caterbody
	found = false
	for key, item := range dbIndex {
		dbItem := item.(reconcile.DBItem)
		if dbItem.ItemName == "02_caterbody" {
			fmt.Printf("FOUND in DB: key=%s, id=%d, sprite_id=%d, item_name=%s\n",
				key, dbItem.ID, dbItem.SpriteID, dbItem.ItemName)
			found = true
		}
	}
	if !found {
		fmt.Println("NOT FOUND in DB by item_name search")
		// Try by sprite_id key
		if item, ok := dbIndex["44449608"]; ok {
			dbItem := item.(reconcile.DBItem)
			fmt.Printf("FOUND by key 44449608: item_name=%s\n", dbItem.ItemName)
		} else {
			fmt.Println("NOT FOUND by key 44449608 either")
		}
	}

	// Test 3: Check classname mapping
	fmt.Println("\n=== TEST 3: Classname Mapping ===")
	storageSet, err := adapter.LoadStorageSet(ctx, client, cfg.Storage.Bucket, "bundled/furniture", ".nitro")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total storage items loaded: %d\n", len(storageSet))

	// Check if 02_caterbody.nitro was mapped
	if _, ok := storageSet["44449608"]; ok {
		fmt.Println("02_caterbody.nitro WAS successfully mapped to key 44449608")
	} else if _, ok := storageSet["02_caterbody"]; ok {
		fmt.Println("02_caterbody.nitro exists but was NOT mapped (key is classname, not ID)")
	} else {
		fmt.Println("02_caterbody.nitro NOT FOUND in storage at all")
	}

	// Test 4: Direct file check
	fmt.Println("\n=== TEST 4: Direct Storage Check ===")
	testKey, ok := adapter.ExtractStorageKey("bundled/furniture/02_caterbody.nitro", ".nitro")
	fmt.Printf("ExtractStorageKey result: key=%s, ok=%v\n", testKey, ok)

	// Save detailed output
	output := map[string]interface{}{
		"gamedata_count": len(gdIndex),
		"db_count":       len(dbIndex),
		"storage_count":  len(storageSet),
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile("debug_reconcile.json", data, 0644)

	fmt.Println("\nDebug complete. Check debug_reconcile.json for details.")
}
