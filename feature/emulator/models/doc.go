// Package emulator contains data models for supported Habbo emulators.
//
// It defines the database schemas for Arcturus, Comet, and Plus emulators as GORM models.
// These models are legally mapped to the database tables (e.g., 'items_base', 'furniture')
// and are used by the reconciliation system to verify database integrity.
//
// # Supported Emulators
//
//   - Arcturus: Uses 'items_base' table.
//   - Comet: Uses 'furniture' table.
//   - Plus: Uses 'furniture' table.
//
// # Usage
//
// The reconcile adapters use these models to type-check and reflect against the
// actual database server during integrity checks.
package models // Using package name 'models' as shown in file structure, though dir is 'emulator/models'. The parent 'emulator' dir seems to just hold this subpackage.
