package drm

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
		{"single connector full wipe", single, "", []string{
			"firmware_class.path=/etc/firmware", "drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"accumulated multi-connector full wipe", accumulated, "", []string{
			"drm.edid_firmware=HDMI-A-1:edid.bin", "video=HDMI-A-1:e",
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e", "firmware_class.path=/etc/firmware",
		}},
		{"connector narrowing excludes others and firmware path", accumulated, "DP-2", []string{
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"ignores unrelated video and foreign firmware path", noise, "", []string{
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"merged drm.edid_firmware token parses both connectors", merged, "", []string{
			"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin", "video=DP-2:e", "video=HDMI-A-1:e",
		}},
		{"connector narrowing with merged token returns whole token", merged, "DP-2", []string{
			"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin", "video=DP-2:e",
		}},
		{"empty cmdline returns nil", "ro quiet splash", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseSunbeamsKargs(tc.cmdline, tc.connector))
		})
	}
}

func TestConnectorsFromKargs(t *testing.T) {
	assert.Equal(t, []string{"DP-2"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e", "firmware_class.path=/etc/firmware",
	}))
	assert.Equal(t, []string{"HDMI-A-1", "DP-2"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=HDMI-A-1:edid.bin", "drm.edid_firmware=DP-2:edid.bin",
	}))
	assert.Equal(t, []string{"DP-2", "HDMI-A-1"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin",
	}))
	assert.Nil(t, ConnectorsFromKargs([]string{"video=DP-2:e", "firmware_class.path=/etc/firmware"}))
}
