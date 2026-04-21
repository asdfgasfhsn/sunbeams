# Contributing

Thanks for considering a contribution. sunbeams is intentionally small — a single static Go binary targeting Bazzite + KDE Plasma — and the goal is to keep it that way.

## Development setup

Everything you need is declared in `flake.nix`. With [nix](https://nixos.org/download/) installed:

```sh
nix develop       # drops you into a shell with go, golangci-lint, goreleaser, edid-decode
```

Without nix, you need Go 1.22+ on PATH. The repo uses Go 1.26.1 locally; CI builds on 1.26.

### Common tasks

```sh
make              # list all targets
make build        # compile sunbeams with ldflags-injected version
make test         # run all tests
make verify-golden # regenerate EDID and diff against the golden fixture
make check        # fmt + lint + tests + golden verification
make snapshot     # goreleaser snapshot (cross-compile amd64 + arm64)
```

Every PR must pass `make check`.

## Adding a device

The most common contribution is supporting a new streaming target.

1. Look up the device's **native resolution** and **maximum refresh rate**.
2. Open `internal/config/defaults.toml` and add an entry to the `[[devices]]` list with a stable `slug`, a human-readable `label`, `width`, `height`, `max_refresh`, and `hdr` flag.
3. Add the mode (and a 60 Hz fallback) to the `[[modes]]` list if the resolution isn't already present.
4. Run `make verify-golden` — this will **fail** because adding a mode changes the EDID bytes.
5. Regenerate the golden fixture deliberately:
   ```sh
   ./sunbeams generate --output-dir testdata
   mv testdata/virtual_display.bin testdata/virtual_display_reference.bin
   ```
6. Re-run `make check`; all tests should pass.
7. Submit a PR with:
   - Device name and vendor spec page URL
   - Your TOML diff
   - The new EDID size (if the extension-block count changed)
   - Ideally, `edid-decode testdata/virtual_display_reference.bin` output confirming the new mode is advertised

## Code guidelines

- **Minimal dependencies.** Stdlib + `github.com/BurntSushi/toml` + `github.com/stretchr/testify`. Justify any new dependency in the PR description.
- **No CGO.** Release binaries are statically linked (`CGO_ENABLED=0`).
- **Binary packing uses `encoding/binary`** for multi-byte values; bit-packed DTD/CTA fields use manual byte assignment.
- **Functions that produce EDID sub-structures return `[]byte`**, not `*[]byte` or `bytes.Buffer`.
- **Pixel clock arithmetic:** `int(math.Round(float64(hTotal*vTotal*refreshHz) / 1000.0))` — never integer division.
- **Errors wrap** with `%w` so callers can `errors.Is` / `errors.As`.
- **No panics in library code.** `panic` is only acceptable in `EncodeManufacturerID` where the TOML schema guarantees valid input.

## Testing discipline

Write the failing test first, observe the failure, implement the minimum, observe the pass, commit. No exceptions for byte-level EDID code — the golden-file test catches structural drift, but each primitive needs its own unit test so regressions are debuggable.

## Issue reports

If a generated EDID doesn't work on your hardware, please include:

- GPU vendor / model
- `kscreen-doctor -o` output
- `edid-decode virtual_display.bin` output (install via your package manager, or run `sunbeams generate --validate`)
- Which resolution is failing
- Whether you're using Desktop Mode or Game Mode (gamescope)
- `sunbeams version` output so we know which release you're on

## Release process

Maintainers: see [RELEASING.md](./RELEASING.md).
