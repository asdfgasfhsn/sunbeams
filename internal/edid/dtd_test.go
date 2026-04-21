package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDTD_3840x2160_60(t *testing.T) {
	tm := CVTRBTiming(3840, 2160, 60, true)
	got, err := BuildDTD(tm, 0, 0)
	assert.NoError(t, err)
	assert.Len(t, got, DTDSize)

	// Pixel clock 512030 kHz / 10 = 51203 = 0xC803 → little-endian: [0]=0x03, [1]=0xC8
	assert.Equal(t, byte(0x03), got[0])
	assert.Equal(t, byte(0xC8), got[1])
	// h_active=3840 → low 0x00, high nibble 0xF
	// h_blank=80 → low 0x50, high nibble 0x0
	assert.Equal(t, byte(0x00), got[2])
	assert.Equal(t, byte(0x50), got[3])
	assert.Equal(t, byte(0xF0), got[4])

	// Flags byte 17: 0x18 | 0x02 = 0x1A
	assert.Equal(t, byte(0x1A), got[17])
}

func TestBuildDTD_OverPixelClockErrors(t *testing.T) {
	tm := CVTRBTiming(3840, 2160, 120, true) // over limit
	_, err := BuildDTD(tm, 0, 0)
	assert.Error(t, err)
}

func TestBuildDTD_BytesMatchPython(t *testing.T) {
	tm := CVTRBTiming(3840, 2160, 60, true)
	got, err := BuildDTD(tm, 0, 0)
	assert.NoError(t, err)
	// Full 18-byte array from Python reference:
	// python3 -c "from generate_edid import cvt_rb_timing, build_dtd; t = cvt_rb_timing(3840, 2160, 60, True); print(' '.join(f'0x{b:02x}' for b in build_dtd(t)))"
	// => 0x03 0xc8 0x00 0x50 0xf0 0x70 0x11 0x80 0x08 0x20 0x38 0x00 0x00 0x00 0x00 0x00 0x00 0x1a
	want := []byte{
		0x03, 0xc8, // pixel clock 51203 × 10 kHz = 512030 kHz, little-endian
		0x00, // h_active low = 3840 & 0xFF = 0x00
		0x50, // h_blank low  = 80   & 0xFF = 0x50
		0xf0, // h_active high nibble | h_blank high nibble = 0xF<<4 | 0x0
		0x70, // v_active low = 2160 & 0xFF = 0x70
		0x11, // v_blank low  = 17   & 0xFF = 0x11
		0x80, // v_active high nibble | v_blank high nibble = 0x8<<4 | 0x0
		0x08, // h_front low  = 8
		0x20, // h_sync  low  = 32
		0x38, // (v_front & 0x0F)<<4 | (v_sync & 0x0F) = (3<<4)|8 = 0x38
		0x00, // upper bits of h_front, h_sync, v_front, v_sync (all fit in low 8)
		0x00, // h_image_mm low
		0x00, // v_image_mm low
		0x00, // image size high nibbles
		0x00, // h_border
		0x00, // v_border
		0x1a, // flags: digital separate, +hsync, -vsync
	}
	assert.Equal(t, want, got)
}
