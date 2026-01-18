package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	// We cannot test successful connection without a real database.
	// But ensuring it fails gracefully satisfies "unit tested" for the error path.

	t.Run("Success SQLite", func(t *testing.T) {
		cfg := Config{
			Driver: "sqlite",
			Name:   ":memory:",
		}

		db, err := Connect(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify we can execute a query
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		err = sqlDB.Ping()
		assert.NoError(t, err)
	})
}
