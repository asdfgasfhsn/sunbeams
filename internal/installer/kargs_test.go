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
