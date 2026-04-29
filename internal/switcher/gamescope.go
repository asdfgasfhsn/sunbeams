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

	// Test seams. Production code leaves these nil; the methods below
	// substitute production implementations on first use.
	runHelper    func(action, connector string) error
	modesCfgPath func(home string) string
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

// SwitchOff implementation arrives in Task 7.
func (g *GamescopeStrategy) SwitchOff(outs Outputs) error {
	return fmt.Errorf("debugfs SwitchOff not implemented yet")
}
