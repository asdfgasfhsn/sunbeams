package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadDefaults()
	require.NoError(t, err)
	assert.Equal(t, "VRT", cfg.EDID.ManufacturerID)
	assert.Equal(t, uint16(0x2025), cfg.EDID.ProductCode)
	assert.Equal(t, "VirtStream", cfg.EDID.MonitorName)
	assert.Len(t, cfg.Modes, 38)
	assert.Len(t, cfg.Devices, 9)
	assert.Len(t, cfg.StandardTimings, 8)
	assert.Equal(t,
		[]int{97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1},
		cfg.CTA.VICCodes)
	assert.Equal(t, []int{97, 118, 117}, cfg.CTA.Y420VICs)
	// First mode is 4K@60
	assert.Equal(t, 3840, cfg.Modes[0].Width)
	assert.Equal(t, 2160, cfg.Modes[0].Height)
	assert.Equal(t, 60, cfg.Modes[0].Refresh)
}
