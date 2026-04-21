package switcher

import (
	"testing"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/stretchr/testify/assert"
)

func modes() []config.Mode {
	return []config.Mode{
		{Width: 3840, Height: 2160, Refresh: 60},
		{Width: 3840, Height: 2160, Refresh: 120},
		{Width: 1920, Height: 1080, Refresh: 60},
		{Width: 1920, Height: 1080, Refresh: 120},
		{Width: 1920, Height: 1080, Refresh: 144},
		{Width: 960, Height: 544, Refresh: 60},
	}
}

func TestFindBestMode_ExactMatch(t *testing.T) {
	assert.Equal(t, "3840x2160@60", FindBestMode(modes(), 3840, 2160, 60))
	assert.Equal(t, "1920x1080@144", FindBestMode(modes(), 1920, 1080, 144))
}

func TestFindBestMode_ClosestRefresh(t *testing.T) {
	assert.Equal(t, "1920x1080@120", FindBestMode(modes(), 1920, 1080, 110))
	assert.Equal(t, "1920x1080@144", FindBestMode(modes(), 1920, 1080, 150))
}

func TestFindBestMode_NoResolutionMatch(t *testing.T) {
	// Asks for 2560x1440@60 — not in list. Closest-overall fallback.
	got := FindBestMode(modes(), 2560, 1440, 60)
	assert.NotEmpty(t, got)
}
