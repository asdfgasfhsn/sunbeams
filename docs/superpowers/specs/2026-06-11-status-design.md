# `sunbeams status` — Design

**Date:** 2026-06-11
**Status:** Approved (pending implementation)

## Problem

There is no way to ask sunbeams which display connectors currently carry the
virtual EDID. After the connector-accumulation bug (see the uninstall design),
a user could not see that *both* `HDMI-1` and `DP-2` were injected. A read-only
diagnostic subcommand would have surfaced that immediately, and more generally
answers "is my virtual display set up, and is it live yet?"

## Goal

Add `sunbeams status`: a read-only, **no-root** command that reports, per
connector, whether the sunbeams EDID is configured, active this boot, and
actually loaded — distinguishing "will apply after reboot" from "is applied".

## Non-Goals

- No mutation of any system state (that's `install`/`uninstall`).
- No change to EDID generation or the golden fixture.
- No mocking of `rpm-ostree` (consistent with existing convention); the sysfs
  and `/proc/cmdline` reads ARE testable via injected paths, and are covered.

## State Signals (all world-readable, no root)

- **Configured** — a sunbeams `drm.edid_firmware=<conn>:edid.bin` karg is
  present in `rpm-ostree kargs` (the persistent intent; same source
  install/uninstall act on).
- **This boot** — the same karg is present in `/proc/cmdline` (what the running
  kernel actually booted with). Comparing this against the configured set is
  what distinguishes "reboot pending" from "live".
- **EDID loaded** — the connector's live `/sys/class/drm/<card-conn>/edid` bytes
  equal the installed `/etc/firmware/edid.bin` (ground truth — exactly what the
  firmware loader injects).

If `rpm-ostree` is unavailable, fall back to `/proc/cmdline` as the configured
source and note that reboot-pending detection is unavailable.

## Architecture

Lives in the **`installer` package** (`internal/installer/status.go`), reusing
its unexported `scanConnectorsAt`, the `FirmwareDir`/`EDIDName` constants, and
`CurrentKargs`/`ParseSunbeamsKargs`. Reporting on install state belongs with the
code that owns that state.

### Data model

```go
type ConnectorStatus struct {
	Name       string // "DP-2"
	Connected  bool   // sysfs status == "connected"
	Configured bool   // sunbeams karg in rpm-ostree kargs
	BootActive bool   // sunbeams karg in /proc/cmdline
	EDIDLoaded bool   // sysfs edid == /etc/firmware/edid.bin
	Verdict    string // synthesized (see classify)
}

type Report struct {
	Connectors      []ConnectorStatus
	FirmwarePresent bool
	FirmwareBytes   int    // size of /etc/firmware/edid.bin, 0 if absent
	RebootDetectable bool  // false when rpm-ostree was unavailable
}
```

### Pure functions (unit-tested)

- **`classify(cs ConnectorStatus, firmwarePresent bool) string`** — verdict:
  - `Configured && BootActive && EDIDLoaded` → `✓ active`
  - `Configured && !BootActive` → `⏳ configured — reboot pending`
  - `Configured && BootActive && !EDIDLoaded` → `⚠ booted but EDID not loaded (connector disconnected or KMS skipped it)`
  - `!firmwarePresent && Configured` → `⚠ no /etc/firmware/edid.bin — install incomplete`
  - `EDIDLoaded && !Configured` → `active but not configured (orphan — re-run install or uninstall)`

  Evaluation order: the firmware-missing check takes precedence over the
  active/loaded checks (you cannot be "active" with no firmware file to load);
  the orphan check applies only when `!Configured`.
- **`connectorsFromKargs(kargs []string) []string`** — extracts connector names
  from `drm.edid_firmware=<conn>:edid.bin` tokens (handles merged comma form),
  de-duplicated, in first-seen order.

### I/O layer (thin)

- **`Status(sysfsRoot, cmdlinePath, firmwarePath string) (Report, error)`** —
  orchestrator. All three paths are injectable (firmwarePath too, so the byte
  comparison is testable without touching the real `/etc/firmware`); the CLI
  passes `"/sys/class/drm"`, `"/proc/cmdline"`, and the installed firmware path.
  1. `configured := connectorsFromKargs(ParseSunbeamsKargs(<rpm-ostree or fallback>, ""))`
  2. `boot := connectorsFromKargs(ParseSunbeamsKargs(<read cmdlinePath>, ""))`
  3. `firmware, _ := os.ReadFile(/etc/firmware/edid.bin)` → presence + size
  4. Walk `sysfsRoot` once: for each `card*-{HDMI,DP}*` entry, capture connector
     name, `status`, and `edid` bytes. (Dedicated walk — like `scanConnectorsAt`
     but also reads `edid`; the dir name is needed to locate the edid file.)
  5. For the union of `configured ∪ boot` (and any connector whose edid matches
     but isn't configured — orphan), build a `ConnectorStatus`, compare edid
     bytes to firmware bytes, and call `classify`.
  6. Sort connectors by name for stable output.
- A missing `sysfsRoot` (e.g. macOS) returns a sentinel so the CLI can print the
  unsupported-platform message and exit 0.

### CLI & output (`cmd/sunbeams/main.go`, `help.go`)

`sunbeams status` — no flags, no root. Output:

```
EDID injection status   (firmware: /etc/firmware/edid.bin, 768 bytes)

  CONNECTOR   CONNECTED      CONFIGURED   THIS BOOT   EDID LOADED   STATE
  DP-2        disconnected   yes          yes         yes           ✓ active
  HDMI-A-1    connected      yes          no          —             ⏳ reboot pending

2 connector(s) carry the sunbeams EDID.
```

Degradation:
- Not Linux / no `/sys/class/drm` → `display status is only available on Linux with DRM/KMS` and exit 0.
- No configured/active/orphan connectors → `No sunbeams EDID injection found.`
- When `rpm-ostree` was unavailable, append a one-line note that reboot-pending
  state could not be determined.

Add `status` to `topLevelHelp` COMMANDS, COMMON INVOCATIONS, and a
`subcommandHelps["status"]` entry. `status --help` renders the standard page.

## Testing

- **Pure:** `classify` (every verdict branch incl. precedence), `connectorsFromKargs`
  (single, merged, multi, none).
- **I/O:** `Status` against a temp `sysfsRoot` containing fake
  `card0-DP-2/{status,edid}` and `card0-HDMI-A-1/{status,edid}` files, a temp
  cmdline file, and a temp firmware file — asserts the full `Report` including
  real byte comparison and verdicts. (More coverage than the installer
  shell-outs because these reads are injectable.) Note: `Status` reads the
  firmware file from the real `/etc/firmware` path via the package constants; the
  byte-comparison logic is exercised by factoring the compare into a helper that
  takes the firmware bytes as an argument, so the test supplies them directly.
- **CLI:** `status --help` added to `TestHelp_TopLevel` (keyword) and
  `TestHelp_Subcommands`; a `status` smoke that exits 0 off-Linux.

## Edge Cases

- **Empty/short sysfs edid:** kernel exposes a 0-byte `edid` when nothing is
  loaded; byte comparison against the 768-byte firmware naturally yields
  `EDIDLoaded == false`.
- **Connector configured but absent from sysfs** (hardware removed): listed with
  `Connected == false`, `EDIDLoaded == false` → `booted but EDID not loaded` (or
  `reboot pending` if not in `/proc/cmdline`).
- **Firmware file missing but kargs present:** every configured connector gets
  the `install incomplete` verdict.
