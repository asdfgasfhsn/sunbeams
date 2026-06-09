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

	rebootRequired := false
	removedAny := false

	// 1. Kernel args
	cmdline, err := CurrentKargs()
	if err != nil {
		return err
	}
	kargs := ParseSunbeamsKargs(cmdline, connector)
	// A merged drm.edid_firmware token references multiple connectors in one
	// karg and cannot be removed for a single connector via --delete-if-present
	// (deleting it would also drop the others). Refuse and point the user at a
	// full uninstall. This cannot arise from sunbeams' own install (which emits
	// one token per connector) but can if kargs were hand-edited.
	if connector != "" {
		for _, k := range kargs {
			if strings.HasPrefix(k, "drm.edid_firmware=") && strings.Contains(k, ",") {
				return fmt.Errorf("connector %s shares a merged kernel argument %q with other connectors "+
					"and cannot be removed alone — run a full 'sunbeams uninstall' (without --connector)", connector, k)
			}
		}
	}
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
			rebootRequired = true
			removedAny = true
		}
	} else {
		fmt.Fprintln(stdout, "No sunbeams kernel arguments found.") //nolint:errcheck // progress to stdout; unactionable
	}

	// The firmware file and user service are shared/global — only handle them
	// on a full wipe, not when narrowing to a single connector.
	if connector == "" {
		// 2. EDID firmware file. Removal failure is non-fatal: warn and continue
		// so the final summary (and any reboot notice) still prints.
		edidPath := filepath.Join(FirmwareDir, EDIDName)
		if _, statErr := os.Stat(edidPath); statErr == nil {
			if confirm(fmt.Sprintf("Remove EDID firmware file %s? [y/N]: ", edidPath)) {
				if err := os.Remove(edidPath); err != nil {
					fmt.Fprintf(stdout, "⚠ Could not remove %s: %v\n", edidPath, err) //nolint:errcheck // warning to stdout; unactionable
				} else {
					fmt.Fprintf(stdout, "✓ Removed %s\n", edidPath) //nolint:errcheck // progress to stdout; unactionable
					removedAny = true
				}
			}
		}

		// 3. systemd user service + xrandr script. Unlike installer.go (which
		// must write and so treats a missing SUDO_USER as fatal), uninstall has
		// nothing to do here without it, so we silently skip.
		if realUser := os.Getenv("SUDO_USER"); realUser != "" {
			if u, lookErr := user.Lookup(realUser); lookErr == nil {
				unitPath := filepath.Join(u.HomeDir, ".config", "systemd", "user", "virtual-display-modes.service")
				scriptPath := filepath.Join(u.HomeDir, ".local", "bin", "add-virtual-display-modes.sh")
				_, unitErr := os.Stat(unitPath)
				_, scriptErr := os.Stat(scriptPath)
				if unitErr == nil || scriptErr == nil {
					if confirm("Remove systemd user service and xrandr mode script? [y/N]: ") {
						var removed []string
						if unitErr == nil {
							if err := os.Remove(unitPath); err != nil {
								fmt.Fprintf(stdout, "⚠ Could not remove %s: %v\n", unitPath, err) //nolint:errcheck // warning to stdout; unactionable
							} else {
								removed = append(removed, "user service")
							}
						}
						if scriptErr == nil {
							if err := os.Remove(scriptPath); err != nil {
								fmt.Fprintf(stdout, "⚠ Could not remove %s: %v\n", scriptPath, err) //nolint:errcheck // warning to stdout; unactionable
							} else {
								removed = append(removed, "xrandr script")
							}
						}
						if len(removed) > 0 {
							fmt.Fprintf(stdout, "✓ Removed %s\n", strings.Join(removed, " and "))               //nolint:errcheck // progress to stdout; unactionable
							fmt.Fprintln(stdout, "  Run: systemctl --user daemon-reload (as the desktop user)") //nolint:errcheck // progress to stdout; unactionable
							removedAny = true
						}
					}
				}
			}
		}
	}

	switch {
	case rebootRequired:
		fmt.Fprintln(stdout, "\nReboot required for kernel argument changes to take effect.") //nolint:errcheck // progress to stdout; unactionable
	case removedAny:
		fmt.Fprintln(stdout, "✓ Done.") //nolint:errcheck // progress to stdout; unactionable
	default:
		fmt.Fprintln(stdout, "Nothing removed.") //nolint:errcheck // progress to stdout; unactionable
	}
	return nil
}
