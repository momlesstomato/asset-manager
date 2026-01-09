package integrity

import (
	"context"

	"asset-manager/core/storage"
	"asset-manager/feature/integrity/checks"

	"asset-manager/feature/furniture"
	"asset-manager/feature/furniture/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service handles integrity checks.
type Service struct {
	client   storage.Client
	bucket   string
	logger   *zap.Logger
	db       *gorm.DB
	emulator string
}

// NewService creates a new integrity service.
func NewService(client storage.Client, bucket string, logger *zap.Logger, db *gorm.DB, emulator string) *Service {
	return &Service{
		client:   client,
		bucket:   bucket,
		logger:   logger,
		db:       db,
		emulator: emulator,
	}
}

// CheckStructure returns a list of missing folders.
func (s *Service) CheckStructure(ctx context.Context) ([]string, error) {
	return checks.CheckStructure(ctx, s.client, s.bucket)
}

// FixStructure creates the missing folders.
func (s *Service) FixStructure(ctx context.Context, missing []string) error {
	return checks.FixStructure(ctx, s.client, s.bucket, s.logger, missing)
}

// CheckGameData returns a list of missing files in the gamedata folder.
func (s *Service) CheckGameData(ctx context.Context) ([]string, error) {
	return checks.CheckGameData(ctx, s.client, s.bucket)
}

// CheckBundled returns a list of missing bundled folders.
func (s *Service) CheckBundled(ctx context.Context) ([]string, error) {
	return checks.CheckBundled(ctx, s.client, s.bucket)
}

// FixBundled creates the missing bundled folders.
func (s *Service) FixBundled(ctx context.Context, missing []string) error {
	return checks.FixBundled(ctx, s.client, s.bucket, s.logger, missing)
}

// CheckFurniture performs an integrity check on furniture assets.
func (s *Service) CheckFurniture(ctx context.Context, checkDB bool) (*models.Report, error) {
	var db *gorm.DB
	if checkDB {
		db = s.db
	}
	return furniture.CheckIntegrity(ctx, s.client, s.bucket, db, s.emulator)
}

// CheckServer performs an integrity check on the emulator database schema.
func (s *Service) CheckServer() (*checks.ServerReport, error) {
	if s.db == nil {
		return nil, nil // Or specific error? "Database not connected"
	}
	return checks.CheckServerIntegrity(s.db, s.emulator)
}

// GetFurnitureDetail returns detailed integrity info for a single furniture item.
func (s *Service) GetFurnitureDetail(ctx context.Context, identifier string) (*models.FurnitureDetailReport, error) {
	return furniture.CheckFurnitureItem(ctx, s.client, s.bucket, s.db, s.emulator, identifier)
}
