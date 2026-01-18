package integrity

import (
	"asset-manager/core/logger"
	"asset-manager/feature/integrity/checks"
	"sync"

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

	// Sync routes
	syncGroup := app.Group("/sync")
	syncGroup.Post("/furniture", h.HandleFurnitureSync)
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
// HandleIntegrityCheck triggers all integrity checks concurrently for faster execution.
// It aggregates the results from each check into a combined report.
func (h *Handler) HandleIntegrityCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Triggering all integrity checks concurrently")

	ctx := c.Context()
	type result struct {
		key  string
		data interface{}
	}
	results := make(chan result, 5)
	var wg sync.WaitGroup
	wg.Add(5)
	// Structure
	go func() {
		defer wg.Done()
		if missing, err := h.service.CheckStructure(ctx); err != nil {
			results <- result{"structure", map[string]interface{}{"status": "error", "error": err.Error()}}
		} else {
			results <- result{"structure", map[string]interface{}{"status": "ok", "missing": missing}}
		}
	}()
	// Bundled
	go func() {
		defer wg.Done()
		if missing, err := h.service.CheckBundled(ctx); err != nil {
			results <- result{"bundled", map[string]interface{}{"status": "error", "error": err.Error()}}
		} else {
			results <- result{"bundled", map[string]interface{}{"status": "ok", "missing": missing}}
		}
	}()
	// GameData
	go func() {
		defer wg.Done()
		if missing, err := h.service.CheckGameData(ctx); err != nil {
			results <- result{"gamedata", map[string]interface{}{"status": "error", "error": err.Error()}}
		} else {
			results <- result{"gamedata", map[string]interface{}{"status": "ok", "missing": missing}}
		}
	}()
	// Server
	go func() {
		defer wg.Done()
		if srvReport, err := h.service.CheckServer(); err != nil {
			results <- result{"server", map[string]interface{}{"status": "error", "error": err.Error()}}
		} else {
			results <- result{"server", srvReport}
		}
	}()
	// Furniture (requires DB)
	go func() {
		defer wg.Done()
		if furnReport, err := h.service.CheckFurniture(ctx); err != nil {
			results <- result{"furniture", map[string]interface{}{"status": "error", "error": err.Error()}}
		} else {
			results <- result{"furniture", furnReport}
		}
	}()
	wg.Wait()
	close(results)
	report := make(map[string]interface{})
	for r := range results {
		report[r.key] = r.data
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
// @Description Perform deep integrity check on furniture assets across FurniData, Storage, and Database
// @Tags integrity
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Furniture Report"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /integrity/furniture [get]
func (h *Handler) HandleFurnitureCheck(c *fiber.Ctx) error {
	l := logger.WithRayID(h.service.logger, c)
	l.Info("Starting furniture integrity check")

	report, err := h.service.CheckFurniture(c.Context())
	if err != nil {
		l.Error("Furniture check failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	l.Info("Furniture check completed",
		zap.Int("total_assets", report.TotalAssets),
		zap.Int("storage_missing", report.StorageMissing))

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
