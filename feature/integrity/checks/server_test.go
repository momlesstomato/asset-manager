package checks

import (
	"regexp"
	"strings"
	"testing"

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

func TestCheckServerIntegrity_UnknownEmulator(t *testing.T) {
	db, _ := setupMockDB(t)
	report, err := CheckServerIntegrity(db, "unknown")
	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Equal(t, "unknown emulator model: unknown", err.Error())
}

func TestCheckServerIntegrity_NilDB(t *testing.T) {
	report, err := CheckServerIntegrity(nil, "arcturus")
	assert.Error(t, err)
	assert.Nil(t, report)
}

func TestCheckServerIntegrity_Arcturus_Success(t *testing.T) {
	db, mock := setupMockDB(t)

	// Mock SHOW COLUMNS for items_base
	// Arcturus has lots of columns, we simulate a matching set.
	// For simplicity in test, we'll verify it matches a subset we return from mock.
	// Actually, the check loops over logic model fields and expects them in DB.
	// So we must provide ALL columns defined in models.ArcturusItemsBase to pass "ok".

	rows := sqlmock.NewRows([]string{"Field", "Type", "Null", "Key", "Default", "Extra"})
	// Add a few critical ones to match model expectations
	rows.AddRow("id", "int(11)", "NO", "PRI", nil, "auto_increment")
	rows.AddRow("sprite_id", "int(11)", "YES", "", "0", "")
	rows.AddRow("item_name", "varchar(70)", "YES", "", "0", "")
	// ... Ideally we add all. To avoid huge test boilerplate, we can test "Missing" scenario mostly.
	// Or we create a helper to add all from a list.
	// Let's test a case where "public_name" is missing to verify failure logic.

	mock.ExpectQuery("SHOW COLUMNS FROM `items_base`").WillReturnRows(rows)

	report, err := CheckServerIntegrity(db, "arcturus")
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.False(t, report.Matched) // Should fail because we only matched 3 columns out of many

	// Assert specific missing
	tbl, ok := report.Tables["items_base"]
	assert.True(t, ok)
	assert.Equal(t, "error", tbl.Status)
	assert.Contains(t, tbl.MissingColumns, "public_name")
}

func TestCheckServerIntegrity_TypeMismatch(t *testing.T) {
	db, mock := setupMockDB(t)

	// ItemsBase
	rows := sqlmock.NewRows([]string{"Field", "Type", "Null", "Key", "Default", "Extra"})
	rows.AddRow("id", "int(11)", "NO", "PRI", nil, "")
	rows.AddRow("item_name", "int(11)", "YES", "", "0", "") // Expect varchar, give int

	mock.ExpectQuery("SHOW COLUMNS FROM `items_base`").WillReturnRows(rows)

	report2, err := CheckServerIntegrity(db, "arcturus")
	assert.NoError(t, err)

	// Verify item_name mismatch
	tbl2 := report2.Tables["items_base"]

	// Add debugging
	if len(tbl2.TypeMismatches) == 0 {
		t.Logf("No mismatches found. Actual Columns reported by inspector might be missing?")
	}

	foundMismatch := false
	for _, m := range tbl2.TypeMismatches {
		// "item_name: expected varchar(70), got int(11)"
		if regexp.MustCompile(`item_name: expected varchar\(70\), got int\(11\)`).MatchString(m) {
			foundMismatch = true
		}
	}
	assert.True(t, foundMismatch, "Should detect type mismatch for item_name. Got: %v", tbl2.TypeMismatches)
}

func TestParseGormTags(t *testing.T) {
	col := parseGormColumn("column:id;primaryKey")
	assert.Equal(t, "id", col)

	col2 := parseGormColumn("primaryKey;column:item_name;type:varchar(100)")
	assert.Equal(t, "item_name", col2)

	typ := parseGormType("column:id;type:int(11)")
	assert.Equal(t, "int(11)", typ)

	typ2 := parseGormType("column:id")
	assert.Equal(t, "", typ2)
}

func TestCheckServerIntegrity_Arcturus_RareEnum(t *testing.T) {
	db, mock := setupMockDB(t)

	// Simulate DB having restricted enum
	rows := sqlmock.NewRows([]string{"Field", "Type", "Null", "Key", "Default", "Extra"})
	rows.AddRow("rare", "enum('0','1')", "NO", "", "0", "")

	mock.ExpectQuery("SHOW COLUMNS FROM `items_base`").WillReturnRows(rows)

	report, err := CheckServerIntegrity(db, "arcturus")
	assert.NoError(t, err)

	tbl := report.Tables["items_base"]

	// Should PASS because we relaxed enum checking.
	// DB has enum('0','1'), model has enum('0'-'4').
	// The new logic skips strict value checking if both are enums.

	foundMismatch := false
	for _, m := range tbl.TypeMismatches {
		if strings.Contains(m, "rare") {
			foundMismatch = true
		}
	}
	assert.False(t, foundMismatch, "Should NOT detect mismatch for slightly different rare enum. Got: %v", tbl.TypeMismatches)
}
