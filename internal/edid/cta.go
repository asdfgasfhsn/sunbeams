package edid

// CTAVideoDataBlock builds a CTA-861 Video Data Block (tag code 2).
// VIC codes 1-127 are emitted as-is; codes 128+ are ignored (matches Python behavior in practice).
func CTAVideoDataBlock(vics []int) []byte {
	payload := make([]byte, 0, len(vics))
	for _, v := range vics {
		if v >= 1 && v <= 127 {
			payload = append(payload, byte(v&0x7F))
		}
	}
	if len(payload) > 31 {
		payload = payload[:31]
	}
	header := byte((0x02 << 5) | (len(payload) & 0x1F))
	return append([]byte{header}, payload...)
}

// CTAHDMIVSDB builds the HDMI Vendor-Specific Data Block (OUI 00-0C-03).
func CTAHDMIVSDB(maxTMDSMHz int) []byte {
	tmds := maxTMDSMHz / 5
	if tmds > 255 {
		tmds = 255
	}
	payload := []byte{
		0x03, 0x0C, 0x00,
		0x10, 0x00,
		0b01111000,
		byte(tmds),
	}
	header := byte((0x03 << 5) | (len(payload) & 0x1F))
	return append([]byte{header}, payload...)
}

// CTAHFVSDB builds the HDMI Forum Vendor-Specific Data Block (OUI C4-5D-D8).
func CTAHFVSDB(maxFRLRate int) []byte {
	payload := []byte{
		0xD8, 0x5D, 0xC4,
		0x01,
		0x00,
		byte(0b11000000 | (maxFRLRate & 0x07)),
		byte(0b00010000),
	}
	header := byte((0x03 << 5) | (len(payload) & 0x1F))
	return append([]byte{header}, payload...)
}

// CTAHDRStaticMetadata builds the HDR Static Metadata Data Block (extended tag 6).
func CTAHDRStaticMetadata() []byte {
	payload := []byte{0x0F, 0x01, 105, 90, 20}
	length := 1 + len(payload)
	header := byte((0x07 << 5) | (length & 0x1F))
	return append([]byte{header, 0x06}, payload...)
}

// CTAColorimetry builds the Colorimetry Data Block (extended tag 5).
func CTAColorimetry() []byte {
	payload := []byte{0b11100111, 0b11000000}
	length := 1 + len(payload)
	header := byte((0x07 << 5) | (length & 0x1F))
	return append([]byte{header, 0x05}, payload...)
}

// CTAVCDB builds the Video Capability Data Block (extended tag 0).
func CTAVCDB() []byte {
	payload := []byte{0b01000101}
	length := 1 + len(payload)
	header := byte((0x07 << 5) | (length & 0x1F))
	return append([]byte{header, 0x00}, payload...)
}

// CTAY420CMDB builds the YCbCr 4:2:0 Capability Map Data Block (extended tag 0x0F).
// The bitmap flags which VICs in the preceding VDB also support YCbCr 4:2:0 sampling.
// VICs listed in y420VICs that are not present in vdbVICs are silently skipped.
func CTAY420CMDB(vdbVICs []int, y420VICs []int) []byte {
	position := make(map[int]int, len(vdbVICs))
	for i, v := range vdbVICs {
		position[v] = i
	}
	// Compute bitmap width: enough bytes to cover the largest flagged position.
	maxBit := -1
	for _, v := range y420VICs {
		if p, ok := position[v]; ok && p > maxBit {
			maxBit = p
		}
	}
	if maxBit < 0 {
		// No matching VICs; emit an empty bitmap (payload = ext tag only).
		header := byte((0x07 << 5) | 1)
		return []byte{header, 0x0F}
	}
	bitmap := make([]byte, (maxBit/8)+1)
	for _, v := range y420VICs {
		if p, ok := position[v]; ok {
			bitmap[p/8] |= 1 << (p % 8)
		}
	}
	payloadLen := 1 + len(bitmap)
	header := byte((0x07 << 5) | (payloadLen & 0x1F))
	out := make([]byte, 0, 1+payloadLen)
	out = append(out, header, 0x0F)
	out = append(out, bitmap...)
	return out
}
