package installer

import (
	"fmt"
	"os/exec"
)

// BuildKargs returns the kernel arguments required to load the EDID.
func BuildKargs(firmwareDir, output, edidName string) []string {
	return []string{
		"firmware_class.path=" + firmwareDir,
		fmt.Sprintf("drm.edid_firmware=%s:%s", output, edidName),
		fmt.Sprintf("video=%s:e", output),
	}
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
