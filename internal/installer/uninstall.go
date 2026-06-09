package installer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Uninstall interactively removes what install added: sunbeams kernel args,
// the EDID firmware file, and the systemd user service + xrandr script.
// When connector is non-empty, only that connector's kargs are considered
// (the shared firmware file and user service are left alone). When assumeYes
// is true, every detected item is removed without prompting.
func Uninstall(connector string, assumeYes bool, stdin io.Reader, stdout io.Writer) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstall must run as root (sudo)")
	}

	r := bufio.NewReader(stdin)
	confirm := func(prompt string) bool {
		if assumeYes {
			return true
		}
		fmt.Fprint(stdout, prompt) //nolint:errcheck // prompt to stdout; unactionable
		line, _ := r.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		return line == "y" || line == "yes"
	}

	removedAny := false

	// 1. Kernel args
	cmdline, err := CurrentKargs()
	if err != nil {
		return err
	}
	kargs := ParseSunbeamsKargs(cmdline, connector)
	if len(kargs) > 0 {
		fmt.Fprintln(stdout, "Found sunbeams kernel arguments:") //nolint:errcheck // progress to stdout; unactionable
		for _, k := range kargs {
			fmt.Fprintf(stdout, "  %s\n", k) //nolint:errcheck // progress to stdout; unactionable
		}
		if confirm("Remove these kernel arguments? [y/N]: ") {
			if err := DeleteKargs(kargs); err != nil {
				return err
			}
			fmt.Fprintln(stdout, "✓ Kernel arguments removed") //nolint:errcheck // progress to stdout; unactionable
			removedAny = true
		}
	} else {
		fmt.Fprintln(stdout, "No sunbeams kernel arguments found.") //nolint:errcheck // progress to stdout; unactionable
	}

	// The firmware file and user service are shared/global — only handle them
	// on a full wipe, not when narrowing to a single connector.
	if connector == "" {
		// 2. EDID firmware file
		edidPath := filepath.Join(FirmwareDir, EDIDName)
		if _, statErr := os.Stat(edidPath); statErr == nil {
			if confirm(fmt.Sprintf("Remove EDID firmware file %s? [y/N]: ", edidPath)) {
				if err := os.Remove(edidPath); err != nil {
					return err
				}
				fmt.Fprintf(stdout, "✓ Removed %s\n", edidPath) //nolint:errcheck // progress to stdout; unactionable
				removedAny = true
			}
		}

		// 3. systemd user service + xrandr script
		if realUser := os.Getenv("SUDO_USER"); realUser != "" {
			if u, lookErr := user.Lookup(realUser); lookErr == nil {
				unitPath := filepath.Join(u.HomeDir, ".config", "systemd", "user", "virtual-display-modes.service")
				scriptPath := filepath.Join(u.HomeDir, ".local", "bin", "add-virtual-display-modes.sh")
				_, unitErr := os.Stat(unitPath)
				_, scriptErr := os.Stat(scriptPath)
				if unitErr == nil || scriptErr == nil {
					if confirm("Remove systemd user service and xrandr mode script? [y/N]: ") {
						if unitErr == nil {
							if err := os.Remove(unitPath); err != nil {
								return err
							}
						}
						if scriptErr == nil {
							if err := os.Remove(scriptPath); err != nil {
								return err
							}
						}
						fmt.Fprintln(stdout, "✓ Removed user service and script")                           //nolint:errcheck // progress to stdout; unactionable
						fmt.Fprintln(stdout, "  Run: systemctl --user daemon-reload (as the desktop user)") //nolint:errcheck // progress to stdout; unactionable
						removedAny = true
					}
				}
			}
		} else {
			fmt.Fprintln(stdout, "Note: SUDO_USER not set — skipping user service check.") //nolint:errcheck // progress to stdout; unactionable
		}
	}

	if removedAny {
		fmt.Fprintln(stdout, "\nReboot required for kernel argument changes to take effect.") //nolint:errcheck // progress to stdout; unactionable
	} else {
		fmt.Fprintln(stdout, "Nothing removed.") //nolint:errcheck // progress to stdout; unactionable
	}
	return nil
}
