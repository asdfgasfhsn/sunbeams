package switcher

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleKscreenOutput = `Output: 1 HDMI-A-1
	enabled
	connected
	priority 1
	Modes:
	  3840x2160@60 preferred current
	  1920x1080@60
Output: 2 DP-1
	disabled
	disconnected
	Modes:
	  1920x1080@60
`

func TestExtractConnectorSection_Virtual(t *testing.T) {
	sec := extractConnectorSection(sampleKscreenOutput, "HDMI-A-1")
	assert.Contains(t, sec, "Output: 1 HDMI-A-1")
	assert.Contains(t, sec, "3840x2160@60 preferred current")
	assert.NotContains(t, sec, "Output: 2 DP-1")
}

func TestExtractConnectorSection_Physical(t *testing.T) {
	sec := extractConnectorSection(sampleKscreenOutput, "DP-1")
	assert.Contains(t, sec, "Output: 2 DP-1")
	assert.Contains(t, sec, "disconnected")
	assert.NotContains(t, sec, "HDMI-A-1")
}

func TestExtractConnectorSection_Missing(t *testing.T) {
	sec := extractConnectorSection(sampleKscreenOutput, "eDP-1")
	assert.Empty(t, strings.TrimSpace(sec))
}

func TestMatchMode_ExactFlag(t *testing.T) {
	m := MatchMode(modes(), 3840, 2160, 60)
	assert.Equal(t, "3840x2160@60", m.String())
	assert.True(t, m.Exact)
	assert.True(t, m.ExactResolution)
	assert.Equal(t, 0, m.DeltaRefresh)
}

func TestMatchMode_SnappedRefresh(t *testing.T) {
	m := MatchMode(modes(), 1920, 1080, 110)
	assert.Equal(t, "1920x1080@120", m.String())
	assert.False(t, m.Exact)
	assert.True(t, m.ExactResolution)
	assert.Equal(t, 10, m.DeltaRefresh)
}

func TestMatchMode_ClosestOverall(t *testing.T) {
	m := MatchMode(modes(), 2560, 1440, 60)
	assert.False(t, m.Exact)
	assert.False(t, m.ExactResolution)
	// DeltaWidth/Height should be populated when no resolution hit.
	assert.True(t, m.DeltaWidth > 0 || m.DeltaHeight > 0)
}
