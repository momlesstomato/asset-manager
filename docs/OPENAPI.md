# OpenAPI Documentation Standard

## Overview
This project mandates the use of OpenAPI (Swagger) to document all API endpoints. This ensures that the API is discoverable, testable, and well-understood by consumers.

## Requirements

1.  **Exhaustive Documentation**: Every exposed HTTP endpoint MUST have corresponding Swagger annotations.
2.  **Detail Level**:
    -   **Description**: Clear summary of what the endpoint does.
    -   **Parameters**: specific details for every path parameter, query parameter, and header.
    -   **Request Body**: precise JSON schema definitions for request payloads.
    -   **Responses**: exact JSON schema definitions for all possible response codes (200, 400, 404, 500, etc.).
    -   **Authentication**: explicit declaration of required security schemes (e.g., API Key, Bearer Token).
3.  **Availability**: The `swagger.json` file and Swagger UI MUST be available when the server is running.
    -   URL: `/swagger/index.html` (or similar)
4.  **Consistency**: Follow the architectural guidelines in `ARCHITECTURE.md`.

## Annotations (swaggo/swag)
We use [swaggo/swag](https://github.com/swaggo/swag) to generate the Swagger documentation from Go comments.

### Example

```go
// HandleGetAsset retrieves an asset by ID.
// @Summary Get Asset
// @Description Fetch a specific asset by its unique identifier.
// @Tags assets
// @Accept json
// @Produce json
// @Param id path string true "Asset ID"
// @Param X-API-Key header string true "API Key"
// @Success 200 {object} models.Asset "Successful retrieval"
// @Failure 404 {object} models.ErrorResponse "Asset not found"
// @Failure 500 {object} models.ErrorResponse "Internal Server Error"
// @Router /assets/{id} [get]
func (h *Handler) HandleGetAsset(c *fiber.Ctx) error {
    // ...
}
```

## Generation
To generate the `docs` folder containing `swagger.json` and `swagger.yaml`, run:

```bash
swag init -g cmd/server/main.go --output docs/swagger
```
*(Adjust the entry point path as per your project structure)*

## 1:1 Parity
The API MUST expose functionality equivalent to the CLI commands where applicable. Ensure that all integrity checks available via `go run main.go integrity ...` are also accessible via HTTP endpoints.
