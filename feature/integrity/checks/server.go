package checks

import (
	"fmt"
	"reflect"
	"strings"

	"asset-manager/core/database"
	"asset-manager/feature/emulator/models"

	"gorm.io/gorm"
)

// ServerReport strictly types the result of a server integrity check.
type ServerReport struct {
	Emulator string                 `json:"emulator"`
	Matched  bool                   `json:"matched"`
	Tables   map[string]TableReport `json:"tables"`
	Errors   []string               `json:"errors"`
}

type TableReport struct {
	MissingColumns []string `json:"missing_columns"`
	TypeMismatches []string `json:"type_mismatches"`
	Status         string   `json:"status"` // "ok", "error"
}

// CheckServerIntegrity verifies the database schema using GORM models as the source of truth.
func CheckServerIntegrity(db *gorm.DB, emulator string) (*ServerReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// 1. Resolve Model based on Emulator
	var model interface{}
	switch emulator {
	case "arcturus":
		model = models.ArcturusItemsBase{}
	case "plus":
		model = models.PlusFurniture{}
	case "comet":
		model = models.CometFurniture{}
	default:
		return nil, fmt.Errorf("unknown emulator model: %s", emulator)
	}

	report := &ServerReport{
		Emulator: emulator,
		Tables:   make(map[string]TableReport),
		Matched:  true,
	}

	// 2. Reflect on the model to build Expected Schema
	val := reflect.TypeOf(model)
	if val.Kind() == reflect.Struct {
		// A model might map to one table usually.
		// Get table name from TableName() method if exists, or snake_case struct name.
		// For simplicity, let's instantiate and check TableName interface.
		tableName := ""
		if tabler, ok := reflect.New(val).Interface().(interface{ TableName() string }); ok {
			tableName = tabler.TableName()
		} else {
			// Fallback or error? All our models implement TableName.
			return nil, fmt.Errorf("model for %s does not implement TableName", emulator)
		}

		tblReport := TableReport{
			MissingColumns: []string{},
			TypeMismatches: []string{},
			Status:         "ok",
		}

		// Get Actual DB Columns
		actualCols, err := database.GetTableColumns(db, tableName)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to inspect table %s: %v", tableName, err))
			report.Matched = false
			return report, nil // Partial fail
		}

		actualMap := make(map[string]database.ColumnInfo)
		for _, col := range actualCols {
			actualMap[col.Field] = col
		}

		// Loop through Struct Fields to check against DB
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			gormTag := field.Tag.Get("gorm")

			// Parse column name from tag
			colName := parseGormColumn(gormTag)
			if colName == "" {
				continue // access default or skip
			}
			expType := parseGormType(gormTag)

			// Check existence
			actCol, exists := actualMap[colName]
			if !exists {
				tblReport.MissingColumns = append(tblReport.MissingColumns, colName)
				tblReport.Status = "error"
				report.Matched = false
				continue
			}

			// Check Type (if defined in GORM tag)
			if expType != "" {
				// Normalize expected type
				expType = strings.ToLower(expType)
				// Relaxed Enum Check: If both are enums, consider it a match
				// This avoids issues with value ordering or "0"-"4" vs "0","1" validation
				if strings.HasPrefix(expType, "enum") && strings.HasPrefix(actCol.Type, "enum") {
					continue
				}

				// Soft check
				if !strings.Contains(actCol.Type, expType) {
					// Check if enum?
					// If GORM tag says "primaryKey" or similar without type, we skip type check.
					// Only check if "type:..." is present.
					mismatch := fmt.Sprintf("%s: expected %s, got %s", colName, expType, actCol.Type)
					tblReport.TypeMismatches = append(tblReport.TypeMismatches, mismatch)
					tblReport.Status = "error"
					report.Matched = false
				}
			}
		}

		report.Tables[tableName] = tblReport
	}

	return report, nil
}

// Helpers to parse simple GORM tags
func parseGormColumn(tag string) string {
	parts := strings.Split(tag, ";")
	for _, p := range parts {
		if strings.HasPrefix(p, "column:") {
			return strings.TrimPrefix(p, "column:")
		}
	}
	return ""
}

func parseGormType(tag string) string {
	parts := strings.Split(tag, ";")
	for _, p := range parts {
		if strings.HasPrefix(p, "type:") {
			return strings.TrimPrefix(p, "type:")
		}
	}
	return ""
}
