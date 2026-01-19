package integrity

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"asset-manager/core/reconcile"
	"asset-manager/core/storage"
	"asset-manager/feature/furniture/models"
	furnitureAdp "asset-manager/feature/furniture/reconcile"

	"gorm.io/gorm"
)

// CheckIntegrity performs a high-performance integrity check of bundled furniture.
// This function uses the new reconcile engine for better performance and maintainability.
func CheckIntegrity(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string) (*models.Report, error) {
	startTime := time.Now()

	// Check if bucket exists
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s not found", bucket)
	}

	// Create furniture adapter and spec
	adapter := furnitureAdp.NewAdapter()
	spec := &reconcile.Spec{
		Adapter:            adapter,
		CacheTTL:           0, // No caching for full scan
		StoragePrefix:      "bundled/furniture",
		StorageExtension:   ".nitro",
		GamedataPaths:      []string{"roomitemtypes.furnitype", "wallitemtypes.furnitype"},
		GamedataObjectName: "gamedata/FurnitureData.json",
		ServerProfile:      emulator,
	}

	// Run reconciliation
	results, err := reconcile.ReconcileAll(ctx, spec, db, client, bucket)
	if err != nil {
		return nil, fmt.Errorf("reconciliation failed: %w", err)
	}

	// Convert reconcile results to existing Report format
	report := convertToReport(results)
	report.GeneratedAt = time.Now().Format(time.RFC3339)
	report.ExecutionTime = time.Since(startTime).String()

	return report, nil
}

// CheckFurnitureItem performs a detailed integrity check for a single item.
// This function uses the new reconcile engine for targeted reconciliation.
func CheckFurnitureItem(ctx context.Context, client storage.Client, bucket string, db *gorm.DB, emulator string, identifier string) (*models.FurnitureDetailReport, error) {
	// Create adapter and spec
	adapter := furnitureAdp.NewAdapter()
	spec := &reconcile.Spec{
		Adapter:            adapter,
		CacheTTL:           0, // No caching for one-off CLI check
		StoragePrefix:      "bundled/furniture",
		StorageExtension:   ".nitro",
		GamedataPaths:      []string{"roomitemtypes.furnitype", "wallitemtypes.furnitype"},
		GamedataObjectName: "gamedata/FurnitureData.json",
		ServerProfile:      emulator,
	}

	// Clean identifier
	searchIdentifier := identifier
	if strings.HasSuffix(identifier, ".nitro") {
		searchIdentifier = strings.TrimSuffix(identifier, ".nitro")
	}

	// Build query
	query := reconcile.Query{
		ID:        searchIdentifier,
		Name:      searchIdentifier,
		Classname: searchIdentifier,
	}

	// Run targeted reconciliation
	result, err := reconcile.ReconcileOne(ctx, spec, db, client, bucket, query)
	if err != nil {
		return nil, fmt.Errorf("targeted reconciliation failed: %w", err)
	}

	// Convert to detail report
	return convertToDetailReport(result), nil
}

// convertToReport converts reconcile results to the existing Report format.
func convertToReport(results []reconcile.ReconcileResult) *models.Report {
	var missingAssets []string
	var unregisteredAssets []string
	var malformedAssets []string
	var parameterMismatches []string

	totalExpected := 0
	totalFound := 0

	for _, r := range results {
		// Count expected items (in gamedata)
		if r.GamedataPresent {
			totalExpected++
		}

		// Count found items (in storage)
		if r.StoragePresent {
			totalFound++
		}

		// Missing assets: in gamedata but not in storage
		if r.GamedataPresent && !r.StoragePresent {
			// Generate filename from classname if available
			filename := r.Name + ".nitro"
			// Try to extract classname from name if possible
			if r.Name != "" {
				filename = r.Name + ".nitro"
			}
			missingAssets = append(missingAssets, filename)
		}

		// Unregistered assets: in storage but not in gamedata
		if r.StoragePresent && !r.GamedataPresent {
			filename := r.Name + ".nitro"
			if r.Name == "" {
				filename = r.ID + ".nitro"
			}
			unregisteredAssets = append(unregisteredAssets, filename)
		}

		// Parameter mismatches
		for _, mismatch := range r.Mismatch {
			msg := fmt.Sprintf("ID %s: %s", r.ID, mismatch)
			parameterMismatches = append(parameterMismatches, msg)
		}

		// Malformed items would have been filtered during loading
		// For now, we don't have a way to detect malformed items in the new system
		// This could be added to the adapter if needed
	}

	return &models.Report{
		TotalExpected:       totalExpected,
		TotalFound:          totalFound,
		MissingAssets:       missingAssets,
		UnregisteredAssets:  unregisteredAssets,
		MalformedAssets:     malformedAssets,
		ParameterMismatches: parameterMismatches,
	}
}

// convertToDetailReport converts a single reconcile result to a detail report.
func convertToDetailReport(result *reconcile.ReconcileResult) *models.FurnitureDetailReport {
	report := &models.FurnitureDetailReport{
		InFurniData:     result.GamedataPresent,
		InDB:            result.DBPresent,
		FileExists:      result.StoragePresent,
		IntegrityStatus: "PASS",
		Name:            result.Name,
		Mismatches:      make([]string, 0),
	}

	// Try to parse ID as int
	var id int
	if _, err := fmt.Sscanf(result.ID, "%d", &id); err == nil {
		report.ID = id
	}

	// Set classname and nitro file
	if result.Name != "" {
		report.ClassName = result.Name
		report.NitroFile = result.Name + ".nitro"
	}

	// Determine status
	if !result.GamedataPresent {
		report.Mismatches = append(report.Mismatches, "Missing in FurniData")
		report.IntegrityStatus = "FAIL"
	}
	if !result.DBPresent && result.GamedataPresent {
		report.Mismatches = append(report.Mismatches, "Missing in Database")
		report.IntegrityStatus = "FAIL"
	}
	if !result.StoragePresent {
		report.Mismatches = append(report.Mismatches, "Missing .nitro file in storage")
		report.IntegrityStatus = "FAIL"
	}

	// Add field mismatches
	report.Mismatches = append(report.Mismatches, result.Mismatch...)

	if len(report.Mismatches) > 0 && report.IntegrityStatus == "PASS" {
		report.IntegrityStatus = "WARNING"
	}

	return report
}

// CheckIntegrityWithDB extends the integrity check to include database verification.
// This is kept for backward compatibility but now delegates to the reconcile engine.
func CheckIntegrityWithDB(ctx context.Context, furniData *models.FurnitureData, db *gorm.DB, emulator string) ([]string, error) {
	// Convert FurnitureData to reconcile GDItems
	adapter := furnitureAdp.NewAdapter()

	// Load DB index
	profile := emulator
	if profile == "" {
		profile = "arcturus"
	}
	dbIndex, err := adapter.LoadDBIndex(ctx, db, profile)
	if err != nil {
		return nil, err
	}

	// Build GD index from provided furniData
	gdIndex := make(map[string]reconcile.GDItem)
	for _, item := range furniData.RoomItemTypes.FurniType {
		key := strconv.Itoa(item.ID)
		gdIndex[key] = furnitureAdp.GDItem{
			ID:         item.ID,
			ClassName:  item.ClassName,
			Name:       item.Name,
			XDim:       item.XDim,
			YDim:       item.YDim,
			CanSitOn:   item.CanSitOn,
			CanStandOn: item.CanStandOn,
			CanLayOn:   item.CanLayOn,
		}
	}
	for _, item := range furniData.WallItemTypes.FurniType {
		key := strconv.Itoa(item.ID)
		gdIndex[key] = furnitureAdp.GDItem{
			ID:         item.ID,
			ClassName:  item.ClassName,
			Name:       item.Name,
			XDim:       item.XDim,
			YDim:       item.YDim,
			CanSitOn:   item.CanSitOn,
			CanStandOn: item.CanStandOn,
			CanLayOn:   item.CanLayOn,
		}
	}

	var mismatches []string

	// Compare items in both indices
	for key, gdItem := range gdIndex {
		dbItem, exists := dbIndex[key]
		if !exists {
			mismatches = append(mismatches, fmt.Sprintf("ID %s: missing in database", key))
			continue
		}

		// Compare fields using adapter
		itemMismatches := adapter.CompareFields(dbItem, gdItem)
		for _, mismatch := range itemMismatches {
			mismatches = append(mismatches, fmt.Sprintf("ID %s: %s", key, mismatch))
		}
	}

	return mismatches, nil
}

// GetDBFurnitureItem fetches a single furniture item from the database by ID or ClassName.
// This is kept for backward compatibility.
func GetDBFurnitureItem(db *gorm.DB, emulator string, identifier string) (*models.DBFurnitureItem, error) {
	adapter := furnitureAdp.NewAdapter()

	query := reconcile.Query{
		ID:        identifier,
		Name:      identifier,
		Classname: identifier,
	}

	dbItem, err := adapter.QueryDB(context.Background(), db, emulator, query)
	if err != nil {
		return nil, err
	}

	if dbItem == nil {
		return nil, gorm.ErrRecordNotFound
	}

	// Convert to models.DBFurnitureItem
	item := dbItem.(furnitureAdp.DBItem)
	return &models.DBFurnitureItem{
		ID:         item.ID,
		ItemName:   item.ItemName,
		PublicName: item.PublicName,
		Width:      item.Width,
		Length:     item.Length,
		CanSit:     item.CanSit,
		CanWalk:    item.CanWalk,
		CanLay:     item.CanLay,
	}, nil
}
