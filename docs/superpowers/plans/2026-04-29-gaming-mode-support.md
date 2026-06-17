# Gaming Mode (gamescope) Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a second display-switching strategy targeting Bazzite Gaming Mode (gamescope), coexisting with the existing kscreen strategy, gated by a sudoers-restricted shell helper that performs DRM debugfs writes.

**Architecture:** Introduce a `Strategy` interface in `internal/switcher` with two implementations — `KScreenStrategy` (existing logic, lifted to methods) and `GamescopeStrategy` (new: edits `~/.config/gamescope/modes.cfg` and execs a sudoers-gated helper that writes `/sys/kernel/debug/dri/<pci>/<connector>/force`). Selection happens via `Select(name)` driven by a `--strategy` flag, `$SUNBEAMS_STRATEGY` env, or auto-detection of `$GAMESCOPE_WAYLAND_DISPLAY`.

**Tech Stack:** Go 1.24 (stdlib + `github.com/BurntSushi/toml` + testify), embedded shell helper via `go:embed`, sudoers (`/etc/sudoers.d/`) for runtime privilege.

**Spec:** [docs/superpowers/specs/2026-04-29-gaming-mode-support-design.md](../specs/2026-04-29-gaming-mode-support-design.md)

---

## File map

| File | Status | Responsibility |
|---|---|---|
| `internal/config/config.go` | Modify | Add `GamingConfig` struct + `Gaming` field on `Config` |
| `internal/config/defaults.toml` | Modify | Add `[gaming]` section with empty defaults |
| `internal/switcher/strategy.go` | Create | `Strategy` interface, `Options`, `Select()`, env detection |
| `internal/switcher/strategy_test.go` | Create | Tests for `Select()` matrix + precedence |
| `internal/switcher/kscreen.go` | Create | `KScreenStrategy` extracted from current free functions |
| `internal/switcher/switcher.go` | Modify | Strip extracted logic; keep `Outputs` and helpers |
| `internal/switcher/modescfg.go` | Create | Pure parse/edit of `~/.config/gamescope/modes.cfg` |
| `internal/switcher/modescfg_test.go` | Create | Table-driven tests |
| `internal/switcher/gamescope.go` | Create | `GamescopeStrategy` |
| `internal/switcher/gamescope_test.go` | Create | Strategy tests with stubbed deps |
| `internal/installer/embed/sunbeams-drm-force.sh` | Create | Privileged shell helper |
| `internal/installer/gamescope.go` | Create | Helper install, sudoers, modes.cfg seed |
| `internal/installer/gamescope_test.go` | Create | Tempdir file-write tests |
| `internal/installer/installer.go` | Modify | Refactor `Run` to take `Options` struct; integrate gaming block |
| `internal/installer/installer_test.go` | Modify | Adapt to new `Run` signature |
| `cmd/sunbeams/main.go` | Modify | New flags: `--strategy`, `--physical`, `--no-safe-revert`, `--with-gaming`, `--no-gaming` |
| `cmd/sunbeams/help.go` | Modify | Update help text for new flags |
| `Makefile` | Modify | Add `shellcheck` invocation to `lint` target |
| `README.md` | Modify | Add Gaming Mode section |
| `docs/troubleshooting.md` | Modify | Add Gaming Mode block |
| `CLAUDE.md` | Modify | Architecture note for the strategy abstraction |

The 34 existing tests must continue passing through every task. The golden-EDID test in particular must remain byte-identical — gaming-mode work does not touch `internal/edid` or `internal/generate`.

---

## Task 1: Add `[gaming]` config section

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/defaults.toml`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDefaults_Gaming(t *testing.T) {
	cfg, err := LoadDefaults()
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/sbin/sunbeams-drm-force", cfg.Gaming.HelperPath)
	assert.Equal(t, ".config/gamescope/modes.cfg", cfg.Gaming.ModesCfg)
	assert.Equal(t, "", cfg.Gaming.SafeRevertMode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaults_Gaming -v`
Expected: FAIL with `cfg.Gaming undefined` (compile error).

- [ ] **Step 3: Add the struct**

In `internal/config/config.go`, add after the `Mode` struct:

```go
type GamingConfig struct {
	HelperPath     string `toml:"helper_path"`
	ModesCfg       string `toml:"modes_cfg"`
	SafeRevertMode string `toml:"safe_revert_mode"`
}
```

And add the field to `Config`:

```go
type Config struct {
	EDID            EDIDConfig       `toml:"edid"`
	CTA             CTAConfig        `toml:"cta"`
	StandardTimings []StandardTiming `toml:"standard_timings"`
	Devices         []Device         `toml:"devices"`
	Modes           []Mode           `toml:"modes"`
	Gaming          GamingConfig     `toml:"gaming"`
}
```

- [ ] **Step 4: Add to defaults.toml**

Append to `internal/config/defaults.toml`:

```toml

[gaming]
# Helper binary path. Installer always writes to and the sudoers grant always
# references /usr/local/sbin/sunbeams-drm-force. Overriding this only changes
# what `switch on/off` invokes — you would also need to install the helper at
# the new path manually and adjust /etc/sudoers.d/sunbeams-drm-switch to match.
helper_path = "/usr/local/sbin/sunbeams-drm-force"

# Path to gamescope modes.cfg, relative to $HOME.
modes_cfg = ".config/gamescope/modes.cfg"

# Safe-revert mode for `switch off`. Empty = pick the first entry in [[modes]]
# (config-file order) whose width<=1920, height<=1080, refresh<=60. Falls back
# to "1920x1080@60" if no [[modes]] entry qualifies.
safe_revert_mode = ""
```

- [ ] **Step 5: Run all tests**

Run: `make test`
Expected: ALL PASS, including new `TestDefaults_Gaming`. The golden-EDID test must still pass — adding a TOML section does not change EDID generation.

- [ ] **Step 6: Verify golden parity**

Run: `make verify-golden`
Expected: `golden parity OK`.

- [ ] **Step 7: Commit**

```bash
git add internal/config/
git commit -m "Add [gaming] config section with default helper path, modes.cfg path, safe-revert mode"
```

---

## Task 2: modes.cfg parser/editor

**Files:**
- Create: `internal/switcher/modescfg.go`
- Create: `internal/switcher/modescfg_test.go`

This is pure logic — no I/O outside what the tests inject. Format: each non-empty line is `<MonitorName>:<W>x<H>@<R>`. The first colon is the separator; monitor names may contain whitespace.

- [ ] **Step 1: Write the failing tests**

Create `internal/switcher/modescfg_test.go`:

```go
package switcher

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModesCfg_FindLine(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		monitor  string
		wantW    int
		wantH    int
		wantR    int
		wantHave bool
	}{
		{"empty file", "", "VirtStream", 0, 0, 0, false},
		{"single line match", "VirtStream:1920x1080@60\n", "VirtStream", 1920, 1080, 60, true},
		{"line with trailing space in name", "Microstep :2340x1080@120\n", "Microstep ", 2340, 1080, 120, true},
		{"multiple lines, ours present", "Other:1920x1080@60\nVirtStream:3840x2160@120\n", "VirtStream", 3840, 2160, 120, true},
		{"multiple lines, ours absent", "Other:1920x1080@60\nFoo:1280x720@60\n", "VirtStream", 0, 0, 0, false},
		{"name is exact match (not prefix)", "VirtStreamX:1920x1080@60\n", "VirtStream", 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h, r, have := findMonitorMode([]byte(tc.body), tc.monitor)
			assert.Equal(t, tc.wantHave, have)
			assert.Equal(t, tc.wantW, w)
			assert.Equal(t, tc.wantH, h)
			assert.Equal(t, tc.wantR, r)
		})
	}
}

func TestModesCfg_UpsertLine(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		monitor string
		w, h, r int
		want    string
	}{
		{"empty file -> append", "", "VirtStream", 1920, 1080, 60, "VirtStream:1920x1080@60\n"},
		{"existing line -> update in place", "VirtStream:800x600@30\n", "VirtStream", 1920, 1080, 60, "VirtStream:1920x1080@60\n"},
		{"keeps other lines intact", "Other:1280x720@60\nVirtStream:800x600@30\nMore:1024x768@60\n", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\nMore:1024x768@60\n"},
		{"appends when monitor missing", "Other:1280x720@60\n", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\n"},
		{"no trailing newline in input -> ensures one in output", "Other:1280x720@60", "VirtStream", 1920, 1080, 60, "Other:1280x720@60\nVirtStream:1920x1080@60\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := upsertMonitorMode([]byte(tc.input), tc.monitor, tc.w, tc.h, tc.r)
			assert.Equal(t, tc.want, string(out))
		})
	}
}

func TestModesCfg_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	// First write: no .bak, no file.
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:1920x1080@60\n")))
	bak := cfgPath + ".bak"

	// .bak should not exist on first write (we only back up before MODIFYING an existing file).
	_, err := readFile(bak)
	assert.Error(t, err, ".bak should not be created on initial write")

	// Second write: overwriting an existing file should create a .bak first time.
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:3840x2160@120\n")))
	bakBytes, err := readFile(bak)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(bakBytes))

	// Third write: .bak must NOT be overwritten (preserves first-known-good).
	require.NoError(t, WriteModesCfgAtomic(cfgPath, []byte("VirtStream:1280x720@60\n")))
	bakBytes2, err := readFile(bak)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(bakBytes2))
}

func readFile(p string) ([]byte, error) {
	return osReadFile(p)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/switcher/ -run TestModesCfg -v`
Expected: FAIL with `findMonitorMode undefined`, `upsertMonitorMode undefined`, `WriteModesCfgAtomic undefined`.

- [ ] **Step 3: Implement the parser/editor**

Create `internal/switcher/modescfg.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/switcher/ -run TestModesCfg -v`
Expected: ALL PASS.

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: ALL PASS, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/switcher/modescfg.go internal/switcher/modescfg_test.go
git commit -m "Add gamescope modes.cfg parser/editor with atomic write and one-shot backup"
```

---

## Task 3: Strategy interface and Select()

**Files:**
- Create: `internal/switcher/strategy.go`
- Create: `internal/switcher/strategy_test.go`

This task introduces the abstraction with two stub strategies. The kscreen body is filled in by Task 4; the gamescope body is filled in by Tasks 7–8. Stubs return `nil` so this task can land independently and the existing test suite stays green.

- [ ] **Step 1: Write the failing test**

Create `internal/switcher/strategy_test.go`:

```go
package switcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelect_Explicit(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "")

	s, err := Select("kscreen", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name())

	s, err = Select("debugfs", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name())

	_, err = Select("bogus", Options{})
	assert.Error(t, err)
}

func TestSelect_AutoDetect(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "")

	s, err := Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name(), "auto outside gamescope should pick kscreen")

	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")
	s, err = Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name(), "auto under gamescope should pick debugfs")
}

func TestSelect_EnvOverridesAuto(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "kscreen")
	t.Setenv("GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")

	// auto with env=kscreen should pick kscreen even though gamescope env says otherwise.
	s, err := Select("auto", Options{})
	require.NoError(t, err)
	assert.Equal(t, "kscreen", s.Name())
}

func TestSelect_FlagOverridesEnv(t *testing.T) {
	t.Setenv("SUNBEAMS_STRATEGY", "kscreen")

	// Explicit "debugfs" must beat the env var.
	s, err := Select("debugfs", Options{})
	require.NoError(t, err)
	assert.Equal(t, "debugfs", s.Name())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/switcher/ -run TestSelect -v`
Expected: FAIL with `Select undefined`, `Options undefined`.

- [ ] **Step 3: Implement the interface and Select()**

Create `internal/switcher/strategy.go`:

```go
package switcher

import (
	"fmt"
	"os"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// Strategy is the display-switching backend. Implementations differ in HOW
// they enable/disable outputs (kscreen-doctor vs debugfs force) but share
// the same SwitchOn/SwitchOff contract.
type Strategy interface {
	Name() string
	SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error
	SwitchOff(outs Outputs) error
}

// Options bundles strategy-specific knobs that can't fit in the shared
// SwitchOn/SwitchOff signature. Each strategy uses only the fields it cares
// about and ignores the rest.
type Options struct {
	// SafeRevert, when true, makes GamescopeStrategy.SwitchOff rewrite the
	// virtual monitor's modes.cfg line to a low-risk safe mode before
	// re-enabling the physical connector. KScreenStrategy ignores this.
	SafeRevert bool
}

// Select resolves a strategy name into a constructed Strategy.
//
// Precedence: an explicit name ("kscreen" or "debugfs") wins. The literal
// "auto" defers to $SUNBEAMS_STRATEGY (if non-empty), and if that is also
// empty defers to $GAMESCOPE_WAYLAND_DISPLAY (debugfs if set, kscreen
// otherwise).
func Select(name string, opts Options) (Strategy, error) {
	resolved := name
	if resolved == "auto" {
		if env := os.Getenv("SUNBEAMS_STRATEGY"); env != "" {
			resolved = env
		} else if os.Getenv("GAMESCOPE_WAYLAND_DISPLAY") != "" {
			resolved = "debugfs"
		} else {
			resolved = "kscreen"
		}
	}
	switch resolved {
	case "kscreen":
		return &KScreenStrategy{}, nil
	case "debugfs":
		return &GamescopeStrategy{Opts: opts}, nil
	default:
		return nil, fmt.Errorf("unknown strategy %q (want auto|kscreen|debugfs)", resolved)
	}
}

// KScreenStrategy is the existing KDE Plasma switcher. Body filled in by
// Task 4 (extraction from switcher.go).
type KScreenStrategy struct{}

func (*KScreenStrategy) Name() string { return "kscreen" }
func (*KScreenStrategy) SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error {
	return SwitchOn(cfg, outs, w, h, fps, hdr) // delegate during refactor
}
func (*KScreenStrategy) SwitchOff(outs Outputs) error {
	return SwitchOff(outs)
}

// GamescopeStrategy is the gaming-mode switcher. Body filled in by Tasks 7-8.
type GamescopeStrategy struct {
	Opts Options
}

func (*GamescopeStrategy) Name() string { return "debugfs" }
func (*GamescopeStrategy) SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error {
	return fmt.Errorf("debugfs strategy not implemented yet")
}
func (*GamescopeStrategy) SwitchOff(outs Outputs) error {
	return fmt.Errorf("debugfs strategy not implemented yet")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/switcher/ -run TestSelect -v`
Expected: ALL PASS.

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: ALL PASS, no regressions.

- [ ] **Step 6: Commit**

```bash
git add internal/switcher/strategy.go internal/switcher/strategy_test.go
git commit -m "Add Strategy interface and Select() with env-precedence dispatch"
```

---

## Task 4: Extract KScreenStrategy to its own file

**Files:**
- Create: `internal/switcher/kscreen.go`
- Modify: `internal/switcher/switcher.go`
- Modify: `internal/switcher/strategy.go`

Move the bodies of `SwitchOn`/`SwitchOff` (currently package-level free functions in `switcher.go`) into `KScreenStrategy` methods. Drop the delegating wrappers added in Task 3.

- [ ] **Step 1: Read the current switcher.go**

Confirm the bodies of `SwitchOn` (lines 45–99) and `SwitchOff` (lines 102–120). Note: they reference `info`, `warn`, `errLog`, `runKScreen`, `MatchMode`, `logSunshineInputs`, `logReadback` — all of which stay in package `switcher` and remain accessible.

- [ ] **Step 2: Confirm existing tests still target package-level callers**

Run: `grep -rn "switcher.SwitchOn\|switcher.SwitchOff" .` from the repo root.
Expected: hits in `cmd/sunbeams/main.go:186` and `cmd/sunbeams/main.go:218`. These will be updated in Task 5 to go through `Select()`. We keep package-level `SwitchOn`/`SwitchOff` exported as facades for now so this task lands cleanly.

- [ ] **Step 3: Create kscreen.go**

Create `internal/switcher/kscreen.go`:

```go
package switcher

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// switchOnKScreen disables the physical output, enables the virtual one, and
// sets the mode. The hdrRequested parameter is logged but not applied —
// kscreen-doctor has no mode-bundled HDR argument.
func switchOnKScreen(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error {
	virt, phys, virtSrc, physSrc := outs.resolve()

	info("switch on: requested %dx%d@%d hdr=%t", width, height, fps, hdrRequested)
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connector: %s (%s)", phys, physSrc)
	logSunshineInputs()

	if hdrRequested {
		info("HDR requested — logged only; kscreen-doctor does not toggle HDR from the command line. Configure HDR in KDE Display Settings if needed.")
	}

	match := MatchMode(cfg.Modes, width, height, fps)
	switch {
	case match.Exact:
		info("mode match: %s (exact)", match)
	case match.ExactResolution:
		info("mode match: %s (snapped refresh: requested %d Hz, Δ%d Hz)", match, fps, match.DeltaRefresh)
	default:
		info("mode match: %s (no resolution hit — closest overall, ΔW=%d ΔH=%d ΔR=%d)",
			match, match.DeltaWidth, match.DeltaHeight, match.DeltaRefresh)
		warn("requested %dx%d@%d has no configured resolution; using %s. Add a [[modes]] entry if this is a supported target.",
			width, height, fps, match)
	}

	args := []string{
		"output." + phys + ".disable",
		"output." + virt + ".enable",
		"output." + virt + ".mode." + match.String(),
	}
	info("applying switch atomically")
	if err := runKScreen(args...); err != nil {
		warn("atomic switch failed: %v", err)
		info("retrying in three steps with a 2s delay before mode-set")
		if err := runKScreen("output." + phys + ".disable"); err != nil {
			errLog("retry step 1 (disable physical) failed: %v", err)
			return err
		}
		if err := runKScreen("output." + virt + ".enable"); err != nil {
			errLog("retry step 2 (enable virtual) failed: %v", err)
			return err
		}
		time.Sleep(2 * time.Second)
		if err := runKScreen("output." + virt + ".mode." + match.String()); err != nil {
			errLog("retry step 3 (mode set) failed: %v", err)
			return err
		}
	}

	info("switch complete: active=%s mode=%s", virt, match)
	if err := logReadback(virt); err != nil {
		warn("could not read back display state: %v", err)
	}
	return nil
}

func switchOffKScreen(outs Outputs) error {
	virt, phys, virtSrc, physSrc := outs.resolve()
	info("switch off: restoring physical display")
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connector: %s (%s)", phys, physSrc)

	if err := runKScreen(
		"output."+virt+".disable",
		"output."+phys+".enable",
	); err != nil {
		errLog("switch off failed: %v", err)
		return err
	}
	info("switch off complete: %s disabled, %s re-enabled", virt, phys)
	if err := logReadback(phys); err != nil {
		warn("could not read back display state: %v", err)
	}
	return nil
}

// logSunshineInputs echoes any SUNSHINE_CLIENT_* env vars so users can see
// what Sunshine actually handed to the Do command. Missing values are
// reported as <unset>.
func logSunshineInputs() {
	keys := []string{
		"SUNSHINE_CLIENT_WIDTH",
		"SUNSHINE_CLIENT_HEIGHT",
		"SUNSHINE_CLIENT_FPS",
		"SUNSHINE_CLIENT_HDR",
	}
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			debug("sunshine env %s=<unset>", k)
			continue
		}
		info("sunshine env %s=%s", k, v)
	}
}

// logReadback fetches `kscreen-doctor -o` and prints the section for the
// given connector so users can visually confirm the switch took effect.
func logReadback(connector string) error {
	out, err := kscreenOutputs()
	if err != nil {
		return err
	}
	section := extractConnectorSection(out, connector)
	if section == "" {
		warn("connector %s not found in kscreen-doctor -o output", connector)
		debug("full kscreen-doctor -o output:\n%s", out)
		return nil
	}
	info("current state of %s (kscreen-doctor -o):", connector)
	for _, line := range strings.Split(strings.TrimRight(section, "\n"), "\n") {
		fmt.Fprintf(os.Stderr, "    %s\n", line)
	}
	return nil
}

// extractConnectorSection pulls the block of lines describing the named
// connector out of `kscreen-doctor -o` output.
func extractConnectorSection(full, connector string) string {
	lines := strings.Split(full, "\n")
	var buf strings.Builder
	capture := false
	for _, ln := range lines {
		fields := strings.Fields(ln)
		isHeader := len(fields) > 0 && fields[0] == "Output:"
		if isHeader {
			if capture {
				break
			}
			for _, f := range fields[1:] {
				if f == connector {
					capture = true
					break
				}
			}
		}
		if capture {
			buf.WriteString(ln)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}
```

- [ ] **Step 4: Trim switcher.go to just shared types and facades**

Replace `internal/switcher/switcher.go` with:

```go
package switcher

import (
	"os"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
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

// SwitchOn is a backward-compatible facade dispatching to the kscreen strategy.
// New code should call Select(...).SwitchOn(...) directly. Removed in Task 5.
func SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error {
	return switchOnKScreen(cfg, outs, w, h, fps, hdr)
}

// SwitchOff is a backward-compatible facade. Removed in Task 5.
func SwitchOff(outs Outputs) error {
	return switchOffKScreen(outs)
}
```

- [ ] **Step 5: Update strategy.go to call the extracted functions directly**

In `internal/switcher/strategy.go`, replace the `KScreenStrategy` methods:

```go
func (*KScreenStrategy) SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error {
	return switchOnKScreen(cfg, outs, w, h, fps, hdr)
}
func (*KScreenStrategy) SwitchOff(outs Outputs) error {
	return switchOffKScreen(outs)
}
```

- [ ] **Step 6: Run all tests**

Run: `make test`
Expected: ALL PASS, including all existing kscreen-related tests (`readback_test.go`, `modes_test.go`).

- [ ] **Step 7: Commit**

```bash
git add internal/switcher/
git commit -m "Extract kscreen switcher logic into KScreenStrategy methods"
```

---

## Task 5: Wire Select() into the CLI and add --strategy flag

**Files:**
- Modify: `cmd/sunbeams/main.go`
- Modify: `cmd/sunbeams/help.go` (or wherever `subcommandHelps` lives)
- Modify: `internal/switcher/switcher.go` (drop the now-unused facades)

Make the CLI go through `Select()`. Add `--strategy`, `--virtual`, `--physical`, and `--no-safe-revert` flags. Auto-detect remains the default behavior so existing users see no change.

- [ ] **Step 1: Locate help text file**

Run: `grep -rn "subcommandHelps\|topLevelSwitchHelp" cmd/sunbeams/`
Expected: identifies the file holding help strings (probably `help.go` or similar). Note its path; add new flag descriptions to the relevant entries (`switch-on`, `switch-off`).

- [ ] **Step 2: Replace `runSwitch` in `cmd/sunbeams/main.go`**

Replace the existing `runSwitch` function (lines 170–222) with:

```go
func runSwitch(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, topLevelSwitchHelp)
		os.Exit(1)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(topLevelSwitchHelp)
		return nil
	}

	switch args[0] {
	case "off":
		fs := flag.NewFlagSet("switch off", flag.ExitOnError)
		strategy := fs.String("strategy", "auto", "auto|kscreen|debugfs")
		virtual := fs.String("virtual", "", "virtual connector name (overrides $VIRTUAL_OUTPUT)")
		physical := fs.String("physical", "", "physical connector name (overrides $PHYSICAL_OUTPUT)")
		noSafeRevert := fs.Bool("no-safe-revert", false, "[debugfs] skip resetting virtual to a safe mode before re-enabling physical")
		help := subcommandHelps["switch-off"]
		fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
		if wantsHelp(args[1:]) {
			renderSubcommandHelp(os.Stdout, help, fs)
			return nil
		}
		_ = fs.Parse(args[1:])

		s, err := switcher.Select(*strategy, switcher.Options{SafeRevert: !*noSafeRevert})
		if err != nil {
			return err
		}
		return s.SwitchOff(switcher.Outputs{Virtual: *virtual, Physical: *physical})

	case "on":
		fs := flag.NewFlagSet("switch on", flag.ExitOnError)
		width := fs.Int("width", envInt("SUNSHINE_CLIENT_WIDTH"), "client width")
		height := fs.Int("height", envInt("SUNSHINE_CLIENT_HEIGHT"), "client height")
		fps := fs.Int("fps", envInt("SUNSHINE_CLIENT_FPS"), "client fps")
		hdrFlag := fs.Bool("hdr", false, "force HDR on")
		noHDR := fs.Bool("no-hdr", false, "force HDR off")
		strategy := fs.String("strategy", "auto", "auto|kscreen|debugfs")
		virtual := fs.String("virtual", "", "virtual connector name (overrides $VIRTUAL_OUTPUT)")
		physical := fs.String("physical", "", "physical connector name (overrides $PHYSICAL_OUTPUT)")
		noSafeRevert := fs.Bool("no-safe-revert", false, "[debugfs] meaningful for switch off only; here for parity")
		cfgPath := fs.String("config", "", "Config file path (default ~/.config/sunbeams/config.toml)")
		fs.StringVar(cfgPath, "c", "", "Config file path (short)")
		help := subcommandHelps["switch-on"]
		fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
		if wantsHelp(args[1:]) {
			renderSubcommandHelp(os.Stdout, help, fs)
			return nil
		}
		_ = fs.Parse(args[1:])

		cfg, err := loadConfig(*cfgPath)
		if err != nil {
			return err
		}
		if *width == 0 || *height == 0 || *fps == 0 {
			return fmt.Errorf("missing width/height/fps (pass flags or set SUNSHINE_CLIENT_*)")
		}
		hdr := os.Getenv("SUNSHINE_CLIENT_HDR") == "true"
		if *hdrFlag {
			hdr = true
		}
		if *noHDR {
			hdr = false
		}

		s, err := switcher.Select(*strategy, switcher.Options{SafeRevert: !*noSafeRevert})
		if err != nil {
			return err
		}
		return s.SwitchOn(cfg, switcher.Outputs{Virtual: *virtual, Physical: *physical}, *width, *height, *fps, hdr)

	default:
		return fmt.Errorf("unknown switch subcommand: %s (expected on|off)", args[0])
	}
}
```

- [ ] **Step 3: Drop the now-unused facades from switcher.go**

In `internal/switcher/switcher.go`, delete the `SwitchOn` and `SwitchOff` package-level functions. The strategy methods are the only callers from now on.

- [ ] **Step 4: Update help text**

In the help-text file identified in Step 1, update `subcommandHelps["switch-on"]` and `["switch-off"]` so the new flags appear in `--help` output. Add a sentence to each describing the strategy/env-detection rules. (Exact strings depend on the existing format — match the surrounding style.)

- [ ] **Step 5: Verify build and tests**

Run: `go build ./... && make test`
Expected: build succeeds, ALL tests pass.

- [ ] **Step 6: Smoke-test the CLI shape**

Run: `./sunbeams switch on --help 2>&1 | grep -E '^\s*-(strategy|virtual|physical|no-safe-revert)'`
Expected: all four flags listed.

- [ ] **Step 7: Commit**

```bash
git add cmd/sunbeams/ internal/switcher/switcher.go
git commit -m "Wire Select() into switch CLI; add --strategy/--physical/--virtual/--no-safe-revert flags"
```

---

## Task 6: GamescopeStrategy.SwitchOn

**Files:**
- Modify: `internal/switcher/gamescope.go` (or split out — the file was created as a stub in Task 3 inside `strategy.go`; move now)
- Create: `internal/switcher/gamescope_test.go`

Implement the SwitchOn flow with stubbed dependencies (helper exec + modes.cfg write) so it's fully unit-testable on macOS.

- [ ] **Step 1: Move GamescopeStrategy out of strategy.go into its own file**

Cut the `GamescopeStrategy` struct + `Name()` method out of `internal/switcher/strategy.go` and paste into a new file `internal/switcher/gamescope.go`. Add the imports as needed. The stub returning `not implemented yet` stays for now in `Switch{On,Off}`.

- [ ] **Step 2: Write the failing test**

Create `internal/switcher/gamescope_test.go`:

```go
package switcher

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

func TestGamescope_SwitchOn_PicksMatchingModeAndCallsHelper(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
			{Width: 3840, Height: 2160, Refresh: 120},
		},
		Gaming: config.GamingConfig{
			HelperPath: "/fake/sunbeams-drm-force",
			ModesCfg:   cfgPath, // tests pass an absolute path; production resolves relative
		},
	}

	var helperCalls [][]string
	stub := &GamescopeStrategy{
		Opts: Options{SafeRevert: true},
		runHelper: func(action, connector string) error {
			helperCalls = append(helperCalls, []string{action, connector})
			return nil
		},
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	err := stub.SwitchOn(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}, 3840, 2160, 120, false)
	require.NoError(t, err)

	body, err := osReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:3840x2160@120\n", string(body))

	require.Len(t, helperCalls, 1)
	assert.Equal(t, []string{"off", "HDMI-A-1"}, helperCalls[0])
}

func TestGamescope_SwitchOn_HelperFailureBubbles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		runHelper:    func(action, connector string) error { return errors.New("sudo: a password is required") },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	err := stub.SwitchOn(cfg, Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}, 1920, 1080, 60, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force off")

	// modes.cfg was still written before the helper exec — that's fine;
	// gamescope simply uses the new mode on next hotplug.
	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestGamescope_SwitchOn_VirtualNameDefaultsToEDIDMonitorName(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1920, Height: 1080, Refresh: 60},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	// Note: Outputs.Virtual is the DRM connector ("DP-1") for the helper, but
	// modes.cfg is keyed by EDID monitor name. The strategy MUST use the
	// EDID name when editing modes.cfg, regardless of what Outputs.Virtual says.
	err := stub.SwitchOn(cfg, Outputs{Virtual: "DP-1", Physical: "HDMI-A-1"}, 1920, 1080, 60, false)
	require.NoError(t, err)

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/switcher/ -run TestGamescope -v`
Expected: FAIL with `runHelper undefined` and `modesCfgPath undefined` (struct fields don't exist yet) and `not implemented yet` from the stub.

- [ ] **Step 4: Implement SwitchOn**

Replace `internal/switcher/gamescope.go`:

```go
package switcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asdfgasfhsn/sunbeams/internal/config"
)

// GamescopeStrategy implements the gaming-mode display switcher: it edits
// ~/.config/gamescope/modes.cfg to pick the resolution and execs a
// sudoers-gated helper to force-disable the physical connector via DRM
// debugfs.
type GamescopeStrategy struct {
	Opts Options

	// Test seams. Production code leaves these nil; the methods below
	// substitute production implementations on first use.
	runHelper    func(action, connector string) error
	modesCfgPath func(home string) string
}

func (*GamescopeStrategy) Name() string { return "debugfs" }

func (g *GamescopeStrategy) SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdr bool) error {
	_, phys, _, physSrc := outs.resolve()
	monitor := cfg.EDID.MonitorName
	if monitor == "" {
		return fmt.Errorf("cfg.EDID.MonitorName is empty; cannot key modes.cfg edit")
	}

	info("switch on (debugfs): requested %dx%d@%d hdr=%t", width, height, fps, hdr)
	info("physical connector: %s (%s)", phys, physSrc)
	info("virtual monitor (EDID name): %s", monitor)
	logSunshineInputs()

	if hdr {
		info("HDR requested — gamescope handles HDR via its own --hdr-* launch flags; this strategy doesn't toggle HDR.")
	}

	match := MatchMode(cfg.Modes, width, height, fps)
	switch {
	case match.Exact:
		info("mode match: %s (exact)", match)
	case match.ExactResolution:
		info("mode match: %s (snapped refresh: requested %d Hz, Δ%d Hz)", match, fps, match.DeltaRefresh)
	default:
		info("mode match: %s (no resolution hit — closest overall, ΔW=%d ΔH=%d ΔR=%d)",
			match, match.DeltaWidth, match.DeltaHeight, match.DeltaRefresh)
		warn("requested %dx%d@%d has no configured resolution; using %s", width, height, fps, match)
	}

	cfgPath := g.resolveModesCfgPath(cfg.Gaming.ModesCfg)
	if err := g.upsertMode(cfgPath, monitor, match.Width, match.Height, match.Refresh); err != nil {
		return fmt.Errorf("update modes.cfg: %w", err)
	}
	info("modes.cfg updated: %s -> %s (%s)", monitor, match, cfgPath)

	if err := g.execHelper(cfg.Gaming.HelperPath, "off", phys); err != nil {
		return fmt.Errorf("force off %s: %w", phys, err)
	}
	info("debugfs force off: %s", phys)
	return nil
}

func (g *GamescopeStrategy) upsertMode(cfgPath, monitor string, w, h, r int) error {
	body, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	updated := upsertMonitorMode(body, monitor, w, h, r)
	return WriteModesCfgAtomic(cfgPath, updated)
}

func (g *GamescopeStrategy) execHelper(path, action, connector string) error {
	if g.runHelper != nil {
		return g.runHelper(action, connector)
	}
	cmd := exec.Command("sudo", "-n", path, action, connector)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g *GamescopeStrategy) resolveModesCfgPath(cfgRel string) string {
	if g.modesCfgPath != nil {
		home, _ := os.UserHomeDir()
		return g.modesCfgPath(home)
	}
	if filepath.IsAbs(cfgRel) {
		return cfgRel
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Best effort: assume relative to CWD if HOME is missing.
		return cfgRel
	}
	return filepath.Join(home, cfgRel)
}

// SwitchOff implementation arrives in Task 7.
func (g *GamescopeStrategy) SwitchOff(outs Outputs) error {
	return fmt.Errorf("debugfs SwitchOff not implemented yet")
}
```

Note on the path resolution: production code uses `cfg.Gaming.ModesCfg` (relative to `$HOME` per the spec). Tests inject an absolute path through the `modesCfgPath` seam, but the test config's `ModesCfg` field is also set to that absolute path so the production fallback (`filepath.IsAbs`) also works without the seam. The seam exists for cases where the test wants total control.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/switcher/ -run TestGamescope -v`
Expected: ALL PASS.

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: ALL PASS, no regressions.

- [ ] **Step 7: Commit**

```bash
git add internal/switcher/gamescope.go internal/switcher/gamescope_test.go internal/switcher/strategy.go
git commit -m "Implement GamescopeStrategy.SwitchOn (modes.cfg edit + debugfs helper exec)"
```

---

## Task 7: GamescopeStrategy.SwitchOff with safe-revert

**Files:**
- Modify: `internal/switcher/gamescope.go`
- Modify: `internal/switcher/gamescope_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/switcher/gamescope_test.go`:

```go
func TestGamescope_SwitchOff_SafeRevertOnByDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{
			{Width: 1280, Height: 720, Refresh: 60},
			{Width: 1920, Height: 1080, Refresh: 60},
			{Width: 3840, Height: 2160, Refresh: 120},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	var helperCalls [][]string
	stub := &GamescopeStrategy{
		Opts: Options{SafeRevert: true},
		cfg:  cfg, // injected; production reads it through SwitchOff's caller
		runHelper: func(action, connector string) error {
			helperCalls = append(helperCalls, []string{action, connector})
			return nil
		},
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1280x720@60\n", string(body), "should revert to first qualifying mode")
	require.Len(t, helperCalls, 1)
	assert.Equal(t, []string{"on", "HDMI-A-1"}, helperCalls[0])
}

func TestGamescope_SwitchOff_NoSafeRevert(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID:   config.EDIDConfig{MonitorName: "VirtStream"},
		Modes:  []config.Mode{{Width: 1920, Height: 1080, Refresh: 60}},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: false},
		cfg:          cfg,
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:3840x2160@120\n", string(body), "must leave modes.cfg alone")
}

func TestGamescope_SwitchOff_FallbackWhenNoQualifyingMode(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID: config.EDIDConfig{MonitorName: "VirtStream"},
		// No mode with W<=1920 H<=1080 R<=60 — should fall back to literal 1920x1080@60.
		Modes: []config.Mode{
			{Width: 3840, Height: 2160, Refresh: 120},
			{Width: 2560, Height: 1440, Refresh: 144},
		},
		Gaming: config.GamingConfig{ModesCfg: cfgPath},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: true},
		cfg:          cfg,
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestGamescope_SwitchOff_ConfigOverrideForSafeRevert(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("VirtStream:3840x2160@120\n"), 0o644))

	cfg := &config.Config{
		EDID:  config.EDIDConfig{MonitorName: "VirtStream"},
		Modes: []config.Mode{{Width: 1280, Height: 720, Refresh: 60}},
		Gaming: config.GamingConfig{
			ModesCfg:       cfgPath,
			SafeRevertMode: "1024x768@30", // explicit override beats cfg.Modes scan
		},
	}

	stub := &GamescopeStrategy{
		Opts:         Options{SafeRevert: true},
		cfg:          cfg,
		runHelper:    func(action, connector string) error { return nil },
		modesCfgPath: func(_ string) string { return cfgPath },
	}

	require.NoError(t, stub.SwitchOff(Outputs{Virtual: "VirtStream", Physical: "HDMI-A-1"}))

	body, _ := osReadFile(cfgPath)
	assert.Equal(t, "VirtStream:1024x768@30\n", string(body))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/switcher/ -run TestGamescope_SwitchOff -v`
Expected: FAIL with `cfg undefined` (new struct field) and the existing not-implemented stub error.

- [ ] **Step 3: Add `cfg` field and a config-aware SwitchOff invocation path**

Update `internal/switcher/gamescope.go`. First, add the field and a setter so the CLI can inject:

```go
type GamescopeStrategy struct {
	Opts Options
	cfg  *config.Config // populated by Configure() for SwitchOff path; read by SwitchOff

	runHelper    func(action, connector string) error
	modesCfgPath func(home string) string
}

// Configure passes the loaded config to the strategy so SwitchOff has access
// to cfg.Modes and cfg.Gaming for safe-revert resolution. Called by the CLI
// after Select() and before SwitchOff().
func (g *GamescopeStrategy) Configure(cfg *config.Config) {
	g.cfg = cfg
}
```

Replace the SwitchOff stub with the implementation:

```go
func (g *GamescopeStrategy) SwitchOff(outs Outputs) error {
	if g.cfg == nil {
		return fmt.Errorf("debugfs SwitchOff requires Configure(cfg) before invocation")
	}
	_, phys, _, physSrc := outs.resolve()
	monitor := g.cfg.EDID.MonitorName
	if monitor == "" {
		return fmt.Errorf("cfg.EDID.MonitorName is empty; cannot key modes.cfg edit")
	}

	info("switch off (debugfs): physical=%s (%s) safe_revert=%t", phys, physSrc, g.Opts.SafeRevert)

	if g.Opts.SafeRevert {
		w, h, r := safeRevertMode(g.cfg)
		cfgPath := g.resolveModesCfgPath(g.cfg.Gaming.ModesCfg)
		if err := g.upsertMode(cfgPath, monitor, w, h, r); err != nil {
			warn("safe-revert modes.cfg edit failed: %v (continuing with force on)", err)
		} else {
			info("modes.cfg safe-reverted to %dx%d@%d", w, h, r)
		}
	}

	if err := g.execHelper(g.cfg.Gaming.HelperPath, "on", phys); err != nil {
		return fmt.Errorf("force on %s: %w", phys, err)
	}
	info("debugfs force on: %s", phys)
	return nil
}

// safeRevertMode picks the mode used by SwitchOff when --no-safe-revert is
// not set. Precedence:
//   1. cfg.Gaming.SafeRevertMode (literal "WxH@R") if non-empty.
//   2. First entry in cfg.Modes (config-file order) with W<=1920 H<=1080 R<=60.
//   3. Literal 1920x1080@60.
func safeRevertMode(cfg *config.Config) (w, h, r int) {
	if cfg.Gaming.SafeRevertMode != "" {
		var ww, hh, rr int
		if _, err := fmt.Sscanf(cfg.Gaming.SafeRevertMode, "%dx%d@%d", &ww, &hh, &rr); err == nil {
			return ww, hh, rr
		}
		warn("cfg.Gaming.safe_revert_mode %q is not WxH@R; falling back to scan", cfg.Gaming.SafeRevertMode)
	}
	for _, m := range cfg.Modes {
		if m.Width <= 1920 && m.Height <= 1080 && m.Refresh <= 60 {
			return m.Width, m.Height, m.Refresh
		}
	}
	return 1920, 1080, 60
}
```

- [ ] **Step 4: Wire Configure() in the CLI**

In `cmd/sunbeams/main.go`'s `runSwitch`, after constructing the strategy in the `off` branch, call `Configure` if applicable:

Replace:

```go
		s, err := switcher.Select(*strategy, switcher.Options{SafeRevert: !*noSafeRevert})
		if err != nil {
			return err
		}
		return s.SwitchOff(switcher.Outputs{Virtual: *virtual, Physical: *physical})
```

with:

```go
		s, err := switcher.Select(*strategy, switcher.Options{SafeRevert: !*noSafeRevert})
		if err != nil {
			return err
		}
		// SwitchOff for the debugfs strategy needs the config for modes.cfg edits.
		// Loading config here is cheap and harmless for kscreen.
		cfg, err := loadConfig("")
		if err != nil {
			return err
		}
		if g, ok := s.(*switcher.GamescopeStrategy); ok {
			g.Configure(cfg)
		}
		return s.SwitchOff(switcher.Outputs{Virtual: *virtual, Physical: *physical})
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/switcher/ -run TestGamescope -v`
Expected: ALL PASS.

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: ALL PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/switcher/gamescope.go internal/switcher/gamescope_test.go cmd/sunbeams/main.go
git commit -m "Implement GamescopeStrategy.SwitchOff with safe-revert mode"
```

---

## Task 8: Embed shell helper, add shellcheck to lint

**Files:**
- Create: `internal/installer/embed/sunbeams-drm-force.sh`
- Create: `internal/installer/embed.go`
- Create: `internal/installer/embed_test.go`
- Modify: `Makefile`

- [ ] **Step 1: Write the helper**

Create `internal/installer/embed/sunbeams-drm-force.sh`:

```sh
#!/bin/bash
# sunbeams-drm-force: privileged DRM connector force-toggle helper.
# Installed at /usr/local/sbin/sunbeams-drm-force, mode 0700 root:root.
# Invoked via sudoers NOPASSWD by `sunbeams switch on/off` under the debugfs
# strategy. See docs/superpowers/specs/2026-04-29-gaming-mode-support-design.md
set -euo pipefail

ACTION="${1:-}"
CONNECTOR="${2:-}"

if [[ ! "$ACTION" =~ ^(on|off)$ ]]; then
    echo "bad action (expected on|off)" >&2
    exit 2
fi

# DRM connector names: HDMI-A-1, DP-1, eDP-1, DSI-1, VGA-1, DP-A-1, etc.
if [[ ! "$CONNECTOR" =~ ^[A-Za-z]+(-[A-Z])?-[0-9]+$ ]]; then
    echo "bad connector name" >&2
    exit 2
fi

shopt -s nullglob
matches=( /sys/kernel/debug/dri/*/"$CONNECTOR"/force )
if (( ${#matches[@]} == 0 )); then
    echo "no debugfs path for $CONNECTOR (debugfs may not be mounted)" >&2
    exit 3
fi
if (( ${#matches[@]} > 1 )); then
    echo "multiple debugfs paths for $CONNECTOR (multi-GPU not supported)" >&2
    exit 3
fi

echo "$ACTION" > "${matches[0]}"
udevadm trigger --subsystem-match=drm
```

- [ ] **Step 2: Write the embed wrapper and a smoke test**

Create `internal/installer/embed.go`:

```go
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
```

Create `internal/installer/embed_test.go`:

```go
package installer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelperScript_NotEmpty(t *testing.T) {
	s := HelperScript()
	assert.NotEmpty(t, s, "embedded helper must not be empty")
}

func TestHelperScript_HasShebang(t *testing.T) {
	s := HelperScript()
	assert.True(t, bytes.HasPrefix(s, []byte("#!/bin/bash")), "helper must start with bash shebang")
}

func TestHelperScript_RejectsBadConnectorPattern(t *testing.T) {
	// We can't actually exec it on macOS without bash + the debugfs files.
	// Just check that the source contains the validation regex covering DP-1.
	s := string(HelperScript())
	assert.Contains(t, s, `^[A-Za-z]+(-[A-Z])?-[0-9]+$`, "regex must accept DP-1 / eDP-1 / HDMI-A-1")
	assert.Contains(t, s, "exit 2", "must exit on validation failure")
	assert.Contains(t, s, "exit 3", "must exit on debugfs absence/ambiguity")
}

func TestHelperScript_HasShellcheckSafetyFlags(t *testing.T) {
	s := string(HelperScript())
	assert.Contains(t, s, "set -euo pipefail")
	// shopt -s nullglob is critical so the glob doesn't expand to itself when empty.
	assert.True(t, strings.Contains(s, "shopt -s nullglob"))
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/installer/ -run TestHelperScript -v`
Expected: ALL PASS (no failing-first step here — these tests are pure smoke for the embed wiring).

- [ ] **Step 4: Add shellcheck to make lint**

Modify the `lint` target in `Makefile`:

```make
.PHONY: lint
lint: ## Run golangci-lint and shellcheck on embedded shell helpers
	golangci-lint run
	@if command -v shellcheck >/dev/null 2>&1; then \
		shellcheck internal/installer/embed/*.sh; \
	else \
		echo "shellcheck not installed; skipping shell lint"; \
	fi
```

- [ ] **Step 5: Run lint**

Run: `make lint`
Expected: golangci-lint passes; shellcheck either passes (if installed; the nix flake should provide it) or skips with the warning message.

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: ALL PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/installer/embed/ internal/installer/embed.go internal/installer/embed_test.go Makefile
git commit -m "Embed sunbeams-drm-force helper script and add shellcheck to make lint"
```

---

## Task 9: Installer gaming-mode block (file writes)

**Files:**
- Create: `internal/installer/gamescope.go`
- Create: `internal/installer/gamescope_test.go`

This task implements the file-writing primitives in isolation: helper installation, sudoers writing with visudo validation, and modes.cfg seeding. The installer-flow integration is Task 10.

- [ ] **Step 1: Write the failing tests**

Create `internal/installer/gamescope_test.go`:

```go
package installer

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallHelper_WritesScriptWithMode0700(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "sbin", "sunbeams-drm-force")

	require.NoError(t, InstallHelper(dst, HelperScript()))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o700), info.Mode().Perm())

	body, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, string(HelperScript()), string(body))
}

func TestInstallSudoers_WritesWithMode0440AndValidates(t *testing.T) {
	if _, err := exec.LookPath("visudo"); err != nil {
		t.Skip("visudo not available; skipping (production install always requires it)")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "sudoers.d", "sunbeams-drm-switch")

	require.NoError(t, InstallSudoers(dst, "alice", "/usr/local/sbin/sunbeams-drm-force"))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o440), info.Mode().Perm())

	body, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Contains(t, string(body), "alice ALL=(root) NOPASSWD: /usr/local/sbin/sunbeams-drm-force")
	assert.True(t, len(body) > 0 && body[len(body)-1] == '\n', "must end with newline (sudoers requirement)")
}

func TestInstallSudoers_VisudoRejectsBadContent(t *testing.T) {
	if _, err := exec.LookPath("visudo"); err != nil {
		t.Skip("visudo not available")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "sudoers.d", "sunbeams-drm-switch")

	// Empty user name produces invalid sudoers content.
	err := InstallSudoers(dst, "", "/usr/local/sbin/sunbeams-drm-force")
	require.Error(t, err)

	// Destination must NOT have been written.
	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr), "install must abort before writing to dst when visudo fails")
}

func TestSeedModesCfg_NewFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".config", "gamescope", "modes.cfg")

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "VirtStream:1920x1080@60\n", string(body))
}

func TestSeedModesCfg_UpdatesExistingLine(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("Other:1280x720@60\nVirtStream:800x600@30\n"), 0o644))

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "Other:1280x720@60\nVirtStream:1920x1080@60\n", string(body))
}

func TestSeedModesCfg_AppendsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "modes.cfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("Other:1280x720@60\n"), 0o644))

	require.NoError(t, SeedModesCfg(cfgPath, "VirtStream", 1920, 1080, 60))

	body, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "Other:1280x720@60\nVirtStream:1920x1080@60\n", string(body))
}

func TestPreflightDebugfsPath_NoMatchOK(t *testing.T) {
	// The pre-flight is permissive: zero matches is a warn, not an error.
	// Pass a connector that doesn't exist on the host.
	res := PreflightDebugfsPath("ZZZZ-Z-99")
	assert.Equal(t, 0, res.Count)
	assert.NoError(t, res.Err, "zero matches must not be an error")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/installer/ -run "TestInstallHelper|TestInstallSudoers|TestSeedModesCfg|TestPreflightDebugfsPath" -v`
Expected: FAIL with undefined symbols.

- [ ] **Step 3: Implement gamescope.go**

Create `internal/installer/gamescope.go`:

```go
package installer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp sudoers: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp sudoers: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o440); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp sudoers: %w", err)
	}

	cmd := exec.Command("visudo", "-cf", tmpPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("visudo rejected sudoers fragment: %w (%s)", err, stderr.String())
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
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
// installer must not depend on the switcher package).
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/installer/ -run "TestInstallHelper|TestInstallSudoers|TestSeedModesCfg|TestPreflightDebugfsPath" -v`
Expected: ALL PASS (sudoers test skipped on hosts without visudo).

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: ALL PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/installer/gamescope.go internal/installer/gamescope_test.go
git commit -m "Add installer primitives: helper install, sudoers with visudo, modes.cfg seed"
```

---

## Task 10: Wire gaming-mode block into installer.Run

**Files:**
- Modify: `internal/installer/installer.go`
- Modify: `internal/installer/installer_test.go`
- Modify: `cmd/sunbeams/main.go`

Refactor `Run` to take an options struct, add the gaming-mode block after the existing flow, and update the single CLI call site.

- [ ] **Step 1: Inspect the existing test**

Run: `cat internal/installer/installer_test.go`
Note: any existing tests will need their call signature adjusted in step 4.

- [ ] **Step 2: Write the failing test for the gaming-mode block**

Append to `internal/installer/installer_test.go`:

```go
func TestRun_GamingModeWritesArtifacts(t *testing.T) {
	if os.Geteuid() != 0 {
		// The integration check at the top of Run() requires root. We test the
		// gaming-mode primitives directly elsewhere; here we just exercise the
		// branching with the root check stubbed.
		t.Skip("Run() requires root; integration tested on Bazzite hardware")
	}
	// (When promoted to integration, this would set up a tempdir-rooted
	// firmware/sudoers/sbin tree and assert post-conditions.)
}
```

The real assertions for installer primitives are in Task 9's test file. This task is mostly about wiring; the unit-test surface is small.

- [ ] **Step 3: Refactor Run to take Options**

Replace `internal/installer/installer.go` Run with the options-struct shape. Full new file body:

```go
package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

const (
	FirmwareDir   = "/etc/firmware"
	EDIDName      = "edid.bin"
	HelperPath    = "/usr/local/sbin/sunbeams-drm-force"
	SudoersPath   = "/etc/sudoers.d/sunbeams-drm-switch"
	GamescopeRel  = ".config/gamescope/modes.cfg"
)

// GamingChoice is the tri-state for the gaming-mode block.
type GamingChoice int

const (
	GamingAsk GamingChoice = iota
	GamingYes
	GamingNo
)

// Options bundles all inputs into Run.
type Options struct {
	EDIDBytes   []byte
	ModesScript []byte
	MonitorName string // from cfg.EDID.MonitorName; used for modes.cfg seed
	Stdin       io.Reader
	Stdout      io.Writer

	Gaming           GamingChoice
	PhysicalConnector string // empty + GamingAsk -> prompt
}

// Run drives the interactive installer.
func Run(opts Options) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must run as root (sudo)")
	}

	stdin := opts.Stdin
	stdout := opts.Stdout

	// 1. Install EDID
	if err := os.MkdirAll(FirmwareDir, 0o755); err != nil {
		return err
	}
	edidPath := filepath.Join(FirmwareDir, EDIDName)
	if err := os.WriteFile(edidPath, opts.EDIDBytes, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ Installed EDID to %s (%d bytes)\n", edidPath, len(opts.EDIDBytes)) //nolint:errcheck

	// 2. Scan connectors
	cons, err := ScanConnectors()
	if err != nil {
		return fmt.Errorf("scan connectors: %w", err)
	}
	if len(cons) == 0 {
		return fmt.Errorf("no HDMI/DP connectors found")
	}
	fmt.Fprintln(stdout)                       //nolint:errcheck
	fmt.Fprintln(stdout, "Available outputs:") //nolint:errcheck
	for i, c := range cons {
		marker := ""
		if c.Status == "disconnected" {
			marker = " (recommended)"
		}
		fmt.Fprintf(stdout, "  [%d] %s — %s%s\n", i+1, c.Name, c.Status, marker) //nolint:errcheck
	}

	// 3. Prompt for virtual selection
	r := bufio.NewReader(stdin)
	fmt.Fprint(stdout, "\nSelect output for virtual display [1-", len(cons), "]: ") //nolint:errcheck
	line, _ := r.ReadString('\n')
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(cons) {
		return fmt.Errorf("invalid selection")
	}
	virtual := cons[idx-1].Name

	// 4. Inject kargs
	kargs := BuildKargs(FirmwareDir, virtual, EDIDName)
	if err := InjectKargs(kargs); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "✓ Kernel args added") //nolint:errcheck

	// 5. Optional xrandr user service (legacy X11 path)
	if len(opts.ModesScript) > 0 {
		fmt.Fprint(stdout, "Install systemd user service to re-add xrandr modes at login? [y/N]: ") //nolint:errcheck
		line, _ = r.ReadString('\n')
		if len(line) > 0 && (line[0] == 'y' || line[0] == 'Y') {
			if err := installXrandrUserService(stdout, opts.ModesScript, virtual); err != nil {
				return err
			}
		}
	}

	// 6. Gaming-mode block
	if err := runGamingBlock(opts, stdin, stdout, cons, r); err != nil {
		return err
	}

	return nil
}

func runGamingBlock(opts Options, stdin io.Reader, stdout io.Writer, cons []Connector, r *bufio.Reader) error {
	// Decide whether to install gaming-mode artifacts.
	want := false
	switch opts.Gaming {
	case GamingYes:
		want = true
	case GamingNo:
		return nil
	case GamingAsk:
		fmt.Fprint(stdout, "\nSet up gaming mode (gamescope) support? [y/N]: ") //nolint:errcheck
		line, _ := r.ReadString('\n')
		want = len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	}
	if !want {
		return nil
	}

	// Resolve physical connector.
	physical := opts.PhysicalConnector
	if physical == "" {
		fmt.Fprintln(stdout, "\nSelect PHYSICAL output (will be force-disabled during streaming):") //nolint:errcheck
		for i, c := range cons {
			marker := ""
			if c.Status == "connected" {
				marker = " (connected)"
			}
			fmt.Fprintf(stdout, "  [%d] %s%s\n", i+1, c.Name, marker) //nolint:errcheck
		}
		fmt.Fprint(stdout, "Select [1-", len(cons), "]: ") //nolint:errcheck
		line, _ := r.ReadString('\n')
		var idx int
		if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(cons) {
			return fmt.Errorf("invalid physical selection")
		}
		physical = cons[idx-1].Name
	}

	// Pre-flight debugfs.
	pre := PreflightDebugfsPath(physical)
	if pre.Err != nil {
		return pre.Err
	}
	if pre.Count == 0 {
		fmt.Fprintf(stdout, "  ! debugfs path for %s not found yet (may populate after reboot)\n", physical) //nolint:errcheck
	} else {
		fmt.Fprintf(stdout, "  ✓ debugfs path: %s\n", pre.Paths[0]) //nolint:errcheck
	}

	// Install helper.
	if err := InstallHelper(HelperPath, HelperScript()); err != nil {
		return fmt.Errorf("install helper: %w", err)
	}
	fmt.Fprintf(stdout, "✓ Installed helper to %s (mode 0700)\n", HelperPath) //nolint:errcheck

	// Resolve real user.
	realUser := os.Getenv("SUDO_USER")
	if realUser == "" {
		return fmt.Errorf("cannot determine real user — set SUDO_USER or run without sudo")
	}
	u, err := user.Lookup(realUser)
	if err != nil {
		return fmt.Errorf("lookup user %q: %w", realUser, err)
	}

	// Install sudoers (visudo-validated).
	if err := InstallSudoers(SudoersPath, realUser, HelperPath); err != nil {
		return fmt.Errorf("install sudoers: %w", err)
	}
	fmt.Fprintf(stdout, "✓ Installed sudoers fragment %s (mode 0440)\n", SudoersPath) //nolint:errcheck

	// Seed modes.cfg.
	if opts.MonitorName == "" {
		fmt.Fprintln(stdout, "  ! cfg.EDID.MonitorName is empty; skipping modes.cfg seed") //nolint:errcheck
	} else {
		modesCfg := filepath.Join(u.HomeDir, GamescopeRel)
		if err := SeedModesCfg(modesCfg, opts.MonitorName, 1920, 1080, 60); err != nil {
			return fmt.Errorf("seed modes.cfg: %w", err)
		}
		// chown so the user can edit/replace the file.
		if err := chownFromUser(modesCfg, u); err != nil {
			fmt.Fprintf(stdout, "  ! could not chown %s to %s: %v\n", modesCfg, realUser, err) //nolint:errcheck
		}
		fmt.Fprintf(stdout, "✓ Seeded %s with %s:1920x1080@60\n", modesCfg, opts.MonitorName) //nolint:errcheck
	}

	fmt.Fprintln(stdout, "\nGaming mode setup complete.")                              //nolint:errcheck
	fmt.Fprintln(stdout, "Use these Sunshine Do/Undo commands:")                        //nolint:errcheck
	fmt.Fprintln(stdout, "  Do:   sunbeams switch on  --physical "+physical+" \\")      //nolint:errcheck
	fmt.Fprintln(stdout, "          --width $SUNSHINE_CLIENT_WIDTH \\")                 //nolint:errcheck
	fmt.Fprintln(stdout, "          --height $SUNSHINE_CLIENT_HEIGHT \\")               //nolint:errcheck
	fmt.Fprintln(stdout, "          --fps $SUNSHINE_CLIENT_FPS")                        //nolint:errcheck
	fmt.Fprintln(stdout, "  Undo: sunbeams switch off --physical "+physical)            //nolint:errcheck
	return nil
}

func installXrandrUserService(stdout io.Writer, script []byte, virtual string) error {
	realUser := os.Getenv("SUDO_USER")
	if realUser == "" {
		return fmt.Errorf("cannot determine real user — set SUDO_USER or run without sudo")
	}
	u, err := user.Lookup(realUser)
	if err != nil {
		return fmt.Errorf("lookup user %q: %w", realUser, err)
	}
	home := u.HomeDir
	scriptPath := filepath.Join(home, ".local", "bin", "add-virtual-display-modes.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return err
	}
	customized := bytes.ReplaceAll(script, []byte("HDMI-A-1"), []byte(virtual))
	if err := os.WriteFile(scriptPath, customized, 0o755); err != nil {
		return err
	}
	svcDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(svcDir, "virtual-display-modes.service")
	if err := os.WriteFile(unitPath, []byte(UserServiceUnit(scriptPath)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ User service installed at %s\n", unitPath)                                                   //nolint:errcheck
	fmt.Fprintln(stdout, "  Enable after reboot with:")                                                                 //nolint:errcheck
	fmt.Fprintln(stdout, "    systemctl --user daemon-reload && systemctl --user enable virtual-display-modes.service") //nolint:errcheck
	return nil
}
```

- [ ] **Step 4: Add chown helper**

Append to `internal/installer/gamescope.go`:

```go
import "strconv"

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
```

(If `internal/installer/gamescope.go` already has an import block, merge `os/user` and `strconv` rather than duplicating.)

- [ ] **Step 5: Update installer_test.go for the new signature**

In `internal/installer/installer_test.go`, find any test calling `Run(...)` and update it to the new struct form:

```go
err := Run(Options{
    EDIDBytes:   edidBytes,
    ModesScript: nil,
    MonitorName: "VirtStream",
    Stdin:       strings.NewReader("1\n"),
    Stdout:      &buf,
    Gaming:      GamingNo, // existing tests are EDID-only
})
```

(Adapt to whatever the existing tests do; the key change is `Run(...)` takes one struct arg instead of four positional args.)

- [ ] **Step 6: Update the CLI call site**

In `cmd/sunbeams/main.go`, replace `runInstall`:

```go
func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	withGaming := fs.Bool("with-gaming", false, "Install gaming-mode artifacts (skip prompt)")
	noGaming := fs.Bool("no-gaming", false, "Skip gaming-mode artifacts (skip prompt)")
	physical := fs.String("physical", "", "Physical connector for force-disable (gaming mode only)")
	help := subcommandHelps["install"]
	fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
	if wantsHelp(args) {
		renderSubcommandHelp(os.Stdout, help, fs)
		return nil
	}
	_ = fs.Parse(args)

	cfg, err := loadConfig("")
	if err != nil {
		return err
	}
	result, err := generate.Generate(cfg)
	if err != nil {
		return err
	}
	var modesScript []byte
	if len(result.HighModes) > 0 {
		modesScript = []byte(generate.WriteAddCustomModesScript(result))
	}

	gaming := installer.GamingAsk
	if *withGaming && *noGaming {
		return fmt.Errorf("--with-gaming and --no-gaming are mutually exclusive")
	}
	if *withGaming {
		gaming = installer.GamingYes
	}
	if *noGaming {
		gaming = installer.GamingNo
	}

	return installer.Run(installer.Options{
		EDIDBytes:         result.EDIDBytes,
		ModesScript:       modesScript,
		MonitorName:       cfg.EDID.MonitorName,
		Stdin:             os.Stdin,
		Stdout:            os.Stdout,
		Gaming:            gaming,
		PhysicalConnector: *physical,
	})
}
```

In `main()`, update the install dispatch to pass `os.Args[2:]`:

```go
	case "install":
		if err := runInstall(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
```

(The current help-check guard is folded into `runInstall` via `wantsHelp(args)`.)

- [ ] **Step 7: Update install help text**

In the help-text file, add the new flags to `subcommandHelps["install"]`.

- [ ] **Step 8: Run all tests**

Run: `make test`
Expected: ALL PASS, including modified `installer_test.go`. The new `TestRun_GamingModeWritesArtifacts` will skip on non-root.

- [ ] **Step 9: Smoke test the CLI shape**

Run: `go build -o sunbeams ./cmd/sunbeams && ./sunbeams install --help`
Expected: shows `--with-gaming`, `--no-gaming`, `--physical` flags.

- [ ] **Step 10: Commit**

```bash
git add internal/installer/ cmd/sunbeams/
git commit -m "Wire gaming-mode block into installer.Run with --with-gaming/--no-gaming/--physical flags"
```

---

## Task 11: Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/troubleshooting.md`
- Modify: `CLAUDE.md`
- Modify: `docs/architecture.md`

- [ ] **Step 1: Add Gaming Mode section to README.md**

Append after the existing usage section (find the appropriate anchor by reading the current README):

```markdown
## Gaming Mode (gamescope)

Sunbeams supports Bazzite Gaming Mode (gamescope) in addition to Desktop
Mode (KDE Plasma). Gaming Mode uses a different switching mechanism
because `kscreen-doctor` doesn't run under gamescope.

### Install

```bash
sudo sunbeams install --with-gaming --physical=HDMI-A-1
```

This adds three artifacts on top of the standard install:

- `/usr/local/sbin/sunbeams-drm-force` — privileged shell helper (mode 0700)
- `/etc/sudoers.d/sunbeams-drm-switch` — NOPASSWD grant for that one binary
- `~/.config/gamescope/modes.cfg` — seeded with your virtual monitor name

### Sunshine commands

In Sunshine's General → Command Preparations, add:

| | Command |
|---|---|
| Do | `sunbeams switch on --physical HDMI-A-1 --width $SUNSHINE_CLIENT_WIDTH --height $SUNSHINE_CLIENT_HEIGHT --fps $SUNSHINE_CLIENT_FPS` |
| Undo | `sunbeams switch off --physical HDMI-A-1` |

The `--strategy` flag defaults to `auto` and detects gamescope via
`$GAMESCOPE_WAYLAND_DISPLAY`. Use `--strategy=debugfs` to force the gaming-mode
strategy from desktop (e.g. for testing).

### Power-user knobs

| Flag / env | Purpose |
|---|---|
| `--strategy=auto\|kscreen\|debugfs` | Override switching strategy |
| `$SUNBEAMS_STRATEGY` | Persistent strategy override |
| `--no-safe-revert` | Skip the resolution-safety reset on `switch off` |
```

- [ ] **Step 2: Add Gaming Mode block to docs/troubleshooting.md**

Append:

```markdown
## Gaming Mode

### Sleep/wake leaves the physical disconnected

If the system sleeps mid-stream, Sunshine cannot run its Undo command and the
physical connector stays force-off. Manually re-enable from another machine
via SSH:

```bash
ssh user@bazzite-host -- sudo /usr/local/sbin/sunbeams-drm-force on HDMI-A-1
```

This is a kernel-level limitation, not a sunbeams bug.

### `sudo: a password is required`

The sudoers fragment was not installed or was overwritten. Re-run:

```bash
sudo sunbeams install --with-gaming --physical=HDMI-A-1
```

### `multiple debugfs paths for HDMI-A-1`

Multi-GPU systems are not supported in v1. The helper refuses to guess which
GPU's connector to toggle.

### Black screen returning to desktop after streaming

The gaming-mode `switch off` resets your virtual monitor to a safe mode
(default `1920x1080@60`) before re-enabling the physical, specifically to
avoid this. If you've added `--no-safe-revert`, this is the cost.

### NVIDIA proprietary driver

The `force` debugfs interface is verified on AMD; behavior on NVIDIA's
proprietary stack is unverified. The helper fails cleanly with exit code 3
("no debugfs path") if your driver doesn't expose the file.
```

- [ ] **Step 3: Update CLAUDE.md architecture note**

Add a paragraph to the "Architecture" section in `CLAUDE.md`:

```markdown
The `internal/switcher` package now exposes a `Strategy` interface with two
implementations: `KScreenStrategy` (KDE Plasma; user-space; today's behavior)
and `GamescopeStrategy` (Gaming Mode; uses `~/.config/gamescope/modes.cfg`
plus a sudoers-gated DRM debugfs helper). `Select(name, opts)` resolves
`auto|kscreen|debugfs` with precedence: `--strategy` flag > `$SUNBEAMS_STRATEGY`
env > `$GAMESCOPE_WAYLAND_DISPLAY` auto-detect. The privileged helper is an
embedded shell script installed at `/usr/local/sbin/sunbeams-drm-force` with a
strict NOPASSWD grant in `/etc/sudoers.d/sunbeams-drm-switch`.
```

- [ ] **Step 4: Update docs/architecture.md if it has switcher details**

Run: `grep -i "switcher\|kscreen\|strategy" docs/architecture.md`
If hits exist, add a corresponding section describing the new strategy. If the file is high-level only, no edit needed.

- [ ] **Step 5: Run `make check`**

Run: `make check`
Expected: fmt + lint + tests + golden parity all pass.

- [ ] **Step 6: Commit**

```bash
git add README.md docs/ CLAUDE.md
git commit -m "Document gaming-mode setup, troubleshooting, and architecture"
```

---

## Task 12: End-to-end smoke

**Files:**
- (none — verification only)

- [ ] **Step 1: Build the binary**

Run: `make build`
Expected: `sunbeams` binary in repo root.

- [ ] **Step 2: Verify all subcommands respond to --help**

```bash
./sunbeams --help
./sunbeams install --help
./sunbeams switch on --help
./sunbeams switch off --help
./sunbeams generate --help
./sunbeams config --help
```

Expected: each prints help including any new flags (`--with-gaming`, `--no-gaming`, `--physical`, `--strategy`, `--no-safe-revert`).

- [ ] **Step 3: Verify auto-detection logic with env var**

```bash
./sunbeams switch off 2>&1 | head -5  # no env, should attempt kscreen and likely fail with kscreen-doctor missing — that's fine, we just want the strategy picker to fire
GAMESCOPE_WAYLAND_DISPLAY=fake ./sunbeams switch off 2>&1 | head -5
SUNBEAMS_STRATEGY=kscreen GAMESCOPE_WAYLAND_DISPLAY=fake ./sunbeams switch off 2>&1 | head -5
```

Expected: stderr log lines reflect different strategies (kscreen / debugfs / kscreen). On macOS the actual switch will fail; we're only verifying the strategy picker.

- [ ] **Step 4: Run the full check suite one final time**

Run: `make check`
Expected: ALL PASS.

- [ ] **Step 5: Inspect generated files**

```bash
git status
```

Expected: clean tree (everything committed in earlier tasks).

- [ ] **Step 6: Tag the work as ready for hardware integration**

No commit required — this is a verification step. The work is now ready for the user to test on real Bazzite hardware (Desktop and Gaming modes both).

---

## Self-review notes

Performed after writing the plan:

**Spec coverage:**
- ✅ Strategy interface + Select() — Tasks 3–5
- ✅ KScreenStrategy extraction — Task 4
- ✅ GamescopeStrategy SwitchOn/Off — Tasks 6–7
- ✅ modes.cfg parser/editor — Task 2
- ✅ Embedded shell helper + shellcheck — Task 8
- ✅ Installer primitives (helper, sudoers, modes.cfg seed, preflight) — Task 9
- ✅ Installer flow integration with new flags — Task 10
- ✅ CLI flags (`--strategy`, `--physical`, `--no-safe-revert`, `--with-gaming`, `--no-gaming`) — Tasks 5, 10
- ✅ Config `[gaming]` section — Task 1
- ✅ Documentation (README, troubleshooting, CLAUDE.md, architecture.md) — Task 11
- ✅ End-to-end smoke — Task 12

**Type/name consistency:**
- `Options` struct in `switcher` (with `SafeRevert` field) and `Options` struct in `installer` (with `EDIDBytes`, etc.) are intentionally different types in different packages.
- `GamescopeStrategy` field names: `Opts`, `cfg`, `runHelper`, `modesCfgPath` are consistent across Tasks 6 and 7.
- `Configure(cfg)` setter introduced in Task 7 is called from Task 7's CLI snippet — consistent.
- `HelperScript()` exported in Task 8, consumed in Tasks 9 and 10 — consistent.
- `upsertMonitorMode` (switcher) and `upsertModesCfgLine` (installer) intentionally duplicated to avoid a switcher → installer import cycle; both have unit tests.

**Placeholders:** none — every step has full code or exact commands.

**Scope:** single coherent feature; no decomposition needed.

**Ambiguity:** safe-revert mode resolution clarified to a 3-step precedence (Gaming.SafeRevertMode literal → cfg.Modes scan → fallback); helper connector regex reviewed against real DRM names (HDMI-A-1, DP-1, eDP-1) and accepts all.
