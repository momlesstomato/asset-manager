// Package database handles database connections and schema inspection.
//
// It provides a wrapper around GORM (Go Object Relational Mapping) to properly configure
// MySQL connections based on the application's configuration.
//
// # Connect
//
// The generic Connect function establishes a connection to the database. It is agnostic
// to the specific emulator schema (Arcturus, Comet, Plus) regarding connection establishment,
// but the Schema Inspector relies on knowing the expected schema.
//
// # Schema Inspection
//
// The package includes tools to inspect the database schema, which is crucial for
// the Server Integrity Check. It allows retrieving table columns and verifying matches
// against expected models defined in feature packages.
//
// # Usage
//
//	db, err := database.Connect(cfg.Database)
//	if err != nil {
//	    log.Fatal("Database connection failed", err)
//	}
//
//	columns, err := database.GetTableColumns(db, "items_base")
package database
