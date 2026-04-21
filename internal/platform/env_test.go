package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWaylandEnv_Defaults(t *testing.T) {
	env := WaylandEnv(1000, nil)
	assert.Contains(t, env, "XDG_RUNTIME_DIR=/run/user/1000")
	assert.Contains(t, env, "WAYLAND_DISPLAY=wayland-0")
	assert.Contains(t, env, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus")
	assert.Contains(t, env, "QT_QPA_PLATFORM=wayland")
}

func TestWaylandEnv_KeepsExisting(t *testing.T) {
	existing := []string{"FOO=bar", "WAYLAND_DISPLAY=wayland-1"}
	env := WaylandEnv(42, existing)
	assert.Contains(t, env, "FOO=bar")
	assert.Contains(t, env, "WAYLAND_DISPLAY=wayland-1")
	// Should not override
	for _, e := range env {
		if e == "WAYLAND_DISPLAY=wayland-0" {
			t.Fatal("default overrode existing")
		}
	}
}
