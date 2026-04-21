package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCVTRBTimingV2_3840x2160_60(t *testing.T) {
	got := CVTRBTiming(3840, 2160, 60, true)
	assert.Equal(t, 3920, got.HTotal)
	assert.Equal(t, 2177, got.VTotal)
	assert.Equal(t, 80, got.HBlank)
	assert.Equal(t, 17, got.VBlank)
	assert.Equal(t, 512030, got.PixelClockKHz)
}

func TestCVTRBTimingV2_3840x2160_120(t *testing.T) {
	got := CVTRBTiming(3840, 2160, 120, true)
	assert.Greater(t, got.PixelClockKHz, MaxDTDPixClkKHz)
}

func TestCVTRBTimingV2_1920x1080_60(t *testing.T) {
	got := CVTRBTiming(1920, 1080, 60, true)
	assert.Equal(t, 2000, got.HTotal)
	assert.Equal(t, 1097, got.VTotal)
	assert.Equal(t, 131640, got.PixelClockKHz)
}

func TestCVTRBAgainstPython(t *testing.T) {
	cases := []struct {
		w, h, r int
		pixKHz  int
	}{
		{3840, 2160, 60, 512030},
		{3840, 2160, 120, 1024061},
		{3440, 1440, 60, 307718},
		{2560, 1440, 60, 230789},
		{1920, 1080, 60, 131640},
		{1280, 720, 60, 60139},
		{960, 544, 60, 35006},
		{480, 272, 60, 9710},
	}
	for _, c := range cases {
		got := CVTRBTiming(c.w, c.h, c.r, true)
		assert.Equal(t, c.pixKHz, got.PixelClockKHz,
			"pixel clock mismatch for %dx%d@%d", c.w, c.h, c.r)
	}
}
