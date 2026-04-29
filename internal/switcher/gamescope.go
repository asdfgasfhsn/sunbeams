package switcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// GamescopeStrategy implements the gaming-mode display switcher: it edits
// ~/.config/gamescope/modes.cfg to pick the resolution and execs a
// sudoers-gated helper to force-disable the physical connector via DRM
// debugfs.
type GamescopeStrategy struct {
	Opts Options
	cfg  *config.Config // populated by Configure() for SwitchOff path; read by SwitchOff

	// Test seams. Production code leaves these nil; the methods below
	// substitute production implementations on first use.
	runHelper    func(action, connector string) error
	modesCfgPath func(home string) string
}

// Configure passes the loaded config to the strategy so SwitchOff has access
// to cfg.Modes and cfg.Gaming for safe-revert resolution. Called by the CLI
// after Select() and before SwitchOff().
func (g *GamescopeStrategy) Configure(cfg *config.Config) {
	g.cfg = cfg
}

func (*GamescopeStrategy) Name() string { return "debugfs" }

func (g *GamescopeStrategy) SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdr bool) error {
	_, phys, _, physSrc := outs.resolve()
	monitor := cfg.EDID.MonitorName
	if monitor == "" {
		return fmt.Errorf("cfg.EDID.MonitorName is empty; cannot key modes.cfg edit")
	}

	info("switch on (debugfs): requested %dx%d@%d hdr=%t", width, height, fps, hdr)
	info("physical connector: %s (%s)", phys, physSrc)
	info("virtual monitor (EDID name): %s", monitor)
	logSunshineInputs()

	if hdr {
		info("HDR requested — gamescope handles HDR via its own --hdr-* launch flags; this strategy doesn't toggle HDR.")
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
		warn("requested %dx%d@%d has no configured resolution; using %s", width, height, fps, match)
	}

	cfgPath := g.resolveModesCfgPath(cfg.Gaming.ModesCfg)
	if err := g.upsertMode(cfgPath, monitor, match.Width, match.Height, match.Refresh); err != nil {
		return fmt.Errorf("update modes.cfg: %w", err)
	}
	info("modes.cfg updated: %s -> %s (%s)", monitor, match, cfgPath)

	if err := g.execHelper(cfg.Gaming.HelperPath, "off", phys); err != nil {
		return fmt.Errorf("force off %s: %w", phys, err)
	}
	info("debugfs force off: %s", phys)
	return nil
}

func (g *GamescopeStrategy) upsertMode(cfgPath, monitor string, w, h, r int) error {
	body, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	updated := upsertMonitorMode(body, monitor, w, h, r)
	return WriteModesCfgAtomic(cfgPath, updated)
}

func (g *GamescopeStrategy) execHelper(path, action, connector string) error {
	if g.runHelper != nil {
		return g.runHelper(action, connector)
	}
	cmd := exec.Command("sudo", "-n", path, action, connector)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g *GamescopeStrategy) resolveModesCfgPath(cfgRel string) string {
	if g.modesCfgPath != nil {
		home, _ := os.UserHomeDir()
		return g.modesCfgPath(home)
	}
	if filepath.IsAbs(cfgRel) {
		return cfgRel
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Best effort: assume relative to CWD if HOME is missing.
		return cfgRel
	}
	return filepath.Join(home, cfgRel)
}

func (g *GamescopeStrategy) SwitchOff(outs Outputs) error {
	if g.cfg == nil {
		return fmt.Errorf("debugfs SwitchOff requires Configure(cfg) before invocation")
	}
	_, phys, _, physSrc := outs.resolve()
	monitor := g.cfg.EDID.MonitorName
	if monitor == "" {
		return fmt.Errorf("cfg.EDID.MonitorName is empty; cannot key modes.cfg edit")
	}

	info("switch off (debugfs): physical=%s (%s) safe_revert=%t", phys, physSrc, g.Opts.SafeRevert)

	if g.Opts.SafeRevert {
		w, h, r := safeRevertMode(g.cfg)
		cfgPath := g.resolveModesCfgPath(g.cfg.Gaming.ModesCfg)
		if err := g.upsertMode(cfgPath, monitor, w, h, r); err != nil {
			warn("safe-revert modes.cfg edit failed: %v (continuing with force on)", err)
		} else {
			info("modes.cfg safe-reverted to %dx%d@%d", w, h, r)
		}
	}

	if err := g.execHelper(g.cfg.Gaming.HelperPath, "on", phys); err != nil {
		return fmt.Errorf("force on %s: %w", phys, err)
	}
	info("debugfs force on: %s", phys)
	return nil
}

// safeRevertMode picks the mode used by SwitchOff when --no-safe-revert is
// not set. Precedence:
//  1. cfg.Gaming.SafeRevertMode (literal "WxH@R") if non-empty.
//  2. First entry in cfg.Modes (config-file order) with W<=1920 H<=1080 R<=60.
//  3. Literal 1920x1080@60.
func safeRevertMode(cfg *config.Config) (w, h, r int) {
	if cfg.Gaming.SafeRevertMode != "" {
		var ww, hh, rr int
		if _, err := fmt.Sscanf(cfg.Gaming.SafeRevertMode, "%dx%d@%d", &ww, &hh, &rr); err == nil {
			return ww, hh, rr
		}
		warn("cfg.Gaming.safe_revert_mode %q is not WxH@R; falling back to scan", cfg.Gaming.SafeRevertMode)
	}
	for _, m := range cfg.Modes {
		if m.Width <= 1920 && m.Height <= 1080 && m.Refresh <= 60 {
			return m.Width, m.Height, m.Refresh
		}
	}
	return 1920, 1080, 60
}
