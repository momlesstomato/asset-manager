// Package logger provides a structured logging facility based on Zap.
//
// It offers a configured logger instance that supports different environments (development vs production)
// and integrates seamlessly with the Fiber web framework.
//
// # Context Awareness
//
// The logger is designed to be context-aware, specifically regarding RayIDs (Request IDs).
// The WithRayID helper extracts the RayID from a Fiber context and attaches it to the
// log entry, ensuring that all logs related to a specific request can be correlated.
//
// # Configuration
//
// The package supports configuration for:
//   - Level: debug, info, warn, error
//   - Encoding: json (production) or console (development)
//
// # Usage
//
//	log, _ := logger.New(&config.LogConfig{Level: "info"})
//	log.Info("Server started")
//
//	// In a request handler:
//	l := logger.WithRayID(log, c)
//	l.Error("Handler failed", zap.Error(err))
package logger
