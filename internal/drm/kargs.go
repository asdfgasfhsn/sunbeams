package drm

import (
	"fmt"
	"strings"
)

const (
	FirmwareDir = "/etc/firmware"
	EDIDName    = "edid.bin"
)

// BuildKargs returns the kernel arguments required to load the EDID.
func BuildKargs(firmwareDir, output, edidName string) []string {
	return []string{
		"firmware_class.path=" + firmwareDir,
		fmt.Sprintf("drm.edid_firmware=%s:%s", output, edidName),
		fmt.Sprintf("video=%s:e", output),
	}
}

// ParseSunbeamsKargs scans a kernel command line and returns the exact karg
// tokens that sunbeams installed: drm.edid_firmware entries for our EDID file,
// the matching video=<conn>:e tokens, and (on a full wipe) the shared
// firmware_class.path token. When connector is non-empty, results are narrowed
// to that connector and firmware_class.path is excluded.
func ParseSunbeamsKargs(cmdline, connector string) []string {
	const edidPrefix = "drm.edid_firmware="
	const videoPrefix = "video="
	fwToken := "firmware_class.path=" + FirmwareDir

	tokens := strings.Fields(cmdline)

	// First pass: connectors that carry a sunbeams EDID injection.
	sunbeamsConn := map[string]bool{}
	for _, tok := range tokens {
		if !strings.HasPrefix(tok, edidPrefix) {
			continue
		}
		for _, pair := range strings.Split(strings.TrimPrefix(tok, edidPrefix), ",") {
			conn, file, ok := strings.Cut(pair, ":")
			if ok && file == EDIDName {
				sunbeamsConn[conn] = true
			}
		}
	}

	var out []string
	for _, tok := range tokens {
		switch {
		case strings.HasPrefix(tok, edidPrefix):
			// When connector is non-empty and the token is the merged form
			// (multiple connectors in one token), the whole token is returned —
			// callers cannot surgically delete one connector from a merged token,
			// so a full wipe (connector == "") is required in that situation.
			for _, pair := range strings.Split(strings.TrimPrefix(tok, edidPrefix), ",") {
				conn, file, ok := strings.Cut(pair, ":")
				if ok && file == EDIDName && (connector == "" || conn == connector) {
					out = append(out, tok)
					break
				}
			}
		case strings.HasPrefix(tok, videoPrefix):
			conn, mode, ok := strings.Cut(strings.TrimPrefix(tok, videoPrefix), ":")
			if ok && mode == "e" && sunbeamsConn[conn] && (connector == "" || conn == connector) {
				out = append(out, tok)
			}
		case tok == fwToken && connector == "":
			out = append(out, tok)
		}
	}
	return out
}

// ConnectorsFromKargs extracts connector names from drm.edid_firmware tokens
// (handles the merged comma form), de-duplicated, in first-seen order.
func ConnectorsFromKargs(kargs []string) []string {
	const edidPrefix = "drm.edid_firmware="
	seen := map[string]bool{}
	var out []string
	for _, tok := range kargs {
		if !strings.HasPrefix(tok, edidPrefix) {
			continue
		}
		for _, pair := range strings.Split(strings.TrimPrefix(tok, edidPrefix), ",") {
			conn, file, ok := strings.Cut(pair, ":")
			if ok && file == EDIDName && !seen[conn] {
				seen[conn] = true
				out = append(out, conn)
			}
		}
	}
	return out
}
