package generate

import (
	"fmt"
	"strings"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// WriteAddCustomModesScript produces the add_custom_modes.sh bash script.
func WriteAddCustomModesScript(r *Result) string {
	var sb strings.Builder
	sb.WriteString(`#!/usr/bin/env bash
#
# add_custom_modes.sh
# Adds display modes whose pixel clock exceeds the EDID DTD limit
# (>655 MHz) — e.g. 4K@144Hz, ultrawide@144Hz, MacBook native@120Hz.
#
# ============================================================
# X11 ONLY — does NOT work under Wayland.
# ============================================================
# Bazzite Desktop ships KDE Plasma on Wayland by default, where
# this script has no effect: xrandr only talks to the X server's
# RandR extension, and Wayland compositors (kwin_wayland) ignore
# that state. XWayland will run the commands without error but
# no new modes will appear in kscreen-doctor, System Settings,
# or Sunshine's mode list.
#
# To use this script you must log into an X11 Plasma session
# (select "Plasma (X11)" at the SDDM login screen) before running
# it. On a stock Wayland-only setup these high-bandwidth modes
# are not reachable via xrandr — rely on the VIC-coded modes in
# the EDID (e.g. 4K@120Hz is VIC 118) instead.
#
# These modes ARE covered by the EDID range limits, but since they
# can't be encoded as DTDs, the GPU driver won't auto-create them.
#
# Usage:  chmod +x add_custom_modes.sh && ./add_custom_modes.sh [OUTPUT]
#
# Adjust OUTPUT to match your virtual display connector:
OUTPUT="${1:-HDMI-A-1}"

if [ -n "${WAYLAND_DISPLAY:-}" ] || [ "${XDG_SESSION_TYPE:-}" = "wayland" ]; then
  echo "WARNING: Wayland session detected (WAYLAND_DISPLAY=${WAYLAND_DISPLAY:-unset}, XDG_SESSION_TYPE=${XDG_SESSION_TYPE:-unset})." >&2
  echo "         xrandr mode additions will not take effect under Wayland." >&2
  echo "         Log into an X11 Plasma session (select 'Plasma (X11)' at SDDM) and re-run." >&2
  exit 1
fi

set -euo pipefail

`)
	for _, hm := range r.HighModes {
		line, name := XRandrModeline(hm.Timing)
		fmt.Fprintf(&sb, "# %dx%d@%dHz — %s\n", hm.Timing.HActive, hm.Timing.VActive, hm.Timing.Refresh, hm.Mode.Description)
		fmt.Fprintf(&sb, "if ! xrandr --query | grep -q \"%s\"; then\n", name)
		fmt.Fprintf(&sb, "  %s\n", line)
		fmt.Fprintf(&sb, "  xrandr --addmode \"$OUTPUT\" \"%s\"\n", name)
		sb.WriteString("fi\n\n")
	}
	sb.WriteString(`echo "✓ Custom modes added to $OUTPUT"
echo "  Available modes:"
xrandr --output "$OUTPUT" --query 2>/dev/null | head -30
`)
	return sb.String()
}

// WriteSunshineCommands produces the sunshine_commands.txt reference file.
func WriteSunshineCommands(cfg *config.Config) string {
	var sb strings.Builder
	sb.WriteString(`# Sunshine Do/Undo Command Reference
# ===================================
#
# The recommended way to configure Sunshine is with "sunbeams switch".
# It reads SUNSHINE_CLIENT_WIDTH/HEIGHT/FPS/HDR automatically.
#
# Sunshine global prep commands:
#   Do command:   sunbeams switch on
#   Undo command: sunbeams switch off
#
# Override resolution:  sunbeams switch on --width 3840 --height 2160 --fps 120
# List known devices:   sunbeams devices
#
# Set VIRTUAL_OUTPUT / PHYSICAL_OUTPUT env vars to override connector
# defaults (HDMI-A-1 / DP-1).
#
# The individual kscreen-doctor commands are listed below for reference.
# Replace HDMI-A-1 with your actual virtual display output.
# Replace DP-1 with your actual physical display output.

`)
	for _, d := range cfg.Devices {
		mode := fmt.Sprintf("%dx%d@%d", d.Width, d.Height, d.MaxRefresh)
		fmt.Fprintf(&sb, "# --- %s (%s) ---\n", d.Label, d.Slug)
		sb.WriteString("# Do:\n")
		fmt.Fprintf(&sb, "/usr/bin/kscreen-doctor output.DP-1.disable && /usr/bin/kscreen-doctor output.HDMI-A-1.enable && /usr/bin/kscreen-doctor output.HDMI-A-1.mode.%s\n", mode)
		sb.WriteString("# Undo:\n")
		sb.WriteString("/usr/bin/kscreen-doctor output.HDMI-A-1.disable && /usr/bin/kscreen-doctor output.DP-1.enable\n\n")
	}
	return sb.String()
}
