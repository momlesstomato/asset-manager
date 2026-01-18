package furniture

import (
	"context"
	"testing"

	"asset-manager/feature/furniture/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open mock sql db: %v", err)
	}

	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open gorm db: %v", err)
	}

	return gormDB, mock
}

func TestCheckIntegrityWithDB_Arcturus(t *testing.T) {
	db, mock := setupMockDB(t)

	// Mock FurniData
	furniData := &models.FurnitureData{
		RoomItemTypes: struct {
			FurniType []models.FurnitureItem `json:"furnitype"`
		}{
			FurniType: []models.FurnitureItem{
				{ID: 1, Name: "Chair", ClassName: "chair", XDim: 1, YDim: 1, CanSitOn: true},
				{ID: 2, Name: "Table", ClassName: "table", XDim: 2, YDim: 2, CanStandOn: true},
			},
		},
	}

	// Mock DB Rows for Arcturus
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "allow_stack", "allow_sit", "allow_lay", "allow_walk", "type", "interaction_type"})
	rows.AddRow(1, 1, "chair", "Chair", 1, 1, 1.0, 0, 1, 0, 0, "s", "default")
	rows.AddRow(2, 2, "table", "Table", 2, 2, 1.0, 1, 0, 0, 1, "s", "default")

	mock.ExpectQuery("SELECT \\* FROM `items_base`").WillReturnRows(rows)

	mismatches, err := CheckIntegrityWithDB(context.Background(), furniData, db, "arcturus")
	assert.NoError(t, err)
	assert.Empty(t, mismatches)
}

func TestCheckIntegrityWithDB_Mismatch(t *testing.T) {
	db, mock := setupMockDB(t)

	// Mock FurniData
	furniData := &models.FurnitureData{
		RoomItemTypes: struct {
			FurniType []models.FurnitureItem `json:"furnitype"`
		}{
			FurniType: []models.FurnitureItem{
				{ID: 1, Name: "Chair", ClassName: "chair", XDim: 1, YDim: 1},
			},
		},
	}

	// Mock DB Rows with mismatched data (Width 2 instead of 1, Name "Big Chair")
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "allow_stack", "allow_sit", "allow_lay", "allow_walk", "type", "interaction_type"})
	rows.AddRow(1, 1, "chair", "Big Chair", 2, 1, 1.0, 0, 1, 0, 0, "s", "default")

	mock.ExpectQuery("SELECT \\* FROM `items_base`").WillReturnRows(rows)

	mismatches, err := CheckIntegrityWithDB(context.Background(), furniData, db, "arcturus")
	assert.NoError(t, err)
	assert.NotEmpty(t, mismatches)
	assert.Contains(t, mismatches[0], "name mismatch")
	assert.Contains(t, mismatches[1], "width mismatch")
}

func TestCheckIntegrityWithDB_Comet(t *testing.T) {
	db, mock := setupMockDB(t)

	furniData := &models.FurnitureData{
		RoomItemTypes: struct {
			FurniType []models.FurnitureItem `json:"furnitype"`
		}{
			FurniType: []models.FurnitureItem{
				{ID: 10, Name: "Lamp", ClassName: "lamp", XDim: 1, YDim: 1},
			},
		},
	}

	// Mock DB Rows for Comet (Note string fields for bools/ints in Comet struct)
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "can_stack", "can_sit", "can_lay", "is_walkable", "type", "interaction_type"})
	rows.AddRow(10, 10, "lamp", "Lamp", 1, 1, "1.0", "1", "0", "0", "0", "s", "default")

	mock.ExpectQuery("SELECT \\* FROM `furniture`").WillReturnRows(rows)

	mismatches, err := CheckIntegrityWithDB(context.Background(), furniData, db, "comet")
	assert.NoError(t, err)
	assert.Empty(t, mismatches)
}

func TestCheckIntegrityWithDB_Plus(t *testing.T) {
	db, mock := setupMockDB(t)

	furniData := &models.FurnitureData{
		RoomItemTypes: struct {
			FurniType []models.FurnitureItem `json:"furnitype"`
		}{
			FurniType: []models.FurnitureItem{
				{ID: 20, Name: "Rug", ClassName: "rug", XDim: 2, YDim: 2, CanStandOn: true},
			},
		},
	}

	// Mock DB Rows for Plus
	rows := sqlmock.NewRows([]string{"id", "sprite_id", "item_name", "public_name", "width", "length", "stack_height", "can_stack", "can_sit", "is_walkable", "type", "interaction_type", "is_rare"})
	rows.AddRow(20, 20, "rug", "Rug", 2, 2, 0.01, 1, 0, 1, "s", "default", 0)

	mock.ExpectQuery("SELECT \\* FROM `furniture`").WillReturnRows(rows)

	mismatches, err := CheckIntegrityWithDB(context.Background(), furniData, db, "plus")
	assert.NoError(t, err)
	assert.Empty(t, mismatches)
}
