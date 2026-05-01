package switcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelect_Explicit(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "")

	s, err := Select("kscreen", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name())

	s, err = Select("debugfs", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name())

	_, err = Select("bogus", Options{})
	assert.Error(t, err)
}

func TestSelect_AutoDetect(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "")

	s, err := Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name(), "auto outside gamescope should pick kscreen")

	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")
	s, err = Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name(), "auto under gamescope should pick debugfs")
}

func TestSelect_EnvOverridesAuto(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "kscreen")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")

	// auto with env=kscreen should pick kscreen even though gamescope env says otherwise.
	s, err := Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name())
}

func TestSelect_FlagOverridesEnv(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "kscreen")

	// Explicit "debugfs" must beat the env var.
	s, err := Select("debugfs", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name())
}
