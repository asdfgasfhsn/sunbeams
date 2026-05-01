package installer

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallHelper_WritesScriptWithMode0700(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "sbin", "sunbeams-drm-force")

	require.NoError(t, InstallHelper(dst, HelperScript()))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o700), info.Mode().Perm())

	body, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, string(HelperScript()), string(body))
}

func TestInstallSudoers_WritesWithMode0440AndValidates(t *testing.T) {
	if _, err := exec.LookPath("visudo"); err != nil {
		t.Skip("visudo not available; skipping (production install always requires it)")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "sudoers.d", "sunbeams-drm-switch")

	require.NoError(t, InstallSudoers(dst, "alice", "/usr/local/sbin/sunbeams-drm-force"))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o440), info.Mode().Perm())

	body, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Contains(t, string(body), "alice ALL=(root) NOPASSWD: /usr/local/sbin/sunbeams-drm-force")
	assert.True(t, len(body) > 0 && body[len(body)-1] == '\n', "must end with newline (sudoers requirement)")
}

func TestInstallSudoers_RejectsEmptyUser(t *testing.T) {
	// The empty-user guard short-circuits before visudo runs. This pins the
	// guard's behavior: error is returned, dst is not created. A future test
	// could exercise the visudo-rejection path with non-empty malformed input.
	dir := t.TempDir()
	dst := filepath.Join(dir, "sudoers.d", "sunbeams-drm-switch")

	err := InstallSudoers(dst, "", "/usr/local/sbin/sunbeams-drm-force")
	require.Error(t, err)

	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr), "install must abort before writing to dst when guard fires")
}

func TestSeedModesCfg_NewFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".config", "gamescope", "modes.cfg")

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestSeedModesCfg_UpdatesExistingLine(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("Other:1280x720@60\nVirtStream:800x600@30\n"), 0o644))

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "Other:1280x720@60\nVirtStream:1920x1080@60\n", string(body))
}

func TestSeedModesCfg_AppendsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("Other:1280x720@60\n"), 0o644))

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "Other:1280x720@60\nVirtStream:1920x1080@60\n", string(body))
}

func TestPreflightDebugfsPath_NoMatchOK(t *testing.T) {
	// The pre-flight is permissive: zero matches is a warn, not an error.
	// Pass a connector that doesn't exist on the host.
	res := PreflightDebugfsPath("ZZZZ-Z-99")
	assert.Equal(t, 0, res.Count)
	assert.NoError(t, res.Err, "zero matches must not be an error")
}
