package models_test

import (
	"testing"

	"asset-manager/feature/emulator/models"

	"github.com/stretchr/testify/assert"
)

func TestEmulatorModels(t *testing.T) {
	t.Run("Arcturus", func(t *testing.T) {
		m := models.ArcturusItemsBase{}
		assert.Equal(t, "items_base", m.TableName())
		// Instantiate fields to ensure they exist/copmpile
		m.ID = 1
		m.ItemName = "test"
	})

	t.Run("Comet", func(t *testing.T) {
		m := models.CometFurniture{}
		assert.Equal(t, "furniture", m.TableName())
		m.ID = 1
		m.ItemName = "test"
	})

	t.Run("Plus", func(t *testing.T) {
		m := models.PlusFurniture{}
		assert.Equal(t, "furniture", m.TableName())
		m.ID = 1
		m.ItemName = "test"
	})
}
