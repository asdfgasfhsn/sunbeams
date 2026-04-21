package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCTAVideoDataBlock(t *testing.T) {
	vics := []int{97, 118, 16, 63, 4, 47, 31, 19, 1}
	got := CTAVideoDataBlock(vics)
	// Header: (0x02<<5) | 9 = 0x49
	assert.Equal(t, byte(0x49), got[0])
	assert.Equal(t, []byte{97, 118, 16, 63, 4, 47, 31, 19, 1}, got[1:])
}

func TestCTAHDMIVSDB(t *testing.T) {
	got := CTAHDMIVSDB(600)
	assert.Equal(t, byte(0x67), got[0]) // (0x03<<5)|7
	assert.Equal(t, []byte{0x03, 0x0C, 0x00}, got[1:4])
	assert.Equal(t, byte(0x10), got[4])
	assert.Equal(t, byte(0x00), got[5])
	assert.Equal(t, byte(0b01111000), got[6])
	assert.Equal(t, byte(120), got[7]) // 600/5
}

func TestCTAHFVSDB(t *testing.T) {
	got := CTAHFVSDB(6)
	assert.Equal(t, byte(0x67), got[0])
	assert.Equal(t, []byte{0xD8, 0x5D, 0xC4}, got[1:4])
	assert.Equal(t, byte(0x01), got[4])
	assert.Equal(t, byte(0x00), got[5])
	assert.Equal(t, byte(0b11000000|6), got[6])
	assert.Equal(t, byte(0b00010000), got[7])
}

func TestCTAHDRStaticMetadata(t *testing.T) {
	got := CTAHDRStaticMetadata()
	assert.Equal(t, byte(0xE6), got[0]) // (0x07<<5)|6
	assert.Equal(t, byte(0x06), got[1])
	assert.Equal(t, byte(0x0F), got[2])
	assert.Equal(t, byte(0x01), got[3])
	assert.Equal(t, byte(105), got[4])
	assert.Equal(t, byte(90), got[5])
	assert.Equal(t, byte(20), got[6])
}

func TestCTAColorimetry(t *testing.T) {
	got := CTAColorimetry()
	assert.Equal(t, byte(0xE3), got[0])
	assert.Equal(t, byte(0x05), got[1])
	assert.Equal(t, byte(0b11100111), got[2])
	assert.Equal(t, byte(0b11000000), got[3])
}

func TestCTAVCDB(t *testing.T) {
	got := CTAVCDB()
	assert.Equal(t, byte(0xE2), got[0])
	assert.Equal(t, byte(0x00), got[1])
	assert.Equal(t, byte(0b01000101), got[2])
}

func TestCTABlocks_BytesMatchPython(t *testing.T) {
	// Python reference: cta_video_data_block([97, 118, 16, 63, 4, 47, 31, 19, 1])
	assert.Equal(t, []byte{0x49, 0x61, 0x76, 0x10, 0x3f, 0x04, 0x2f, 0x1f, 0x13, 0x01}, CTAVideoDataBlock([]int{97, 118, 16, 63, 4, 47, 31, 19, 1}))
	// Python reference: cta_hdmi_vsdb(600)
	assert.Equal(t, []byte{0x67, 0x03, 0x0c, 0x00, 0x10, 0x00, 0x78, 0x78}, CTAHDMIVSDB(600))
	// Python reference: cta_hf_vsdb(6)
	assert.Equal(t, []byte{0x67, 0xd8, 0x5d, 0xc4, 0x01, 0x00, 0xc6, 0x10}, CTAHFVSDB(6))
	// Python reference: cta_hdr_static_metadata_block()
	assert.Equal(t, []byte{0xe6, 0x06, 0x0f, 0x01, 0x69, 0x5a, 0x14}, CTAHDRStaticMetadata())
	// Python reference: cta_colorimetry_block()
	assert.Equal(t, []byte{0xe3, 0x05, 0xe7, 0xc0}, CTAColorimetry())
	// Python reference: cta_vcdb()
	assert.Equal(t, []byte{0xe2, 0x00, 0x45}, CTAVCDB())
	// CTAY420CMDB reference: VDB [97,118,117,96,95,94,93,16,63,34,33,32,31,4,47,19,1]
	// with y420 [97,118,117] → header 0xE2, ext tag 0x0F, bitmap 0x07.
	assert.Equal(t, []byte{0xe2, 0x0f, 0x07}, CTAY420CMDB(
		[]int{97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1},
		[]int{97, 118, 117},
	))
}

func TestCTAY420CMDB(t *testing.T) {
	// VDB order matches defaults.toml after Round 1.
	vdb := []int{97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1}
	y420 := []int{97, 118, 117}
	got := CTAY420CMDB(vdb, y420)

	// Extended-tag block: header (tag 7), extended tag 0x0F, 1-byte bitmap.
	// Bit positions: 97 -> 0, 118 -> 1, 117 -> 2 → 0b00000111 = 0x07.
	// Header = (0x07 << 5) | payload_length (payload includes ext tag + bitmap = 2).
	assert.Len(t, got, 3, "block length")
	assert.Equal(t, byte(0xE2), got[0], "header byte")
	assert.Equal(t, byte(0x0F), got[1], "extended tag byte")
	assert.Equal(t, byte(0x07), got[2], "bitmap byte")
}

func TestCTAY420CMDB_SkipsMissingVICs(t *testing.T) {
	// If a y420 VIC isn't present in the VDB, skip silently.
	vdb := []int{97, 16}
	y420 := []int{97, 117, 118} // 117, 118 not in VDB
	got := CTAY420CMDB(vdb, y420)

	// Only position 0 (VIC 97) gets flagged.
	// Still one bitmap byte because max flagged position = 0.
	assert.Equal(t, byte(0xE2), got[0])
	assert.Equal(t, byte(0x0F), got[1])
	assert.Equal(t, byte(0x01), got[2])
}
