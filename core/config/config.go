package config

import (
	"reflect"
	"strings"

	"asset-manager/core/database"
	"asset-manager/core/logger"
	"asset-manager/core/server"
	"asset-manager/core/storage"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
// It is divided into partial configurations for better modularity.
type Config struct {
	// Server holds configuration for the HTTP server.
	Server server.Config `mapstructure:"server"`
	// Storage holds configuration for the object storage (e.g., S3, Minio).
	Storage storage.Config `mapstructure:"storage"`
	// Log holds configuration for the logger.
	Log logger.Config `mapstructure:"log"`
	// Database holds configuration for the database connection.
	Database database.Config `mapstructure:"database"`
}

// LoadConfig loads configuration from environment variables and .env file.
func LoadConfig(path string) (*Config, error) {
	// 1. Load .env file if it exists
	// We construct the path to .env
	envPath := path + "/.env"
	if path == "." {
		envPath = ".env"
	}

	// Ignore error if file doesn't exist (e.g. production)
	_ = godotenv.Overload(envPath)

	v := viper.New()

	// Recursively parse struct tags to set default values
	bindValues(v, Config{}, "")

	// Map environment variables to nested keys (e.g. SERVER_PORT -> server.port)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// bindValues uses reflection to iterate over the struct and set default values in Viper
// based on the 'default' and 'mapstructure' tags.
func bindValues(v *viper.Viper, iface any, prefix string) {
	t := reflect.TypeOf(iface)

	// If it's a pointer, get the element
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")

		// Skip if no tag
		if tag == "" {
			continue
		}

		// Build the key
		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}

		// If it's a nested struct, recurse
		if field.Type.Kind() == reflect.Struct {
			bindValues(v, reflect.New(field.Type).Elem().Interface(), key)
			continue
		}

		defaultValue := field.Tag.Get("default")
		// Always set default (even if empty) to register the key for AutomaticEnv
		v.SetDefault(key, defaultValue)
	}
}
