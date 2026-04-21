# Validating the EDID

The easiest way to validate is to pass `--validate` to `sunbeams generate` — it runs `edid-decode` automatically if the tool is on PATH, streams its output, and fails the command if `edid-decode` reports any errors. On macOS or other systems where `edid-decode` is not installed, the flag prints a skip message and exits cleanly.

If you prefer to run it manually:

```bash
edid-decode virtual_display.bin
```

On Bazzite/Fedora:

```bash
sudo dnf install edid-decode
```

## Golden fixture

The EDID bytes shipped by `sunbeams generate` are pinned by `testdata/virtual_display_reference.bin` — a byte-for-byte fixture the test suite compares against. If you change `internal/config/defaults.toml` (add a device/mode, tweak EDID parameters), the `make verify-golden` check will fail until you regenerate the fixture deliberately. See [`../testdata/README.md`](../testdata/README.md) for the procedure, and [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for how to land a device addition cleanly.
