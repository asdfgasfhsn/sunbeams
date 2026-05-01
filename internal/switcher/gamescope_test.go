package switcher

import (
	"errors"
	"os"
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

func TestGamescope_SwitchOff_SafeRevertOnByDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1280, Height: 720, Refresh: 60},
			{Width: 1920, Height: 1080, Refresh: 60},
			{Width: 3840, Height: 2160, Refresh: 120},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
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

	require.NoError(t, stub.SwitchOff(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1280x720@60\n", string(body), "should revert to first qualifying mode")
	require.Len(t, helperCalls, 1)
	assert.Equal(t, []string{"on", "HDMI-A-1"}, helperCalls[0])
}

func TestGamescope_SwitchOff_NoSafeRevert(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID:   config.EDIDConfig{MonitorName: "VirtStream"},
		Modes:  []config.Mode{{Width: 1920, Height: 1080, Refresh: 60}},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: false},
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:3840x2160@120\n", string(body), "must leave modes.cfg alone")
}

func TestGamescope_SwitchOff_FallbackWhenNoQualifyingMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		// No mode with W<=1920 H<=1080 R<=60 — should fall back to literal 1920x1080@60.
		Modes: []config.Mode{
			{Width: 3840, Height: 2160, Refresh: 120},
			{Width: 2560, Height: 1440, Refresh: 144},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: true},
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestGamescope_SwitchOff_ConfigOverrideForSafeRevert(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID:  config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{{Width: 1280, Height: 720, Refresh: 60}},
		Gaming: config.GamingConfig{
			ModesCfg:       cfgPath,
			SafeRevertMode: "1024x768@30", // explicit override beats cfg.Modes scan
		},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: true},
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1024x768@30\n", string(body))
}
