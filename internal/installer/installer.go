package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

const (
	FirmwareDir  = "/etc/firmware"
	EDIDName     = "edid.bin"
	HelperPath   = "/usr/local/sbin/sunbeams-drm-force"
	SudoersPath  = "/etc/sudoers.d/sunbeams-drm-switch"
	GamescopeRel = ".config/gamescope/modes.cfg"
)

// GamingChoice is the tri-state for the gaming-mode block.
type GamingChoice int

const (
	GamingAsk GamingChoice = iota
	GamingYes
	GamingNo
)

// Options bundles all inputs into Run.
type Options struct {
	EDIDBytes   []byte
	ModesScript []byte
	MonitorName string // from cfg.EDID.MonitorName; used for modes.cfg seed
	Stdin       io.Reader
	Stdout      io.Writer

	Gaming            GamingChoice
	PhysicalConnector string // empty + GamingAsk -> prompt
}

// Run drives the interactive installer.
func Run(opts Options) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must run as root (sudo)")
	}

	stdin := opts.Stdin
	stdout := opts.Stdout

	// 1. Install EDID
	if err := os.MkdirAll(FirmwareDir, 0o755); err != nil {
		return err
	}
	edidPath := filepath.Join(FirmwareDir, EDIDName)
	if err := os.WriteFile(edidPath, opts.EDIDBytes, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ Installed EDID to %s (%d bytes)\n", edidPath, len(opts.EDIDBytes)) //nolint:errcheck

	// 2. Scan connectors
	cons, err := ScanConnectors()
	if err != nil {
		return fmt.Errorf("scan connectors: %w", err)
	}
	if len(cons) == 0 {
		return fmt.Errorf("no HDMI/DP connectors found")
	}
	fmt.Fprintln(stdout)                       //nolint:errcheck
	fmt.Fprintln(stdout, "Available outputs:") //nolint:errcheck
	for i, c := range cons {
		marker := ""
		if c.Status == "disconnected" {
			marker = " (recommended)"
		}
		fmt.Fprintf(stdout, "  [%d] %s — %s%s\n", i+1, c.Name, c.Status, marker) //nolint:errcheck
	}

	// 3. Prompt for virtual selection
	r := bufio.NewReader(stdin)
	fmt.Fprint(stdout, "\nSelect output for virtual display [1-", len(cons), "]: ") //nolint:errcheck
	line, _ := r.ReadString('\n')
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(cons) {
		return fmt.Errorf("invalid selection")
	}
	virtual := cons[idx-1].Name

	// 4. Inject kargs
	kargs := BuildKargs(FirmwareDir, virtual, EDIDName)
	if err := InjectKargs(kargs); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "✓ Kernel args added") //nolint:errcheck

	// 5. Optional xrandr user service (legacy X11 path)
	if len(opts.ModesScript) > 0 {
		fmt.Fprint(stdout, "Install systemd user service to re-add xrandr modes at login? [y/N]: ") //nolint:errcheck
		line, _ = r.ReadString('\n')
		if len(line) > 0 && (line[0] == 'y' || line[0] == 'Y') {
			if err := installXrandrUserService(stdout, opts.ModesScript, virtual); err != nil {
				return err
			}
		}
	}

	// 6. Gaming-mode block
	if err := runGamingBlock(opts, cons, r, stdout); err != nil {
		return err
	}

	return nil
}

func runGamingBlock(opts Options, cons []Connector, r *bufio.Reader, stdout io.Writer) error {
	// Decide whether to install gaming-mode artifacts.
	want := false
	switch opts.Gaming {
	case GamingYes:
		want = true
	case GamingNo:
		return nil
	case GamingAsk:
		fmt.Fprint(stdout, "\nSet up gaming mode (gamescope) support? [y/N]: ") //nolint:errcheck
		line, _ := r.ReadString('\n')
		want = len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	}
	if !want {
		return nil
	}

	// Resolve physical connector.
	physical := opts.PhysicalConnector
	if physical == "" {
		fmt.Fprintln(stdout, "\nSelect PHYSICAL output (will be force-disabled during streaming):") //nolint:errcheck
		for i, c := range cons {
			marker := ""
			if c.Status == "connected" {
				marker = " (connected)"
			}
			fmt.Fprintf(stdout, "  [%d] %s%s\n", i+1, c.Name, marker) //nolint:errcheck
		}
		fmt.Fprint(stdout, "Select [1-", len(cons), "]: ") //nolint:errcheck
		line, _ := r.ReadString('\n')
		var idx int
		if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(cons) {
			return fmt.Errorf("invalid physical selection")
		}
		physical = cons[idx-1].Name
	}

	// Pre-flight debugfs.
	pre := PreflightDebugfsPath(physical)
	if pre.Err != nil {
		return pre.Err
	}
	if pre.Count == 0 {
		fmt.Fprintf(stdout, "  ! debugfs path for %s not found yet (may populate after reboot)\n", physical) //nolint:errcheck
	} else {
		fmt.Fprintf(stdout, "  ✓ debugfs path: %s\n", pre.Paths[0]) //nolint:errcheck
	}

	// Install helper.
	if err := InstallHelper(HelperPath, HelperScript()); err != nil {
		return fmt.Errorf("install helper: %w", err)
	}
	fmt.Fprintf(stdout, "✓ Installed helper to %s (mode 0700)\n", HelperPath) //nolint:errcheck

	// Resolve real user.
	realUser := os.Getenv("SUDO_USER")
	if realUser == "" {
		return fmt.Errorf("cannot determine real user — set SUDO_USER or run without sudo")
	}
	u, err := user.Lookup(realUser)
	if err != nil {
		return fmt.Errorf("lookup user %q: %w", realUser, err)
	}

	// Install sudoers (visudo-validated).
	if err := InstallSudoers(SudoersPath, realUser, HelperPath); err != nil {
		return fmt.Errorf("install sudoers: %w", err)
	}
	fmt.Fprintf(stdout, "✓ Installed sudoers fragment %s (mode 0440)\n", SudoersPath) //nolint:errcheck

	// Seed modes.cfg.
	if opts.MonitorName == "" {
		fmt.Fprintln(stdout, "  ! cfg.EDID.MonitorName is empty; skipping modes.cfg seed") //nolint:errcheck
	} else {
		modesCfg := filepath.Join(u.HomeDir, GamescopeRel)
		if err := SeedModesCfg(modesCfg, opts.MonitorName, 1920, 1080, 60); err != nil {
			return fmt.Errorf("seed modes.cfg: %w", err)
		}
		// chown so the user can edit/replace the file.
		if err := chownFromUser(modesCfg, u); err != nil {
			fmt.Fprintf(stdout, "  ! could not chown %s to %s: %v\n", modesCfg, realUser, err) //nolint:errcheck
		}
		fmt.Fprintf(stdout, "✓ Seeded %s with %s:1920x1080@60\n", modesCfg, opts.MonitorName) //nolint:errcheck
	}

	fmt.Fprintln(stdout, "\nGaming mode setup complete.")                          //nolint:errcheck
	fmt.Fprintln(stdout, "Use these Sunshine Do/Undo commands:")                   //nolint:errcheck
	fmt.Fprintln(stdout, "  Do:   sunbeams switch on  --physical "+physical+" \\") //nolint:errcheck
	fmt.Fprintln(stdout, "          --width $SUNSHINE_CLIENT_WIDTH \\")            //nolint:errcheck
	fmt.Fprintln(stdout, "          --height $SUNSHINE_CLIENT_HEIGHT \\")          //nolint:errcheck
	fmt.Fprintln(stdout, "          --fps $SUNSHINE_CLIENT_FPS")                   //nolint:errcheck
	fmt.Fprintln(stdout, "  Undo: sunbeams switch off --physical "+physical)       //nolint:errcheck
	return nil
}

func installXrandrUserService(stdout io.Writer, script []byte, virtual string) error {
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
	customized := bytes.ReplaceAll(script, []byte("HDMI-A-1"), []byte(virtual))
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
	fmt.Fprintf(stdout, "✓ User service installed at %s\n", unitPath)                                                   //nolint:errcheck
	fmt.Fprintln(stdout, "  Enable after reboot with:")                                                                 //nolint:errcheck
	fmt.Fprintln(stdout, "    systemctl --user daemon-reload && systemctl --user enable virtual-display-modes.service") //nolint:errcheck
	return nil
}
