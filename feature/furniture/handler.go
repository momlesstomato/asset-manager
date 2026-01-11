package furniture

import (
	"asset-manager/core/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for furniture.
type Handler struct {
	service *Service
}

// NewHandler creates a new HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the furniture routes.
func (h *Handler) RegisterRoutes(app fiber.Router) {
	group := app.Group("/furniture")
	group.Get("/:identifier", h.HandleGetFurnitureDetail)
}

// HandleGetFurnitureDetail returns a detailed report for a single furniture item.
// @Summary Get Furniture Detail
// @Description Get detailed integrity report for a specific furniture item.
// @Tags furniture
// @Accept json
// @Produce json
// @Param identifier path string true "Furniture Identifier (e.g. 'f_couch')"
// @Success 200 {object} models.FurnitureDetailReport "Furniture Detail"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /furniture/{identifier} [get]
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
