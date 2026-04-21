package edid

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBaseBlock(t *testing.T) {
	tm := CVTRBTiming(3840, 2160, 60, true)
	dtd1, _ := BuildDTD(tm, 0, 0)
	dtd2 := BuildMonitorNameDescriptor("VirtStream")
	dtd3 := BuildRangeLimitsDescriptor(24, 300, 15, 400, 1700)
	dtd4 := dtd2 // placeholder — repeat name is fine for this unit test

	st := make([][]byte, 8)
	for i := range st {
		st[i] = []byte{0x01, 0x01}
	}

	got, err := BuildBaseBlock(BaseBlockParams{
		ManufacturerID: "VRT",
		ProductCode:    0x2025,
		Serial:         1,
		Week:           1,
		Year:           2025,
		DTDs:           [4][]byte{dtd1, dtd2, dtd3, dtd4},
		StandardTiming: [8][]byte{st[0], st[1], st[2], st[3], st[4], st[5], st[6], st[7]},
		NumExtensions:  5,
	})
	assert.NoError(t, err)
	assert.Len(t, got, BlockSize)
	// Header
	assert.Equal(t, []byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}, got[:8])
	// Version 1.4
	assert.Equal(t, byte(1), got[18])
	assert.Equal(t, byte(4), got[19])
	assert.Equal(t, byte(5), got[126])
	// Checksum sums to 0 mod 256
	var s int
	for _, b := range got {
		s += int(b)
	}
	assert.Equal(t, 0, s%256)
}

func TestBuildBaseBlock_MatchesGoldenFirst128(t *testing.T) {
	// Reproduce the exact base block the Python generator produces.
	// Base DTDs: 3840x2160@60 (first), 3440x1440@60 (second)
	dtd1, err := BuildDTD(CVTRBTiming(3840, 2160, 60, true), 0, 0)
	require.NoError(t, err)
	dtd2, err := BuildDTD(CVTRBTiming(3440, 1440, 60, true), 0, 0)
	require.NoError(t, err)
	name := BuildMonitorNameDescriptor("VirtStream")
	rng := BuildRangeLimitsDescriptor(24, 300, 15, 400, 1700)

	// Standard timings: same 8 that generate_edid.py uses
	stdDefs := [][3]int{
		{1920, 1080, 60}, {1920, 1080, 120},
		{2560, 1440, 60}, {2560, 1440, 120},
		{1920, 1200, 60}, {1280, 720, 60},
		{1600, 900, 60}, {1680, 1050, 60},
	}
	var std [8][]byte
	for i, def := range stdDefs {
		std[i] = BuildStandardTiming(def[0], GuessAspectCode(def[0], def[1]), def[2])
	}

	got, err := BuildBaseBlock(BaseBlockParams{
		ManufacturerID: "VRT",
		ProductCode:    0x2025,
		Serial:         1,
		Week:           1,
		Year:           2025,
		DTDs:           [4][]byte{dtd1, dtd2, name, rng},
		StandardTiming: std,
		NumExtensions:  5,
	})
	require.NoError(t, err)

	ref, err := os.ReadFile("../../testdata/virtual_display_reference.bin")
	require.NoError(t, err)
	assert.Equal(t, ref[:128], got, "base block differs from golden reference")
}
