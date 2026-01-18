# Integrity Checks

## Overview
The Asset Manager includes tools to verify the integrity of the storage bucket structure. This ensures that all required directories exist for the Nitro client to function correctly.

## Required Structure
The S3 bucket must contain the following top-level "folders" (prefixes):
- `bundled/`
- `c_images/`
- `dcr/`
- `gamedata/`
- `images/`
- `logos/`
- `sounds/`

## Usage

### CLI
Check integrity:
```bash
go run main.go integrity
```

Check and fix missing folders:
```bash
go run main.go integrity structure --fix
```

### HTTP API
Check integrity (requires API Key):
```bash
curl -H "X-API-Key: <key>" http://localhost:8080/integrity
```

Check and fix (requires API Key):
```bash
curl -H "X-API-Key: <key>" http://localhost:8080/integrity/structure?fix=true
```
