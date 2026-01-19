package integrity

import (
	"asset-manager/core/storage/mocks"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoader(t *testing.T) {
	mockClient := new(mocks.Client)
	logger := zap.NewNop()
	// Pass nil db for this test as we don't access it unless we use the service
	feature := NewFeature(mockClient, "test-bucket", logger, nil, "")

	assert.Equal(t, "integrity", feature.Name())
	assert.True(t, feature.IsEnabled())

	app := fiber.New()
	err := feature.Load(app)
	assert.NoError(t, err)
}
