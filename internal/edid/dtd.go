package edid

import "fmt"

// BuildDTD packs an 18-byte Detailed Timing Descriptor.
// hImageMM and vImageMM are physical dimensions in mm (0 = unknown).
func BuildDTD(t Timing, hImageMM, vImageMM int) ([]byte, error) {
	pc := t.PixelClockKHz / 10
	if pc > 0xFFFF {
		return nil, fmt.Errorf("pixel clock %.2f MHz exceeds DTD max (655.35 MHz) for %dx%d@%dHz",
			float64(t.PixelClockKHz)/1000, t.HActive, t.VActive, t.Refresh)
	}

	b := make([]byte, DTDSize)
	b[0] = byte(pc & 0xFF)
	b[1] = byte((pc >> 8) & 0xFF)

	b[2] = byte(t.HActive & 0xFF)
	b[3] = byte(t.HBlank & 0xFF)
	b[4] = byte(((t.HActive>>8)&0x0F)<<4) | byte((t.HBlank>>8)&0x0F)

	b[5] = byte(t.VActive & 0xFF)
	b[6] = byte(t.VBlank & 0xFF)
	b[7] = byte(((t.VActive>>8)&0x0F)<<4) | byte((t.VBlank>>8)&0x0F)

	b[8] = byte(t.HFront & 0xFF)
	b[9] = byte(t.HSync & 0xFF)
	b[10] = byte((t.VFront&0x0F)<<4) | byte(t.VSync&0x0F)
	b[11] = byte(((t.HFront>>8)&0x03)<<6) |
		byte(((t.HSync>>8)&0x03)<<4) |
		byte(((t.VFront>>4)&0x03)<<2) |
		byte((t.VSync>>4)&0x03)

	b[12] = byte(hImageMM & 0xFF)
	b[13] = byte(vImageMM & 0xFF)
	b[14] = byte(((hImageMM>>8)&0x0F)<<4) | byte((vImageMM>>8)&0x0F)

	b[15] = 0
	b[16] = 0
	b[17] = 0x18 | 0x02 // digital separate, +hsync, -vsync

	return b, nil
}
