package generate

import (
	"fmt"
	"math"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/asdfgasfhsn/sunbeams/internal/edid"
)

// Result is the in-memory output of the generator.
type Result struct {
	EDIDBytes  []byte
	DTDModes   []ResolvedMode // modes that fit in a DTD
	HighModes  []ResolvedMode // modes needing xrandr --newmode
	NumExtBlks int
	DTDsPlaced int
}

// ResolvedMode pairs a Mode with its computed timing.
type ResolvedMode struct {
	Mode   config.Mode
	Timing edid.Timing
}

// Generate builds the EDID binary from the given config.
//
// The config must contain a mode with width=3440, height=1440, refresh=60 —
// it is used as the second DTD in the base EDID block. If not found, an error is returned.
func Generate(cfg *config.Config) (*Result, error) {
	if len(cfg.Modes) == 0 {
		return nil, fmt.Errorf("config has no modes")
	}

	// 1. Compute timings for every mode
	var dtdModes, highModes []ResolvedMode
	for _, m := range cfg.Modes {
		t := edid.CVTRBTiming(m.Width, m.Height, m.Refresh, true)
		rm := ResolvedMode{Mode: m, Timing: t}
		if t.PixelClockKHz <= edid.MaxDTDPixClkKHz {
			dtdModes = append(dtdModes, rm)
		} else {
			highModes = append(highModes, rm)
		}
	}
	if len(dtdModes) == 0 {
		return nil, fmt.Errorf("no DTD-capable modes")
	}

	// 2. Standard timings
	var stdTimings [8][]byte
	for i := range stdTimings {
		stdTimings[i] = []byte{0x01, 0x01}
	}
	for i, s := range cfg.StandardTimings {
		if i >= 8 {
			break
		}
		aspect := edid.GuessAspectCode(s.Width, s.Height)
		stdTimings[i] = edid.BuildStandardTiming(s.Width, aspect, s.Refresh)
	}

	// 3. Base block DTDs:
	//    slot 1: first DTD-capable mode (usually 4K@60)
	//    slot 2: ultrawide@60 (3440x1440@60) — find it
	//    slot 3: monitor name
	//    slot 4: range limits
	base1, err := edid.BuildDTD(dtdModes[0].Timing, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("base DTD1: %w", err)
	}

	var uwMode *ResolvedMode
	for i := range dtdModes {
		t := dtdModes[i].Timing
		if t.HActive == 3440 && t.VActive == 1440 && t.Refresh == 60 {
			uwMode = &dtdModes[i]
			break
		}
	}
	if uwMode == nil {
		return nil, fmt.Errorf("config must contain 3440x1440@60 mode (used as base block DTD slot 2); add it to [[modes]] in your override")
	}
	base2, err := edid.BuildDTD(uwMode.Timing, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("base DTD2: %w", err)
	}

	nameDesc := edid.BuildMonitorNameDescriptor(cfg.EDID.MonitorName)
	rangeDesc := edid.BuildRangeLimitsDescriptor(
		cfg.EDID.MinVRate, cfg.EDID.MaxVRate,
		cfg.EDID.MinHRate, cfg.EDID.MaxHRate,
		cfg.EDID.MaxPixelClockMHz,
	)

	// 4. Remaining DTD modes (skip the two used in base block)
	var remaining []ResolvedMode
	for _, rm := range dtdModes {
		t := rm.Timing
		if t.HActive == 3840 && t.VActive == 2160 && t.Refresh == 60 {
			continue
		}
		if t.HActive == 3440 && t.VActive == 1440 && t.Refresh == 60 {
			continue
		}
		remaining = append(remaining, rm)
	}
	remainingDTDs := make([][]byte, 0, len(remaining))
	for _, rm := range remaining {
		d, err := edid.BuildDTD(rm.Timing, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("remaining DTD %dx%d@%d: %w",
				rm.Timing.HActive, rm.Timing.VActive, rm.Timing.Refresh, err)
		}
		remainingDTDs = append(remainingDTDs, d)
	}

	// 5. CTA data blocks in fixed tag order.
	// Y420CMDB must follow the VDB so its bitmap indexes align with VDB VIC positions.
	dataBlocks := [][]byte{
		edid.CTAVideoDataBlock(cfg.CTA.VICCodes),
		edid.CTAHDMIVSDB(cfg.EDID.MaxTMDSMHz),
		edid.CTAHFVSDB(cfg.EDID.MaxFRLRate),
		edid.CTAHDRStaticMetadata(),
		edid.CTAY420CMDB(cfg.CTA.VICCodes, cfg.CTA.Y420VICs),
		edid.CTAColorimetry(),
		edid.CTAVCDB(),
	}

	// 6. Compute extension count
	totalDBBytes := 0
	for _, db := range dataBlocks {
		totalDBBytes += len(db)
	}
	cta1DTDCapacity := (123 - totalDBBytes) / edid.DTDSize
	overflow := len(remainingDTDs) - cta1DTDCapacity
	additional := 0
	if overflow > 0 {
		additional = int(math.Ceil(float64(overflow) / float64(123/edid.DTDSize)))
	}
	numExt := 1 + additional

	// 7. Base block
	base, err := edid.BuildBaseBlock(edid.BaseBlockParams{
		ManufacturerID: cfg.EDID.ManufacturerID,
		ProductCode:    cfg.EDID.ProductCode,
		Serial:         cfg.EDID.Serial,
		Week:           cfg.EDID.Week,
		Year:           cfg.EDID.Year,
		DTDs:           [4][]byte{base1, base2, nameDesc, rangeDesc},
		StandardTiming: stdTimings,
		NumExtensions:  byte(numExt),
	})
	if err != nil {
		return nil, fmt.Errorf("base block: %w", err)
	}

	// 8. CTA extension blocks
	out := make([]byte, 0, edid.BlockSize*(1+numExt))
	out = append(out, base...)
	queue := remainingDTDs
	first, placed, err := edid.BuildCTABlock(dataBlocks, queue)
	if err != nil {
		return nil, fmt.Errorf("CTA block 1: %w", err)
	}
	out = append(out, first...)
	queue = queue[placed:]
	for len(queue) > 0 {
		blk, p, err := edid.BuildCTABlock(nil, queue)
		if err != nil {
			return nil, fmt.Errorf("CTA block overflow: %w", err)
		}
		out = append(out, blk...)
		queue = queue[p:]
	}

	return &Result{
		EDIDBytes:  out,
		DTDModes:   dtdModes,
		HighModes:  highModes,
		NumExtBlks: numExt,
		DTDsPlaced: 2 + len(remainingDTDs),
	}, nil
}
