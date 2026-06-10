package installer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
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

// buildReport assembles a Report from the configured/boot connector lists, the
// installed firmware bytes, and the raw sysfs reads. A connector is included if
// it is configured, active this boot, or (when firmware is present) carries the
// firmware's exact EDID bytes (orphan detection). Output is sorted by name.
func buildReport(configured, boot []string, firmwareBytes []byte, firmwarePresent bool, conns map[string]sysfsConn) Report {
	cfgSet := toSet(configured)
	bootSet := toSet(boot)

	names := map[string]bool{}
	for n := range cfgSet {
		names[n] = true
	}
	for n := range bootSet {
		names[n] = true
	}
	if firmwarePresent {
		for n, c := range conns {
			if bytes.Equal(c.EDID, firmwareBytes) {
				names[n] = true
			}
		}
	}

	var list []ConnectorStatus
	for n := range names {
		c, ok := conns[n]
		cs := ConnectorStatus{
			Name:       n,
			Connected:  ok && c.Status == "connected",
			Configured: cfgSet[n],
			BootActive: bootSet[n],
			EDIDLoaded: ok && firmwarePresent && bytes.Equal(c.EDID, firmwareBytes),
		}
		cs.Verdict = classify(cs, firmwarePresent)
		list = append(list, cs)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

	size := 0
	if firmwarePresent {
		size = len(firmwareBytes)
	}
	return Report{Connectors: list, FirmwarePresent: firmwarePresent, FirmwareBytes: size}
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// scanConnectorEDID walks a DRM sysfs root and returns each HDMI/DP connector's
// status and live EDID bytes, keyed by connector name (e.g. "DP-2"). Returns
// ErrNoSysfs if the root does not exist.
func scanConnectorEDID(root string) (map[string]sysfsConn, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSysfs
		}
		return nil, err
	}
	out := map[string]sysfsConn{}
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
		out[connector] = sysfsConn{Status: strings.TrimSpace(string(st)), EDID: edid}
	}
	return out, nil
}

// Status reports the sunbeams EDID-injection state for every connector. All
// three paths are injectable for testing; the CLI passes "/sys/class/drm",
// "/proc/cmdline", and the installed firmware path. Requires no root.
func Status(sysfsRoot, cmdlinePath, firmwarePath string) (Report, error) {
	conns, err := scanConnectorEDID(sysfsRoot)
	if err != nil {
		return Report{}, err
	}

	// Configured source: prefer rpm-ostree kargs (persistent intent). If
	// unavailable, fall back to /proc/cmdline and flag reboot-pending as
	// undetectable.
	cmdlineRaw, _ := os.ReadFile(cmdlinePath)
	rebootDetectable := true
	configuredRaw, err := CurrentKargs()
	if err != nil {
		configuredRaw = string(cmdlineRaw)
		rebootDetectable = false
	}

	configured := connectorsFromKargs(ParseSunbeamsKargs(configuredRaw, ""))
	boot := connectorsFromKargs(ParseSunbeamsKargs(string(cmdlineRaw), ""))

	firmwareBytes, fwErr := os.ReadFile(firmwarePath)
	firmwarePresent := fwErr == nil

	rep := buildReport(configured, boot, firmwareBytes, firmwarePresent, conns)
	rep.RebootDetectable = rebootDetectable
	return rep, nil
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
