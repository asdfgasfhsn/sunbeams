package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCTABlock_DataBlocksOnly(t *testing.T) {
	dbs := [][]byte{
		CTAVideoDataBlock([]int{97}),
		CTAColorimetry(),
	}
	got, placed, err := BuildCTABlock(dbs, nil)
	assert.NoError(t, err)
	assert.Len(t, got, BlockSize)
	assert.Equal(t, 0, placed)
	assert.Equal(t, byte(0x02), got[0])
	assert.Equal(t, byte(0x03), got[1])
	// video(len=2: header+1 VIC) + colorimetry(len=4: header+ext_tag+2 payload) = 6 bytes
	assert.Equal(t, byte(4+6), got[2])
	assert.Equal(t, byte(0b11100011), got[3])
	// Checksum valid
	var s int
	for _, b := range got {
		s += int(b)
	}
	assert.Equal(t, 0, s%256)
}

func TestBuildCTABlock_DTDsOnly(t *testing.T) {
	tm := CVTRBTiming(1920, 1080, 60, true)
	dtd, _ := BuildDTD(tm, 0, 0)
	got, placed, err := BuildCTABlock(nil, [][]byte{dtd, dtd, dtd, dtd, dtd, dtd, dtd, dtd})
	assert.NoError(t, err)
	// dtd_offset = 4 (no data blocks), 128-1-4 = 123 bytes for DTDs → 6 fit (108 bytes)
	assert.Equal(t, 6, placed)
	assert.Len(t, got, BlockSize)
}
