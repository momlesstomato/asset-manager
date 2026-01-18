package furniture

import (
	"context"
	"fmt"
	"strings"

	"asset-manager/feature/furniture/models"

	"gorm.io/gorm"
)

// CheckIntegrityWithDB extends the integrity check to include database verification.
func CheckIntegrityWithDB(ctx context.Context, furniData *models.FurnitureData, db *gorm.DB, emulator string) ([]string, error) {
	dbAssets, err := getDBAssets(db, emulator)
	if err != nil {
		return nil, err
	}

	return compareFurnitureWithDB(furniData, dbAssets), nil
}

func getDBAssets(db *gorm.DB, emulator string) (map[int]models.DBFurnitureItem, error) {
	assets := make(map[int]models.DBFurnitureItem)

	switch strings.ToLower(emulator) {
	case "arcturus":
		var items []models.ArcturusItemsBase
		if err := db.Find(&items).Error; err != nil {
			return nil, err
		}
		for _, item := range items {
			assets[item.ID] = item.ToNormalized()
		}
	case "comet":
		var items []models.CometFurniture
		if err := db.Find(&items).Error; err != nil {
			return nil, err
		}
		for _, item := range items {
			assets[item.ID] = item.ToNormalized()
		}
	case "plus":
		var items []models.PlusFurniture
		if err := db.Find(&items).Error; err != nil {
			return nil, err
		}
		for _, item := range items {
			assets[item.ID] = item.ToNormalized()
		}
	default:
		return nil, fmt.Errorf("unsupported emulator: %s", emulator)
	}

	return assets, nil
}

func compareFurnitureWithDB(fd *models.FurnitureData, dbAssets map[int]models.DBFurnitureItem) []string {
	var mismatches []string

	processItems := func(items []models.FurnitureItem) {
		for _, item := range items {
			dbItem, exists := dbAssets[item.ID]
			if !exists {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: missing in database", item.ID))
				continue
			}

			// Compare fields
			if item.Name != dbItem.PublicName {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: name mismatch (json: '%s', db: '%s')", item.ID, item.Name, dbItem.PublicName))
			}
			if item.ClassName != dbItem.ItemName {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: classname mismatch (json: '%s', db: '%s')", item.ID, item.ClassName, dbItem.ItemName))
			}
			if item.XDim != dbItem.Width {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: width mismatch (json: %d, db: %d)", item.ID, item.XDim, dbItem.Width))
			}
			if item.YDim != dbItem.Length {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: length mismatch (json: %d, db: %d)", item.ID, item.YDim, dbItem.Length))
			}

			// Boolean flags
			// Note: Some emulators might interpret these differently, checking strict equality based on docs.
			if item.CanSitOn != dbItem.CanSit {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: can_sit mismatch (json: %v, db: %v)", item.ID, item.CanSitOn, dbItem.CanSit))
			}
			if item.CanStandOn != dbItem.CanWalk {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: can_walk/stand mismatch (json: %v, db: %v)", item.ID, item.CanStandOn, dbItem.CanWalk))
			}
			if item.CanLayOn != dbItem.CanLay {
				mismatches = append(mismatches, fmt.Sprintf("ID %d: can_lay mismatch (json: %v, db: %v)", item.ID, item.CanLayOn, dbItem.CanLay))
			}
		}
	}

	processItems(fd.RoomItemTypes.FurniType)
	processItems(fd.WallItemTypes.FurniType)

	return mismatches
}
