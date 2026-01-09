package furniture

import (
	"context"
	"fmt"
	"io"
	"strings"

	"sync"
	"time"

	"github.com/goccy/go-json"

	"asset-manager/core/storage"
	"asset-manager/feature/furniture/models"

	"github.com/minio/minio-go/v7"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// CheckIntegrity performs a high-performance integrity check of bundled furniture.
func CheckIntegrity(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string) (*models.Report, error) {
	startTime := time.Now()

	furniData, err := loadFurnitureData(ctx, client, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to load FurnitureData.json: %w", err)
	}

	expectedFiles, malformedAssets := getExpectedFilesAndValidate(furniData)

	actualFiles, err := getActualFiles(ctx, client, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to list bundled furniture: %w", err)
	}

	report := compareFurniture(expectedFiles, actualFiles)
	report.MalformedAssets = malformedAssets

	if db != nil && emulator != "" {
		mismatches, err := CheckIntegrityWithDB(ctx, furniData, db, emulator)
		// If DB check fails (e.g. connection error), should we fail whole check or just report error?
		// Let's report it or fail. failing seems safer to alert user.
		if err != nil {
			return nil, fmt.Errorf("database integrity check failed: %w", err)
		}
		report.ParameterMismatches = mismatches
	}

	report.GeneratedAt = time.Now().Format(time.RFC3339)
	report.ExecutionTime = time.Since(startTime).String()

	return report, nil
}

func loadFurnitureData(ctx context.Context, client storage.Client, bucket string) (*models.FurnitureData, error) {
	objName := "gamedata/FurnitureData.json"

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s not found", bucket)
	}

	reader, err := client.GetObject(ctx, bucket, objName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read FurnitureData.json: %w", err)
	}

	var fd models.FurnitureData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, fmt.Errorf("failed to parse FurnitureData.json: %w", err)
	}

	return &fd, nil
}

func getExpectedFilesAndValidate(fd *models.FurnitureData) (map[string]bool, []string) {
	expected := make(map[string]bool)
	var malformed []string

	processItems := func(items []models.FurnitureItem) {
		for _, item := range items {
			if msg := item.Validate(); msg != "" {
				// Identify item by ID if possible, else index?
				// Using ID if > 0, else just "Unknown item"
				idStr := fmt.Sprintf("%d", item.ID)
				if item.ID == 0 {
					idStr = "?"
				}
				malformed = append(malformed, fmt.Sprintf("ID %s: %s", idStr, msg))
				continue
			}

			name := item.ClassName
			if idx := strings.Index(name, "*"); idx != -1 {
				name = name[:idx]
			}
			expected[name+".nitro"] = true
		}
	}

	processItems(fd.RoomItemTypes.FurniType)
	processItems(fd.WallItemTypes.FurniType)

	return expected, malformed
}

func getActualFiles(ctx context.Context, client storage.Client, bucket string) (map[string]bool, error) {
	actual := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Prefixes to scan: a-z, A-Z, 0-9, _, and -
	prefixes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

	errCh := make(chan error, 1)

	for _, char := range prefixes {
		prefix := fmt.Sprintf("bundled/furniture/%c", char)
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			// Check if context canceled or error occurred
			select {
			case <-ctx.Done():
				return
			case <-errCh:
				return
			default:
			}

			opts := minio.ListObjectsOptions{
				Prefix:    p,
				Recursive: true,
			}

			for obj := range client.ListObjects(ctx, bucket, opts) {
				if obj.Err != nil {
					select {
					case errCh <- obj.Err:
					default:
					}
					return
				}

				filename := strings.TrimPrefix(obj.Key, "bundled/furniture/")
				if filename == "" || strings.HasSuffix(filename, "/") {
					continue
				}

				mu.Lock()
				actual[filename] = true
				mu.Unlock()
			}
		}(prefix)
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	return actual, nil
}

func compareFurniture(expected map[string]bool, actual map[string]bool) *models.Report {
	report := &models.Report{
		TotalExpected:      len(expected),
		TotalFound:         len(actual),
		MissingAssets:      make([]string, 0),
		UnregisteredAssets: make([]string, 0),
		MalformedAssets:    make([]string, 0),
	}

	// Check missing (In Furnidata but not in Storage)
	for file := range expected {
		if !actual[file] {
			report.MissingAssets = append(report.MissingAssets, file)
		}
	}

	// Check extra (In Storage but not in Furnidata)
	for file := range actual {
		if !expected[file] {
			report.UnregisteredAssets = append(report.UnregisteredAssets, file)
		}
	}

	return report
}

// CheckFurnitureItem performs a detailed integrity check for a single item.
func CheckFurnitureItem(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string, identifier string) (*models.FurnitureDetailReport, error) {
	report := &models.FurnitureDetailReport{
		IntegrityStatus: "PASS",
	}

	// Clean identifier for DB/Storage search (remove .nitro suffix if present)
	searchIdentifier := identifier
	if strings.HasSuffix(identifier, ".nitro") {
		searchIdentifier = strings.TrimSuffix(identifier, ".nitro")
	}

	var (
		furniData *models.FurnitureData
		dbItem    *models.DBFurnitureItem
	)

	// 1. Parallel Fetch: Load FurniData and Check DB
	g, ctxGroup := errgroup.WithContext(ctx)

	// Fetch FurniData
	g.Go(func() error {
		var err error
		furniData, err = loadFurnitureData(ctxGroup, client, bucket)
		if err != nil {
			return fmt.Errorf("failed to load FurnitureData: %w", err)
		}
		return nil
	})

	// Fetch DB Item (if db is present)
	if db != nil && emulator != "" {
		g.Go(func() error {
			var err error
			dbItem, err = GetDBFurnitureItem(db, emulator, searchIdentifier)
			if err != nil && err != gorm.ErrRecordNotFound {
				return fmt.Errorf("db lookup failed: %w", err)
			}
			// Ignore RecordNotFound here, treat as dbItem == nil
			return nil
		})
	}

	// Wait for fetches
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 2. Find in FurniData
	var item *models.FurnitureItem
	// Check by ID (if numeric)
	var id int
	isNumericId := false
	if _, err := fmt.Sscanf(identifier, "%d", &id); err == nil && id > 0 {
		isNumericId = true
		report.ID = id
	}

	findInList := func(list []models.FurnitureItem) *models.FurnitureItem {
		for _, idx := range list {
			if isNumericId && idx.ID == id {
				return &idx
			}
			// Use searchIdentifier (cleaned) for comparison
			if strings.EqualFold(idx.ClassName, searchIdentifier) {
				return &idx
			}
			// Check if identifier matches name? Use cleaned just in case
			if strings.EqualFold(idx.Name, searchIdentifier) {
				return &idx
			}
		}
		return nil
	}

	if found := findInList(furniData.RoomItemTypes.FurniType); found != nil {
		item = found
	} else if found := findInList(furniData.WallItemTypes.FurniType); found != nil {
		item = found
	}

	if item != nil {
		report.InFurniData = true
		report.ID = item.ID
		report.ClassName = item.ClassName
		report.Name = item.Name

		// Clean classname for file check
		cleanName := item.ClassName
		if idx := strings.Index(cleanName, "*"); idx != -1 {
			cleanName = cleanName[:idx]
		}
		report.NitroFile = cleanName + ".nitro"
	} else {
		report.InFurniData = false
		// If not in furnidata, we might still find it in DB or Storage if identifier was file/classname
		// But if identifier was ID, we set it.
		if isNumericId {
			report.ID = id
		} else {
			report.ClassName = searchIdentifier
			report.NitroFile = searchIdentifier + ".nitro"
		}
	}

	// 3. Process DB Result (Already fetched)
	if db != nil && emulator != "" {
		if dbItem != nil {
			report.InDB = true
			if report.ID == 0 {
				report.ID = dbItem.ID
			}
			if report.Name == "" {
				report.Name = dbItem.PublicName
			}
			if report.ClassName == "" {
				report.ClassName = dbItem.ItemName
				cleanName := dbItem.ItemName
				if idx := strings.Index(cleanName, "*"); idx != -1 {
					cleanName = cleanName[:idx]
				}
				report.NitroFile = cleanName + ".nitro"
			}

			// Compare if we have both
			if item != nil {
				// We reuse the logic broadly, but focused.
				if item.Name != dbItem.PublicName {
					report.Mismatches = append(report.Mismatches, fmt.Sprintf("Name mismatch: FurniData='%s', DB='%s'", item.Name, dbItem.PublicName))
				}
				if item.ClassName != dbItem.ItemName {
					report.Mismatches = append(report.Mismatches, fmt.Sprintf("ClassName mismatch: FurniData='%s', DB='%s'", item.ClassName, dbItem.ItemName))
				}
				if item.XDim != dbItem.Width {
					report.Mismatches = append(report.Mismatches, fmt.Sprintf("Width mismatch: FurniData=%d, DB=%d", item.XDim, dbItem.Width))
				}
				if item.YDim != dbItem.Length {
					report.Mismatches = append(report.Mismatches, fmt.Sprintf("Length mismatch: FurniData=%d, DB=%d", item.YDim, dbItem.Length))
				}
				// Can add more detailed checks here if needed
			}
		} else {
			report.InDB = false
		}
	}

	// 4. Check Storage
	if report.NitroFile != "" {
		filename := report.NitroFile
		pathsToCheck := []string{
			fmt.Sprintf("bundled/furniture/%s", filename), // Check flat path first
		}

		if len(filename) > 0 {
			firstChar := string(filename[0])
			pathsToCheck = append(pathsToCheck,
				fmt.Sprintf("bundled/furniture/%s/%s", strings.ToLower(firstChar), filename),
				fmt.Sprintf("bundled/furniture/%s/%s", strings.ToUpper(firstChar), filename),
			)
		}

		foundFile := false
		for _, path := range pathsToCheck {
			// Check existence using ListObjects (StatObject not in interface)
			opts := minio.ListObjectsOptions{
				Prefix:    path,
				Recursive: false,
				MaxKeys:   1,
			}
			for obj := range client.ListObjects(ctx, bucket, opts) {
				if obj.Err == nil && obj.Key == path {
					foundFile = true
					break
				}
			}
			if foundFile {
				break
			}
		}
		report.FileExists = foundFile
	}

	// Calculate Status
	if !report.InFurniData {
		report.Mismatches = append(report.Mismatches, "Missing in FurniData")
		report.IntegrityStatus = "FAIL"
	}
	if !report.InDB && db != nil {
		report.Mismatches = append(report.Mismatches, "Missing in Database")
		report.IntegrityStatus = "FAIL"
	}
	if !report.FileExists {
		report.Mismatches = append(report.Mismatches, "Missing .nitro file in storage")
		report.IntegrityStatus = "FAIL"
	}
	if len(report.Mismatches) > 0 && report.IntegrityStatus == "PASS" {
		report.IntegrityStatus = "WARNING"
	}

	return report, nil
}
