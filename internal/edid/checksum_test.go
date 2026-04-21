package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksum(t *testing.T) {
	// Known EDID header bytes — checksum of zero block is 0
	assert.Equal(t, byte(0), Checksum(make([]byte, 127)))
	// sum = 1, checksum = 255
	assert.Equal(t, byte(255), Checksum([]byte{1}))
	// sum = 256, checksum = 0
	assert.Equal(t, byte(0), Checksum([]byte{128, 128}))
}

func TestEncodeManufacturerID(t *testing.T) {
	// "VRT" = V(22) R(18) T(20)
	// val = (22<<10) | (18<<5) | 20 = 22528 | 576 | 20 = 23124 = 0x5A54
	got := EncodeManufacturerID("VRT")
	assert.Equal(t, []byte{0x5A, 0x54}, got)
}
