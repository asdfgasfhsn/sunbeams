# Auto-Target the Live Virtual Display — Design

**Date:** 2026-06-11
**Status:** Approved (pending implementation)

## Problem

`switch on`/`off` choose which connector to drive purely from `Outputs.resolve()`:
`Outputs.Virtual` (never set by the CLI) → `VIRTUAL_OUTPUT` env → hardcoded
default `HDMI-A-1` (and `DP-1` for physical). The CLI passes an empty
`switcher.Outputs{}`, so the only way to target a non-default connector (e.g.
`DP-2`) is to thread `VIRTUAL_OUTPUT`/`PHYSICAL_OUTPUT` env vars through every
Sunshine Prep Command. The default mismatches real hardware and the physical
disable can fail on headless boxes. The app should instead detect where the
virtual EDID actually lives and target it automatically on every execution.

## Goals

- `switch on`/`off` auto-detect the virtual connector from the running system,
  with no env var required.
- Detection is deterministic; ambiguity fails loudly rather than guessing.
- Physical handling becomes robust (auto-detected; skipped when headless).
- Establish a clean `internal/drm` layer for read-only DRM/sysfs/karg knowledge,
  shared by `installer` and `switcher`.

## Non-Goals

- No new CLI flags (a future `--virtual`/`--physical` can slot into the same
  precedence; not built here).
- No change to EDID generation — golden fixture stays byte-identical.
- No change to the `rpm-ostree` shell-outs' behavior.

## Decisions (from brainstorming)

1. **Detection signal:** the `drm.edid_firmware` connector(s) in `/proc/cmdline`
   (boot-stable, no root), disambiguated by live `/sys/.../edid` byte-match
   against `/etc/firmware/edid.bin` when more than one is configured.
2. **Precedence:** explicit `Outputs.Virtual` → `VIRTUAL_OUTPUT` env →
   auto-detect. The old hardcoded `HDMI-A-1`/`DP-1` defaults are removed.
3. **Detection failure:** zero matches, or still-ambiguous after the EDID
   cross-check → **fail with a clear, actionable error** (no silent fallback).
4. **Physical:** auto-detect = every currently-`connected` connector that isn't
   the virtual; skip the disable entirely when none are connected (headless).
   `PHYSICAL_OUTPUT` still overrides.
5. **Structure:** extract a shared `internal/drm` package (chosen over
   `switcher`-imports-`installer`).

## Architecture

### New `internal/drm` package — read-only DRM/sysfs/karg primitives

No `rpm-ostree`, no dependency on `installer`/`switcher` (so no import cycle).
Moved out of `installer`, with export-casing where now cross-package:

- `connector.go`: `Connector{Name,Status}`, `ScanConnectors()`,
  `scanConnectorsAt(root)` (moved from `installer/connectors.go`).
- `sysfs.go`: `SysfsConn{Status,EDID}`, `ScanConnectorEDID(root)`, `ErrNoSysfs`
  (moved from `installer/status.go`; `sysfsConn`→`SysfsConn`,
  `scanConnectorEDID`→`ScanConnectorEDID`).
- `kargs.go`: `BuildKargs`, `ParseSunbeamsKargs`, `ConnectorsFromKargs`
  (`connectorsFromKargs`→exported) + constants `FirmwareDir` (`/etc/firmware`)
  and `EDIDName` (`edid.bin`) (moved from `installer`).
- `detect.go`: **new** `DetectVirtual(sysfsRoot, cmdlinePath, firmwarePath string) (string, error)`.

`DetectVirtual` logic:
1. `configured := ConnectorsFromKargs(ParseSunbeamsKargs(<read cmdlinePath>, ""))`.
2. `len == 0` → error: `no virtual display configured: no sunbeams drm.edid_firmware in <cmdlinePath> — run 'sudo sunbeams install' (and reboot)`.
3. `len == 1` → return it (boot-stable; need not be presenting right now).
4. `len > 1` → read firmware; if missing/empty → error naming the candidates and
   suggesting `VIRTUAL_OUTPUT`. Else `ScanConnectorEDID`, keep candidates whose
   `EDID` is non-empty and `bytes.Equal` the firmware. Exactly one → return it;
   otherwise → error: `could not determine virtual display: <c1>, <c2> carry the sunbeams EDID — set VIRTUAL_OUTPUT to pick one, or run 'sunbeams status'`.

All paths injectable; fully unit-testable with temp dirs.

### `installer` (now imports `drm`)

Keeps the **mutating / orchestration** code: `Run`, `Uninstall`, `Status` +
`classify` + `buildReport` + `Report`/`ConnectorStatus`, and the `rpm-ostree`
shell-outs `InjectKargs`/`CurrentKargs`/`DeleteKargs`. Every reference to a moved
primitive/constant becomes `drm.*` (e.g. `drm.FirmwareDir`, `drm.EDIDName`,
`drm.ScanConnectors`, `drm.ParseSunbeamsKargs`, `drm.ConnectorsFromKargs`,
`drm.ScanConnectorEDID`, `drm.SysfsConn`, `drm.ErrNoSysfs`, `drm.BuildKargs`).
`buildReport` now takes `map[string]drm.SysfsConn`. Behavior is unchanged.

### `switcher` (now imports `drm`)

`resolve()` is replaced by `resolveOutputs(outs Outputs) (virt string, phys []string, virtSrc, physSrc string, err error)`:

- **Virtual:** `outs.Virtual` (`"flag"`) → `VIRTUAL_OUTPUT` (`"env:VIRTUAL_OUTPUT"`)
  → `detectVirtual()` (`"auto"`). If detect errors, return the error.
- **Physical:** `outs.Physical` (`"flag"`, single) → `PHYSICAL_OUTPUT`
  (`"env:PHYSICAL_OUTPUT"`, single) → auto: `scanConnectors()` filtered to
  `Status == "connected" && Name != virt`, sorted (`"auto"`). A scan error is
  non-fatal (warn, empty list). Empty list ⇒ disable step skipped.

Detection seams are package-level vars for test stubbing:
```go
var detectVirtual = func() (string, error) {
    return drm.DetectVirtual("/sys/class/drm", "/proc/cmdline",
        filepath.Join(drm.FirmwareDir, drm.EDIDName))
}
var scanConnectors = drm.ScanConnectors
```

`SwitchOn(cfg, outs, w,h,fps,hdr)`:
- resolve (return err on detect failure);
- args: `output.<p>.disable` for each `p` in `phys`, then `output.<virt>.enable`,
  then `output.<virt>.mode.<match>`;
- same atomic-then-retry structure as today (retry disables each physical,
  enables virtual, sleeps, sets mode).

`SwitchOff(outs)`: resolve; `output.<virt>.disable` then `output.<p>.enable` for
each `p`. Empty `phys` ⇒ only the virtual is disabled.

Logging reports the resolved virtual + source and the physical list + source.

### `cmd/sunbeams/main.go`

`runStatus` swaps `installer.FirmwareDir/EDIDName` → `drm.*` and
`installer.ErrNoSysfs` → `drm.ErrNoSysfs` (add the `drm` import). `runSwitch`
still passes `switcher.Outputs{}`; no CLI surface change. The connector names
produced by detection (`/sys/class/drm` names like `DP-2`) are the same strings
kscreen-doctor uses for its outputs, so they feed `output.<name>.*` directly —
consistent with how the current hardcoded `HDMI-A-1`/`DP-1` were already used.

## Testing

- **`drm` package:** migrate the existing pure tests (`TestBuildKargs`,
  `TestParseSunbeamsKargs`, `TestConnectorsFromKargs`, `TestScanConnectorEDID*`,
  `TestScanConnectorsReadsSysfs`) into `internal/drm`, updated for the new
  package + exported names. Add `TestDetectVirtual`: single candidate; >1
  disambiguated by EDID match; >1 ambiguous → error; 0 → error; missing/empty
  firmware with >1 candidates → error.
- **`installer`:** remaining tests (`classify`, `buildReport`, `Status`,
  `Run`/`Uninstall` root gates, kargs shell-out helpers) stay green against
  `drm.*`. `buildReport`/`Status` tests update type/symbol references only.
- **`switcher`:** new `TestResolveOutputs` cases with stubbed
  `detectVirtual`/`scanConnectors`: auto virtual + auto physical; `VIRTUAL_OUTPUT`
  overrides auto; `PHYSICAL_OUTPUT` overrides; headless (no connected → empty
  physical); multi-physical (two connected non-virtual → both); detect error
  propagates; physical scan error is non-fatal.
- **Full:** `make check` (lint, golden parity) + `make test-race` stay green.

## Edge Cases

- **Headless:** `phys` empty → `switch on` enables virtual + sets mode, no
  disable; `switch off` only disables virtual.
- **Accumulation (>1 configured):** disambiguated by EDID byte-match; if still
  ambiguous, hard error pointing at `VIRTUAL_OUTPUT` / `sunbeams status`.
- **Pre-reboot / not yet installed:** `DetectVirtual` errors (no
  `drm.edid_firmware` in `/proc/cmdline`), telling the user to install + reboot.
- **`kscreen-doctor` name vs sysfs name:** assumed identical (both are the DRM
  connector name); documented assumption, matching current behavior.
- **Multiple connected physicals:** all are disabled on `on` and re-enabled on
  `off` (more correct than the old single-physical model).
