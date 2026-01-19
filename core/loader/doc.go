// Package loader provides the plugin-like feature loading system.
//
// It allows the application to register and initialize features (modules) dynamically.
// Each feature implements the Feature interface, which defines its lifecycle hooks
// and route registration logic.
//
// # Feature Interface
//
//	type Feature interface {
//	    Name() string
//	    IsEnabled() bool
//	    Load(app fiber.Router) error
//	}
//
// # Manager
//
// The Manager struct holds the registry of available features. It handles:
//   - Registration of features via Register()
//   - Initialization and loading of enabled features via LoadAll()
//
// This architecture promotes modularity, allowing features like 'furniture', 'integrity',
// or future modules to be developed and tested in isolation.
package loader
