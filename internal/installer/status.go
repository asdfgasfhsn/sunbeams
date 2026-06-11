package installer

import (
	"bytes"
	"os"
	"sort"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)

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
func buildReport(configured, boot []string, firmwareBytes []byte, firmwarePresent bool, conns map[string]drm.SysfsConn) Report {
	cfgSet := toSet(configured)
	bootSet := toSet(boot)

	names := map[string]bool{}
	for n := range cfgSet {
		names[n] = true
	}
	for n := range bootSet {
		names[n] = true
	}
	if firmwarePresent && len(firmwareBytes) > 0 {
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
			EDIDLoaded: ok && firmwarePresent && len(firmwareBytes) > 0 && bytes.Equal(c.EDID, firmwareBytes),
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

// Status reports the sunbeams EDID-injection state for every connector. All
// three paths are injectable for testing; the CLI passes "/sys/class/drm",
// "/proc/cmdline", and the installed firmware path. Requires no root.
func Status(sysfsRoot, cmdlinePath, firmwarePath string) (Report, error) {
	conns, err := drm.ScanConnectorEDID(sysfsRoot)
	if err != nil {
		return Report{}, err
	}

	// Configured source: prefer rpm-ostree kargs (persistent intent). If
	// unavailable, fall back to /proc/cmdline and flag reboot-pending (and
	// removal-staged) detection as undetectable — both sets become identical.
	cmdlineRaw, _ := os.ReadFile(cmdlinePath)
	rebootDetectable := true
	configuredRaw, err := CurrentKargs()
	if err != nil {
		configuredRaw = string(cmdlineRaw)
		rebootDetectable = false
	}

	configured := drm.ConnectorsFromKargs(drm.ParseSunbeamsKargs(configuredRaw, ""))
	boot := drm.ConnectorsFromKargs(drm.ParseSunbeamsKargs(string(cmdlineRaw), ""))

	firmwareBytes, fwErr := os.ReadFile(firmwarePath)
	firmwarePresent := fwErr == nil && len(firmwareBytes) > 0

	rep := buildReport(configured, boot, firmwareBytes, firmwarePresent, conns)
	rep.RebootDetectable = rebootDetectable
	return rep, nil
}
