package switcher

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)

func withStubs(t *testing.T, virt string, virtErr error, cons []drm.Connector) {
	t.Helper()
	origDetect, origScan := detectVirtual, scanConnectors
	detectVirtual = func() (string, error) { return virt, virtErr }
	scanConnectors = func() ([]drm.Connector, error) { return cons, nil }
	t.Cleanup(func() { detectVirtual, scanConnectors = origDetect, origScan })
}

func TestResolveOutputs_AutoVirtualAndPhysical(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	t.Setenv("PHYSICAL_OUTPUT", "")
	withStubs(t, "DP-2", nil, []drm.Connector{
		{Name: "DP-2", Status: "disconnected"},
		{Name: "HDMI-A-1", Status: "connected"},
		{Name: "DP-1", Status: "connected"},
	})
	virt, phys, vsrc, psrc, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-2", virt)
	assert.Equal(t, "auto", vsrc)
	sort.Strings(phys)
	assert.Equal(t, []string{"DP-1", "HDMI-A-1"}, phys) // connected, non-virtual
	assert.Equal(t, "auto", psrc)
}

func TestResolveOutputs_EnvOverridesAuto(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "DP-3")
	t.Setenv("PHYSICAL_OUTPUT", "HDMI-A-2")
	withStubs(t, "DP-2", nil, []drm.Connector{{Name: "DP-1", Status: "connected"}})
	virt, phys, vsrc, psrc, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-3", virt)
	assert.Equal(t, "env:VIRTUAL_OUTPUT", vsrc)
	assert.Equal(t, []string{"HDMI-A-2"}, phys)
	assert.Equal(t, "env:PHYSICAL_OUTPUT", psrc)
}

func TestResolveOutputs_Headless(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	t.Setenv("PHYSICAL_OUTPUT", "")
	withStubs(t, "DP-2", nil, []drm.Connector{
		{Name: "DP-2", Status: "disconnected"},
	})
	virt, phys, _, _, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-2", virt)
	assert.Empty(t, phys) // nothing connected → skip disable
}

func TestResolveOutputs_DetectErrorPropagates(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	withStubs(t, "", assert.AnError, nil)
	_, _, _, _, err := resolveOutputs(Outputs{})
	require.Error(t, err)
}
