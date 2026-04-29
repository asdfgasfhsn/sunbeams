package installer

import _ "embed"

//go:embed embed/sunbeams-drm-force.sh
var helperScript []byte

// HelperScript returns the embedded helper bytes. Exposed for tests and for
// the gaming-mode install path.
func HelperScript() []byte {
	out := make([]byte, len(helperScript))
	copy(out, helperScript)
	return out
}
