package edid

import "fmt"

// Checksum returns (256 - sum(b) % 256) % 256.
func Checksum(b []byte) byte {
	var s int
	for _, x := range b {
		s += int(x)
	}
	return byte((256 - s%256) % 256)
}

// EncodeManufacturerID encodes a 3-letter uppercase PNP ID into 2 big-endian bytes.
func EncodeManufacturerID(code string) []byte {
	if len(code) != 3 {
		panic(fmt.Sprintf("manufacturer id must be 3 letters, got %q", code))
	}
	for _, c := range code {
		if c < 'A' || c > 'Z' {
			panic(fmt.Sprintf("manufacturer id must be uppercase ASCII, got %q", code))
		}
	}
	val := (uint16(code[0]-64) << 10) | (uint16(code[1]-64) << 5) | uint16(code[2]-64)
	return []byte{byte(val >> 8), byte(val & 0xFF)}
}
