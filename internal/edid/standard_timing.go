package edid

import "math"

// GuessAspectCode returns the EDID standard-timing aspect ratio code.
// 0=16:10, 1=4:3, 2=5:4, 3=16:9 (default fallback).
func GuessAspectCode(h, v int) int {
	ratio := float64(h) / float64(v)
	switch {
	case math.Abs(ratio-16.0/10.0) < 0.05:
		return 0
	case math.Abs(ratio-4.0/3.0) < 0.05:
		return 1
	case math.Abs(ratio-5.0/4.0) < 0.05:
		return 2
	default:
		return 3
	}
}

// BuildStandardTiming returns the 2-byte standard timing identifier,
// or {0x01, 0x01} if the parameters are out of range.
func BuildStandardTiming(hPixels, aspectCode, refreshHz int) []byte {
	if hPixels < 256 || hPixels > 2288 || refreshHz < 60 {
		return []byte{0x01, 0x01}
	}
	b0 := (hPixels / 8) - 31
	b1 := (aspectCode << 6) | ((refreshHz - 60) & 0x3F)
	return []byte{byte(b0 & 0xFF), byte(b1 & 0xFF)}
}
