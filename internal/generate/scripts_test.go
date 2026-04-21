package generate

import (
	"strings"
	"testing"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/asdfgasfhsn/sunbeams/internal/edid"
	"github.com/stretchr/testify/require"
)

func TestWriteAddCustomModesScript(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	r, err := Generate(cfg)
	require.NoError(t, err)
	script := WriteAddCustomModesScript(r)
	// Must be a bash script
	if !strings.HasPrefix(script, "#!/usr/bin/env bash") {
		t.Fatalf("missing shebang")
	}
	// Must cover all high-clock modes
	for _, hm := range r.HighModes {
		if !strings.Contains(script, xrandrModelineName(hm.Timing)) {
			t.Errorf("script missing mode %dx%d@%d", hm.Timing.HActive, hm.Timing.VActive, hm.Timing.Refresh)
		}
	}
}

func TestWriteSunshineCommands(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	s := WriteSunshineCommands(cfg)
	for _, d := range cfg.Devices {
		if !strings.Contains(s, d.Slug) {
			t.Errorf("missing device %s", d.Slug)
		}
	}
}

// xrandrModelineName is a helper used by tests to extract the mode name.
func xrandrModelineName(t edid.Timing) string {
	_, name := XRandrModeline(t)
	return name
}
