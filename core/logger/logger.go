package logger

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap logger based on the configuration.
func New(cfg *Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	var config zap.Config

	if cfg.Level == "debug" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Set format based on configuration
	if cfg.Format == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.DisableStacktrace = true
	} else {
		config.Encoding = "json"
	}

	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.MessageKey = "message"

	logger, err = config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// WithRayID returns a logger with the ray_id field set from the Fiber context.
func WithRayID(l *zap.Logger, c *fiber.Ctx) *zap.Logger {
	rid := c.Locals("ray_id")
	if str, ok := rid.(string); ok && str != "" {
		return l.With(zap.String("ray_id", str))
	}
	return l
}
