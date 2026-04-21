package edid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuessAspectCode(t *testing.T) {
	assert.Equal(t, 0, GuessAspectCode(1920, 1200)) // 16:10
	assert.Equal(t, 1, GuessAspectCode(1024, 768))  // 4:3
	assert.Equal(t, 2, GuessAspectCode(1280, 1024)) // 5:4
	assert.Equal(t, 3, GuessAspectCode(1920, 1080)) // 16:9
}

func TestBuildStandardTiming(t *testing.T) {
	// 1920x1080@60, aspect 16:9 (code 3)
	// byte0 = 1920/8 - 31 = 240 - 31 = 209 = 0xD1
	// byte1 = (3<<6) | ((60-60)&0x3F) = 0xC0
	got := BuildStandardTiming(1920, 3, 60)
	assert.Equal(t, []byte{0xD1, 0xC0}, got)

	// out-of-range returns unused marker
	assert.Equal(t, []byte{0x01, 0x01}, BuildStandardTiming(100, 0, 60))
	assert.Equal(t, []byte{0x01, 0x01}, BuildStandardTiming(3000, 0, 60))
	assert.Equal(t, []byte{0x01, 0x01}, BuildStandardTiming(1920, 0, 30))
}
