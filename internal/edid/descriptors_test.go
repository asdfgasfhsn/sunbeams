package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMonitorNameDescriptor(t *testing.T) {
	got := BuildMonitorNameDescriptor("VirtStream")
	assert.Len(t, got, DTDSize)
	assert.Equal(t, []byte{0, 0, 0}, got[:3])
	assert.Equal(t, byte(0xFC), got[3])
	assert.Equal(t, byte('V'), got[5])
	assert.Equal(t, byte('m'), got[14])
	assert.Equal(t, byte(0x0A), got[15])
	assert.Equal(t, byte(0x20), got[16])
	assert.Equal(t, byte(0x20), got[17])
}

func TestRangeLimitsDescriptor(t *testing.T) {
	got := BuildRangeLimitsDescriptor(24, 300, 15, 400, 1700)
	assert.Len(t, got, DTDSize)
	assert.Equal(t, byte(0xFD), got[3])
	assert.Equal(t, byte(0x02|0x08), got[4])
	assert.Equal(t, byte(24), got[5])
	assert.Equal(t, byte(300-255), got[6])
	assert.Equal(t, byte(15), got[7])
	assert.Equal(t, byte(400-255), got[8])
	assert.Equal(t, byte(1700/10), got[9])
	assert.Equal(t, byte(0x00), got[10])
	assert.Equal(t, byte(0x0A), got[11])
	assert.Equal(t, byte(0x20), got[12])
}

func TestMonitorNameDescriptor_BytesMatchPython(t *testing.T) {
	got := BuildMonitorNameDescriptor("VirtStream")
	want := []byte{0x00, 0x00, 0x00, 0xfc, 0x00, 0x56, 0x69, 0x72, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x0a, 0x20, 0x20}
	assert.Equal(t, want, got)
}

func TestRangeLimitsDescriptor_BytesMatchPython(t *testing.T) {
	got := BuildRangeLimitsDescriptor(24, 300, 15, 400, 1700)
	want := []byte{0x00, 0x00, 0x00, 0xfd, 0x0a, 0x18, 0x2d, 0x0f, 0x91, 0xaa, 0x00, 0x0a, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00}
	assert.Equal(t, want, got)
}
