package edid

import "fmt"

// BuildCTABlock packs a CTA-861 extension block: header + data blocks + as
// many DTDs as fit. Returns the block bytes, number of DTDs placed, and any error.
func BuildCTABlock(dataBlocks [][]byte, dtds [][]byte) ([]byte, int, error) {
	b := make([]byte, BlockSize)
	b[0] = 0x02
	b[1] = 0x03

	var payload []byte
	for _, db := range dataBlocks {
		payload = append(payload, db...)
	}
	dtdOffset := 4 + len(payload)
	if dtdOffset > BlockSize-1 {
		return nil, 0, fmt.Errorf("CTA data blocks (%d bytes) exceed block capacity", len(payload))
	}
	b[2] = byte(dtdOffset)
	b[3] = 0b11100011

	copy(b[4:4+len(payload)], payload)

	pos := dtdOffset
	placed := 0
	for _, dtd := range dtds {
		if pos+DTDSize > BlockSize-1 {
			break
		}
		copy(b[pos:pos+DTDSize], dtd)
		pos += DTDSize
		placed++
	}

	b[127] = Checksum(b[:127])
	return b, placed, nil
}
