package installer

import (
	"fmt"
	"os/exec"
	"strings"
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

// InjectKargs appends each karg using rpm-ostree if available, otherwise
// returns an error instructing the user to edit grub manually.
func InjectKargs(kargs []string) error {
	if _, err := exec.LookPath("rpm-ostree"); err == nil {
		for _, k := range kargs {
			cmd := exec.Command("rpm-ostree", "kargs", "--append-if-missing="+k)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("rpm-ostree kargs %q: %w\n%s", k, err, out)
			}
		}
		return nil
	}
	return fmt.Errorf("rpm-ostree not found — add these kargs manually to /etc/default/grub: %v", kargs)
}

// CurrentKargs returns the current kernel command line via rpm-ostree.
func CurrentKargs() (string, error) {
	if _, err := exec.LookPath("rpm-ostree"); err != nil {
		return "", fmt.Errorf("rpm-ostree not found — cannot read kernel args")
	}
	out, err := exec.Command("rpm-ostree", "kargs").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rpm-ostree kargs: %w\n%s", err, out)
	}
	return string(out), nil
}

// DeleteKargs removes each karg token via rpm-ostree --delete-if-present.
func DeleteKargs(kargs []string) error {
	if len(kargs) == 0 {
		return nil
	}
	if _, err := exec.LookPath("rpm-ostree"); err != nil {
		return fmt.Errorf("rpm-ostree not found — remove these kargs manually from /etc/default/grub: %v", kargs)
	}
	for _, k := range kargs {
		cmd := exec.Command("rpm-ostree", "kargs", "--delete-if-present="+k)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rpm-ostree kargs --delete-if-present %q: %w\n%s", k, err, out)
		}
	}
	return nil
}
