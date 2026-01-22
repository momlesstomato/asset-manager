package storage

// Config holds configuration for the storage provider.
type Config struct {
	// Endpoint is the URL of the storage service.
	Endpoint string `mapstructure:"endpoint" default:"localhost:9000"`
	// AccessKey is the access key ID for authentication.
	AccessKey string `mapstructure:"access_key" default:"minioadmin"`
	// SecretKey is the secret access key for authentication.
	SecretKey string `mapstructure:"secret_key" default:"minioadmin"`
	// UseSSL indicates whether to use SSL/TLS for connections.
	UseSSL bool `mapstructure:"use_ssl" default:"false"`
	// Bucket is the name of the bucket to store assets in.
	Bucket string `mapstructure:"bucket" default:"assets"`
	// Region is the location of the bucket (e.g., us-east-1).
	Region string `mapstructure:"region" default:""`
	// TimeoutSeconds is the connection timeout in seconds.
	TimeoutSeconds int `mapstructure:"timeout_seconds" default:"30"`
}
