package furniture

import (
	"fmt"
	"strings"

	"asset-manager/feature/furniture/models"

	"gorm.io/gorm"
)

// GetDBFurnitureItem fetches a single furniture item from the database by ID or ClassName.
func GetDBFurnitureItem(db *gorm.DB, emulator string, identifier string) (*models.DBFurnitureItem, error) {
	var item models.DBFurnitureItem
	found := false

	// Try to look up by ID if identifier is numeric
	var id int
	if _, err := fmt.Sscanf(identifier, "%d", &id); err == nil && id > 0 {
		switch strings.ToLower(emulator) {
		case "arcturus":
			var dbItem models.ArcturusItemsBase
			if err := db.First(&dbItem, id).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		case "comet":
			var dbItem models.CometFurniture
			if err := db.First(&dbItem, id).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		case "plus":
			var dbItem models.PlusFurniture
			if err := db.First(&dbItem, id).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
	}

	// If not found by ID, try ClassName (item_name) OR PublicName (name)
	if !found {
		switch strings.ToLower(emulator) {
		case "arcturus":
			var dbItem models.ArcturusItemsBase
			if err := db.Where("item_name = ? OR public_name = ?", identifier, identifier).First(&dbItem).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		case "comet":
			var dbItem models.CometFurniture
			if err := db.Where("item_name = ? OR public_name = ?", identifier, identifier).First(&dbItem).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		case "plus":
			var dbItem models.PlusFurniture
			if err := db.Where("item_name = ? OR public_name = ?", identifier, identifier).First(&dbItem).Error; err == nil {
				item = dbItem.ToNormalized()
				found = true
			} else if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
	}

	if !found {
		return nil, nil
	}

	return &item, nil
}
