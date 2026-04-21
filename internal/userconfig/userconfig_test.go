package userconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithOverride_MonitorName(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(override, []byte(`
[edid]
monitor_name = "Overridden"
`), 0o644))

	cfg, err := LoadWithOverride(override)
	require.NoError(t, err)
	assert.Equal(t, "Overridden", cfg.EDID.MonitorName)
	// Non-overridden fields come from defaults
	assert.Equal(t, "VRT", cfg.EDID.ManufacturerID)
	assert.NotEmpty(t, cfg.Modes)
}

func TestLoadWithOverride_DevicesReplaceDefaults(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(override, []byte(`
[[devices]]
slug = "only-one"
label = "Only device"
width = 1920
height = 1080
max_refresh = 60
hdr = false
`), 0o644))

	cfg, err := LoadWithOverride(override)
	require.NoError(t, err)
	assert.Len(t, cfg.Devices, 1)
	assert.Equal(t, "only-one", cfg.Devices[0].Slug)
}

func TestLoadWithOverride_Missing(t *testing.T) {
	cfg, err := LoadWithOverride(filepath.Join(t.TempDir(), "nope.toml"))
	require.NoError(t, err)
	assert.Equal(t, "VRT", cfg.EDID.ManufacturerID)
}
