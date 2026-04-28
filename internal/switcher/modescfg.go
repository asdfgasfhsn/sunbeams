package switcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// osReadFile is a small indirection so tests in this package can call it
// without importing os themselves.
var osReadFile = os.ReadFile

// findMonitorMode returns the width/height/refresh recorded for the named
// monitor, or have=false if the monitor is not present. The match is exact
// (the part of the line before the first colon must equal `monitor`).
func findMonitorMode(body []byte, monitor string) (w, h, r int, have bool) {
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		if string(line[:idx]) != monitor {
			continue
		}
		var ww, hh, rr int
		if _, err := fmt.Sscanf(string(line[idx+1:]), "%dx%d@%d", &ww, &hh, &rr); err == nil {
			return ww, hh, rr, true
		}
	}
	return 0, 0, 0, false
}

// upsertMonitorMode returns a new file body with the line for `monitor`
// updated to W x H @ R. If the monitor's line is absent, the new line is
// appended. Other lines are preserved verbatim.
func upsertMonitorMode(body []byte, monitor string, w, h, r int) []byte {
	newLine := fmt.Sprintf("%s:%dx%d@%d", monitor, w, h, r)

	// Normalize: ensure trailing newline so split/rejoin round-trips.
	if len(body) > 0 && body[len(body)-1] != '\n' {
		body = append(body, '\n')
	}

	lines := bytes.Split(body, []byte{'\n'})
	// bytes.Split with a trailing newline produces a final empty element.
	// We'll strip and re-add at the end.
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

// WriteModesCfgAtomic writes body to path via temp+rename. Before the first
// modification of an existing file, it copies the existing content to
// path+".bak". The .bak is never overwritten once created — it preserves
// the first-known-good file content.
func WriteModesCfgAtomic(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create modes.cfg dir: %w", err)
	}
	if existing, err := os.ReadFile(path); err == nil {
		bak := path + ".bak"
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			if err := os.WriteFile(bak, existing, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", bak, err)
			}
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".modes.cfg.tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}
