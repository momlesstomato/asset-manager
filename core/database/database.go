package database

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to the MySQL database.
// It returns a *gorm.DB connection or an error if the connection fails.
// This is an optional connection, so callers should handle the error gracefully.
func Connect(cfg Config) (*gorm.DB, error) {
	// Use net/url to encode password if special characters exist, but standard mysql driver DSN format:
	// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	// If special chars in password, might mess up.
	// But go-sql-driver/mysql handles raw strings if passed correctly?
	// No, parsing logic splits by first @ or last @.
	// Documentation says: "Special characters in the password should be URL encoded."

	// Create user:password string with encoding if needed.
	// But wait, url.UserPassword("u", "p").String() returns u:p encoded.

	userInfo := url.UserPassword(cfg.User, cfg.Password).String()

	dsn := fmt.Sprintf("%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		userInfo, cfg.Host, cfg.Port, cfg.Name)

	// Suppress GORM logging for cleaner optional warnings in main logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool settings to avoid typical issues
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
