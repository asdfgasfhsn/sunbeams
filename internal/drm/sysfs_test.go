package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConnector(t *testing.T, root, dir, status string, edid []byte) {
	t.Helper()
	d := filepath.Join(root, dir)
	require.NoError(t, os.MkdirAll(d, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(d, "status"), []byte(status), 0o644))
	if edid != nil {
		require.NoError(t, os.WriteFile(filepath.Join(d, "edid"), edid, 0o644))
	}
}

func TestScanConnectorEDID(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", []byte("OURS"))
	writeConnector(t, root, "card0-eDP-1", "connected\n", []byte("laptop")) // not HDMI/DP — ignored

	got, err := ScanConnectorEDID(root)
	require.NoError(t, err)
	require.Contains(t, got, "DP-2")
	assert.Equal(t, "disconnected", got["DP-2"].Status)
	assert.Equal(t, []byte("OURS"), got["DP-2"].EDID)
	assert.NotContains(t, got, "eDP-1")
}

func TestScanConnectorEDID_NoSysfs(t *testing.T) {
	_, err := ScanConnectorEDID(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.ErrorIs(t, err, ErrNoSysfs)
}
