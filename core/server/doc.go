// Package server holds the HTTP server configuration and constants.
//
// While the main application entry point handles the server startup, this package
// defines the configuration structures and valid values for server settings,
// such as supported emulator types.
//
// # Configuration
//
// The Config struct defines the HTTP port, API key, and the target emulator
// (Arcturus, Plus, Comet).
//
// # Usage
//
// This package is primarily used by the core/config package to embed server settings
// and by feature adapters to validate emulator types.
package server
