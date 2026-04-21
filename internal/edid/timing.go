package edid

import "math"

// Timing holds all horizontal/vertical timing parameters and the pixel clock.
type Timing struct {
	HActive, VActive int
	HBlank, VBlank   int
	HFront, HSync    int
	HBack, VFront    int
	VSync, VBack     int
	HTotal, VTotal   int
	Refresh          int
	PixelClockKHz    int
}

// CVTRBTiming computes CVT Reduced Blanking timings. If useRB2 is false, CVT-RB v1 is used.
func CVTRBTiming(hActive, vActive, refreshHz int, useRB2 bool) Timing {
	var hBlank, hFront, hSync, hBack, vFront, vSync, vBack int
	if useRB2 {
		hBlank, hFront, hSync, hBack = CVTRB2HBlank, CVTRB2HFront, CVTRB2HSync, CVTRB2HBack
		vFront, vSync, vBack = CVTRB2VFront, CVTRB2VSync, CVTRB2VBack
	} else {
		hBlank, hFront, hSync, hBack = CVTRB1HBlank, CVTRB1HFront, CVTRB1HSync, CVTRB1HBack
		vFront, vSync = 3, 5
		computed := int(math.Ceil(460e-6*float64(refreshHz))) - vFront - vSync
		if computed > 6 {
			vBack = computed
		} else {
			vBack = 6
		}
	}

	vBlank := vFront + vSync + vBack
	hTotal := hActive + hBlank
	vTotal := vActive + vBlank
	pixClk := int(math.Round(float64(hTotal*vTotal*refreshHz) / 1000.0))

	return Timing{
		HActive: hActive, VActive: vActive,
		HBlank: hBlank, VBlank: vBlank,
		HFront: hFront, HSync: hSync, HBack: hBack,
		VFront: vFront, VSync: vSync, VBack: vBack,
		HTotal: hTotal, VTotal: vTotal,
		Refresh:       refreshHz,
		PixelClockKHz: pixClk,
	}
}
