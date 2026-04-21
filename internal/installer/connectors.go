package installer

import (
	"os"
	"path/filepath"
	"strings"
)

type Connector struct {
	Name   string
	Status string
}

// ScanConnectors reads /sys/class/drm for HDMI/DP connectors.
func ScanConnectors() ([]Connector, error) {
	return scanConnectorsAt("/sys/class/drm")
}

func scanConnectorsAt(root string) ([]Connector, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Connector
	for _, e := range entries {
		name := e.Name()
		// name like "card0-HDMI-A-1"
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
		out = append(out, Connector{
			Name:   connector,
			Status: strings.TrimSpace(string(st)),
		})
	}
	return out, nil
}
