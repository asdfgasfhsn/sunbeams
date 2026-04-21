package switcher

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// Outputs names the connectors involved. Empty fields fall back to env
// VIRTUAL_OUTPUT / PHYSICAL_OUTPUT and finally to HDMI-A-1 / DP-1.
type Outputs struct {
	Virtual  string
	Physical string
}

// resolve returns the final virtual/physical connector names along with a
// human-readable source tag for each ("flag", "env:VIRTUAL_OUTPUT", "default").
func (o Outputs) resolve() (virt, phys, virtSrc, physSrc string) {
	virt, virtSrc = o.Virtual, "flag"
	if virt == "" {
		if v := os.Getenv("VIRTUAL_OUTPUT"); v != "" {
			virt, virtSrc = v, "env:VIRTUAL_OUTPUT"
		} else {
			virt, virtSrc = "HDMI-A-1", "default"
		}
	}
	phys, physSrc = o.Physical, "flag"
	if phys == "" {
		if v := os.Getenv("PHYSICAL_OUTPUT"); v != "" {
			phys, physSrc = v, "env:PHYSICAL_OUTPUT"
		} else {
			phys, physSrc = "DP-1", "default"
		}
	}
	return
}

// SwitchOn disables the physical output, enables the virtual one, and sets
// the mode. The hdrRequested parameter is logged but not applied — KDE HDR
// toggling is handled via user settings or external tools (kscreen-doctor
// has no mode-bundled HDR argument).
func SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error {
	virt, phys, virtSrc, physSrc := outs.resolve()

	info("switch on: requested %dx%d@%d hdr=%t", width, height, fps, hdrRequested)
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connector: %s (%s)", phys, physSrc)
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

	args := []string{
		"output." + phys + ".disable",
		"output." + virt + ".enable",
		"output." + virt + ".mode." + match.String(),
	}
	info("applying switch atomically")
	if err := runKScreen(args...); err != nil {
		warn("atomic switch failed: %v", err)
		info("retrying in three steps with a 2s delay before mode-set")
		if err := runKScreen("output." + phys + ".disable"); err != nil {
			errLog("retry step 1 (disable physical) failed: %v", err)
			return err
		}
		if err := runKScreen("output." + virt + ".enable"); err != nil {
			errLog("retry step 2 (enable virtual) failed: %v", err)
			return err
		}
		time.Sleep(2 * time.Second)
		if err := runKScreen("output." + virt + ".mode." + match.String()); err != nil {
			errLog("retry step 3 (mode set) failed: %v", err)
			return err
		}
	}

	info("switch complete: active=%s mode=%s", virt, match)
	if err := logReadback(virt); err != nil {
		warn("could not read back display state: %v", err)
	}
	return nil
}

// SwitchOff disables the virtual output and re-enables the physical one.
func SwitchOff(outs Outputs) error {
	virt, phys, virtSrc, physSrc := outs.resolve()
	info("switch off: restoring physical display")
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connector: %s (%s)", phys, physSrc)

	if err := runKScreen(
		"output."+virt+".disable",
		"output."+phys+".enable",
	); err != nil {
		errLog("switch off failed: %v", err)
		return err
	}
	info("switch off complete: %s disabled, %s re-enabled", virt, phys)
	if err := logReadback(phys); err != nil {
		warn("could not read back display state: %v", err)
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
