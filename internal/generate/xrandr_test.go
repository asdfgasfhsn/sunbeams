package generate

import (
	"testing"

	"github.com/asdfgasfhsn/sunbeams/internal/edid"
	"github.com/stretchr/testify/assert"
)

func TestXRandrModeline(t *testing.T) {
	tm := edid.CVTRBTiming(3840, 2160, 120, true)
	line, name := XRandrModeline(tm)
	assert.Equal(t, "3840x2160_120", name)
	// Sanity: starts with xrandr --newmode and includes the name
	assert.Contains(t, line, "xrandr --newmode \"3840x2160_120\"")
	assert.Contains(t, line, "+hsync -vsync")
	// Parity check: exact value from Python reference
	assert.Equal(t, `xrandr --newmode "3840x2160_120" 1024.06 3840 3848 3880 3920 2160 2163 2171 2177 +hsync -vsync`, line)
}
