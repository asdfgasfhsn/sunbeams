package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/asdfgasfhsn/sunbeams/internal/edid"
	"github.com/asdfgasfhsn/sunbeams/internal/generate"
	"github.com/asdfgasfhsn/sunbeams/internal/installer"
	"github.com/asdfgasfhsn/sunbeams/internal/switcher"
	"github.com/asdfgasfhsn/sunbeams/internal/userconfig"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, topLevelHelp)
		os.Exit(1)
	}
	cmd := os.Args[1]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		fmt.Print(topLevelHelp)
		return
	}
	switch cmd {
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "switch":
		if err := runSwitch(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "devices":
		if wantsHelp(os.Args[2:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["devices"], nil)
			return
		}
		if err := runDevices(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "modes":
		if wantsHelp(os.Args[2:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["modes"], nil)
			return
		}
		if err := runModes(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "config":
		if err := runConfig(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "install":
		if wantsHelp(os.Args[2:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["install"], nil)
			return
		}
		if err := runInstall(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := runUninstall(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "version":
		if wantsHelp(os.Args[2:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["version"], nil)
			return
		}
		fmt.Printf("sunbeams %s (commit %s, built %s)\n", version, commit, date)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, topLevelHelp)
		os.Exit(1)
	}
}

// loadConfig resolves the user config file and returns the merged config.
// If overridePath is empty, the default ~/.config/sunbeams/config.toml is used.
// A missing config file is not an error.
func loadConfig(overridePath string) (*config.Config, error) {
	if overridePath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			overridePath = filepath.Join(home, ".config", "sunbeams", "config.toml")
		}
	}
	return userconfig.LoadWithOverride(overridePath)
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	outputDir := fs.String("output-dir", ".", "Output directory")
	fs.StringVar(outputDir, "o", ".", "Output directory (short)")
	cfgPath := fs.String("config", "", "Config file path (default ~/.config/sunbeams/config.toml)")
	fs.StringVar(cfgPath, "c", "", "Config file path (short)")
	noScripts := fs.Bool("no-scripts", false, "Skip helper script generation")
	validate := fs.Bool("validate", false, "Run edid-decode on the output if available")
	help := subcommandHelps["generate"]
	fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
	if wantsHelp(args) {
		renderSubcommandHelp(os.Stdout, help, fs)
		return nil
	}
	_ = fs.Parse(args)

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	result, err := generate.Generate(cfg)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	edidPath := filepath.Join(*outputDir, "virtual_display.bin")
	if err := os.WriteFile(edidPath, result.EDIDBytes, 0o644); err != nil {
		return fmt.Errorf("write EDID: %w", err)
	}
	fmt.Printf("✓ EDID written to %s (%d bytes, %d blocks)\n",
		edidPath, len(result.EDIDBytes), len(result.EDIDBytes)/128)

	if !*noScripts {
		if len(result.HighModes) > 0 {
			p := filepath.Join(*outputDir, "add_custom_modes.sh")
			if err := os.WriteFile(p, []byte(generate.WriteAddCustomModesScript(result)), 0o755); err != nil {
				return fmt.Errorf("write script: %w", err)
			}
			fmt.Printf("✓ xrandr helper written to %s (%d modes)\n", p, len(result.HighModes))
		}
		p := filepath.Join(*outputDir, "sunshine_commands.txt")
		if err := os.WriteFile(p, []byte(generate.WriteSunshineCommands(cfg)), 0o644); err != nil {
			return fmt.Errorf("write sunshine commands: %w", err)
		}
		fmt.Printf("✓ Sunshine reference written to %s\n", p)
	}

	if *validate {
		path, err := exec.LookPath("edid-decode")
		if err != nil {
			fmt.Println("edid-decode not on PATH; skipping validation")
			return nil
		}
		cmd := exec.Command(path, edidPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("edid-decode: %w", err)
		}
	}

	return nil
}

func runSwitch(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, topLevelSwitchHelp)
		os.Exit(1)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(topLevelSwitchHelp)
		return nil
	}

	switch args[0] {
	case "off":
		if wantsHelp(args[1:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["switch-off"], nil)
			return nil
		}
		return switcher.SwitchOff(switcher.Outputs{})
	case "on":
		fs := flag.NewFlagSet("switch on", flag.ExitOnError)
		width := fs.Int("width", envInt("SUNSHINE_CLIENT_WIDTH"), "client width")
		height := fs.Int("height", envInt("SUNSHINE_CLIENT_HEIGHT"), "client height")
		fps := fs.Int("fps", envInt("SUNSHINE_CLIENT_FPS"), "client fps")
		hdrFlag := fs.Bool("hdr", false, "force HDR on")
		noHDR := fs.Bool("no-hdr", false, "force HDR off")
		cfgPath := fs.String("config", "", "Config file path (default ~/.config/sunbeams/config.toml)")
		fs.StringVar(cfgPath, "c", "", "Config file path (short)")
		help := subcommandHelps["switch-on"]
		fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
		if wantsHelp(args[1:]) {
			renderSubcommandHelp(os.Stdout, help, fs)
			return nil
		}
		_ = fs.Parse(args[1:])

		cfg, err := loadConfig(*cfgPath)
		if err != nil {
			return err
		}
		if *width == 0 || *height == 0 || *fps == 0 {
			return fmt.Errorf("missing width/height/fps (pass flags or set SUNSHINE_CLIENT_*)")
		}
		hdr := os.Getenv("SUNSHINE_CLIENT_HDR") == "true"
		if *hdrFlag {
			hdr = true
		}
		if *noHDR {
			hdr = false
		}
		return switcher.SwitchOn(cfg, switcher.Outputs{}, *width, *height, *fps, hdr)
	default:
		return fmt.Errorf("unknown switch subcommand: %s (expected on|off)", args[0])
	}
}

func envInt(key string) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

func runDevices() error {
	cfg, err := loadConfig("")
	if err != nil {
		return err
	}
	maxSlug, maxLabel := 0, 0
	for _, d := range cfg.Devices {
		if len(d.Slug) > maxSlug {
			maxSlug = len(d.Slug)
		}
		if len(d.Label) > maxLabel {
			maxLabel = len(d.Label)
		}
	}
	fmt.Println("Available devices:")
	fmt.Println()
	for _, d := range cfg.Devices {
		fmt.Printf("  %-*s  %-*s  %dx%d@%d\n",
			maxSlug, d.Slug, maxLabel, d.Label, d.Width, d.Height, d.MaxRefresh)
	}
	return nil
}

func runModes() error {
	cfg, err := loadConfig("")
	if err != nil {
		return err
	}
	fmt.Println("All configured modes:")
	for _, m := range cfg.Modes {
		t := edid.CVTRBTiming(m.Width, m.Height, m.Refresh, true)
		marker := "DTD"
		if t.PixelClockKHz > edid.MaxDTDPixClkKHz {
			marker = "xrandr"
		}
		fmt.Printf("  %4dx%-4d @%3dHz  %8.2f MHz  [%6s]  %s\n",
			m.Width, m.Height, m.Refresh,
			float64(t.PixelClockKHz)/1000.0, marker, m.Description)
	}
	return nil
}

func runConfig(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, topLevelConfigHelp)
		os.Exit(1)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(topLevelConfigHelp)
		return nil
	}
	switch args[0] {
	case "init":
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".config", "sunbeams")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(dir, "config.toml")
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("%s already exists — remove it first", dst)
		}
		if err := os.WriteFile(dst, config.DefaultsTOML(), 0o644); err != nil {
			return err
		}
		fmt.Printf("✓ wrote %s\n", dst)
		return nil
	case "show":
		fs := flag.NewFlagSet("config show", flag.ExitOnError)
		cfgPath := fs.String("config", "", "Config file path (default ~/.config/sunbeams/config.toml)")
		fs.StringVar(cfgPath, "c", "", "Config file path (short)")
		help := subcommandHelps["config-show"]
		fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
		if wantsHelp(args[1:]) {
			renderSubcommandHelp(os.Stdout, help, fs)
			return nil
		}
		_ = fs.Parse(args[1:])
		cfg, err := loadConfig(*cfgPath)
		if err != nil {
			return err
		}
		enc := toml.NewEncoder(os.Stdout)
		return enc.Encode(cfg)
	default:
		return fmt.Errorf("unknown config subcommand: %s (expected init|show)", args[0])
	}
}

func runInstall() error {
	cfg, err := loadConfig("")
	if err != nil {
		return err
	}
	result, err := generate.Generate(cfg)
	if err != nil {
		return err
	}
	var modesScript []byte
	if len(result.HighModes) > 0 {
		modesScript = []byte(generate.WriteAddCustomModesScript(result))
	}
	return installer.Run(result.EDIDBytes, modesScript, os.Stdin, os.Stdout)
}

func runUninstall(args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	connector := fs.String("connector", "", "Only remove kargs for this connector (default: all)")
	yes := fs.Bool("yes", false, "Remove everything detected without prompting")
	fs.BoolVar(yes, "y", false, "Remove everything detected without prompting (short)")
	help := subcommandHelps["uninstall"]
	fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
	if wantsHelp(args) {
		renderSubcommandHelp(os.Stdout, help, fs)
		return nil
	}
	_ = fs.Parse(args)
	return installer.Uninstall(*connector, *yes, os.Stdin, os.Stdout)
}

func runStatus(args []string) error {
	if wantsHelp(args) {
		renderSubcommandHelp(os.Stdout, subcommandHelps["status"], nil)
		return nil
	}
	fwPath := filepath.Join(installer.FirmwareDir, installer.EDIDName)
	rep, err := installer.Status("/sys/class/drm", "/proc/cmdline", fwPath)
	if errors.Is(err, installer.ErrNoSysfs) {
		fmt.Println("display status is only available on Linux with DRM/KMS")
		return nil
	}
	if err != nil {
		return err
	}

	if rep.FirmwarePresent {
		fmt.Printf("EDID injection status   (firmware: %s, %d bytes)\n\n", fwPath, rep.FirmwareBytes)
	} else {
		fmt.Printf("EDID injection status   (firmware: %s — MISSING)\n\n", fwPath)
	}

	if len(rep.Connectors) == 0 {
		fmt.Println("No sunbeams EDID injection found.")
		return nil
	}

	yn := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	fmt.Printf("  %-11s %-14s %-12s %-11s %-13s %s\n",
		"CONNECTOR", "CONNECTED", "CONFIGURED", "THIS BOOT", "EDID LOADED", "STATE")
	for _, c := range rep.Connectors {
		connected := "disconnected"
		if c.Connected {
			connected = "connected"
		}
		fmt.Printf("  %-11s %-14s %-12s %-11s %-13s %s\n",
			c.Name, connected, yn(c.Configured), yn(c.BootActive), yn(c.EDIDLoaded), c.Verdict)
	}
	fmt.Printf("\n%d connector(s) carry the sunbeams EDID.\n", len(rep.Connectors))
	if !rep.RebootDetectable {
		fmt.Println("(rpm-ostree unavailable — 'reboot pending' / 'removal staged' states could not be determined.)")
	}
	return nil
}
