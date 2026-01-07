package server

// Config holds configuration for the HTTP server.
type Config struct {
	// Port is the port where the server will listen.
	Port string `mapstructure:"port" default:"8080"`
	// ApiKey is the secret key required to access the API.
	ApiKey string `mapstructure:"api_key" default:""`
	// Emulator specifies the emulator type (arcturus, plusemu, comet).
	Emulator string `mapstructure:"emulator" default:"arcturus"`
}

const (
	EmulatorArcturus = "arcturus"
	EmulatorPlus     = "plusemu"
	EmulatorComet    = "comet"
)

// IsValidEmulator checks if the configured emulator is valid.
func (c Config) IsValidEmulator() bool {
	switch c.Emulator {
	case EmulatorArcturus, EmulatorPlus, EmulatorComet:
		return true
	default:
		return false
	}
}
