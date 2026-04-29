package switcher

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

func TestGamescope_SwitchOn_PicksMatchingModeAndCallsHelper(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
			{Width: 3840, Height: 2160, Refresh: 120},
		},
		Gaming: config.GamingConfig{
			HelperPath: "/fake/sunbeams-drm-force",
			ModesCfg:   cfgPath, // tests pass an absolute path; production resolves relative
		},
	}

	var helperCalls [][]string
	stub := &GamescopeStrategy{
		Opts: Options{SafeRevert: true},
		runHelper: func(action, connector string) error {
			helperCalls = append(helperCalls, []string{action, connector})
			return nil
		},
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	err := stub.SwitchOn(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}, 3840, 2160, 120, false)
	require.NoError(t, err)

	body, err := osReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:3840x2160@120\n", string(body))

	require.Len(t, helperCalls, 1)
	assert.Equal(t, []string{"off", "HDMI-A-1"}, helperCalls[0])
}

func TestGamescope_SwitchOn_HelperFailureBubbles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		runHelper:    func(action, connector string) error { return errors.New("sudo: a password is required") },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	err := stub.SwitchOn(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}, 1920, 1080, 60, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force off")

	// modes.cfg was still written before the helper exec — that's fine;
	// gamescope simply uses the new mode on next hotplug.
	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestGamescope_SwitchOn_VirtualNameDefaultsToEDIDMonitorName(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	// Note: Outputs.Virtual is the DRM connector ("DP-1") for the helper, but
	// modes.cfg is keyed by EDID monitor name. The strategy MUST use the
	// EDID name when editing modes.cfg, regardless of what Outputs.Virtual says.
	err := stub.SwitchOn(cfg, Outputs{Virtual: "DP-1", Physical: "HDMI-A-1"}, 1920, 1080, 60, false)
	require.NoError(t, err)

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}
