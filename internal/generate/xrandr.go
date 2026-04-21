package generate

import (
	"fmt"

	"github.com/asdfgasfhsn/sunbeams/internal/edid"
)

// XRandrModeline returns the `xrandr --newmode` line and the mode name.
func XRandrModeline(t edid.Timing) (string, string) {
	name := fmt.Sprintf("%dx%d_%d", t.HActive, t.VActive, t.Refresh)
	clkMHz := float64(t.PixelClockKHz) / 1000.0
	hSyncEnd := t.HActive + t.HFront + t.HSync
	vSyncEnd := t.VActive + t.VFront + t.VSync
	line := fmt.Sprintf(
		`xrandr --newmode "%s" %.2f %d %d %d %d %d %d %d %d +hsync -vsync`,
		name, clkMHz,
		t.HActive, t.HActive+t.HFront, hSyncEnd, t.HTotal,
		t.VActive, t.VActive+t.VFront, vSyncEnd, t.VTotal,
	)
	return line, name
}
