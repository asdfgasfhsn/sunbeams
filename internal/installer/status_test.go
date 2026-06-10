package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name            string
		cs              ConnectorStatus
		firmwarePresent bool
		want            string
	}{
		{
			name:            "active",
			cs:              ConnectorStatus{Configured: true, BootActive: true, EDIDLoaded: true},
			firmwarePresent: true,
			want:            "✓ active",
		},
		{
			name:            "reboot pending",
			cs:              ConnectorStatus{Configured: true, BootActive: false},
			firmwarePresent: true,
			want:            "⏳ configured — reboot pending",
		},
		{
			name:            "booted but not loaded",
			cs:              ConnectorStatus{Configured: true, BootActive: true, EDIDLoaded: false},
			firmwarePresent: true,
			want:            "⚠ booted but EDID not loaded (connector disconnected or KMS skipped it)",
		},
		{
			name:            "install incomplete takes precedence over reboot pending",
			cs:              ConnectorStatus{Configured: true, BootActive: false},
			firmwarePresent: false,
			want:            "⚠ no /etc/firmware/edid.bin — install incomplete",
		},
		{
			name:            "removal staged",
			cs:              ConnectorStatus{Configured: false, BootActive: true, EDIDLoaded: true},
			firmwarePresent: true,
			want:            "⏳ removal staged — reboot to clear (still active this boot)",
		},
		{
			name:            "orphan",
			cs:              ConnectorStatus{Configured: false, BootActive: false, EDIDLoaded: true},
			firmwarePresent: true,
			want:            "active but not configured (orphan — re-run install or uninstall)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classify(tc.cs, tc.firmwarePresent))
		})
	}
}

func TestConnectorsFromKargs(t *testing.T) {
	assert.Equal(t, []string{"DP-2"}, connectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e", "firmware_class.path=/etc/firmware",
	}))
	assert.Equal(t, []string{"HDMI-A-1", "DP-2"}, connectorsFromKargs([]string{
		"drm.edid_firmware=HDMI-A-1:edid.bin", "drm.edid_firmware=DP-2:edid.bin",
	}))
	assert.Equal(t, []string{"DP-2", "HDMI-A-1"}, connectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin",
	}))
	assert.Nil(t, connectorsFromKargs([]string{"video=DP-2:e", "firmware_class.path=/etc/firmware"}))
}

func TestBuildReport(t *testing.T) {
	fw := []byte("EDID-BYTES-768")
	conns := map[string]sysfsConn{
		"DP-2":     {Status: "disconnected", EDID: fw},                  // configured+boot+loaded
		"HDMI-A-1": {Status: "connected", EDID: []byte("real-monitor")}, // configured, not yet booted
	}
	rep := buildReport(
		[]string{"DP-2", "HDMI-A-1"}, // configured
		[]string{"DP-2"},             // boot
		fw, true, conns,
	)

	assert.True(t, rep.FirmwarePresent)
	assert.Equal(t, len(fw), rep.FirmwareBytes)
	assert.Len(t, rep.Connectors, 2)

	// Sorted by name: DP-2 before HDMI-A-1.
	dp := rep.Connectors[0]
	assert.Equal(t, "DP-2", dp.Name)
	assert.False(t, dp.Connected)
	assert.True(t, dp.Configured)
	assert.True(t, dp.BootActive)
	assert.True(t, dp.EDIDLoaded)
	assert.Equal(t, "✓ active", dp.Verdict)

	hd := rep.Connectors[1]
	assert.Equal(t, "HDMI-A-1", hd.Name)
	assert.True(t, hd.Connected)
	assert.True(t, hd.Configured)
	assert.False(t, hd.BootActive)
	assert.False(t, hd.EDIDLoaded)
	assert.Equal(t, "⏳ configured — reboot pending", hd.Verdict)
}

func TestBuildReport_OrphanFromMatchingEDID(t *testing.T) {
	fw := []byte("OURS")
	conns := map[string]sysfsConn{
		"DP-1": {Status: "connected", EDID: fw}, // edid matches but not in any kargs
	}
	rep := buildReport(nil, nil, fw, true, conns)
	assert.Len(t, rep.Connectors, 1)
	assert.Equal(t, "DP-1", rep.Connectors[0].Name)
	assert.True(t, rep.Connectors[0].EDIDLoaded)
	assert.Equal(t, "active but not configured (orphan — re-run install or uninstall)", rep.Connectors[0].Verdict)
}

func TestBuildReport_NoFirmwareMarksIncomplete(t *testing.T) {
	conns := map[string]sysfsConn{"DP-2": {Status: "disconnected", EDID: nil}}
	rep := buildReport([]string{"DP-2"}, []string{"DP-2"}, nil, false, conns)
	assert.False(t, rep.FirmwarePresent)
	assert.Len(t, rep.Connectors, 1)
	assert.False(t, rep.Connectors[0].EDIDLoaded)
	assert.Equal(t, "⚠ no /etc/firmware/edid.bin — install incomplete", rep.Connectors[0].Verdict)
}
