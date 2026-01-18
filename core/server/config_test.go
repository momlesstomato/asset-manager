package server_test

import (
	"testing"

	"asset-manager/core/server"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsValidEmulator(t *testing.T) {
	tests := []struct {
		name     string
		emulator string
		want     bool
	}{
		{"Arcturus", server.EmulatorArcturus, true},
		{"Plus", server.EmulatorPlus, true},
		{"Comet", server.EmulatorComet, true},
		{"Invalid", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := server.Config{Emulator: tt.emulator}
			assert.Equal(t, tt.want, c.IsValidEmulator())
		})
	}
}
