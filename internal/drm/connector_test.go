package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanConnectorsReadsSysfs(t *testing.T) {
	root := t.TempDir()
	mk := func(name, status string) {
		dir := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644))
	}
	mk("card0-HDMI-A-1", "disconnected\n")
	mk("card0-DP-1", "connected\n")
	mk("card0-eDP-1", "connected\n") // not HDMI/DP — ignored

	got, err := scanConnectorsAt(root)
	require.NoError(t, err)
	names := make([]string, len(got))
	for i, c := range got {
		names[i] = c.Name
	}
	assert.Contains(t, names, "HDMI-A-1")
	assert.Contains(t, names, "DP-1")
	assert.NotContains(t, names, "eDP-1")
}
