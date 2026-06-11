package drm

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// DetectVirtual resolves the connector that carries the sunbeams virtual EDID.
// It reads the drm.edid_firmware connectors from cmdlinePath (e.g.
// /proc/cmdline); a single configured connector is returned directly. When more
// than one is configured (the accumulation case), it disambiguates by which
// connector's live sysfs EDID exactly equals the firmware file. It returns an
// actionable error when it cannot resolve a single connector. No root required.
func DetectVirtual(sysfsRoot, cmdlinePath, firmwarePath string) (string, error) {
	cmdlineRaw, _ := os.ReadFile(cmdlinePath)
	configured := ConnectorsFromKargs(ParseSunbeamsKargs(string(cmdlineRaw), ""))

	switch len(configured) {
	case 0:
		return "", fmt.Errorf("no virtual display configured: no sunbeams drm.edid_firmware in %s — run 'sudo sunbeams install' (and reboot)", cmdlinePath)
	case 1:
		return configured[0], nil
	}

	// Multiple configured: disambiguate by live EDID byte-match.
	firmware, err := os.ReadFile(firmwarePath)
	if err != nil || len(firmware) == 0 {
		return "", fmt.Errorf("multiple virtual displays configured (%s) and no firmware file to disambiguate — set VIRTUAL_OUTPUT to choose one", strings.Join(configured, ", "))
	}
	conns, err := ScanConnectorEDID(sysfsRoot)
	if err != nil {
		return "", err
	}
	var matched []string
	for _, c := range configured {
		if sc, ok := conns[c]; ok && len(sc.EDID) > 0 && bytes.Equal(sc.EDID, firmware) {
			matched = append(matched, c)
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	return "", fmt.Errorf("could not determine virtual display: %s carry the sunbeams EDID — set VIRTUAL_OUTPUT to pick one, or run 'sunbeams status'", strings.Join(configured, ", "))
}
