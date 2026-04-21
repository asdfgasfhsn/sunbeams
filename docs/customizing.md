# Customizing Devices and Modes

All device and resolution definitions live in `internal/config/defaults.toml`, which is the single source of truth embedded in the binary. To customize, generate a config file and edit it:

```bash
sunbeams config init
# Edit ~/.config/sunbeams/config.toml
```

To add a new device, add an entry to the `[[devices]]` array:

```toml
[[devices]]
slug = "my-monitor"
label = "My Ultrawide Monitor"
width = 2560
height = 1080
max_refresh = 144
hdr = false
```

Then add the corresponding resolutions to `[[modes]]`:

```toml
[[modes]]
width = 2560
height = 1080
refresh = 144
description = "Ultrawide 1080p — My Monitor"

[[modes]]
width = 2560
height = 1080
refresh = 60
description = "Ultrawide 1080p — 60Hz fallback"
```

Re-run `sunbeams generate` to regenerate the EDID binary and helper files. The new device will automatically appear in `sunbeams devices`.
