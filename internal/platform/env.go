package platform

import (
	"fmt"
	"strings"
)

// WaylandEnv returns an environment slice with Wayland/Qt/D-Bus defaults
// merged over the provided existing env. Existing keys take precedence.
func WaylandEnv(uid int, existing []string) []string {
	defaults := map[string]string{
		"XDG_RUNTIME_DIR":          fmt.Sprintf("/run/user/%d", uid),
		"WAYLAND_DISPLAY":          "wayland-0",
		"DBUS_SESSION_BUS_ADDRESS": fmt.Sprintf("unix:path=/run/user/%d/bus", uid),
		"QT_QPA_PLATFORM":          "wayland",
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(existing)+len(defaults))
	for _, e := range existing {
		if i := strings.IndexByte(e, '='); i > 0 {
			seen[e[:i]] = true
		}
		out = append(out, e)
	}
	for k, v := range defaults {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}
