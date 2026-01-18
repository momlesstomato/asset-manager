package database

// Config holds configuration for the database connection.
type Config struct {
	// Host is the database host.
	Host string `mapstructure:"host" default:"localhost"`
	// Port is the database port.
	Port int `mapstructure:"port" default:"3306"`
	// User is the database user.
	User string `mapstructure:"user" default:"root"`
	// Password is the database password.
	Password string `mapstructure:"password" default:""`
	// Name is the database name.
	Name string `mapstructure:"name" default:"emulator"`
	// Driver is the database driver (mysql, sqlite).
	Driver string `mapstructure:"driver" default:"mysql"`
}
