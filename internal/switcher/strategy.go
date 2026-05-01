package switcher

import (
	"fmt"
	"os"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// Strategy is the display-switching backend. Implementations differ in HOW
// they enable/disable outputs (kscreen-doctor vs debugfs force) but share
// the same SwitchOn/SwitchOff contract.
type Strategy interface {
	Name() string
	SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error
	SwitchOff(cfg *config.Config, outs Outputs) error
}

// Options bundles strategy-specific knobs that can't fit in the shared
// SwitchOn/SwitchOff signature. Each strategy uses only the fields it cares
// about and ignores the rest.
type Options struct {
	// SafeRevert, when true, makes GamescopeStrategy.SwitchOff rewrite the
	// virtual monitor's modes.cfg line to a low-risk safe mode before
	// re-enabling the physical connector. KScreenStrategy ignores this.
	SafeRevert bool
}

// Select resolves a strategy name into a constructed Strategy.
//
// Precedence: an explicit name ("kscreen" or "debugfs") wins. The literal
// "auto" defers to $SUNBEAMS_STRATEGY (if non-empty), and if that is also
// empty defers to $GAMESCOPE_WAYLAND_DISPLAY (debugfs if set, kscreen
// otherwise).
func Select(name string, opts Options) (Strategy, error) {
	resolved := name
	if resolved == "auto" {
		if env := os.Getenv("SUNBEAMS_STRATEGY"); env != "" {
			resolved = env
		} else if os.Getenv("GAMESCOPE_WAYLAND_DISPLAY") != "" {
			resolved = "debugfs"
		} else {
			resolved = "kscreen"
		}
	}
	switch resolved {
	case "kscreen":
		return &KScreenStrategy{}, nil
	case "debugfs":
		return &GamescopeStrategy{Opts: opts}, nil
	default:
		return nil, fmt.Errorf("unknown strategy %q (want auto|kscreen|debugfs)", resolved)
	}
}

// KScreenStrategy drives kscreen-doctor under KDE Plasma/Wayland. Stateless;
// ignores Options.SafeRevert (only meaningful for the gamescope strategy).
type KScreenStrategy struct{}

func (*KScreenStrategy) Name() string { return "kscreen" }
func (*KScreenStrategy) SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error {
	return switchOnKScreen(cfg, outs, w, h, fps, hdr)
}
func (*KScreenStrategy) SwitchOff(_ *config.Config, outs Outputs) error {
	return switchOffKScreen(outs)
}
