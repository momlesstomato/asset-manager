package integrity

import (
	"asset-manager/core/logger"
	"asset-manager/feature/integrity/checks"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for integrity checks.
type Handler struct {
	service *Service
}

// NewHandler creates a new HTTP handler.
func NewHandler(service *Service) *Handler {
	// Force import for Swagger
	var _ = checks.ServerReport{}
	return &Handler{service: service}
}

// RegisterRoutes registers the integrity routes.
func (h *Handler) RegisterRoutes(app fiber.Router) {
	group := app.Group("/integrity")
	group.Get("/", h.HandleIntegrityCheck)
	group.Get("/structure", h.HandleStructureCheck)
	group.Get("/bundled", h.HandleBundleCheck)
	group.Get("/gamedata", h.HandleGameDataCheck)
	group.Get("/furniture", h.HandleFurnitureCheck)
	group.Get("/server", h.HandleServerCheck)
}

// HandleIntegrityCheck triggers all integrity checks.
// @Summary Run All Integrity Checks
// @Description Performs all available integrity checks (Structure, Bundled, GameData, Furniture, Server). This operation may take a long time.
// @Tags integrity
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Combined Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity [get]
func (h *Handler) HandleIntegrityCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Triggering all integrity checks")

	ctx := c.Context()
	report := make(map[string]interface{})

	// Structure
	if missing, err := h.service.CheckStructure(ctx); err != nil {
		report["structure"] = map[string]interface{}{"status": "error", "error": err.Error()}
	} else {
		report["structure"] = map[string]interface{}{"status": "ok", "missing": missing}
	}

	// Bundled
	if missing, err := h.service.CheckBundled(ctx); err != nil {
		report["bundled"] = map[string]interface{}{"status": "error", "error": err.Error()}
	} else {
		report["bundled"] = map[string]interface{}{"status": "ok", "missing": missing}
	}

	// GameData
	if missing, err := h.service.CheckGameData(ctx); err != nil {
		report["gamedata"] = map[string]interface{}{"status": "error", "error": err.Error()}
	} else {
		report["gamedata"] = map[string]interface{}{"status": "ok", "missing": missing}
	}

	// Server
	if srvReport, err := h.service.CheckServer(); err != nil {
		report["server"] = map[string]interface{}{"status": "error", "error": err.Error()}
	} else {
		report["server"] = srvReport
	}

	// Furniture (Slow)
	if furnReport, err := h.service.CheckFurniture(ctx, false); err != nil {
		report["furniture"] = map[string]interface{}{"status": "error", "error": err.Error()}
	} else {
		report["furniture"] = furnReport
	}

	return c.JSON(report)
}

// HandleStructureCheck checks and optionally fixes structure.
// @Summary Check Structure
// @Description Checks if the required folder structure exists in the storage bucket. Optionally fixes missing folders.
// @Tags integrity
// @Accept json
// @Produce json
// @Param fix query boolean false "Fix missing folders"
// @Success 200 {object} map[string]interface{} "Structure Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/structure [get]
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

// HandleBundleCheck checks and optionally fixes bundled folders.
// @Summary Check Bundled Folders
// @Description Checks if the required bundled asset folders exist. Optionally fixes missing folders.
// @Tags integrity
// @Accept json
// @Produce json
// @Param fix query boolean false "Fix missing folders"
// @Success 200 {object} map[string]interface{} "Bundle Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/bundled [get]
func (h *Handler) HandleBundleCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	fix := c.Query("fix") == "true"

	missing, err := h.service.CheckBundled(c.Context())
	if err != nil {
		l.Error("Bundle check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if len(missing) > 0 {
		l.Warn("Missing bundled folders detected", zap.Strings("missing", missing))

		if fix {
			l.Info("Attempting to fix missing bundled folders")
			if err := h.service.FixBundled(c.Context(), missing); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to fix bundled folders",
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

// HandleGameDataCheck checks gamedata files.
// @Summary Check GameData
// @Description Verify that all required GameData JSON files are present.
// @Tags integrity
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "GameData Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/gamedata [get]
func (h *Handler) HandleGameDataCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)

	missing, err := h.service.CheckGameData(c.Context())
	if err != nil {
		l.Error("GameData check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "checked",
		"missing": missing,
	})
}

// HandleFurnitureCheck checks integrity of bundled furniture assets.
// @Summary Check Furniture Assets
// @Description Perform deep integrity check on furniture assets in storage.
// @Tags integrity
// @Accept json
// @Produce json
// @Param db query boolean false "Check Database Integrity too"
// @Success 200 {object} map[string]interface{} "Furniture Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/furniture [get]
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

// HandleServerCheck checks server schema integrity.
// @Summary Check Server Schema
// @Description Checks if the emulator database schema matches the expected models.
// @Tags integrity
// @Accept json
// @Produce json
// @Success 200 {object} checks.ServerReport "Server Check Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/server [get]
func (h *Handler) HandleServerCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Starting server schema check")

	report, err := h.service.CheckServer()
	if err != nil {
		l.Error("Server schema check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(report)
}
