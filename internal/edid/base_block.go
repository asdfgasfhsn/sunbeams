package edid

import (
	"encoding/binary"
	"fmt"
)

// BaseBlockParams holds all parameters needed to assemble a 128-byte base EDID 1.4 block.
type BaseBlockParams struct {
	ManufacturerID string
	ProductCode    uint16
	Serial         uint32
	Week           byte
	Year           int
	DTDs           [4][]byte
	StandardTiming [8][]byte
	NumExtensions  byte
}

// BuildBaseBlock assembles a 128-byte EDID 1.4 base block.
func BuildBaseBlock(p BaseBlockParams) ([]byte, error) {
	b := make([]byte, BlockSize)

	// Header
	copy(b[0:8], []byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00})

	// Manufacturer / Product
	copy(b[8:10], EncodeManufacturerID(p.ManufacturerID))
	binary.LittleEndian.PutUint16(b[10:12], p.ProductCode)
	binary.LittleEndian.PutUint32(b[12:16], p.Serial)
	b[16] = p.Week
	if p.Year < 1990 {
		return nil, fmt.Errorf("year %d too old", p.Year)
	}
	b[17] = byte(p.Year - 1990)

	// EDID version 1.4
	b[18] = 1
	b[19] = 4

	// Basic display parameters
	// Byte 20: Digital input, 10 bpc, undefined interface (virtual)
	b[20] = 0b10100000
	// Image size: 0,0 = variable/projector
	b[21] = 0
	b[22] = 0
	// Gamma 2.20: (2.20 * 100) - 100 = 120
	b[23] = 120

	// Feature support: sRGB default, preferred timing in DTD1, continuous freq
	b[24] = 0b00000111

	// Chromaticity coordinates (sRGB primaries)
	b[25] = 0xEE
	b[26] = 0x91
	b[27] = 0xA3
	b[28] = 0x54
	b[29] = 0x4C
	b[30] = 0x99
	b[31] = 0x26
	b[32] = 0x0F
	b[33] = 0x50
	b[34] = 0x54

	// Established timings (none — we use DTDs instead)
	b[35], b[36], b[37] = 0, 0, 0

	// Standard timings (8 × 2 bytes, starting at offset 38)
	for i, st := range p.StandardTiming {
		if len(st) != 2 {
			return nil, fmt.Errorf("standard timing %d must be 2 bytes", i)
		}
		b[38+i*2] = st[0]
		b[39+i*2] = st[1]
	}

	// Detailed Timing Descriptors (4 × 18 bytes, starting at offset 54)
	for i, dtd := range p.DTDs {
		if len(dtd) != DTDSize {
			return nil, fmt.Errorf("DTD %d must be %d bytes", i, DTDSize)
		}
		copy(b[54+i*DTDSize:54+(i+1)*DTDSize], dtd)
	}

	// Extension count
	b[126] = p.NumExtensions
	// Checksum
	b[127] = Checksum(b[:127])
	return b, nil
}
