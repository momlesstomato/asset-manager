package furniture

import (
	"context"

	"asset-manager/core/storage"
	"asset-manager/feature/furniture/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service handles furniture operations.
type Service struct {
	client   storage.Client
	bucket   string
	logger   *zap.Logger
	db       *gorm.DB
	emulator string
}

// NewService creates a new furniture service.
func NewService(client storage.Client, bucket string, logger *zap.Logger, db *gorm.DB, emulator string) *Service {
	return &Service{
		client:   client,
		bucket:   bucket,
		logger:   logger,
		db:       db,
		emulator: emulator,
	}
}

// GetFurnitureDetail returns detailed integrity info for a single furniture item.
func (s *Service) GetFurnitureDetail(ctx context.Context, identifier string) (*models.FurnitureDetailReport, error) {
	return CheckFurnitureItem(ctx, s.client, s.bucket, s.db, s.emulator, identifier)
}
