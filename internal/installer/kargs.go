package installer

import (
	"fmt"
	"os/exec"
)

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
