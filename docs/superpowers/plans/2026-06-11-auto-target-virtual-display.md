# Auto-Target the Live Virtual Display — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `switch on`/`off` auto-detect the virtual connector (where the sunbeams EDID lives) and the physical connectors to disable, with no env var required.

**Architecture:** Extract read-only DRM/sysfs/karg primitives into a new neutral `internal/drm` package shared by `installer` and `switcher`; add `drm.DetectVirtual`; rework `switcher` resolution to use it. `installer` keeps mutation/orchestration and imports `drm`.

**Tech Stack:** Go 1.24 stdlib (`os`, `bytes`, `errors`, `sort`, `strings`, `path/filepath`, `os/exec`), testify.

---

## File Structure

- **Create** `internal/drm/connector.go`, `sysfs.go`, `kargs.go`, `detect.go` + `*_test.go` — read-only primitives + `DetectVirtual`.
- **Delete** `internal/installer/connectors.go`, `connectors_test.go`, `kargs_test.go` (contents move to `drm`).
- **Edit** `internal/installer/installer.go`, `kargs.go`, `status.go`, `uninstall.go`, `status_test.go` — reference `drm.*`.
- **Edit** `cmd/sunbeams/main.go` — `runStatus` references `drm.*`.
- **Edit** `internal/switcher/switcher.go` (+ new `switcher_test.go`) — auto-targeting.

Module path is `github.com/asdfgasfhsn/sunbeams`, so the new package import is `github.com/asdfgasfhsn/sunbeams/internal/drm`.

This is a behavior-preserving refactor for everything except `switcher`. Lean on `go build ./...` to surface any missed reference. The golden EDID is untouched.

---

## Task 1: Extract `internal/drm` package

**Files:** create `internal/drm/{connector.go,sysfs.go,kargs.go,connector_test.go,sysfs_test.go,kargs_test.go}`; delete `internal/installer/{connectors.go,connectors_test.go,kargs_test.go}`; edit `internal/installer/{installer.go,kargs.go,status.go,uninstall.go,status_test.go}` and `cmd/sunbeams/main.go`.

- [ ] **Step 1: Create the `drm` source files**

`internal/drm/connector.go`:
```go
package drm

import (
	"os"
	"path/filepath"
	"strings"
)

// Connector is a DRM/KMS display connector and its connection status.
type Connector struct {
	Name   string
	Status string
}

// ScanConnectors reads /sys/class/drm for HDMI/DP connectors.
func ScanConnectors() ([]Connector, error) {
	return scanConnectorsAt("/sys/class/drm")
}

func scanConnectorsAt(root string) ([]Connector, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Connector
	for _, e := range entries {
		name := e.Name()
		// name like "card0-HDMI-A-1"
		dash := strings.Index(name, "-")
		if dash < 0 {
			continue
		}
		connector := name[dash+1:]
		if !strings.HasPrefix(connector, "HDMI") && !strings.HasPrefix(connector, "DP") {
			continue
		}
		st, err := os.ReadFile(filepath.Join(root, name, "status"))
		if err != nil {
			continue
		}
		out = append(out, Connector{
			Name:   connector,
			Status: strings.TrimSpace(string(st)),
		})
	}
	return out, nil
}
```

`internal/drm/sysfs.go`:
```go
package drm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoSysfs is returned when the DRM sysfs tree is absent (e.g. on macOS),
// so callers can degrade gracefully.
var ErrNoSysfs = errors.New("no DRM sysfs tree")

// SysfsConn is one connector's raw sysfs read.
type SysfsConn struct {
	Status string
	EDID   []byte
}

// ScanConnectorEDID walks a DRM sysfs root and returns each HDMI/DP connector's
// status and live EDID bytes, keyed by connector name (e.g. "DP-2"). Returns
// ErrNoSysfs if the root does not exist.
func ScanConnectorEDID(root string) (map[string]SysfsConn, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSysfs
		}
		return nil, err
	}
	out := map[string]SysfsConn{}
	for _, e := range entries {
		name := e.Name() // e.g. "card0-DP-2"
		dash := strings.Index(name, "-")
		if dash < 0 {
			continue
		}
		connector := name[dash+1:]
		if !strings.HasPrefix(connector, "HDMI") && !strings.HasPrefix(connector, "DP") {
			continue
		}
		st, err := os.ReadFile(filepath.Join(root, name, "status"))
		if err != nil {
			continue
		}
		edid, _ := os.ReadFile(filepath.Join(root, name, "edid")) // may be absent/empty
		out[connector] = SysfsConn{Status: strings.TrimSpace(string(st)), EDID: edid}
	}
	return out, nil
}
```

`internal/drm/kargs.go`:
```go
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
```

- [ ] **Step 2: Create the `drm` test files** (migrated verbatim, names exported)

`internal/drm/connector_test.go`:
```go
package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanConnectorsReadsSysfs(t *testing.T) {
	root := t.TempDir()
	mk := func(name, status string) {
		dir := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644))
	}
	mk("card0-HDMI-A-1", "disconnected\n")
	mk("card0-DP-1", "connected\n")
	mk("card0-eDP-1", "connected\n") // not HDMI/DP — ignored

	got, err := scanConnectorsAt(root)
	require.NoError(t, err)
	names := make([]string, len(got))
	for i, c := range got {
		names[i] = c.Name
	}
	assert.Contains(t, names, "HDMI-A-1")
	assert.Contains(t, names, "DP-1")
	assert.NotContains(t, names, "eDP-1")
}
```

`internal/drm/sysfs_test.go`:
```go
package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConnector(t *testing.T, root, dir, status string, edid []byte) {
	t.Helper()
	d := filepath.Join(root, dir)
	require.NoError(t, os.MkdirAll(d, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(d, "status"), []byte(status), 0o644))
	if edid != nil {
		require.NoError(t, os.WriteFile(filepath.Join(d, "edid"), edid, 0o644))
	}
}

func TestScanConnectorEDID(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", []byte("OURS"))
	writeConnector(t, root, "card0-eDP-1", "connected\n", []byte("laptop")) // not HDMI/DP — ignored

	got, err := ScanConnectorEDID(root)
	require.NoError(t, err)
	require.Contains(t, got, "DP-2")
	assert.Equal(t, "disconnected", got["DP-2"].Status)
	assert.Equal(t, []byte("OURS"), got["DP-2"].EDID)
	assert.NotContains(t, got, "eDP-1")
}

func TestScanConnectorEDID_NoSysfs(t *testing.T) {
	_, err := ScanConnectorEDID(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.ErrorIs(t, err, ErrNoSysfs)
}
```

`internal/drm/kargs_test.go`:
```go
package drm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildKargs(t *testing.T) {
	args := BuildKargs("/etc/firmware", "HDMI-A-1", "edid.bin")
	assert.Equal(t, []string{
		"firmware_class.path=/etc/firmware",
		"drm.edid_firmware=HDMI-A-1:edid.bin",
		"video=HDMI-A-1:e",
	}, args)
}

func TestParseSunbeamsKargs(t *testing.T) {
	const single = "ro firmware_class.path=/etc/firmware drm.edid_firmware=DP-2:edid.bin video=DP-2:e quiet"
	const accumulated = "ro drm.edid_firmware=HDMI-A-1:edid.bin video=HDMI-A-1:e " +
		"drm.edid_firmware=DP-2:edid.bin video=DP-2:e firmware_class.path=/etc/firmware"
	const merged = "drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin video=DP-2:e video=HDMI-A-1:e"
	const noise = "ro video=eDP-1:1920x1080 firmware_class.path=/some/other " +
		"drm.edid_firmware=DP-2:edid.bin video=DP-2:e"

	cases := []struct {
		name      string
		cmdline   string
		connector string
		want      []string
	}{
		{"single connector full wipe", single, "", []string{
			"firmware_class.path=/etc/firmware", "drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"accumulated multi-connector full wipe", accumulated, "", []string{
			"drm.edid_firmware=HDMI-A-1:edid.bin", "video=HDMI-A-1:e",
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e", "firmware_class.path=/etc/firmware",
		}},
		{"connector narrowing excludes others and firmware path", accumulated, "DP-2", []string{
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"ignores unrelated video and foreign firmware path", noise, "", []string{
			"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e",
		}},
		{"merged drm.edid_firmware token parses both connectors", merged, "", []string{
			"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin", "video=DP-2:e", "video=HDMI-A-1:e",
		}},
		{"connector narrowing with merged token returns whole token", merged, "DP-2", []string{
			"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin", "video=DP-2:e",
		}},
		{"empty cmdline returns nil", "ro quiet splash", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseSunbeamsKargs(tc.cmdline, tc.connector))
		})
	}
}

func TestConnectorsFromKargs(t *testing.T) {
	assert.Equal(t, []string{"DP-2"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin", "video=DP-2:e", "firmware_class.path=/etc/firmware",
	}))
	assert.Equal(t, []string{"HDMI-A-1", "DP-2"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=HDMI-A-1:edid.bin", "drm.edid_firmware=DP-2:edid.bin",
	}))
	assert.Equal(t, []string{"DP-2", "HDMI-A-1"}, ConnectorsFromKargs([]string{
		"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin",
	}))
	assert.Nil(t, ConnectorsFromKargs([]string{"video=DP-2:e", "firmware_class.path=/etc/firmware"}))
}
```

- [ ] **Step 3: Delete the moved installer files**

```bash
git rm internal/installer/connectors.go internal/installer/connectors_test.go internal/installer/kargs_test.go
```

- [ ] **Step 4: Trim `internal/installer/kargs.go` to the rpm-ostree shell-outs only**

Replace the entire file with:
```go
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
```

- [ ] **Step 5: Edit `internal/installer/installer.go`**

(a) Replace the import block + const block (lines 3-17) with:
```go
import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)
```
(the `const ( FirmwareDir ... EDIDName ... )` block is removed entirely — those now live in `drm`).

(b) Update references in `Run`:
- `ParseSunbeamsKargs(cmdline, "")` → `drm.ParseSunbeamsKargs(cmdline, "")`
- `os.MkdirAll(FirmwareDir, 0o755)` → `os.MkdirAll(drm.FirmwareDir, 0o755)`
- `filepath.Join(FirmwareDir, EDIDName)` → `filepath.Join(drm.FirmwareDir, drm.EDIDName)`
- `cons, err := ScanConnectors()` → `cons, err := drm.ScanConnectors()`
- `kargs := BuildKargs(FirmwareDir, output, EDIDName)` → `kargs := drm.BuildKargs(drm.FirmwareDir, output, drm.EDIDName)`

(`CurrentKargs`, `DeleteKargs`, `InjectKargs`, `UserServiceUnit` remain installer-local — unqualified.)

- [ ] **Step 6: Edit `internal/installer/status.go`**

(a) Replace the import block (lines 3-10) with:
```go
import (
	"bytes"
	"os"
	"sort"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)
```

(b) Delete these now-moved declarations: `ErrNoSysfs` (var), `sysfsConn` (type), `scanConnectorEDID` (func), `connectorsFromKargs` (func). Keep `ConnectorStatus`, `Report`, `classify`, `buildReport`, `toSet`, `Status`.

(c) In `buildReport`, change the signature param `conns map[string]sysfsConn` → `conns map[string]drm.SysfsConn`. Body is unchanged.

(d) In `Status`, update:
- `conns, err := scanConnectorEDID(sysfsRoot)` → `conns, err := drm.ScanConnectorEDID(sysfsRoot)`
- `configured := connectorsFromKargs(ParseSunbeamsKargs(configuredRaw, ""))` → `configured := drm.ConnectorsFromKargs(drm.ParseSunbeamsKargs(configuredRaw, ""))`
- `boot := connectorsFromKargs(ParseSunbeamsKargs(string(cmdlineRaw), ""))` → `boot := drm.ConnectorsFromKargs(drm.ParseSunbeamsKargs(string(cmdlineRaw), ""))`

(`CurrentKargs()` stays unqualified — installer-local.)

- [ ] **Step 7: Edit `internal/installer/uninstall.go`**

Add `"github.com/asdfgasfhsn/sunbeams/internal/drm"` to its import block, then update:
- `ParseSunbeamsKargs(cmdline, connector)` → `drm.ParseSunbeamsKargs(cmdline, connector)`
- `filepath.Join(FirmwareDir, EDIDName)` → `filepath.Join(drm.FirmwareDir, drm.EDIDName)`

(`CurrentKargs`, `DeleteKargs` stay unqualified.)

- [ ] **Step 8: Edit `internal/installer/status_test.go`**

(a) Delete `TestConnectorsFromKargs`, `TestScanConnectorEDID`, `TestScanConnectorEDID_NoSysfs` (moved to `drm`). Keep `TestClassify`, all `TestBuildReport*`, all `TestStatus*`, and the `writeConnector` helper (still used by the `TestStatus*` tests).

(b) Add `"github.com/asdfgasfhsn/sunbeams/internal/drm"` to the import block.

(c) In every `TestBuildReport*`, change the map literal type `map[string]sysfsConn{` → `map[string]drm.SysfsConn{`.

(d) In `TestStatus_NoSysfs`, change `assert.ErrorIs(t, err, ErrNoSysfs)` → `assert.ErrorIs(t, err, drm.ErrNoSysfs)`.

- [ ] **Step 9: Edit `cmd/sunbeams/main.go` (`runStatus`)**

Add `"github.com/asdfgasfhsn/sunbeams/internal/drm"` to the import block, then in `runStatus`:
- `fwPath := filepath.Join(installer.FirmwareDir, installer.EDIDName)` → `fwPath := filepath.Join(drm.FirmwareDir, drm.EDIDName)`
- `if errors.Is(err, installer.ErrNoSysfs) {` → `if errors.Is(err, drm.ErrNoSysfs) {`

(`installer.Status(...)` stays.)

- [ ] **Step 10: Build, test, format**

Run: `cd /Users/asdfgasfhsn/src/p/sunbeams && go build ./... && go test ./... && go vet ./... && gofmt -l internal/ cmd/`
Expected: build clean; all packages PASS (including the new `internal/drm`); vet clean; gofmt prints nothing. If the build reports any remaining reference to a moved symbol (e.g. a bare `FirmwareDir`, `ScanConnectors`, `ParseSunbeamsKargs`, `ErrNoSysfs`, `sysfsConn`), qualify it with `drm.` and re-run.

Then `make verify-golden` → `golden parity OK` (no EDID change).

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "refactor: extract internal/drm package for shared DRM/karg primitives"
```
End the commit body with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 2: `drm.DetectVirtual`

**Files:** create `internal/drm/detect.go`, `internal/drm/detect_test.go`.

- [ ] **Step 1: Write the failing test** — `internal/drm/detect_test.go`:

```go
package drm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConnector is defined in sysfs_test.go (same package).

func cmdlineFile(t *testing.T, contents string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cmdline")
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	return p
}

func firmwareFile(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "edid.bin")
	require.NoError(t, os.WriteFile(p, b, 0o644))
	return p
}

func TestDetectVirtual_SingleConfigured(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro drm.edid_firmware=DP-2:edid.bin video=DP-2:e")
	fw := firmwareFile(t, []byte("EDID"))

	got, err := DetectVirtual(root, cmd, fw)
	require.NoError(t, err)
	assert.Equal(t, "DP-2", got)
}

func TestDetectVirtual_MultipleDisambiguatedByEDID(t *testing.T) {
	root := t.TempDir()
	payload := []byte("OUR-EDID")
	writeConnector(t, root, "card0-DP-2", "disconnected\n", payload)        // matches firmware
	writeConnector(t, root, "card0-HDMI-A-1", "connected\n", []byte("real")) // does not
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := firmwareFile(t, payload)

	got, err := DetectVirtual(root, cmd, fw)
	require.NoError(t, err)
	assert.Equal(t, "DP-2", got)
}

func TestDetectVirtual_AmbiguousErrors(t *testing.T) {
	root := t.TempDir()
	payload := []byte("OUR-EDID")
	writeConnector(t, root, "card0-DP-2", "disconnected\n", payload)
	writeConnector(t, root, "card0-HDMI-A-1", "disconnected\n", payload) // both match
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := firmwareFile(t, payload)

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VIRTUAL_OUTPUT")
}

func TestDetectVirtual_NoneConfiguredErrors(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro quiet splash")
	fw := firmwareFile(t, []byte("EDID"))

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install")
}

func TestDetectVirtual_MultipleNoFirmwareErrors(t *testing.T) {
	root := t.TempDir()
	writeConnector(t, root, "card0-DP-2", "disconnected\n", nil)
	writeConnector(t, root, "card0-HDMI-A-1", "disconnected\n", nil)
	cmd := cmdlineFile(t, "ro drm.edid_firmware=HDMI-A-1:edid.bin drm.edid_firmware=DP-2:edid.bin")
	fw := filepath.Join(t.TempDir(), "missing-edid.bin") // does not exist

	_, err := DetectVirtual(root, cmd, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VIRTUAL_OUTPUT")
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/drm/ -run TestDetectVirtual -v`
Expected: FAIL — `undefined: DetectVirtual`.

- [ ] **Step 3: Implement** — `internal/drm/detect.go`:

```go
package drm

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// DetectVirtual resolves the connector that carries the sunbeams virtual EDID.
// It reads the drm.edid_firmware connectors from cmdlinePath (e.g.
// /proc/cmdline); a single configured connector is returned directly. When more
// than one is configured (the accumulation case), it disambiguates by which
// connector's live sysfs EDID exactly equals the firmware file. It returns an
// actionable error when it cannot resolve a single connector. No root required.
func DetectVirtual(sysfsRoot, cmdlinePath, firmwarePath string) (string, error) {
	cmdlineRaw, _ := os.ReadFile(cmdlinePath)
	configured := ConnectorsFromKargs(ParseSunbeamsKargs(string(cmdlineRaw), ""))

	switch len(configured) {
	case 0:
		return "", fmt.Errorf("no virtual display configured: no sunbeams drm.edid_firmware in %s — run 'sudo sunbeams install' (and reboot)", cmdlinePath)
	case 1:
		return configured[0], nil
	}

	// Multiple configured: disambiguate by live EDID byte-match.
	firmware, err := os.ReadFile(firmwarePath)
	if err != nil || len(firmware) == 0 {
		return "", fmt.Errorf("multiple virtual displays configured (%s) and no firmware file to disambiguate — set VIRTUAL_OUTPUT to choose one", strings.Join(configured, ", "))
	}
	conns, err := ScanConnectorEDID(sysfsRoot)
	if err != nil {
		return "", err
	}
	var matched []string
	for _, c := range configured {
		if sc, ok := conns[c]; ok && len(sc.EDID) > 0 && bytes.Equal(sc.EDID, firmware) {
			matched = append(matched, c)
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	return "", fmt.Errorf("could not determine virtual display: %s carry the sunbeams EDID — set VIRTUAL_OUTPUT to pick one, or run 'sunbeams status'", strings.Join(configured, ", "))
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/drm/ -run TestDetectVirtual -v`
Expected: PASS (all 5). `go vet ./internal/drm/` clean; `gofmt -l internal/drm/` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/drm/detect.go internal/drm/detect_test.go
git commit -m "feat(drm): add DetectVirtual for auto-targeting the virtual display"
```
End commit body with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 3: Auto-targeting in `switcher`

**Files:** edit `internal/switcher/switcher.go`; create `internal/switcher/switcher_test.go`.

Context: `SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error` and `SwitchOff(outs Outputs) error` currently call `outs.resolve()` (returns `virt, phys, virtSrc, physSrc string`). `info`/`warn`/`errLog`/`debug` log helpers and `runKScreen`/`MatchMode`/`logReadback`/`logSunshineInputs` already exist in the package. The CLI passes `switcher.Outputs{}`.

- [ ] **Step 1: Write the failing test** — `internal/switcher/switcher_test.go`:

```go
package switcher

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asdfgasfhsn/sunbeams/internal/drm"
)

func withStubs(t *testing.T, virt string, virtErr error, cons []drm.Connector) {
	t.Helper()
	origDetect, origScan := detectVirtual, scanConnectors
	detectVirtual = func() (string, error) { return virt, virtErr }
	scanConnectors = func() ([]drm.Connector, error) { return cons, nil }
	t.Cleanup(func() { detectVirtual, scanConnectors = origDetect, origScan })
}

func TestResolveOutputs_AutoVirtualAndPhysical(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	t.Setenv("PHYSICAL_OUTPUT", "")
	withStubs(t, "DP-2", nil, []drm.Connector{
		{Name: "DP-2", Status: "disconnected"},
		{Name: "HDMI-A-1", Status: "connected"},
		{Name: "DP-1", Status: "connected"},
	})
	virt, phys, vsrc, psrc, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-2", virt)
	assert.Equal(t, "auto", vsrc)
	sort.Strings(phys)
	assert.Equal(t, []string{"DP-1", "HDMI-A-1"}, phys) // connected, non-virtual
	assert.Equal(t, "auto", psrc)
}

func TestResolveOutputs_EnvOverridesAuto(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "DP-3")
	t.Setenv("PHYSICAL_OUTPUT", "HDMI-A-2")
	withStubs(t, "DP-2", nil, []drm.Connector{{Name: "DP-1", Status: "connected"}})
	virt, phys, vsrc, psrc, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-3", virt)
	assert.Equal(t, "env:VIRTUAL_OUTPUT", vsrc)
	assert.Equal(t, []string{"HDMI-A-2"}, phys)
	assert.Equal(t, "env:PHYSICAL_OUTPUT", psrc)
}

func TestResolveOutputs_Headless(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	t.Setenv("PHYSICAL_OUTPUT", "")
	withStubs(t, "DP-2", nil, []drm.Connector{
		{Name: "DP-2", Status: "disconnected"},
	})
	virt, phys, _, _, err := resolveOutputs(Outputs{})
	require.NoError(t, err)
	assert.Equal(t, "DP-2", virt)
	assert.Empty(t, phys) // nothing connected → skip disable
}

func TestResolveOutputs_DetectErrorPropagates(t *testing.T) {
	t.Setenv("VIRTUAL_OUTPUT", "")
	withStubs(t, "", assert.AnError, nil)
	_, _, _, _, err := resolveOutputs(Outputs{})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/switcher/ -run TestResolveOutputs -v`
Expected: FAIL — `undefined: resolveOutputs` / `detectVirtual` / `scanConnectors`.

- [ ] **Step 3: Implement** — in `internal/switcher/switcher.go`:

(a) Add to the import block: `"path/filepath"`, `"sort"`, and `"github.com/asdfgasfhsn/sunbeams/internal/drm"` (keep existing `fmt`, `os`, `strings`, `time`, `config`).

(b) Replace the `resolve` method (the whole `func (o Outputs) resolve() ...` block) with these package-level detection seams + `resolveOutputs`:

```go
// detectVirtual and scanConnectors are package-level seams so tests can stub
// auto-detection without real hardware.
var detectVirtual = func() (string, error) {
	return drm.DetectVirtual("/sys/class/drm", "/proc/cmdline",
		filepath.Join(drm.FirmwareDir, drm.EDIDName))
}

var scanConnectors = drm.ScanConnectors

// resolveOutputs determines the virtual connector and the physical connectors
// to disable. Virtual: explicit flag → VIRTUAL_OUTPUT → auto-detect (error if
// unresolved). Physical: explicit flag → PHYSICAL_OUTPUT → every connected
// connector that is not the virtual one (empty when headless; scan errors are
// non-fatal).
func resolveOutputs(o Outputs) (virt string, phys []string, virtSrc, physSrc string, err error) {
	switch {
	case o.Virtual != "":
		virt, virtSrc = o.Virtual, "flag"
	case os.Getenv("VIRTUAL_OUTPUT") != "":
		virt, virtSrc = os.Getenv("VIRTUAL_OUTPUT"), "env:VIRTUAL_OUTPUT"
	default:
		virt, err = detectVirtual()
		if err != nil {
			return "", nil, "", "", fmt.Errorf("auto-detect virtual display: %w", err)
		}
		virtSrc = "auto"
	}

	switch {
	case o.Physical != "":
		phys, physSrc = []string{o.Physical}, "flag"
	case os.Getenv("PHYSICAL_OUTPUT") != "":
		phys, physSrc = []string{os.Getenv("PHYSICAL_OUTPUT")}, "env:PHYSICAL_OUTPUT"
	default:
		physSrc = "auto"
		cons, scanErr := scanConnectors()
		if scanErr != nil {
			warn("could not scan connectors for physical outputs: %v", scanErr)
			break
		}
		for _, c := range cons {
			if c.Status == "connected" && c.Name != virt {
				phys = append(phys, c.Name)
			}
		}
		sort.Strings(phys)
	}
	return virt, phys, virtSrc, physSrc, nil
}
```

(c) Rewrite `SwitchOn` to use `resolveOutputs` and a physical slice. Replace its body down through the kscreen application with:

```go
func SwitchOn(cfg *config.Config, outs Outputs, width, height, fps int, hdrRequested bool) error {
	virt, phys, virtSrc, physSrc, err := resolveOutputs(outs)
	if err != nil {
		return err
	}

	info("switch on: requested %dx%d@%d hdr=%t", width, height, fps, hdrRequested)
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connectors: %v (%s)", phys, physSrc)
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

	var args []string
	for _, p := range phys {
		args = append(args, "output."+p+".disable")
	}
	args = append(args,
		"output."+virt+".enable",
		"output."+virt+".mode."+match.String(),
	)
	info("applying switch atomically")
	if err := runKScreen(args...); err != nil {
		warn("atomic switch failed: %v", err)
		info("retrying in steps with a 2s delay before mode-set")
		for _, p := range phys {
			if err := runKScreen("output." + p + ".disable"); err != nil {
				errLog("retry step (disable physical %s) failed: %v", p, err)
				return err
			}
		}
		if err := runKScreen("output." + virt + ".enable"); err != nil {
			errLog("retry step (enable virtual) failed: %v", err)
			return err
		}
		time.Sleep(2 * time.Second)
		if err := runKScreen("output." + virt + ".mode." + match.String()); err != nil {
			errLog("retry step (mode set) failed: %v", err)
			return err
		}
	}

	info("switch complete: active=%s mode=%s", virt, match)
	if err := logReadback(virt); err != nil {
		warn("could not read back display state: %v", err)
	}
	return nil
}
```

(d) Rewrite `SwitchOff`:

```go
func SwitchOff(outs Outputs) error {
	virt, phys, virtSrc, physSrc, err := resolveOutputs(outs)
	if err != nil {
		return err
	}
	info("switch off: restoring physical display(s)")
	info("virtual connector:  %s (%s)", virt, virtSrc)
	info("physical connectors: %v (%s)", phys, physSrc)

	args := []string{"output." + virt + ".disable"}
	for _, p := range phys {
		args = append(args, "output."+p+".enable")
	}
	if err := runKScreen(args...); err != nil {
		errLog("switch off failed: %v", err)
		return err
	}
	info("switch off complete: %s disabled, physical(s) %v re-enabled", virt, phys)
	for _, p := range phys {
		if err := logReadback(p); err != nil {
			warn("could not read back display state for %s: %v", p, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/switcher/ -v`
Expected: PASS (`TestResolveOutputs*` + existing `modes_test`/`readback_test`). `go build ./...` clean; `go vet ./internal/switcher/` clean; `gofmt -l internal/switcher/` prints nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/switcher/switcher.go internal/switcher/switcher_test.go
git commit -m "feat(switcher): auto-detect virtual + physical outputs on switch"
```
End commit body with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 4: Full verification

**Files:** none (verification only).

- [ ] **Step 1:** `make check` — fmt, lint (0 issues), all tests across `cmd`, `internal/drm`, `internal/installer`, `internal/switcher`, etc.; golden parity OK.
- [ ] **Step 2:** `make test-race` — no data races.
- [ ] **Step 3:** Smoke on this (macOS) box:
  - `go run ./cmd/sunbeams status` → `display status is only available on Linux with DRM/KMS` (exit 0).
  - `go run ./cmd/sunbeams switch on --width 1920 --height 1080 --fps 60` → fails fast with an auto-detect error (no `/proc/cmdline` drm.edid_firmware on macOS / no kscreen) rather than silently targeting a default — confirm the message mentions the virtual display / install.
- [ ] **Step 4:** `go run ./cmd/sunbeams --help` still lists all commands; `git status` clean.

---

## Self-Review Notes

- **Spec coverage:** `internal/drm` extraction (Task 1) ✓; `DetectVirtual` w/ cmdline-primary + EDID-disambiguation + hard-error (Task 2) ✓; `switcher` precedence (env over auto), physical auto-detect + headless skip + non-fatal scan error, error propagation (Task 3) ✓; tests migrated + added ✓; golden untouched ✓.
- **Type/name consistency:** `drm.FirmwareDir`/`drm.EDIDName`, `drm.ScanConnectors`/`Connector`, `drm.ScanConnectorEDID`/`SysfsConn`/`ErrNoSysfs`, `drm.ParseSunbeamsKargs`/`ConnectorsFromKargs`/`BuildKargs`, `drm.DetectVirtual`, `resolveOutputs`, seams `detectVirtual`/`scanConnectors` — used identically across tasks.
- **No cycle:** `drm` imports only stdlib; `installer` and `switcher` import `drm`; nothing imports `switcher` except `main`.
- **Behavior preserved:** installer/status/uninstall logic unchanged (only symbol qualification); golden EDID and `make verify-golden` stay green.
- **Removed defaults:** the old `HDMI-A-1`/`DP-1` switch defaults are gone — replaced by auto-detect (virtual, errors if unresolved) and connected-scan (physical, may be empty).
