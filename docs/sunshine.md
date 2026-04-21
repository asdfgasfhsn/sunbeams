# Sunshine Configuration

Add these as **global prep commands** in Sunshine → Configuration → General:

```
Do command:   sunbeams switch on
Undo command: sunbeams switch off
```

Sunshine passes `SUNSHINE_CLIENT_WIDTH`, `SUNSHINE_CLIENT_HEIGHT`, `SUNSHINE_CLIENT_FPS`, and `SUNSHINE_CLIENT_HDR` to all Do/Undo commands automatically. The binary reads these and finds the closest matching mode from the configured mode list — so even if a client requests a resolution that doesn't exactly match, it snaps to the best available mode. All display changes (disable physical, enable virtual, set mode) are applied atomically in a single `kscreen-doctor` call to avoid race conditions.

You can override any value from the command line:

```bash
# Force a specific resolution regardless of what the client requests
sunbeams switch on --width 3840 --height 2160 --fps 120 --hdr
```

Set `VIRTUAL_OUTPUT` and `PHYSICAL_OUTPUT` environment variables if your connectors differ from the defaults (`HDMI-A-1` / `DP-1`).

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SUNSHINE_CLIENT_WIDTH` | — | Client width (set by Sunshine) |
| `SUNSHINE_CLIENT_HEIGHT` | — | Client height (set by Sunshine) |
| `SUNSHINE_CLIENT_FPS` | — | Client frame rate (set by Sunshine) |
| `SUNSHINE_CLIENT_HDR` | — | `true` if client requested HDR (set by Sunshine) |
| `VIRTUAL_OUTPUT` | `HDMI-A-1` | DRM connector for the virtual display |
| `PHYSICAL_OUTPUT` | `DP-1` | DRM connector for the physical display |
| `SUNBEAMS_DEBUG` | unset | Set to `1` or `true` to log the Wayland/D-Bus session env passed to `kscreen-doctor` plus unset Sunshine vars. Useful when the switcher runs but the display doesn't change. |

## Logging

`sunbeams switch` logs every step to stderr with a `[sunbeams LEVEL HH:MM:SS]` prefix. A typical on/off cycle captures:

- The requested resolution, refresh, and HDR flag
- The resolved virtual/physical connector names and where each came from (flag / env / default)
- The `SUNSHINE_CLIENT_*` values Sunshine supplied (values logged at `info`, unset keys at `debug`)
- The match decision: exact hit, refresh-snapped (with Δ Hz), or closest-overall fallback (with Δ W/H/R) — plus a warning if no configured resolution matched
- The exact `kscreen-doctor` command issued, and a separate log line per retry step if the atomic call fails
- A readback of `kscreen-doctor -o` filtered to the affected connector after success, so you can confirm the current mode

When HDR is requested via `--hdr` or `SUNSHINE_CLIENT_HDR=true`, sunbeams logs the request and explicitly notes that it is **not** toggled by `kscreen-doctor` — HDR must be enabled in KDE Display Settings separately.

With `SUNBEAMS_DEBUG=1` you also get the `XDG_RUNTIME_DIR` / `WAYLAND_DISPLAY` / `DBUS_SESSION_BUS_ADDRESS` / `XDG_SESSION_TYPE` / `QT_QPA_PLATFORM` / `DISPLAY` values sunbeams passed to every `kscreen-doctor` invocation, plus the command's stdout/stderr. This is the first thing to enable when diagnosing "Sunshine fires the prep command but nothing happens."
