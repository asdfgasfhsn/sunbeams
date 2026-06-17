# Uninstall + Injection Guards — Design

**Date:** 2026-06-09
**Status:** Approved (pending implementation)

## Problem

`sunbeams install` injects an EDID by appending kernel args via
`rpm-ostree kargs --append-if-missing`. It **only ever appends** — there is no
removal path and no idempotency. Re-running the installer and selecting a
different connector leaves the previous connector's kargs in place.

A user who set up on a DisplayPort monitor, then moved the machine to a rack and
re-ran setup selecting `DP-2`, ended up with **both** `HDMI-1` and `DP-2` having
the forced EDID injected (`drm.edid_firmware=<conn>:edid.bin` accumulated for
both). The result is broken display switching.

Two gaps to close:

1. No way to remove an EDID injection (`uninstall`).
2. No guard against injecting onto a connector with a real display attached,
   which overrides that monitor's real EDID.

## Goals

- Provide `sunbeams uninstall` that cleanly removes sunbeams-installed state.
- Make re-running `install` self-healing (offer to clean stale kargs first).
- Warn + confirm before injecting onto a currently-connected connector.

## Non-Goals

- No change to EDID byte generation or the golden fixture.
- No mocking of the existing live shell-outs (`rpm-ostree`, `kscreen-doctor`)
  beyond the new pure parsing function.

## Architecture

### Karg primitives (`internal/installer/kargs.go`)

Split pure logic (unit-tested) from shell-outs (integration-tested on hardware,
matching existing convention).

- **`ParseSunbeamsKargs(cmdline string, connector string) []string`** — *pure.*
  Tokenizes a kernel cmdline string (whitespace-split) and returns the exact
  sunbeams-owned tokens to delete:
  - every `drm.edid_firmware=<conn>:<EDIDName>` token (matched by `EDIDName`,
    i.e. `edid.bin`);
  - every `video=<conn>:e` token whose `<conn>` appears among the connectors
    found in the `drm.edid_firmware` tokens — so an unrelated `video=` set by
    something else is never touched;
  - `firmware_class.path=<FirmwareDir>` — included **only** on a full wipe
    (`connector == ""`), since it is shared across connectors.

  When `connector != ""`, results are narrowed to tokens referencing that
  connector, and `firmware_class.path` is excluded.

- **`CurrentKargs() (string, error)`** — thin shell-out to `rpm-ostree kargs`
  (no subargs) returning the current cmdline string. Untested, per convention.

- **`DeleteKargs(kargs []string) error`** — runs
  `rpm-ostree kargs --delete-if-present=<token>` for each token. Untested
  shell-out. Returns the same "rpm-ostree not found" guidance error as
  `InjectKargs` when the binary is absent.

### Interactive uninstall (`internal/installer/uninstall.go`)

`Uninstall(connector string, assumeYes bool, stdin io.Reader, stdout io.Writer) error`

- Root-gated (`os.Geteuid() != 0`), like `Run`.
- **Detects** installed state and **prompts per group** (each group skipped
  silently if absent). With `assumeYes`, every detected group is removed without
  prompting.
  1. **Kernel args** — `CurrentKargs` → `ParseSunbeamsKargs(cmdline, connector)`.
     Lists discovered tokens (grouped by connector). On confirm → `DeleteKargs`.
  2. **EDID firmware file** — `/etc/firmware/edid.bin`. On confirm → `os.Remove`.
  3. **systemd user service + xrandr script** — resolved via `SUDO_USER` home:
     `~/.config/systemd/user/virtual-display-modes.service` and
     `~/.local/bin/add-virtual-display-modes.sh`. On confirm → attempt
     `systemctl --user disable` (best-effort) and remove both files. If
     `SUDO_USER` is unset, this group is skipped with a note.
- Prints a closing reminder that a **reboot** is required for karg changes to
  take effect.

### Install changes (`internal/installer/installer.go`)

- **Re-run cleanup (self-heal):** at the start of `Run`, call `CurrentKargs` /
  `ParseSunbeamsKargs(cmdline, "")`. If any existing sunbeams kargs are found,
  list them and prompt to remove before injecting the newly-selected connector
  (reusing `DeleteKargs`). Declining proceeds with the existing append-only
  behavior.
- **Connected-connector guard:** after the connector is selected, if its
  `Status == "connected"`, print a warning that injecting will override the real
  monitor's EDID and require an explicit `y/N` confirm before proceeding.
  Disconnected connectors proceed unchanged.

### CLI + help (`cmd/sunbeams/main.go`, `cmd/sunbeams/help.go`)

- Dispatch `uninstall`:
  - `--connector NAME` (optional) — narrow removal to one connector.
  - `--yes` / `-y` — non-interactive; remove everything detected.
  - `-h`/`--help` renders subcommand help.
  - Calls `installer.Uninstall(connector, assumeYes, os.Stdin, os.Stdout)`.
- Add `uninstall` to `topLevelHelp` and `subcommandHelps`.

## Testing

TDD, failing-test-first (per CLAUDE.md). New coverage centers on the pure parser:

- `ParseSunbeamsKargs` table tests:
  - single connector → its three/one tokens;
  - accumulated multi-connector (the `HDMI-1` + `DP-2` regression) → all tokens;
  - `connector` narrowing → only the matching connector's tokens, no
    `firmware_class.path`;
  - ignores unrelated `video=<other>` and foreign `firmware_class.path`;
  - comma-merged `drm.edid_firmware=A:edid.bin,B:edid.bin` value parsed into both
    connectors.
- Existing golden/EDID/E2E tests remain untouched (no byte-format change).

The `Uninstall` driver, `CurrentKargs`, and `DeleteKargs` shell-outs are
integration-tested on live Bazzite hardware, consistent with the existing
`installer.Run` / `switcher` convention noted in CLAUDE.md.

## Edge Cases

- **Merged token narrowing:** a single `drm.edid_firmware=A:...,B:...` token
  cannot be surgically split by `--delete-if-present`. A full wipe deletes the
  whole token cleanly. When `--connector` targets one entry inside a merged
  token, detect this and instruct the user to run a full `uninstall` rather than
  silently mis-deleting. (In practice install appends separate tokens, so this
  is rare.)
- **`firmware_class.path` is shared:** only removed on full wipe, never on a
  `--connector`-narrowed run.
- **Reboot required:** karg changes (install and uninstall) take effect only
  after reboot; both paths say so.
