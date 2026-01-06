package integrity

import (
	"context"

	"asset-manager/core/storage"
	"asset-manager/feature/integrity/checks"

	"go.uber.org/zap"
)

// Service handles integrity checks.
type Service struct {
	client storage.Client
	bucket string
	logger *zap.Logger
}

// NewService creates a new integrity service.
func NewService(client storage.Client, bucket string, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		bucket: bucket,
		logger: logger,
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
