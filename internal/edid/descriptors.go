package edid

// BuildMonitorNameDescriptor returns an 18-byte monitor-name descriptor (tag 0xFC).
func BuildMonitorNameDescriptor(name string) []byte {
	b := make([]byte, DTDSize)
	b[3] = 0xFC
	if len(name) > 13 {
		name = name[:13]
	}
	for i := 0; i < len(name); i++ {
		b[5+i] = name[i]
	}
	if len(name) < 13 {
		b[5+len(name)] = 0x0A
		for i := len(name) + 1; i < 13; i++ {
			b[5+i] = 0x20
		}
	}
	return b
}

// BuildRangeLimitsDescriptor returns an 18-byte display range limits descriptor (tag 0xFD).
//
// minV and minH are capped at 255 — EDID 1.4 has no offset encoding for them.
func BuildRangeLimitsDescriptor(minV, maxV, minH, maxH, maxPixClkMHz int) []byte {
	b := make([]byte, DTDSize)
	b[3] = 0xFD

	var offsetFlags byte
	vMax := maxV
	hMax := maxH
	if maxV > 255 {
		offsetFlags |= 0x02
		vMax = maxV - 255
	}
	if maxH > 255 {
		offsetFlags |= 0x08
		hMax = maxH - 255
	}

	b[4] = offsetFlags
	b[5] = byte(min(minV, 255))
	b[6] = byte(vMax & 0xFF)
	b[7] = byte(min(minH, 255))
	b[8] = byte(hMax & 0xFF)
	b[9] = byte(min(maxPixClkMHz/10, 255))
	// Default GTF payload (matches Python)
	b[10] = 0x00
	b[11] = 0x0A
	b[12] = 0x20
	b[13] = 0x00
	b[14] = 0x00
	b[15] = 0x00
	b[16] = 0x00
	b[17] = 0x00
	return b
}
