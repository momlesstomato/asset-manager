package integrity

import (
	"asset-manager/core/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for integrity checks.
type Handler struct {
	service *Service
}

// NewHandler creates a new HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the integrity routes.
func (h *Handler) RegisterRoutes(app fiber.Router) {
	group := app.Group("/integrity")
	group.Get("/", h.HandleIntegrityCheck)
	group.Get("/structure", h.HandleStructureCheck)
	group.Get("/furniture", h.HandleFurnitureCheck)

	// Separate furniture detail view
	app.Get("/furniture/:identifier", h.HandleGetFurnitureDetail)
}

// HandleIntegrityCheck triggers all integrity checks.
func (h *Handler) HandleIntegrityCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Triggering all integrity checks")

	// Currently only structure check exists in this handler.
	// Users should use specific endpoints for heavier checks.
	missing, err := h.service.CheckStructure(c.Context())
	if err != nil {
		l.Error("Structure check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	status := "ok"
	if len(missing) > 0 {
		status = "warning"
	}

	return c.JSON(fiber.Map{
		"status":  status,
		"missing": missing,
	})
}

// HandleStructureCheck checks and optionally fixes structure.
func (h *Handler) HandleStructureCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	fix := c.Query("fix") == "true"

	missing, err := h.service.CheckStructure(c.Context())
	if err != nil {
		l.Error("Structure check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if len(missing) > 0 {
		l.Warn("Missing folders detected", zap.Strings("missing", missing))

		if fix {
			l.Info("Attempting to fix missing folders")
			if err := h.service.FixStructure(c.Context(), missing); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to fix structure",
					"details": err.Error(),
					"missing": missing,
				})
			}
			return c.JSON(fiber.Map{
				"status": "fixed",
				"fixed":  missing,
			})
		}
	}

	return c.JSON(fiber.Map{
		"status":  "checked",
		"missing": missing,
	})
}

// HandleFurnitureCheck checks integrity of bundled furniture assets.
func (h *Handler) HandleFurnitureCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Starting furniture integrity check")

	checkDB := c.Query("db") == "true"
	report, err := h.service.CheckFurniture(c.Context(), checkDB)
	if err != nil {
		l.Error("Furniture check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	l.Info("Furniture check completed",
		zap.Int("expected", report.TotalExpected),
		zap.Int("found", report.TotalFound))

	return c.JSON(report)
}

// HandleGetFurnitureDetail returns a detailed report for a single furniture item.
func (h *Handler) HandleGetFurnitureDetail(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	l := logger.WithRayID(h.service.logger, c)

	report, err := h.service.GetFurnitureDetail(c.Context(), identifier)
	if err != nil {
		l.Error("Furniture detail check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(report)
}
