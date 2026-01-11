package furniture

import (
	"asset-manager/core/storage"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Feature implements the loader.Feature interface.
type Feature struct {
	service *Service
	handler *Handler
}

// NewFeature creates a new Furniture feature.
func NewFeature(client storage.Client, bucket string, logger *zap.Logger, db *gorm.DB, emulator string) *Feature {
	svc := NewService(client, bucket, logger, db, emulator)
	h := NewHandler(svc)
	return &Feature{service: svc, handler: h}
}

// Name returns the name of the feature.
func (f *Feature) Name() string {
	return "furniture"
}

// IsEnabled checks if the feature is enabled.
func (f *Feature) IsEnabled() bool {
	return true
}

// Load registers the feature's routes.
func (f *Feature) Load(app fiber.Router) error {
	f.handler.RegisterRoutes(app)
	return nil
}
