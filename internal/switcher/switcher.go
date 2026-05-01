package switcher

import (
	"os"
)

// Outputs names the connectors involved. Empty fields fall back to env
// VIRTUAL_OUTPUT / PHYSICAL_OUTPUT and finally to HDMI-A-1 / DP-1.
type Outputs struct {
	Virtual  string
	Physical string
}

// resolve returns the final virtual/physical connector names along with a
// human-readable source tag for each ("flag", "env:VIRTUAL_OUTPUT", "default").
func (o Outputs) resolve() (virt, phys, virtSrc, physSrc string) {
	virt, virtSrc = o.Virtual, "flag"
	if virt == "" {
		if v := os.Getenv("VIRTUAL_OUTPUT"); v != "" {
			virt, virtSrc = v, "env:VIRTUAL_OUTPUT"
		} else {
			virt, virtSrc = "HDMI-A-1", "default"
		}
	}
	phys, physSrc = o.Physical, "flag"
	if phys == "" {
		if v := os.Getenv("PHYSICAL_OUTPUT"); v != "" {
			phys, physSrc = v, "env:PHYSICAL_OUTPUT"
		} else {
			phys, physSrc = "DP-1", "default"
		}
	}
	return
}
