package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConnector is defined in sysfs_test.go (same package).

func cmdlineFile(t *testing.T, contents string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cmdline")
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	return p
}

func firmwareFile(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "edid.bin")
	require.NoError(t, os.WriteFile(p, b, 0o644))
	return p
}

func TestDetectVirtual_SingleConfigured(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro drm.edid_firmware=DP-2:edid.bin video=DP-2:e")
	fw := firmwareFile(t, []byte("EDID"))

	got, err := DetectVirtual(root, cmd, fw)
	require.NoError(t, err)
	assert.Equal(t, "DP-2", got)
}

func TestDetectVirtual_MultipleDisambiguatedByEDID(t *testing.T) {
	root := t.TempDir()
	payload := []byte("OUR-EDID")
	writeConnector(t, root, "card0-DP-2", "disconnected\n", payload)         // matches firmware
	writeConnector(t, root, "card0-HDMI-A-1", "connected\n", []byte("real")) // does not
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := firmwareFile(t, payload)

	got, err := DetectVirtual(root, cmd, fw)
	require.NoError(t, err)
	assert.Equal(t, "DP-2", got)
}

func TestDetectVirtual_AmbiguousErrors(t *testing.T) {
	root := t.TempDir()
	payload := []byte("OUR-EDID")
	writeConnector(t, root, "card0-DP-2", "disconnected\n", payload)
	writeConnector(t, root, "card0-HDMI-A-1", "disconnected\n", payload) // both match
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := firmwareFile(t, payload)

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VIRTUAL_OUTPUT")
}

func TestDetectVirtual_NoneConfiguredErrors(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro quiet splash")
	fw := firmwareFile(t, []byte("EDID"))

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install")
}

func TestDetectVirtual_MultipleNoFirmwareErrors(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	writeConnector(t, root, "card0-HDMI-A-1", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := filepath.Join(t.TempDir(), "missing-edid.bin") // does not exist

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VIRTUAL_OUTPUT")
}
