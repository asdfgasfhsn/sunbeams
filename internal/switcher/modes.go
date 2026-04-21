package switcher

import (
	"fmt"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// MatchResult describes how a requested WxH@R was resolved against the
// configured mode list.
type MatchResult struct {
	Width, Height, Refresh int
	// Exact is true when the configured list contained the exact WxH@R.
	Exact bool
	// ExactResolution is true when WxH matched but refresh had to be snapped.
	ExactResolution bool
	// DeltaRefresh is the absolute difference between requested and chosen
	// refresh when a snap occurred (0 for exact matches).
	DeltaRefresh int
	// DeltaWidth / DeltaHeight are populated only when no resolution match
	// was available and the overall-closest fallback fired.
	DeltaWidth, DeltaHeight int
}

// String returns the kscreen-doctor mode identifier ("WxH@R").
func (m MatchResult) String() string {
	return fmt.Sprintf("%dx%d@%d", m.Width, m.Height, m.Refresh)
}

// MatchMode resolves a requested (width,height,fps) tuple against the
// configured modes and returns details about how the match was made.
func MatchMode(modes []config.Mode, width, height, fps int) MatchResult {
	// Pass 1: exact resolution, closest refresh.
	bestScore := -1
	var best config.Mode
	for _, m := range modes {
		if m.Width == width && m.Height == height {
			score := absInt(m.Refresh - fps)
			if bestScore == -1 || score < bestScore {
				bestScore = score
				best = m
			}
		}
	}
	if bestScore != -1 {
		delta := absInt(best.Refresh - fps)
		return MatchResult{
			Width:           best.Width,
			Height:          best.Height,
			Refresh:         best.Refresh,
			Exact:           delta == 0,
			ExactResolution: true,
			DeltaRefresh:    delta,
		}
	}

	// Pass 2: closest overall (weight refresh distance).
	bestScore = -1
	for _, m := range modes {
		dw := m.Width - width
		dh := m.Height - height
		dr := m.Refresh - fps
		score := dw*dw + dh*dh + dr*dr*10
		if bestScore == -1 || score < bestScore {
			bestScore = score
			best = m
		}
	}
	if bestScore == -1 {
		// No modes configured at all — echo the request back verbatim.
		return MatchResult{Width: width, Height: height, Refresh: fps}
	}
	return MatchResult{
		Width:        best.Width,
		Height:       best.Height,
		Refresh:      best.Refresh,
		DeltaRefresh: absInt(best.Refresh - fps),
		DeltaWidth:   absInt(best.Width - width),
		DeltaHeight:  absInt(best.Height - height),
	}
}

// FindBestMode returns "WxH@R" for the closest matching mode. Kept for
// callers that don't need match metadata.
func FindBestMode(modes []config.Mode, width, height, fps int) string {
	return MatchMode(modes, width, height, fps).String()
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
