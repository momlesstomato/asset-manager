package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupMockDB creates a mock GORM DB for testing.
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

func TestConnect(t *testing.T) {
	t.Run("Invalid Connection", func(t *testing.T) {
		cfg := Config{
			Host:     "localhost",
			Port:     9999, // Unused port
			User:     "root",
			Password: "wrongpassword",
			Name:     "emulator",
		}

		// Connect should fail (timeout or refused)
		// We expect an error.
		db, err := Connect(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestGetTableColumns(t *testing.T) {
	db, sqlMock := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"Field", "Type", "Null", "Key", "Default", "Extra"})
		rows.AddRow("id", "int(11)", "NO", "PRI", nil, "auto_increment")
		rows.AddRow("name", "varchar(255)", "YES", "", nil, "")

		sqlMock.ExpectQuery("SHOW COLUMNS FROM `items`").WillReturnRows(rows)

		columns, err := GetTableColumns(db, "items")
		assert.NoError(t, err)
		assert.Len(t, columns, 2)
		assert.Equal(t, "id", columns[0].Field)
		assert.Equal(t, "int(11)", columns[0].Type)
		assert.Equal(t, "name", columns[1].Field)
		assert.Equal(t, "varchar(255)", columns[1].Type)
	})

	t.Run("Failure", func(t *testing.T) {
		sqlMock.ExpectQuery("SHOW COLUMNS FROM `items`").WillReturnError(assert.AnError)

		columns, err := GetTableColumns(db, "items")
		assert.Error(t, err)
		assert.Nil(t, columns)
	})
}
