# Gaming Mode (gamescope) Support — Design

**Status**: Draft for review
**Date**: 2026-04-29
**Scope**: Add Bazzite Gaming Mode (gamescope) support to sunbeams alongside existing Desktop (KDE Plasma) support.

## Summary

Sunbeams today switches displays via `kscreen-doctor`, which is KDE-only. Bazzite Gaming Mode runs gamescope as the Wayland compositor, where `kscreen-doctor` does not work. This spec adds a second switching strategy that uses the Linux DRM debugfs `force` interface plus gamescope's `~/.config/gamescope/modes.cfg` — both compositor-agnostic mechanisms validated by the [/r/Bazzite Gamemode-streaming guide](https://www.reddit.com/r/Bazzite/comments/1s8eeka/guides_gamemode_streaming_with_apolloartemis_like/).

The existing kscreen strategy is preserved unchanged; the new debugfs strategy coexists. A runtime selector picks the right one based on environment, with explicit override flags for power users.

## Background

The EDID half of sunbeams is compositor-agnostic — `firmware_class.path` + `drm.edid_firmware=` kargs are loaded by the kernel before any compositor exists, so the same EDID firmware works in both modes. What breaks in Gaming Mode is:

1. The `internal/switcher` package (calls `kscreen-doctor`, which is not present under gamescope).
2. The optional xrandr modes user-service (X11-only; gamescope is Wayland).

The Reddit guide solves (1) with two compositor-agnostic techniques:

- **Physical-monitor disable via DRM debugfs**: write `off`/`on` to `/sys/kernel/debug/dri/<pci>/<connector>/force` and trigger `udevadm`. This is a kernel-level fake hotplug. Works on AMD (validated), plausible on Intel, unverified on NVIDIA proprietary.
- **Resolution selection via `~/.config/gamescope/modes.cfg`**: a flat text file gamescope reads on connector hotplug, format `MonitorName:WIDTHxHEIGHT@REFRESH`. Edit the line for the virtual monitor before triggering the hotplug; gamescope picks up the new mode automatically.

Combined, this gives full per-stream resolution + display switching from a Sunshine Do/Undo command — the gaming-mode equivalent of what sunbeams already provides for desktop mode.

## Goals

- Sunbeams works under Gaming Mode with a UX equivalent to today's Desktop Mode: one Do command to switch on with the requested resolution, one Undo to revert.
- The existing Desktop Mode flow is untouched and continues to require no runtime root.
- Power users get explicit knobs (`--strategy`, `$SUNBEAMS_STRATEGY`, `--no-safe-revert`) rather than implicit auto-detection only.
- Privileged operations are minimized to a single 40-line auditable shell helper, gated by a strict NOPASSWD sudoers entry on that one binary.
- Install-time UX matches current pattern: interactive prompts by default, flags for unattended/scripted installs.

## Non-goals (v1)

The following are out of scope for the initial implementation. Each has a recorded rationale and revisit condition so they can be picked up cleanly later.

| Item | Rationale | Revisit when |
|---|---|---|
| `sunbeams uninstall` subcommand | Install is one-time; manual cleanup paths are short and documented. | Users report install-state issues, or a packaging story (RPM, Flatpak) needs scripted teardown. |
| HDR toggling via gamescope | gamescope handles HDR via its own launch flags (`--hdr-enabled`, `--hdr-itm-*`). Sunbeams doesn't manipulate the gamescope command line. | Sunshine env exposes a stable HDR signal *and* gamescope gains a runtime HDR toggle (currently launch-time only). |
| Multiple virtual displays | Reddit guide assumes single virtual; modes.cfg keys by monitor name so multiple lines are technically possible, but switch logic and config schema would need more thought. | A real user case appears (e.g., two streamers from one host). |
| NVIDIA proprietary driver support | `force` debugfs behavior is unverified on NVIDIA proprietary; AMD is the validated path. | An NVIDIA user reports success or failure; tracked as a doc-level known limitation in the meantime. |
| Polkit-based privilege mechanism | sudoers + strict-arg validation matches the Reddit guide and is widely understood. The shell helper is auditable in 10 seconds. Polkit adds two more files (action + rule) for no functional gain at v1. | Bazzite (or a downstream) tightens sudoers handling, e.g., disallows custom NOPASSWD entries by default. |
| gamescope native protocol / D-Bus integration | Doesn't exist upstream yet. modes.cfg is the only documented gamescope public interface for mode preferences. | Upstream gamescope ships a stable D-Bus interface for mode/output control. |
| Sleep/wake recovery during streaming | Kernel-level limitation: a sleep mid-stream leaves the physical "unplugged" because Sunshine can't run its Undo command on suspend. CEC via DP→HDMI converters also fails because the GPU treats the port as unplugged. | The kernel/Sunshine ecosystem grows a robust suspend hook. Documented as a known limitation in the meantime. |

These items live in this spec rather than the project backlog so the rationale stays attached to the design context that produced them. If the backlog (Backlog.md) is adopted for issue tracking, mirroring these as tickets is fine; the spec remains the source of truth for *why* they were deferred.

## Architecture

### Strategy interface

`internal/switcher` gains a small interface:

```go
type Strategy interface {
    Name() string
    SwitchOn(cfg *config.Config, outs Outputs, w, h, fps int, hdr bool) error
    SwitchOff(outs Outputs) error
}

func Select(name string) (Strategy, error)
```

Two implementations:

- **`KScreenStrategy`** — today's `SwitchOn`/`SwitchOff` free functions, lifted to methods. No logic changes.
- **`GamescopeStrategy`** (new) — orchestrates `modes.cfg` edits + `sudo` helper exec.

`Select(name)` resolves to a strategy:

| Input | Result |
|---|---|
| `"auto"` (default) | If `$GAMESCOPE_WAYLAND_DISPLAY` is set → `GamescopeStrategy`; else `KScreenStrategy` |
| `"kscreen"` | `KScreenStrategy` |
| `"debugfs"` | `GamescopeStrategy` |
| anything else | error |

Precedence: `--strategy` flag > `$SUNBEAMS_STRATEGY` env > `"auto"`.

The strategy name `debugfs` describes the underlying mechanism. Trade-off acknowledged: `kscreen` is target-named while `debugfs` is mechanism-named, which is slightly inconsistent. We accept this because (a) `debugfs` accurately signals the low-level/root-required nature, (b) future gaming-mode strategies (e.g., a hypothetical gamescope D-Bus path) would warrant their own name rather than collapsing under a generic "gamescope".

### File layout

| File | Purpose | Approx LOC |
|---|---|---|
| `internal/switcher/strategy.go` (new) | `Strategy` interface, `Select()`, env detection | ~60 |
| `internal/switcher/strategy_test.go` (new) | `Select()` matrix, env-var precedence | ~80 |
| `internal/switcher/gamescope.go` (new) | `GamescopeStrategy` impl | ~150 |
| `internal/switcher/gamescope_test.go` (new) | strategy unit tests with stubbed helper/cfg | ~120 |
| `internal/switcher/modescfg.go` (new) | pure parse/edit of `~/.config/gamescope/modes.cfg` | ~80 |
| `internal/switcher/modescfg_test.go` (new) | table-driven parser tests | ~120 |
| `internal/switcher/switcher.go` (modified) | extract `KScreenStrategy` from free functions | small |
| `internal/installer/gamescope.go` (new) | install helper + sudoers + modes.cfg seed | ~180 |
| `internal/installer/gamescope_test.go` (new) | tempdir-based file-write tests | ~150 |
| `internal/installer/embed/sunbeams-drm-force.sh` (new) | `go:embed`-ded shell helper | ~40 |
| `internal/installer/installer.go` (modified) | wire gaming-mode block + flags | small |
| `internal/config/defaults.toml` (modified) | new `[gaming]` section | small |
| `cmd/sunbeams/main.go` (modified) | new flags on `install` and `switch` | small |
| `README.md` / `docs/` (modified) | gaming-mode setup section | n/a |
| `CLAUDE.md` (modified) | architecture note for the new strategy | small |

Total new code: ~750 LOC, all isolated and unit-testable. Existing 34 tests pass unchanged (the `KScreenStrategy` extraction is mechanical).

### Privileged helper

A 40-line embedded shell script, written to `/usr/local/sbin/sunbeams-drm-force` at install time:

```sh
#!/bin/bash
set -euo pipefail

ACTION="${1:-}"
CONNECTOR="${2:-}"

[[ "$ACTION" =~ ^(on|off)$ ]] || { echo "bad action (expected on|off)" >&2; exit 2; }
# Connector regex covers DRM names: HDMI-A-1, DP-1, eDP-1, DSI-1, VGA-1, etc.
[[ "$CONNECTOR" =~ ^[A-Za-z]+(-[A-Z])?-[0-9]+$ ]] || { echo "bad connector name" >&2; exit 2; }

shopt -s nullglob
matches=( /sys/kernel/debug/dri/*/"$CONNECTOR"/force )
if (( ${#matches[@]} == 0 )); then
    echo "no debugfs path for $CONNECTOR (debugfs may not be mounted)" >&2; exit 3
fi
if (( ${#matches[@]} > 1 )); then
    echo "multiple debugfs paths for $CONNECTOR (multi-GPU not supported)" >&2; exit 3
fi

echo "$ACTION" > "${matches[0]}"
udevadm trigger --subsystem-match=drm
```

Three layers of defense:

1. **sudoers grant** locked to this exact binary path: `<user> ALL=(root) NOPASSWD: /usr/local/sbin/sunbeams-drm-force`.
2. **Helper input validation**: regex-checks both arguments before any privileged action.
3. **Runtime path discovery**: globs to find the debugfs path; refuses on zero or multiple matches. No hardcoded PCI path means the helper survives GPU swaps and kernel device-numbering changes that would silently break the Reddit guide's baked-path approach.

File modes:

- `/usr/local/sbin/sunbeams-drm-force`: `0700 root:root` (only root can exec; sudo runs as root, so this is fine).
- `/etc/sudoers.d/sunbeams-drm-switch`: `0440 root:root` (sudoers requires this). Validated with `visudo -cf <tmpfile>` before atomic move into place; install aborts and leaves system unchanged on validation failure.

Choice of shell over Go: a sudoers-NOPASSWD-targeted binary should be auditable in 10 seconds. `cat /usr/local/sbin/sunbeams-drm-force` reveals exactly what runs as root. A compiled Go binary obscures that and adds a build/embed step for no functional gain.

`shellcheck` runs against the helper as part of `make lint`.

## Data flows

### Install (`sudo sunbeams install [...]`)

```
1. (existing) Write /etc/firmware/edid.bin
2. (existing) Scan connectors → prompt for VIRTUAL output
3. (existing) Inject kargs (firmware_class.path, drm.edid_firmware, video=...)
4. (existing) Optional: install xrandr user service [Y/N]   ← legacy X11

5. (NEW) Gaming-mode block — gated by:
   - --with-gaming flag → install (skip prompt)
   - --no-gaming flag → skip silently
   - neither → interactive "Set up gaming mode support? [y/N]:"

   5a. Resolve PHYSICAL connector:
       - --physical=NAME flag → use it
       - else prompt (default: first connected connector, marked "(connected)")
   5b. Pre-flight glob /sys/kernel/debug/dri/*/<physical>/force:
       - 1 match → log path for diagnostics; helper rediscovers at runtime
       - 0 matches → warn but continue (debugfs may need reboot to populate)
       - >1 matches → abort install (ambiguous; multi-GPU not v1)
   5c. Write /usr/local/sbin/sunbeams-drm-force (0700 root:root) from go:embed
   5d. Write /etc/sudoers.d/sunbeams-drm-switch (0440 root:root):
         <SUDO_USER> ALL=(root) NOPASSWD: /usr/local/sbin/sunbeams-drm-force
       Validate with `visudo -cf <tmpfile>` before atomic move; abort if invalid.
   5e. Resolve SUDO_USER's $HOME, seed ~/.config/gamescope/modes.cfg:
       - Compute monitor name from cfg.EDID.MonitorName
       - File missing → create with single line `<MonitorName>:1920x1080@60`
       - File exists → append-or-update line keyed on `<MonitorName>:` prefix
       - Never touch other lines (user's real monitors)
       - chown to SUDO_USER (we run as root)
```

### Switch on (gaming, debugfs strategy)

```
1. Resolve mode: switcher.MatchMode(cfg.Modes, w, h, fps) → e.g. "1920x1080@60"
2. Edit ~/.config/gamescope/modes.cfg:
   - Read file
   - Find or append the line starting with "<MonitorName>:"
   - Replace value with "WxH@R"
   - Atomic write (temp + rename)
   - Once-per-boot backup to .bak
3. Exec: sudo -n /usr/local/sbin/sunbeams-drm-force off <physical>
   - -n flag = non-interactive; if NOPASSWD missing, fail fast with "run install --with-gaming"
4. Log readback from /sys/class/drm/card?-<physical>/status (expect "disconnected")
5. Existing logSunshineInputs() echoes Sunshine env vars
```

### Switch off (gaming, debugfs strategy)

```
1. Safe-revert (default on; --no-safe-revert disables):
   - Rewrite modes.cfg <MonitorName> line to safe mode
   - Safe mode = first entry in cfg.Modes (config-file order) with
     W<=1920 H<=1080 R<=60; fall back to literal "1920x1080@60" if none qualify
2. Exec: sudo -n /usr/local/sbin/sunbeams-drm-force on <physical>
3. Log readback (expect "connected")
```

The existing kscreen strategy retries with a 2s delay after atomic-switch failure. The debugfs strategy doesn't need that — its writes are sequential and either succeed or fail cleanly.

## CLI surface (final)

```
sunbeams install
    [--with-gaming]      # skip prompt, install gaming-mode artifacts
    [--no-gaming]        # skip prompt, skip gaming-mode artifacts
    [--physical=NAME]    # physical connector for force-disable
    (existing flags unchanged)

sunbeams switch on
    -w INT -h INT -fps INT
    [--virtual=NAME]
    [--physical=NAME]                       # NEW; required for debugfs
    [--strategy=auto|kscreen|debugfs]       # NEW; default auto
    [--no-safe-revert]                      # NEW; debugfs only, switch off only
    (existing flags unchanged)

sunbeams switch off
    [--virtual=NAME]
    [--physical=NAME]
    [--strategy=auto|kscreen|debugfs]
    [--no-safe-revert]

Environment:
    $SUNBEAMS_STRATEGY                  # overrides default; flag overrides env
    $GAMESCOPE_WAYLAND_DISPLAY          # auto-detection signal (read-only)
    $PHYSICAL_OUTPUT, $VIRTUAL_OUTPUT   # existing
    $SUNSHINE_CLIENT_*                  # existing
```

## Config surface (`defaults.toml`)

New section, all keys overridable via `~/.config/sunbeams/config.toml`:

```toml
[gaming]
# Helper binary path. The installer always writes to and the sudoers grant
# always references /usr/local/sbin/sunbeams-drm-force. Overriding this only
# changes what `switch on/off` invokes — you would also need to install the
# helper at the new path manually and adjust /etc/sudoers.d/sunbeams-drm-switch
# to match. Almost no one should change this.
helper_path = "/usr/local/sbin/sunbeams-drm-force"

# Path to gamescope modes.cfg, relative to $HOME
modes_cfg = ".config/gamescope/modes.cfg"

# Safe-revert mode for `switch off`. Empty = pick the first entry in cfg.Modes
# (in config-file order) whose W<=1920, H<=1080, R<=60. Falls back to the
# literal "1920x1080@60" if no cfg.Modes entry qualifies.
safe_revert_mode = ""
```

## Error handling matrix

| Failure | Behavior |
|---|---|
| Helper missing | Error: "gaming-mode strategy selected but helper not installed; run `sudo sunbeams install --with-gaming`" |
| sudo prompts (NOPASSWD missing) | `sudo -n` fails fast; same error as above |
| Helper exits non-zero | Bubble stderr from helper; skip readback |
| modes.cfg parent dir missing | Create it (gamescope hasn't been launched yet) |
| modes.cfg unwritable | Log warn, continue with debugfs force — gamescope will use its previous mode |
| debugfs not mounted (helper exit 3) | Error: "debugfs unavailable; check `mount | grep debugfs` and reboot if missing" |
| Multiple debugfs matches at runtime (helper exit 3) | Error: "ambiguous DRM path for <connector>; multi-GPU not supported in v1" |
| Strategy auto-detect ambiguous (gamescope nested in Plasma) | Trust `$GAMESCOPE_WAYLAND_DISPLAY`; user can override with `--strategy=kscreen` |
| `switch off` after Sunshine session killed mid-stream (sleep/wake) | Helper still works; physical comes back. The wider sleep limitation is documented, not fixed. |
| `visudo -cf` fails during install | Abort install, leave system unchanged, surface visudo's error verbatim |

## Testing strategy

**Unit tests** (all pure, run on macOS/Linux without root):

| Test | Coverage |
|---|---|
| `switcher/strategy_test.go` | `Select()` matrix; env-var/flag precedence; auto-detect under and outside `$GAMESCOPE_WAYLAND_DISPLAY` |
| `switcher/modescfg_test.go` | append-or-update: empty file, our line only, our line + others, others only, malformed lines, BOM/trailing-newline edges |
| `switcher/gamescope_test.go` | `SwitchOn`/`SwitchOff` with stubbed `runHelper` + `editModesCfg`; mode selection, safe-revert toggle, error propagation |
| `installer/gamescope_test.go` | tempdir-based: helper written 0700, sudoers written 0440, modes.cfg seeded with correct monitor name, `visudo -cf` invocation (skipped if absent), idempotent re-install |

**Lint**: `shellcheck` added to `make lint` to cover the embedded helper.

**Integration (deferred to Bazzite hardware)**, consistent with existing un-tested paths (`installer.Run`, `runKScreen`):

- Real `sudo -n` exec
- Real debugfs write
- `udevadm trigger` propagation
- gamescope picking up modes.cfg changes

**Untouched**: golden-file test (`internal/edid` and `internal/generate` are not modified).

## Documentation work (in scope)

- `README.md` gains a "Gaming Mode" section: install command, Sunshine Do/Undo example, the safe-revert default explanation.
- `docs/troubleshooting.md` (already exists) gains a Gaming Mode block:
  - Sleep/wake leaves physical disabled (kernel limitation; doc'd workaround is SSH suspend)
  - Multi-GPU not supported in v1 (helper aborts cleanly)
  - NVIDIA proprietary `force` behavior unverified
  - Black-screen-on-return mitigated by safe-revert default; `--no-safe-revert` if it misfires
- `CLAUDE.md` gains a paragraph on the strategy abstraction so future work has context.

## Open questions / risks

1. **NVIDIA proprietary drivers**: `force` debugfs is a Linux DRM feature; behavior on NVIDIA's proprietary stack is unverified. Mitigation: documented as a known limitation; helper fails cleanly if debugfs path missing rather than silently doing nothing.
2. **debugfs mount perms across distros**: Bazzite mounts debugfs with permissive enough access for root; some hardened distros may restrict this further. Helper failure is visible, not silent.
3. **gamescope config schema stability**: `modes.cfg` is documented in the bazzite-org/gamescope fork but not formally specified upstream. A schema change would break the strategy. Mitigation: append-or-update is conservative (we never rewrite the whole file); a schema break is a one-line fix in `modescfg.go`.
4. **Sunshine session model under gamescope**: gamescope can force-restart Sunshine internally, which is incompatible with some Do/Undo command structures (Sunshine #3860). Our flow avoids this because we don't restart gamescope — we only toggle the physical connector.

## References

- [/r/Bazzite — Gamemode streaming with apollo/artemis like experience](https://www.reddit.com/r/Bazzite/comments/1s8eeka/guides_gamemode_streaming_with_apolloartemis_like/) — the source guide this strategy implements
- [Sunshine on Bazzite with virtual display (gist)](https://gist.github.com/iamthenuggetman/6d0884954653940596d463a48b2f459c) — referenced in the Reddit guide for the EDID-loading half
- [bazzite-org/gamescope](https://github.com/bazzite-org/gamescope) — Bazzite's gamescope fork, source of `modes.cfg` behavior
- [Sunshine #3860: No Steam Overlay in Gamescope Gamemode](https://github.com/LizardByte/Sunshine/issues/3860) — relevant constraint on Do/Undo design
- [HikariKnight/ScopeBuddy](https://github.com/HikariKnight/ScopeBuddy) — alternative gamescope-args injection approach (out of scope, see non-goals)
