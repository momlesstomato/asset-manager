package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmulator(t *testing.T) {
	tests := []struct {
		name     string
		emulator string
		want     bool
	}{
		{"arcturus", EmulatorArcturus, true},
		{"plusemu", EmulatorPlus, true},
		{"comet", EmulatorComet, true},
		{"invalid", "unknown", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Emulator: tt.emulator}
			assert.Equal(t, tt.want, cfg.IsValidEmulator())
		})
	}
}
