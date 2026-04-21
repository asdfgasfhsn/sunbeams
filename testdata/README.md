# testdata/

## `virtual_display_reference.bin`

Frozen golden EDID fixture. The Go implementation must produce this exact byte sequence when run against the defaults in `internal/config/defaults.toml`.

- **Size:** 768 bytes (6 blocks: 1 base + 5 CTA extensions)
- **SHA-256:** `66f8f00bea5c3d13224b8399267e6af8514331f9daaebb850a7289d480930b0f`
- **Generator:** produced by `sunbeams generate` against `internal/config/defaults.toml`.
- **Coverage:** 17 CTA VICs, 30 DTD-capable modes, 8 xrandr-overflow modes, YCbCr 4:2:0 capability map for 4K@60/100/120. See `docs/resolution_gaps.md` for the full coverage rationale.

### Regenerating

Do **not** regenerate casually. The fixture pins the shipping EDID — a change here changes what user GPUs actually advertise.

If the TOML config deliberately changes (new device, new mode, new EDID parameter), update the fixture as part of that PR:

```sh
./sunbeams generate --output-dir testdata
mv testdata/virtual_display.bin testdata/virtual_display_reference.bin
```

Document the change in the commit message, including which modes were added/removed and the new SHA-256.

### Verifying

```sh
make verify-golden
```

Compares a freshly-generated EDID against the reference and fails on any byte mismatch.
