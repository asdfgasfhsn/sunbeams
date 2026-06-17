package drm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoSysfs is returned when the DRM sysfs tree is absent (e.g. on macOS),
// so callers can degrade gracefully.
var ErrNoSysfs = errors.New("no DRM sysfs tree")

// SysfsConn is one connector's raw sysfs read.
type SysfsConn struct {
	Status string
	EDID   []byte
}

// ScanConnectorEDID walks a DRM sysfs root and returns each HDMI/DP connector's
// status and live EDID bytes, keyed by connector name (e.g. "DP-2"). Returns
// ErrNoSysfs if the root does not exist.
func ScanConnectorEDID(root string) (map[string]SysfsConn, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSysfs
		}
		return nil, err
	}
	out := map[string]SysfsConn{}
	for _, e := range entries {
		name := e.Name() // e.g. "card0-DP-2"
		dash := strings.Index(name, "-")
		if dash < 0 {
			continue
		}
		connector := name[dash+1:]
		if !strings.HasPrefix(connector, "HDMI") && !strings.HasPrefix(connector, "DP") {
			continue
		}
		st, err := os.ReadFile(filepath.Join(root, name, "status"))
		if err != nil {
			continue
		}
		edid, _ := os.ReadFile(filepath.Join(root, name, "edid")) // may be absent/empty
		out[connector] = SysfsConn{Status: strings.TrimSpace(string(st)), EDID: edid}
	}
	return out, nil
}
