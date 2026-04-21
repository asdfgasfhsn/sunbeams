package generate

import (
	"os"
	"testing"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoldenEDID(t *testing.T) {
	cfg, err := config.LoadDefaults()
	require.NoError(t, err)

	result, err := Generate(cfg)
	require.NoError(t, err)

	want, err := os.ReadFile("../../testdata/virtual_display_reference.bin")
	require.NoError(t, err)

	assert.Equal(t, len(want), len(result.EDIDBytes),
		"EDID length mismatch: got %d want %d", len(result.EDIDBytes), len(want))
	assert.Equal(t, want, result.EDIDBytes,
		"EDID bytes differ from Python-generated reference")
}

func TestAllBlockChecksumsValid(t *testing.T) {
	cfg, _ := config.LoadDefaults()
	result, err := Generate(cfg)
	require.NoError(t, err)
	require.True(t, len(result.EDIDBytes)%128 == 0)
	for i := 0; i < len(result.EDIDBytes); i += 128 {
		block := result.EDIDBytes[i : i+128]
		var s int
		for _, b := range block {
			s += int(b)
		}
		assert.Equal(t, 0, s%256, "block %d checksum invalid", i/128)
	}
}
