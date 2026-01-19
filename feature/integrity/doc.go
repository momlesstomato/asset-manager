// Package integrity provides comprehensive system health checks.
//
// Unlike the 'furniture' package which focuses on asset content reconciliation,
// this package validates the infrastructure and structural requirements of the Asset Manager.
//
// # Checks Provided
//
//   - Structure: Checks if the required directory structure exists in the storage bucket (e.g., /gamedata, /bundled).
//   - GameData: Verifies the presence of key configuration files like FurnitureData.json and FigureData.json.
//   - Bundled: Checks for the existence of bundled asset directories (e.g., /bundled/furniture, /bundled/clothing).
//   - Server: Validates that the connected database schema matches the expected emulator definition (columns, types).
//   - Furniture: Triggers the furniture reconciliation process (delegates to furniture package/reconcile engine).
//
// # HTTP Endpoints
//
//   - GET /integrity : Runs all checks.
//   - GET /integrity/structure : Runs structure check (supports ?fix=true).
//   - GET /integrity/gamedata : Runs gamedata check.
//   - GET /integrity/bundled : Runs bundle check (supports ?fix=true).
//   - GET /integrity/server : Runs server schema check.
package integrity
