package reconcile

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPrepare_SchemaExpansion(t *testing.T) {
	// Setup sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	// Use MySQL dialector to support MySQL-specific syntax mocking if needed by GORM
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	adapter := NewAdapter()
	adapter.serverProfile = "arcturus" // Profile uses 'items_base' table and maps public_name/item_name

	// Expect ALTER TABLE statements
	// Note: GORM might wrap queries or we might execute raw SQL.
	// The implementation performs: db.Exec("ALTER TABLE items_base MODIFY COLUMN ...")

	mock.ExpectExec("ALTER TABLE items_base MODIFY COLUMN item_name VARCHAR\\(120\\)").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("ALTER TABLE items_base MODIFY COLUMN public_name VARCHAR\\(120\\)").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Run Prepare
	err = adapter.Prepare(context.Background(), gormDB)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
