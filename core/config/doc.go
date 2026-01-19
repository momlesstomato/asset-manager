// Package config provides configuration management for the Asset Manager.
//
// It utilizes Viper for loading configuration from environment variables,
// config files (config.yaml), and command-line flags.
//
// # Configuration Structure
//
// The Config struct is the central repository for all application settings, divided into subsections:
//   - Server: HTTP server settings (port, API key, emulator type)
//   - Database: MySQL connection details
//   - Storage: S3/MinIO credentials and bucket settings
//   - Log: Logging level and format
//
// # Usage
//
//	cfg, err := config.LoadConfig(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(cfg.Server.Port)
package config
