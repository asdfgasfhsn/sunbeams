package installer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

// InstallHelper writes the embedded helper script to dst with mode 0700.
// dst's parent directory is created if missing.
func InstallHelper(dst string, script []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", dst, err)
	}
	return os.WriteFile(dst, script, 0o700)
}

// InstallSudoers writes a sudoers fragment granting `user` NOPASSWD on
// helperPath. The fragment is validated with `visudo -cf` before being
// moved into place; if validation fails, dst is not created.
func InstallSudoers(dst, user, helperPath string) error {
	if user == "" {
		return fmt.Errorf("sudoers user must not be empty")
	}
	body := []byte(fmt.Sprintf("%s ALL=(root) NOPASSWD: %s\n", user, helperPath))

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", dst, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".sunbeams-sudoers-*")
	if err != nil {
		return fmt.Errorf("create temp sudoers: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp sudoers: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp sudoers: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o440); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp sudoers: %w", err)
	}

	cmd := exec.Command("visudo", "-cf", tmpPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("visudo rejected sudoers fragment: %w (%s)", err, stderr.String())
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, dst, err)
	}
	return nil
}

// SeedModesCfg ensures a single line `<monitor>:<W>x<H>@<R>` exists in
// path's gamescope modes.cfg. The line is updated in place if present;
// appended otherwise. Other lines are preserved.
func SeedModesCfg(path, monitor string, w, h, r int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", path, err)
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	body := upsertModesCfgLine(existing, monitor, w, h, r)
	return os.WriteFile(path, body, 0o644)
}

// upsertModesCfgLine is the installer-side mirror of switcher.upsertMonitorMode.
// We duplicate the small amount of logic here to avoid an import cycle (the
// installer must not depend on the switcher package). Keep these two
// functions byte-for-byte equivalent.
func upsertModesCfgLine(body []byte, monitor string, w, h, r int) []byte {
	newLine := fmt.Sprintf("%s:%dx%d@%d", monitor, w, h, r)

	if len(body) > 0 && body[len(body)-1] != '\n' {
		body = append(body, '\n')
	}
	lines := bytes.Split(body, []byte{'\n'})
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	replaced := false
	out := make([][]byte, 0, len(lines)+1)
	for _, line := range lines {
		idx := bytes.IndexByte(line, ':')
		if idx >= 0 && string(line[:idx]) == monitor {
			out = append(out, []byte(newLine))
			replaced = true
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, []byte(newLine))
	}
	return append(bytes.Join(out, []byte{'\n'}), '\n')
}

// PreflightResult reports what we found when checking for a usable debugfs
// path at install time.
type PreflightResult struct {
	Count int
	Paths []string
	Err   error // non-nil only on an actual ambiguity (>1 match); 0 matches is permissive
}

// PreflightDebugfsPath globs /sys/kernel/debug/dri/<pci>/<connector>/force.
// Zero matches is permissive (warn) — the helper rediscovers at runtime.
// Multiple matches is an error (multi-GPU not supported in v1).
func PreflightDebugfsPath(connector string) PreflightResult {
	matches, _ := filepath.Glob(fmt.Sprintf("/sys/kernel/debug/dri/*/%s/force", connector))
	res := PreflightResult{Count: len(matches), Paths: matches}
	if len(matches) > 1 {
		res.Err = fmt.Errorf("multiple debugfs paths for %s; multi-GPU not supported in v1", connector)
	}
	return res
}

// chownFromUser changes the owner of path to the given user's UID/GID.
func chownFromUser(path string, u *user.User) error {
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	return os.Chown(path, uid, gid)
}
