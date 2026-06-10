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
