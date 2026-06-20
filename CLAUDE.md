# Virtual Display EDID Generator

Go CLI that generates custom EDID binaries for headless Linux game streaming with Sunshine/Moonlight, plus a `kscreen-doctor` wrapper for display switching and a guided Bazzite installer. Targets Bazzite (Fedora Atomic) but works on any Linux with DRM/KMS.

## Stack

- Go 1.24 (stdlib + `github.com/BurntSushi/toml` + `github.com/stretchr/testify` for tests)
- Target OS: Bazzite Desktop (rpm-ostree based Fedora Atomic)
- Display stack: KDE Plasma / Wayland / kscreen-doctor
- Streaming: Sunshine server + Moonlight clients
- Build/toolchain: provided by `flake.nix` (`nix develop` or direnv)

## Commands

Run `make` (no args) to list all targets with descriptions. Common ones:

- `make build` — build the `sunbeams` binary with ldflags-injected version/commit/date
- `make test` — run all tests (34 tests across 8 packages)
- `make test-race` — run tests with the race detector
- `make lint` — run `golangci-lint`
- `make verify-golden` — regenerate EDID and diff against `testdata/virtual_display_reference.bin`
- `make check` — fmt + lint + tests + golden-file verification
- `make snapshot` — build a goreleaser snapshot (cross-compile amd64 + arm64)

CLI subcommands: `sunbeams generate|switch|devices|modes|install|uninstall|status|config|version`. Run `./sunbeams` with no args for usage.

## Key Files / Packages

- `cmd/sunbeams/main.go` — CLI dispatch for all subcommands
- `internal/edid/` — EDID primitives: constants, timing (CVT-RB v1/v2), DTD packing, descriptors, standard timing, CTA-861 data blocks, base block, CTA extension block, checksum
- `internal/config/` — `defaults.toml` embedded via `go:embed`; single source of truth for devices, modes, EDID params, CTA VICs, standard timings
- `internal/userconfig/` — merges `~/.config/sunbeams/config.toml` over embedded defaults
- `internal/generate/` — orchestrator (`Generate`) producing the EDID bytes + xrandr modeline + helper scripts (`add_custom_modes.sh`, `sunshine_commands.txt`)
- `internal/switcher/` — `kscreen-doctor` wrapper with Wayland/D-Bus env defaults, `SwitchOn`/`SwitchOff`, `FindBestMode`
- `internal/installer/` — DRM connector scan (`/sys/class/drm`), rpm-ostree kargs injection, systemd user service template, guided `Run` driver
- `internal/platform/` — Wayland/Qt env helper
- `testdata/virtual_display_reference.bin` — frozen golden EDID; every change must preserve byte-for-byte parity unless the TOML config was deliberately updated (see `testdata/README.md`)

## Architecture

The EDID binary format has hard constraints:
- Each block is exactly 128 bytes
- Detailed Timing Descriptors (DTDs) max pixel clock is 655.35 MHz (uint16 × 10kHz)
- Resolutions exceeding this limit (e.g. 4K@120Hz+) need xrandr `--newmode` as a workaround
- Base block holds 2 DTDs + monitor name + range limits
- CTA-861 extension blocks hold VIC codes, HDR metadata, colorimetry, and additional DTDs

`Generate()` splits configured modes into DTD-capable (fit in the 655 MHz limit) vs xrandr-required; emits the DTD set into the base block (slot 1: first 4K@60, slot 2: 3440x1440@60) and remaining CTA extension blocks; writes the xrandr script for the overflow modes.

## Bazzite Filesystem

- `/usr` is immutable — never write firmware there
- EDID installs to `/etc/firmware/` (writable, persists across updates, available early in boot)
- The `firmware_class.path=/etc/firmware` kernel parameter redirects the firmware loader there
- `/etc/firmware` is preferred over `/usr/local/lib/firmware` because it avoids early KMS race conditions

## Conventions

- Minimal external dependencies — TOML parser + testify; everything else is stdlib (`encoding/binary`, `os/exec`, `embed`, `flag`, `math`, `os/user`)
- Static binaries (`CGO_ENABLED=0`), Linux amd64 + arm64 via goreleaser
- All binary packing uses `encoding/binary` for multi-byte values; bit-packed DTD/CTA fields use manual byte assignment
- Pixel clock calculation: `int(math.Round(float64(hTotal*vTotal*refreshHz) / 1000.0))` — **never** integer division (Python parity depends on this)
- Functions that build EDID sub-structures return `[]byte`, never `*[]byte` or `bytes.Buffer`
- Checksum: `(256 - sum(block_bytes) % 256) % 256`
- CLI output: `fmt.Printf` for user-facing progress, `fmt.Fprintln(os.Stderr, ...)` for errors

## Testing

**Discipline:** Write the failing test first, observe the failure, implement the minimum, observe the pass, commit. No exceptions for byte-level EDID code — the golden-file test will catch structural drift, but each primitive needs its own unit test to keep debugging tractable.

**Byte-level parity tests.** When implementing a function that produces bytes (DTDs, descriptors, CTA data blocks, standard timings), add a `TestFoo_Bytes` test that asserts the exact hex dump. The original Python reference is no longer in the repo; if you need to re-derive expected bytes, retrieve it from git history (`git log --all --full-history -- legacy/generate_edid.py`) or cross-check against `edid-decode` output on the golden fixture.

**Golden-file test.** `internal/generate/generate_test.go::TestGoldenEDID` compares the full 768-byte output byte-for-byte against `testdata/virtual_display_reference.bin`. Any change that breaks this test must either fix the regression or re-freeze the reference (requires explicit justification — the reference is the shipping EDID). Run via `make verify-golden`.

**Per-block sanity.** `TestAllBlockChecksumsValid` confirms every 128-byte block sums to 0 mod 256 independently of the golden fixture.

**Integration.** `cmd/sunbeams/main_test.go::TestGenerateE2E` builds the binary, runs `sunbeams generate`, and re-verifies the golden fixture from the compiled output.

**External validation.** On Linux with `edid-decode` on PATH (the nix flake provides it), run `sunbeams generate --validate` or `edid-decode testdata/virtual_display_reference.bin` to inspect the parsed output. On macOS `--validate` skips cleanly with a message.

**Untested paths (intentional).** `installer.Run` beyond the root check, and the kscreen-doctor shell-outs in `switcher.go`, are integration-tested on live Bazzite hardware rather than mocked. Adding coverage there would require extracting `ScanConnectors`/`InjectKargs`/`runKScreen` behind interfaces — worth doing if the paths become defect-prone.
