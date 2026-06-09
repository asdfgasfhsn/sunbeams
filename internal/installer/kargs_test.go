package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildKargs(t *testing.T) {
	args := BuildKargs("/etc/firmware", "HDMI-A-1", "edid.bin")
	assert.Equal(t, []string{
		"firmware_class.path=/etc/firmware",
		"drm.edid_firmware=HDMI-A-1:edid.bin",
		"video=HDMI-A-1:e",
	}, args)
}

func TestParseSunbeamsKargs(t *testing.T) {
	const single = "ro firmware_class.path=/etc/firmware drm.edid_firmware=DP-2:edid.bin video=DP-2:e quiet"
	const accumulated = "ro drm.edid_firmware=HDMI-A-1:edid.bin video=HDMI-A-1:e " +
		"drm.edid_firmware=DP-2:edid.bin video=DP-2:e firmware_class.path=/etc/firmware"
	const merged = "drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin video=DP-2:e video=HDMI-A-1:e"
	const noise = "ro video=eDP-1:1920x1080 firmware_class.path=/some/other " +
		"drm.edid_firmware=DP-2:edid.bin video=DP-2:e"

	cases := []struct {
		name      string
		cmdline   string
		connector string
		want      []string
	}{
		{
			name:    "single connector full wipe",
			cmdline: single,
			want: []string{
				"firmware_class.path=/etc/firmware",
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "accumulated multi-connector full wipe",
			cmdline: accumulated,
			want: []string{
				"drm.edid_firmware=HDMI-A-1:edid.bin",
				"video=HDMI-A-1:e",
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
				"firmware_class.path=/etc/firmware",
			},
		},
		{
			name:      "connector narrowing excludes others and firmware path",
			cmdline:   accumulated,
			connector: "DP-2",
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "ignores unrelated video and foreign firmware path",
			cmdline: noise,
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "merged drm.edid_firmware token parses both connectors",
			cmdline: merged,
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin",
				"video=DP-2:e",
				"video=HDMI-A-1:e",
			},
		},
		{
			name:    "empty cmdline returns nil",
			cmdline: "ro quiet splash",
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSunbeamsKargs(tc.cmdline, tc.connector)
			assert.Equal(t, tc.want, got)
		})
	}
}
