package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)

// Run drives the interactive installer: writes EDID to firmware,
// prompts for connector, injects kernel args, optionally installs
// a user service for xrandr custom modes.
func Run(edidBytes []byte, modesScript []byte, stdin io.Reader, stdout io.Writer) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must run as root (sudo)")
	}

	r := bufio.NewReader(stdin)

	// 0. Detect and offer to clean stale sunbeams kargs (idempotent re-install).
	if cmdline, err := CurrentKargs(); err == nil {
		if stale := drm.ParseSunbeamsKargs(cmdline, ""); len(stale) > 0 {
			fmt.Fprintln(stdout, "Existing sunbeams kernel arguments detected:") //nolint:errcheck // progress to stdout; unactionable
			for _, k := range stale {
				fmt.Fprintf(stdout, "  %s\n", k) //nolint:errcheck // progress to stdout; unactionable
			}
			fmt.Fprint(stdout, "Remove them before injecting the new connector? [Y/n]: ") //nolint:errcheck // prompt to stdout; unactionable
			line, _ := r.ReadString('\n')
			line = strings.ToLower(strings.TrimSpace(line))
			if line == "" || line == "y" || line == "yes" {
				if err := DeleteKargs(stale); err != nil {
					return err
				}
				fmt.Fprintln(stdout, "✓ Removed stale kernel arguments") //nolint:errcheck // progress to stdout; unactionable
			}
		}
	}

	// 1. Install EDID
	if err := os.MkdirAll(drm.FirmwareDir, 0o755); err != nil {
		return err
	}
	edidPath := filepath.Join(drm.FirmwareDir, drm.EDIDName)
	if err := os.WriteFile(edidPath, edidBytes, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ Installed EDID to %s (%d bytes)\n", edidPath, len(edidBytes)) //nolint:errcheck // progress message to stdout; unactionable

	// 2. Scan connectors
	cons, err := drm.ScanConnectors()
	if err != nil {
		return fmt.Errorf("scan connectors: %w", err)
	}
	if len(cons) == 0 {
		return fmt.Errorf("no HDMI/DP connectors found")
	}
	fmt.Fprintln(stdout)                       //nolint:errcheck // progress message to stdout; unactionable
	fmt.Fprintln(stdout, "Available outputs:") //nolint:errcheck // progress message to stdout; unactionable
	for i, c := range cons {
		marker := ""
		if c.Status == "disconnected" {
			marker = " (recommended)"
		}
		fmt.Fprintf(stdout, "  [%d] %s — %s%s\n", i+1, c.Name, c.Status, marker) //nolint:errcheck // progress message to stdout; unactionable
	}

	// 3. Prompt for selection
	fmt.Fprint(stdout, "\nSelect output for virtual display [1-", len(cons), "]: ") //nolint:errcheck // progress message to stdout; unactionable
	line, _ := r.ReadString('\n')
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(cons) {
		return fmt.Errorf("invalid selection")
	}
	selected := cons[idx-1]
	output := selected.Name
	if selected.Status == "connected" {
		fmt.Fprintf(stdout, "\n⚠ %s currently has a display connected. Injecting a forced EDID\n", output) //nolint:errcheck // warning to stdout; unactionable
		fmt.Fprintln(stdout, "  will override that monitor's real EDID.")                                  //nolint:errcheck // warning to stdout; unactionable
		fmt.Fprint(stdout, "Continue anyway? [y/N]: ")                                                     //nolint:errcheck // prompt to stdout; unactionable
		confirmLine, _ := r.ReadString('\n')
		confirmLine = strings.ToLower(strings.TrimSpace(confirmLine))
		if confirmLine != "y" && confirmLine != "yes" {
			return fmt.Errorf("aborted: connector %s is connected", output)
		}
	}

	// 4. Inject kargs
	kargs := drm.BuildKargs(drm.FirmwareDir, output, drm.EDIDName)
	if err := InjectKargs(kargs); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "✓ Kernel args added") //nolint:errcheck // progress message to stdout; unactionable

	// 5. Optional user service
	if len(modesScript) == 0 {
		return nil
	}
	fmt.Fprint(stdout, "Install systemd user service to re-add xrandr modes at login? [y/N]: ") //nolint:errcheck // progress message to stdout; unactionable
	line, _ = r.ReadString('\n')
	if len(line) == 0 || (line[0] != 'y' && line[0] != 'Y') {
		return nil
	}
	realUser := os.Getenv("SUDO_USER")
	if realUser == "" {
		return fmt.Errorf("cannot determine real user — set SUDO_USER or run without sudo")
	}
	u, err := user.Lookup(realUser)
	if err != nil {
		return fmt.Errorf("lookup user %q: %w", realUser, err)
	}
	home := u.HomeDir
	scriptPath := filepath.Join(home, ".local", "bin", "add-virtual-display-modes.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return err
	}
	// Replace default HDMI-A-1 with the chosen output
	customized := bytes.ReplaceAll(modesScript, []byte("HDMI-A-1"), []byte(output))
	if err := os.WriteFile(scriptPath, customized, 0o755); err != nil {
		return err
	}
	svcDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(svcDir, "virtual-display-modes.service")
	if err := os.WriteFile(unitPath, []byte(UserServiceUnit(scriptPath)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ User service installed at %s\n", unitPath)                                                   //nolint:errcheck // progress message to stdout; unactionable
	fmt.Fprintln(stdout, "  Enable after reboot with:")                                                                 //nolint:errcheck // progress message to stdout; unactionable
	fmt.Fprintln(stdout, "    systemctl --user daemon-reload && systemctl --user enable virtual-display-modes.service") //nolint:errcheck // progress message to stdout; unactionable
	return nil
}
