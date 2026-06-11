package switcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)

// Outputs names the connectors involved. Empty fields fall back to env
// VIRTUAL_OUTPUT / PHYSICAL_OUTPUT and finally to auto-detection.
type Outputs struct {
	Virtual  string
	Physical string
}

// detectVirtual and scanConnectors are package-level seams so tests can stub
// auto-detection without real hardware.
var detectVirtual = func() (string, error) {
	return drm.DetectVirtual("/sys/class/drm", "/proc/cmdline",
		filepath.Join(drm.FirmwareDir, drm.EDIDName))
}

var scanConnectors = drm.ScanConnectors

// resolveOutputs determines the virtual connector and the physical connectors
// to disable. Virtual: explicit flag → VIRTUAL_OUTPUT → auto-detect (error if
// unresolved). Physical: explicit flag → PHYSICAL_OUTPUT → every connected
// connector that is not the virtual one (empty when headless; scan errors are
// non-fatal).
func resolveOutputs(o Outputs) (virt string, phys []string, virtSrc, physSrc string, err error) {
	switch {
	case o.Virtual != "":
		virt, virtSrc = o.Virtual, "flag"
	case os.Getenv("VIRTUAL_OUTPUT") != "":
		virt, virtSrc = os.Getenv("VIRTUAL_OUTPUT"), "env:VIRTUAL_OUTPUT"
	default:
		virt, err = detectVirtual()
		if err != nil {
			return "", nil, "", "", fmt.Errorf("auto-detect virtual display: %w", err)
		}
		virtSrc = "auto"
	}

	switch {
	case o.Physical != "":
		phys, physSrc = []string{o.Physical}, "flag"
	case os.Getenv("PHYSICAL_OUTPUT") != "":
		phys, physSrc = []string{os.Getenv("PHYSICAL_OUTPUT")}, "env:PHYSICAL_OUTPUT"
	default:
		physSrc = "auto"
		cons, scanErr := scanConnectors()
		if scanErr != nil {
			warn("could not scan connectors for physical outputs: %v", scanErr)
			break
		}
		for _, c := range cons {
			if c.Status == "connected" && c.Name != virt {
				phys = append(phys, c.Name)
			}
		}
		sort.Strings(phys)
	}
	return virt, phys, virtSrc, physSrc, nil
}

// SwitchOn disables the physical output(s), enables the virtual one, and sets
// the mode. The hdrRequested parameter is logged but not applied — KDE HDR
// toggling is handled via user settings or external tools (kscreen-doctor
// has no mode-bundled HDR argument).
func SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error {
	virt, phys, virtSrc, physSrc, err := resolveOutputs(outs)
	if err != nil {
		return err
	}

	info("switch on: requested %dx%d@%d hdr=%t", width, height, fps, hdrRequested)
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connectors: %v (%s)", phys, physSrc)
	logSunshineInputs()

	if hdrRequested {
		info("HDR requested — logged only; kscreen-doctor does not toggle HDR from the command line. Configure HDR in KDE Display Settings if needed.")
	}

	match := MatchMode(cfg.Modes, width, height, fps)
	switch {
	case match.Exact:
		info("mode match: %s (exact)", match)
	case match.ExactResolution:
		info("mode match: %s (snapped refresh: requested %d Hz, Δ%d Hz)", match, fps, match.DeltaRefresh)
	default:
		info("mode match: %s (no resolution hit — closest overall, ΔW=%d ΔH=%d ΔR=%d)",
			match, match.DeltaWidth, match.DeltaHeight, match.DeltaRefresh)
		warn("requested %dx%d@%d has no configured resolution; using %s. Add a [[modes]] entry if this is a supported target.",
			width, height, fps, match)
	}

	var args []string
	for _, p := range phys {
		args = append(args, "output."+p+".disable")
	}
	args = append(args,
		"output."+virt+".enable",
		"output."+virt+".mode."+match.String(),
	)
	info("applying switch atomically")
	if err := runKScreen(args...); err != nil {
		warn("atomic switch failed: %v", err)
		info("retrying in steps with a 2s delay before mode-set")
		for _, p := range phys {
			if err := runKScreen("output." + p + ".disable"); err != nil {
				errLog("retry step (disable physical %s) failed: %v", p, err)
				return err
			}
		}
		if err := runKScreen("output." + virt + ".enable"); err != nil {
			errLog("retry step (enable virtual) failed: %v", err)
			return err
		}
		time.Sleep(2 * time.Second)
		if err := runKScreen("output." + virt + ".mode." + match.String()); err != nil {
			errLog("retry step (mode set) failed: %v", err)
			return err
		}
	}

	info("switch complete: active=%s mode=%s", virt, match)
	if err := logReadback(virt); err != nil {
		warn("could not read back display state: %v", err)
	}
	return nil
}

// SwitchOff disables the virtual output and re-enables the physical one(s).
func SwitchOff(outs Outputs) error {
	virt, phys, virtSrc, physSrc, err := resolveOutputs(outs)
	if err != nil {
		return err
	}
	info("switch off: restoring physical display(s)")
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connectors: %v (%s)", phys, physSrc)

	args := []string{"output." + virt + ".disable"}
	for _, p := range phys {
		args = append(args, "output."+p+".enable")
	}
	if err := runKScreen(args...); err != nil {
		errLog("switch off failed: %v", err)
		return err
	}
	info("switch off complete: %s disabled, physical(s) %v re-enabled", virt, phys)
	for _, p := range phys {
		if err := logReadback(p); err != nil {
			warn("could not read back display state for %s: %v", p, err)
		}
	}
	return nil
}

// logSunshineInputs echoes any SUNSHINE_CLIENT_* env vars so users can see
// what Sunshine actually handed to the Do command. Missing values are
// reported as <unset>.
func logSunshineInputs() {
	keys := []string{
		"SUNSHINE_CLIENT_WIDTH",
		"SUNSHINE_CLIENT_HEIGHT",
		"SUNSHINE_CLIENT_FPS",
		"SUNSHINE_CLIENT_HDR",
	}
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			debug("sunshine env %s=<unset>", k)
			continue
		}
		info("sunshine env %s=%s", k, v)
	}
}

// logReadback fetches `kscreen-doctor -o` and prints the section for the
// given connector so users can visually confirm the switch took effect.
func logReadback(connector string) error {
	out, err := kscreenOutputs()
	if err != nil {
		return err
	}
	section := extractConnectorSection(out, connector)
	if section == "" {
		warn("connector %s not found in kscreen-doctor -o output", connector)
		debug("full kscreen-doctor -o output:\n%s", out)
		return nil
	}
	info("current state of %s (kscreen-doctor -o):", connector)
	for _, line := range strings.Split(strings.TrimRight(section, "\n"), "\n") {
		fmt.Fprintf(os.Stderr, "    %s\n", line)
	}
	return nil
}

// extractConnectorSection pulls the block of lines describing the named
// connector out of `kscreen-doctor -o` output. Each output is introduced by
// an "Output:" header followed by indented detail lines; this returns from
// the matching header up to (but not including) the next "Output:" header.
func extractConnectorSection(full, connector string) string {
	lines := strings.Split(full, "\n")
	var buf strings.Builder
	capture := false
	for _, ln := range lines {
		fields := strings.Fields(ln)
		isHeader := len(fields) > 0 && fields[0] == "Output:"
		if isHeader {
			if capture {
				break
			}
			for _, f := range fields[1:] {
				if f == connector {
					capture = true
					break
				}
			}
		}
		if capture {
			buf.WriteString(ln)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}
