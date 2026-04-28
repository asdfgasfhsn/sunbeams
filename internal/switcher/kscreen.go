package switcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/asdfgasfhsn/sunbeams/internal/platform"
)

const kscreenDoctor = "/usr/bin/kscreen-doctor"

// switchOnKScreen disables the physical output, enables the virtual one, and
// sets the mode. The hdrRequested parameter is logged but not applied —
// kscreen-doctor has no mode-bundled HDR argument.
func switchOnKScreen(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error {
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

func switchOffKScreen(outs Outputs) error {
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

// runKScreen invokes kscreen-doctor with the given args, injecting
// Wayland/Qt/D-Bus env defaults when missing. Each invocation is logged
// and failure output is captured into the returned error.
func runKScreen(args ...string) error {
	info("kscreen-doctor %s", strings.Join(args, " "))

	cmd := exec.Command(kscreenDoctor, args...)
	cmd.Env = platform.WaylandEnv(os.Getuid(), os.Environ())

	if debugEnabled() {
		logSessionEnv(cmd.Env)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if out := strings.TrimSpace(outBuf.String()); out != "" {
		for _, line := range strings.Split(out, "\n") {
			debug("kscreen stdout: %s", line)
		}
	}
	if errOut := strings.TrimSpace(errBuf.String()); errOut != "" {
		for _, line := range strings.Split(errOut, "\n") {
			if err != nil {
				errLog("kscreen stderr: %s", line)
			} else {
				debug("kscreen stderr: %s", line)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("kscreen-doctor %v: %w (stderr=%q)", args, err, strings.TrimSpace(errBuf.String()))
	}
	return nil
}

// kscreenOutputs runs `kscreen-doctor -o` and returns its stdout. Used for
// post-switch confirmation so users can see which connector is active and at
// what mode. Returns any error from the process verbatim.
func kscreenOutputs() (string, error) {
	cmd := exec.Command(kscreenDoctor, "-o")
	cmd.Env = platform.WaylandEnv(os.Getuid(), os.Environ())
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("kscreen-doctor -o: %w (stderr=%q)", err, strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), nil
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
// connector out of `kscreen-doctor -o` output.
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

// logSessionEnv prints the Wayland/D-Bus/Qt env state being passed to
// kscreen-doctor. Only called under SUNBEAMS_DEBUG=1.
func logSessionEnv(env []string) {
	keys := []string{
		"XDG_RUNTIME_DIR",
		"WAYLAND_DISPLAY",
		"DBUS_SESSION_BUS_ADDRESS",
		"XDG_SESSION_TYPE",
		"QT_QPA_PLATFORM",
		"DISPLAY",
	}
	seen := map[string]string{}
	for _, e := range env {
		for _, k := range keys {
			if strings.HasPrefix(e, k+"=") {
				seen[k] = strings.TrimPrefix(e, k+"=")
			}
		}
	}
	for _, k := range keys {
		v, ok := seen[k]
		if !ok {
			debug("env %s=<unset>", k)
			continue
		}
		debug("env %s=%s", k, v)
	}
}
