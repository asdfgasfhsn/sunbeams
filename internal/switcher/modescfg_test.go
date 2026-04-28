package switcher

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModesCfg_FindLine(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		monitor  string
		wantW    int
		wantH    int
		wantR    int
		wantHave bool
	}{
		{"empty file", "", "VirtStream", 0, 0, 0, false},
		{"single line match", "VirtStream:1920x1080@60\n", "VirtStream", 1920, 1080, 60, true},
		{"line with trailing space in name", "Microstep :2340x1080@120\n", "Microstep ", 2340, 1080, 120, true},
		{"multiple lines, ours present", "Other:1920x1080@60\nVirtStream:3840x2160@120\n", "VirtStream", 3840, 2160, 120, true},
		{"multiple lines, ours absent", "Other:1920x1080@60\nFoo:1280x720@60\n", "VirtStream", 0, 0, 0, false},
		{"name is exact match (not prefix)", "VirtStreamX:1920x1080@60\n", "VirtStream", 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h, r, have := findMonitorMode([]byte(tc.body), tc.monitor)
			assert.Equal(t, tc.wantHave, have)
			assert.Equal(t, tc.wantW, w)
			assert.Equal(t, tc.wantH, h)
			assert.Equal(t, tc.wantR, r)
		})
	}
}

func TestModesCfg_UpsertLine(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		monitor string
		w, h, r int
		want    string
	}{
		{"empty file -> append", "", "VirtStream", 1920, 1080, 60, "VirtStream:1920x1080@60\n"},
		{"existing line -> update in place", "VirtStream:800x600@30\n", "VirtStream", 1920, 1080, 60, "VirtStream:1920x1080@60\n"},
		{"keeps other lines intact", "Other:1280x720@60\nVirtStream:800x600@30\nMore:1024x768@60\n", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\nMore:1024x768@60\n"},
		{"appends when monitor missing", "Other:1280x720@60\n", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\n"},
		{"no trailing newline in input -> ensures one in output", "Other:1280x720@60", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := upsertMonitorMode([]byte(tc.input), tc.monitor, tc.w, tc.h, tc.r)
			assert.Equal(t, tc.want, string(out))
		})
	}
}

func TestModesCfg_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	// First write: no .bak, no file.
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:1920x1080@60\n")))
	bak := cfgPath + ".bak"

	// .bak should not exist on first write (we only back up before MODIFYING an existing file).
	_, err := readFile(bak)
	assert.Error(t, err, ".bak should not be created on initial write")

	// Second write: overwriting an existing file should create a .bak first time.
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:3840x2160@120\n")))
	bakBytes, err := readFile(bak)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(bakBytes))

	// Third write: .bak must NOT be overwritten (preserves first-known-good).
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:1280x720@60\n")))
	bakBytes2, err := readFile(bak)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(bakBytes2))
}

func readFile(p string) ([]byte, error) {
	return osReadFile(p)
}
