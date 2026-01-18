package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ColumnInfo matches the output of SHWO COLUMNS
type ColumnInfo struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default *string // Pointer because NULL default is possible
	Extra   string
}

// GetTableColumns retrieves the column definitions for a given table.
func GetTableColumns(db *gorm.DB, tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo
	// Check dialect
	if db.Dialector.Name() == "sqlite" {
		// SQLite uses PRAGMA table_info
		type SQLiteColumn struct {
			Cid        int
			Name       string
			Type       string
			Notnull    int
			DefaultVal *string
			Pk         int
		}
		var sqliteCols []SQLiteColumn
		if err := db.Raw(fmt.Sprintf("PRAGMA table_info('%s')", tableName)).Scan(&sqliteCols).Error; err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		for _, col := range sqliteCols {
			columns = append(columns, ColumnInfo{
				Field: strings.ToLower(col.Name),
				Type:  strings.ToLower(col.Type),
				// Mapping other fields if needed, but for Integrity check Field and Type are most important
			})
		}
		return columns, nil
	}

	// Use Raw SQL for MySQL "SHOW COLUMNS"
	// Note: GORM's Migrator().ColumnTypes() is an abstraction, but raw might be easier for exact type strings.
	// Let's us Raw.
	err := db.Raw(fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName)).Scan(&columns).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}
	// Normalize types to lowercase
	for i := range columns {
		columns[i].Type = strings.ToLower(columns[i].Type)
		columns[i].Field = strings.ToLower(columns[i].Field)
	}
	return columns, nil
}
