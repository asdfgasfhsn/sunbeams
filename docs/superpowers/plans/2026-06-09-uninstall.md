# Uninstall + Injection Guards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `sunbeams uninstall` to remove an EDID injection, make re-running `install` self-heal stale kargs, and warn before injecting onto a connected display.

**Architecture:** A new pure `ParseSunbeamsKargs` function discovers sunbeams-owned kernel-arg tokens from the live cmdline; thin `CurrentKargs`/`DeleteKargs` shell-outs read and delete them via `rpm-ostree`. An interactive `Uninstall` driver detects and removes kargs + firmware file + user service. `install` reuses the same discovery/delete path to clean stale state and gates connected connectors.

**Tech Stack:** Go 1.24 stdlib (`strings`, `os/exec`, `bufio`, `os`, `os/user`, `path/filepath`), testify for tests.

---

## File Structure

- **Modify** `internal/installer/kargs.go` — add `ParseSunbeamsKargs` (pure), `CurrentKargs`, `DeleteKargs` (shell-outs).
- **Modify** `internal/installer/kargs_test.go` — add `TestParseSunbeamsKargs` table tests.
- **Create** `internal/installer/uninstall.go` — `Uninstall` interactive driver.
- **Create** `internal/installer/uninstall_test.go` — `TestUninstall_RequiresRoot`.
- **Modify** `internal/installer/installer.go` — stale-karg cleanup offer + connected-connector guard.
- **Modify** `cmd/sunbeams/main.go` — `uninstall` dispatch + `runUninstall`.
- **Modify** `cmd/sunbeams/help.go` — `uninstall` in `topLevelHelp` + `subcommandHelps`.
- **Modify** `cmd/sunbeams/main_test.go` — add `uninstall` to help tests.

Conventions to follow (from CLAUDE.md): `CGO_ENABLED=0` stdlib-only; CLI progress via `fmt.Fprintf(stdout, …)`; shell-outs to `rpm-ostree` are integration-tested on hardware, not mocked; pure functions get unit tests. The `//nolint:errcheck` comments on `fmt.Fprint*` to stdout match the existing installer style.

---

## Task 1: Pure karg parser

**Files:**
- Modify: `internal/installer/kargs.go`
- Test: `internal/installer/kargs_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/installer/kargs_test.go`:

```go
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
		{
			name:    "single connector full wipe",
			cmdline: single,
			want: []string{
				"firmware_class.path=/etc/firmware",
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "accumulated multi-connector full wipe",
			cmdline: accumulated,
			want: []string{
				"drm.edid_firmware=HDMI-A-1:edid.bin",
				"video=HDMI-A-1:e",
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
				"firmware_class.path=/etc/firmware",
			},
		},
		{
			name:      "connector narrowing excludes others and firmware path",
			cmdline:   accumulated,
			connector: "DP-2",
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "ignores unrelated video and foreign firmware path",
			cmdline: noise,
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin",
				"video=DP-2:e",
			},
		},
		{
			name:    "merged drm.edid_firmware token parses both connectors",
			cmdline: merged,
			want: []string{
				"drm.edid_firmware=DP-2:edid.bin,HDMI-A-1:edid.bin",
				"video=DP-2:e",
				"video=HDMI-A-1:e",
			},
		},
		{
			name:    "empty cmdline returns nil",
			cmdline: "ro quiet splash",
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSunbeamsKargs(tc.cmdline, tc.connector)
			assert.Equal(t, tc.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/installer/ -run TestParseSunbeamsKargs -v`
Expected: FAIL — `undefined: ParseSunbeamsKargs` (compile error).

- [ ] **Step 3: Write minimal implementation**

Add to `internal/installer/kargs.go`. First update the imports block to add `"strings"`:

```go
import (
	"fmt"
	"os/exec"
	"strings"
)
```

Then append:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/installer/ -run TestParseSunbeamsKargs -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/installer/kargs.go internal/installer/kargs_test.go
git commit -m "feat(installer): add ParseSunbeamsKargs for karg discovery"
```

---

## Task 2: Karg read/delete shell-outs

**Files:**
- Modify: `internal/installer/kargs.go`

No unit test — these shell out to `rpm-ostree` and are integration-tested on live Bazzite, matching the existing `InjectKargs` convention.

- [ ] **Step 1: Write the implementation**

Append to `internal/installer/kargs.go`:

```go
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

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/installer/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/installer/kargs.go
git commit -m "feat(installer): add CurrentKargs and DeleteKargs rpm-ostree helpers"
```

---

## Task 3: Interactive Uninstall driver

**Files:**
- Create: `internal/installer/uninstall.go`
- Create: `internal/installer/uninstall_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/installer/uninstall_test.go`:

```go
package installer

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestUninstall_RequiresRoot mirrors TestRun_RequiresRoot: the only path
// exercisable without real system access is the root check. The rest of
// Uninstall shells out to rpm-ostree/systemctl and is validated on live Bazzite.
func TestUninstall_RequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — root-check path is unreachable, skipping")
	}

	var stdout bytes.Buffer
	stdin := strings.NewReader("")

	err := Uninstall("", false, stdin, &stdout)
	if err == nil {
		t.Fatal("expected Uninstall to return an error when not root, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "root") {
		t.Errorf("error %q does not mention root", err.Error())
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no output before root check, got: %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/installer/ -run TestUninstall_RequiresRoot -v`
Expected: FAIL — `undefined: Uninstall` (compile error).

- [ ] **Step 3: Write minimal implementation**

Create `internal/installer/uninstall.go`:

```go
package installer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Uninstall interactively removes what install added: sunbeams kernel args,
// the EDID firmware file, and the systemd user service + xrandr script.
// When connector is non-empty, only that connector's kargs are considered
// (the shared firmware file and user service are left alone). When assumeYes
// is true, every detected item is removed without prompting.
func Uninstall(connector string, assumeYes bool, stdin io.Reader, stdout io.Writer) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstall must run as root (sudo)")
	}

	r := bufio.NewReader(stdin)
	confirm := func(prompt string) bool {
		if assumeYes {
			return true
		}
		fmt.Fprint(stdout, prompt) //nolint:errcheck // prompt to stdout; unactionable
		line, _ := r.ReadString('\n')
		line = strings.ToLower(strings.TrimSpace(line))
		return line == "y" || line == "yes"
	}

	removedAny := false

	// 1. Kernel args
	cmdline, err := CurrentKargs()
	if err != nil {
		return err
	}
	kargs := ParseSunbeamsKargs(cmdline, connector)
	if len(kargs) > 0 {
		fmt.Fprintln(stdout, "Found sunbeams kernel arguments:") //nolint:errcheck // progress to stdout; unactionable
		for _, k := range kargs {
			fmt.Fprintf(stdout, "  %s\n", k) //nolint:errcheck // progress to stdout; unactionable
		}
		if confirm("Remove these kernel arguments? [y/N]: ") {
			if err := DeleteKargs(kargs); err != nil {
				return err
			}
			fmt.Fprintln(stdout, "✓ Kernel arguments removed") //nolint:errcheck // progress to stdout; unactionable
			removedAny = true
		}
	} else {
		fmt.Fprintln(stdout, "No sunbeams kernel arguments found.") //nolint:errcheck // progress to stdout; unactionable
	}

	// The firmware file and user service are shared/global — only handle them
	// on a full wipe, not when narrowing to a single connector.
	if connector == "" {
		// 2. EDID firmware file
		edidPath := filepath.Join(FirmwareDir, EDIDName)
		if _, statErr := os.Stat(edidPath); statErr == nil {
			if confirm(fmt.Sprintf("Remove EDID firmware file %s? [y/N]: ", edidPath)) {
				if err := os.Remove(edidPath); err != nil {
					return err
				}
				fmt.Fprintf(stdout, "✓ Removed %s\n", edidPath) //nolint:errcheck // progress to stdout; unactionable
				removedAny = true
			}
		}

		// 3. systemd user service + xrandr script
		if realUser := os.Getenv("SUDO_USER"); realUser != "" {
			if u, lookErr := user.Lookup(realUser); lookErr == nil {
				unitPath := filepath.Join(u.HomeDir, ".config", "systemd", "user", "virtual-display-modes.service")
				scriptPath := filepath.Join(u.HomeDir, ".local", "bin", "add-virtual-display-modes.sh")
				_, unitErr := os.Stat(unitPath)
				_, scriptErr := os.Stat(scriptPath)
				if unitErr == nil || scriptErr == nil {
					if confirm("Remove systemd user service and xrandr mode script? [y/N]: ") {
						if unitErr == nil {
							if err := os.Remove(unitPath); err != nil {
								return err
							}
						}
						if scriptErr == nil {
							if err := os.Remove(scriptPath); err != nil {
								return err
							}
						}
						fmt.Fprintln(stdout, "✓ Removed user service and script")                          //nolint:errcheck // progress to stdout; unactionable
						fmt.Fprintln(stdout, "  Run: systemctl --user daemon-reload (as the desktop user)") //nolint:errcheck // progress to stdout; unactionable
						removedAny = true
					}
				}
			}
		} else {
			fmt.Fprintln(stdout, "Note: SUDO_USER not set — skipping user service check.") //nolint:errcheck // progress to stdout; unactionable
		}
	}

	if removedAny {
		fmt.Fprintln(stdout, "\nReboot required for kernel argument changes to take effect.") //nolint:errcheck // progress to stdout; unactionable
	} else {
		fmt.Fprintln(stdout, "Nothing removed.") //nolint:errcheck // progress to stdout; unactionable
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/installer/ -run TestUninstall_RequiresRoot -v`
Expected: PASS (or SKIP if the test runner is root).

- [ ] **Step 5: Commit**

```bash
git add internal/installer/uninstall.go internal/installer/uninstall_test.go
git commit -m "feat(installer): add interactive Uninstall driver"
```

---

## Task 4: Install self-heal + connected-connector guard

**Files:**
- Modify: `internal/installer/installer.go`

No new unit test — `Run` past the root check is integration-tested on hardware (CLAUDE.md). Existing `TestRun_RequiresRoot` must still pass.

- [ ] **Step 1: Add the `strings` import**

In `internal/installer/installer.go`, change the import block from:

```go
import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
)
```

to add `"strings"`:

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
)
```

- [ ] **Step 2: Hoist the reader and add the stale-karg cleanup offer**

In `Run`, the reader `r` is currently created at step 3 (`r := bufio.NewReader(stdin)`). Move its creation to just after the root check and add the cleanup block. Replace this block:

```go
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must run as root (sudo)")
	}

	// 1. Install EDID
```

with:

```go
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must run as root (sudo)")
	}

	r := bufio.NewReader(stdin)

	// 0. Detect and offer to clean stale sunbeams kargs (idempotent re-install).
	if cmdline, err := CurrentKargs(); err == nil {
		if stale := ParseSunbeamsKargs(cmdline, ""); len(stale) > 0 {
			fmt.Fprintln(stdout, "Existing sunbeams kernel arguments detected:") //nolint:errcheck // progress to stdout; unactionable
			for _, k := range stale {
				fmt.Fprintf(stdout, "  %s\n", k) //nolint:errcheck // progress to stdout; unactionable
			}
			fmt.Fprint(stdout, "Remove them before injecting the new connector? [Y/n]: ") //nolint:errcheck // prompt to stdout; unactionable
			line, _ := r.ReadString('\n')
			line = strings.ToLower(strings.TrimSpace(line))
			if line == "" || line == "y" || line == "yes" {
				if err := DeleteKargs(stale); err != nil {
					return err
				}
				fmt.Fprintln(stdout, "✓ Removed stale kernel arguments") //nolint:errcheck // progress to stdout; unactionable
			}
		}
	}

	// 1. Install EDID
```

Then **delete** the now-duplicate reader creation at step 3. Change:

```go
	// 3. Prompt for selection
	fmt.Fprint(stdout, "\nSelect output for virtual display [1-", len(cons), "]: ") //nolint:errcheck // progress message to stdout; unactionable
	r := bufio.NewReader(stdin)
	line, _ := r.ReadString('\n')
```

to (drop the `r :=` line):

```go
	// 3. Prompt for selection
	fmt.Fprint(stdout, "\nSelect output for virtual display [1-", len(cons), "]: ") //nolint:errcheck // progress message to stdout; unactionable
	line, _ := r.ReadString('\n')
```

- [ ] **Step 3: Add the connected-connector guard**

Replace:

```go
	output := cons[idx-1].Name
```

with:

```go
	selected := cons[idx-1]
	output := selected.Name
	if selected.Status == "connected" {
		fmt.Fprintf(stdout, "\n⚠ %s currently has a display connected. Injecting a forced EDID\n", output) //nolint:errcheck // warning to stdout; unactionable
		fmt.Fprintln(stdout, "  will override that monitor's real EDID.")                                  //nolint:errcheck // warning to stdout; unactionable
		fmt.Fprint(stdout, "Continue anyway? [y/N]: ")                                                     //nolint:errcheck // prompt to stdout; unactionable
		confirmLine, _ := r.ReadString('\n')
		confirmLine = strings.ToLower(strings.TrimSpace(confirmLine))
		if confirmLine != "y" && confirmLine != "yes" {
			return fmt.Errorf("aborted: connector %s is connected", output)
		}
	}
```

- [ ] **Step 4: Run installer tests to verify nothing broke**

Run: `go test ./internal/installer/ -v`
Expected: PASS (including `TestRun_RequiresRoot`, `TestParseSunbeamsKargs`, `TestUninstall_RequiresRoot`).

- [ ] **Step 5: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): self-heal stale kargs and guard connected connectors"
```

---

## Task 5: CLI dispatch + help

**Files:**
- Modify: `cmd/sunbeams/main.go`
- Modify: `cmd/sunbeams/help.go`
- Modify: `cmd/sunbeams/main_test.go`

- [ ] **Step 1: Write the failing help tests**

In `cmd/sunbeams/main_test.go`, update `TestHelp_TopLevel`'s keyword list to include `"uninstall"`. Change:

```go
		for _, keyword := range []string{"USAGE:", "COMMANDS:", "generate", "switch", "install", "TYPICAL WORKFLOW", "CONFIGURATION"} {
```

to:

```go
		for _, keyword := range []string{"USAGE:", "COMMANDS:", "generate", "switch", "install", "uninstall", "TYPICAL WORKFLOW", "CONFIGURATION"} {
```

And add a case to `TestHelp_Subcommands`'s `cases` slice (after the `install` entry):

```go
		{[]string{"uninstall", "--help"}, []string{"USAGE:", "DESCRIPTION:", "EXAMPLES:", "--connector"}, true},
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/sunbeams/ -run TestHelp -v`
Expected: FAIL — top-level help missing `"uninstall"`, and the `uninstall_--help` subtest exits non-zero (unknown command).

- [ ] **Step 3: Add the dispatch case and runUninstall in main.go**

In `cmd/sunbeams/main.go`, add a case to the `switch cmd` block, immediately after the `case "install":` block (after its closing `}` for the `runInstall` error check):

```go
	case "uninstall":
		if wantsHelp(os.Args[2:]) {
			renderSubcommandHelp(os.Stdout, subcommandHelps["uninstall"], nil)
			return
		}
		if err := runUninstall(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
```

Note: the `wantsHelp` short-circuit above renders help with a nil FlagSet so `--help` works even though `runUninstall` builds its own FlagSet. Then add the `runUninstall` function after `runInstall` (near the end of the file):

```go
func runUninstall(args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	connector := fs.String("connector", "", "Only remove kargs for this connector (default: all)")
	yes := fs.Bool("yes", false, "Remove everything detected without prompting")
	fs.BoolVar(yes, "y", false, "Remove everything detected without prompting (short)")
	help := subcommandHelps["uninstall"]
	fs.Usage = func() { renderSubcommandHelp(os.Stderr, help, fs) }
	if wantsHelp(args) {
		renderSubcommandHelp(os.Stdout, help, fs)
		return nil
	}
	_ = fs.Parse(args)
	return installer.Uninstall(*connector, *yes, os.Stdin, os.Stdout)
}
```

- [ ] **Step 4: Add help content in help.go**

In `cmd/sunbeams/help.go`, add to `topLevelHelp`'s COMMANDS section, immediately after the `install` line:

```
  uninstall   Remove the EDID injection: kargs, firmware file, user service
```

And add an entry to the `subcommandHelps` map, after the `"install"` entry:

```go
	"uninstall": {
		Name:     "uninstall",
		Synopsis: "sudo sunbeams uninstall [--connector <name>] [-y]",
		Summary:  "Remove a sunbeams EDID injection (requires root, reboot afterwards).",
		Description: "Interactively detects and removes what 'install' added:\n" +
			"  1. sunbeams kernel args (drm.edid_firmware, video, firmware_class.path)\n" +
			"  2. the EDID firmware file at /etc/firmware/edid.bin\n" +
			"  3. the systemd user service and xrandr mode script\n" +
			"Use --connector to remove kargs for a single output only (leaves the\n" +
			"shared firmware file and user service in place). Use -y to remove\n" +
			"everything detected without prompting. A reboot is required for kernel\n" +
			"arg changes to take effect.",
		Examples: []string{
			"sudo sunbeams uninstall",
			"sudo sunbeams uninstall --connector DP-2",
			"sudo sunbeams uninstall -y",
		},
	},
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/sunbeams/ -run TestHelp -v`
Expected: PASS (including the new `uninstall_--help` subtest and the top-level `uninstall` keyword).

- [ ] **Step 6: Commit**

```bash
git add cmd/sunbeams/main.go cmd/sunbeams/help.go cmd/sunbeams/main_test.go
git commit -m "feat(cli): wire up uninstall subcommand and help"
```

---

## Task 6: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Run the full check suite**

Run: `make check`
Expected: fmt clean, lint clean, all tests pass, golden EDID verifies byte-for-byte (the EDID format was not touched, so the golden fixture is unchanged).

- [ ] **Step 2: Run the race detector**

Run: `make test-race`
Expected: PASS, no data races.

- [ ] **Step 3: Manual smoke (non-root, off-target)**

Run: `go run ./cmd/sunbeams uninstall`
Expected: exits with `error: uninstall must run as root (sudo)` (the root gate; rpm-ostree is never reached).

Run: `go run ./cmd/sunbeams uninstall --help`
Expected: prints the uninstall help page with USAGE, DESCRIPTION, FLAGS (`--connector`, `-y`/`--yes`), and EXAMPLES.

- [ ] **Step 4: Commit any formatting fixups**

```bash
git add -A
git commit -m "chore: gofmt/lint fixups for uninstall feature" || echo "nothing to commit"
```

---

## Self-Review Notes

- **Spec coverage:** karg parser (Task 1) ✓; CurrentKargs/DeleteKargs (Task 2) ✓; interactive Uninstall with per-group prompts + `--yes` (Task 3) ✓; install self-heal + connected guard (Task 4) ✓; CLI `uninstall` + `--connector`/`-y` + help (Task 5) ✓; merged-token full-wipe behavior covered by the `merged` test case ✓; firmware_class.path only on full wipe ✓; reboot reminder ✓.
- **Type consistency:** `ParseSunbeamsKargs(cmdline, connector string) []string`, `CurrentKargs() (string, error)`, `DeleteKargs([]string) error`, `Uninstall(connector string, assumeYes bool, io.Reader, io.Writer) error`, `runUninstall([]string) error` — used identically across all tasks.
- **Constants reused:** `FirmwareDir` and `EDIDName` from `installer.go` drive both the parser and the firmware-file path — single source of truth.
- **Merged-token `--connector` edge case:** the spec notes surgical narrowing of a merged token isn't possible with `--delete-if-present`. In practice install emits separate tokens, so this is not implemented; full wipe handles merged tokens correctly (tested). If it ever surfaces, the user runs a full `uninstall`.
