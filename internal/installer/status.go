package installer

import (
	"errors"
	"strings"
)

// ErrNoSysfs is returned by Status when the DRM sysfs tree is absent
// (e.g. on macOS), so callers can degrade gracefully.
var ErrNoSysfs = errors.New("no DRM sysfs tree")

// ConnectorStatus is the per-connector EDID-injection state.
type ConnectorStatus struct {
	Name       string // "DP-2"
	Connected  bool   // sysfs status == "connected"
	Configured bool   // sunbeams karg present in rpm-ostree kargs
	BootActive bool   // sunbeams karg present in /proc/cmdline
	EDIDLoaded bool   // live sysfs edid == /etc/firmware/edid.bin
	Verdict    string // synthesized by classify
}

// Report is the full status result.
type Report struct {
	Connectors       []ConnectorStatus
	FirmwarePresent  bool
	FirmwareBytes    int  // size of the installed EDID, 0 if absent
	RebootDetectable bool // false when rpm-ostree was unavailable
}

// sysfsConn is one connector's raw sysfs read.
type sysfsConn struct {
	Status string
	EDID   []byte
}

// classify synthesizes a human-readable verdict from a connector's flags.
// The install-incomplete check takes precedence: a configured connector with
// no firmware file cannot be "active" regardless of the other flags.
func classify(cs ConnectorStatus, firmwarePresent bool) string {
	switch {
	case cs.Configured && !firmwarePresent:
		return "⚠ no /etc/firmware/edid.bin — install incomplete"
	case cs.Configured && !cs.BootActive:
		return "⏳ configured — reboot pending"
	case cs.Configured && cs.BootActive && cs.EDIDLoaded:
		return "✓ active"
	case cs.Configured && cs.BootActive && !cs.EDIDLoaded:
		return "⚠ booted but EDID not loaded (connector disconnected or KMS skipped it)"
	case !cs.Configured && cs.BootActive:
		return "⏳ removal staged — reboot to clear (still active this boot)"
	case !cs.Configured && cs.EDIDLoaded:
		return "active but not configured (orphan — re-run install or uninstall)"
	default:
		return "unknown"
	}
}

// connectorsFromKargs extracts connector names from drm.edid_firmware tokens
// (handles the merged comma form), de-duplicated, in first-seen order.
func connectorsFromKargs(kargs []string) []string {
	const edidPrefix = "drm.edid_firmware="
	seen := map[string]bool{}
	var out []string
	for _, tok := range kargs {
		if !strings.HasPrefix(tok, edidPrefix) {
			continue
		}
		for _, pair := range strings.Split(strings.TrimPrefix(tok, edidPrefix), ",") {
			conn, file, ok := strings.Cut(pair, ":")
			if ok && file == EDIDName && !seen[conn] {
				seen[conn] = true
				out = append(out, conn)
			}
		}
	}
	return out
}
