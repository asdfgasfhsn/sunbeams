package switcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/asdfgasfhsn/sunbeams/internal/platform"
)

const kscreenDoctor = "/usr/bin/kscreen-doctor"

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
