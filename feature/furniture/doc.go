// Package furniture implements the furniture asset management feature.
//
// It provides functionality to check the integrity of furniture assets by reconciling
// three sources of truth:
//  1. Storage (S3/MinIO): The physical asset files (.nitro).
//  2. Gamedata (JSON): The FurnitureData.json definition file.
//  3. Database: The emulator's furniture definition table.
//
// # Reconcile Adapter
//
// This package utilizes the `core/reconcile` engine via a specialized adapter
// (`core/reconcile/adapters/furniture`). This allows it to perform high-performance,
// concurrent integrity checks across thousands of assets.
//
// # Components
//
//   - Service: Orchestrates the checks and delegates to the integrity/reconcile logic.
//   - Handler: Exposes HTTP endpoints for integrity checks and detail reports.
//   - Loader: Registers the feature with the application.
//
// # HTTP Endpoints
//
//   - GET /furniture/:identifier : Get detailed status for a specific item (e.g. 'f_couch').
package furniture
