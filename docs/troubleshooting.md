# Troubleshooting

**Black screen after reboot:** SSH in and re-enable your physical display:

```bash
export XDG_RUNTIME_DIR=/run/user/1000
export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus
export WAYLAND_DISPLAY=wayland-0
kscreen-doctor output.DP-1.enable
```

`sunbeams switch off` does the same thing if Wayland env vars are already set.

**Virtual display not appearing:** Check that the port name in your kernel args matches an actual port on your GPU. Use `cat /sys/class/drm/card*/status` to verify.

**Wrong port for your GPU:** NVIDIA, AMD, and Intel GPUs name ports differently. NVIDIA may use `DP-0` instead of `DP-1`. Check `ls /sys/class/drm/`.

**Modes not available after adding via xrandr:** Make sure you're adding modes to the correct output name. The xrandr output name must match the DRM connector.

**`add_custom_modes.sh` reports success but new modes don't appear:** You're almost certainly in a Wayland session. `xrandr --newmode` only registers modes with the X server's RandR extension, which Plasma/Wayland ignores. Log out, select **Plasma (X11)** at the SDDM login screen, log back in, and re-run the script. The high-bandwidth modes (4K@144, ultrawide@144, MacBook native@120) are only reachable from X11. 4K@120Hz is encoded as VIC 118 and works on Wayland without the script.

**Only low resolutions available (up to ~1920×1200):** The EDID is likely missing the HDMI Vendor-Specific Data Block. Without it, the kernel assumes a 165 MHz TMDS clock limit (HDMI 1.0 baseline). Regenerate the EDID with `sunbeams generate` which includes both HDMI VSDB and HF-VSDB blocks.

**Checking Sunshine logs:** `sunbeams switch` writes structured log lines to stderr with a `[sunbeams LEVEL HH:MM:SS]` prefix — Sunshine captures these into its own log. Every run includes the requested values, resolved connectors (with source), the mode-match decision, the exact `kscreen-doctor` commands issued, and a `kscreen-doctor -o` read-back of the affected connector.

```bash
# Log file
cat ~/.config/sunshine/sunshine.log

# Or via journalctl
journalctl --user -u sunshine -f
```

**Sunshine fires the prep command but nothing visibly changes:** Re-run with `SUNBEAMS_DEBUG=1` set in Sunshine's environment (or temporarily from a shell). That adds the `XDG_RUNTIME_DIR`, `WAYLAND_DISPLAY`, `DBUS_SESSION_BUS_ADDRESS`, `XDG_SESSION_TYPE`, and `QT_QPA_PLATFORM` values sunbeams actually passed to `kscreen-doctor`, plus the command's stdout/stderr. If any of those are unset or pointing at the wrong user runtime dir, the kscreen call will silently no-op.

**Resolution doesn't match what the client requested:** Look for the `mode match:` log line. `(exact)` means your request was in the configured list verbatim; `(snapped refresh…)` means the resolution was found but the refresh was adjusted by Δ Hz; `(no resolution hit — closest overall…)` means no `[[modes]]` entry covered that resolution and sunbeams fell back to the nearest mode with a warning. Add a `[[modes]]` entry for the resolution you want, then re-run `sunbeams generate` (see [customizing.md](customizing.md)).

**`--hdr` / `SUNSHINE_CLIENT_HDR=true` doesn't enable HDR:** `kscreen-doctor` has no command-line HDR toggle. sunbeams logs the request explicitly (`HDR requested — logged only…`) but cannot apply it. Enable HDR per-output in KDE Display Settings, which persists across switches.

**Bazzite Deck / Game Mode / gamescope:** Not supported. `kscreen-doctor` is not available inside gamescope, so `sunbeams switch` cannot run. Boot into Desktop Mode (KDE Plasma / Wayland) before streaming. The Bazzite Deck images themselves are not a tested target regardless of which mode you boot into — see [Supported platforms](../README.md#supported-platforms). For community gamescope experiments, see the [gamescope workaround](https://gist.github.com/iamthenuggetman/6d0884954653940596d463a48b2f459c#gistcomment-5882590), but no support is provided.

**GNOME-based images (bazzite-gnome, Aurora, Bluefin, Silverblue):** Not supported. The display switcher is KDE/`kscreen-doctor` specific; there is no GNOME/Mutter equivalent in sunbeams today.

## Gaming Mode

### Sleep/wake leaves the physical disconnected

If the system sleeps mid-stream, Sunshine cannot run its Undo command and the physical connector stays force-off. Manually re-enable from another machine via SSH:

```bash
ssh user@bazzite-host -- sudo /usr/local/sbin/sunbeams-drm-force on HDMI-A-1
```

This is a kernel-level limitation, not a sunbeams bug.

### `sudo: a password is required`

The sudoers fragment was not installed or was overwritten. Re-run:

```bash
sudo sunbeams install --with-gaming --physical=HDMI-A-1
```

### `multiple debugfs paths for HDMI-A-1`

Multi-GPU systems are not supported in v1. The helper refuses to guess which GPU's connector to toggle.

### Black screen returning to desktop after streaming

The gaming-mode `switch off` resets the virtual monitor to a safe mode (default 1920x1080@60) before re-enabling the physical, specifically to avoid this. If you've added `--no-safe-revert`, this is the cost.

### NVIDIA proprietary driver

The DRM debugfs `force` interface is verified on AMD; behavior on NVIDIA's proprietary stack is unverified. The helper fails cleanly with exit code 3 ("no debugfs path") if your driver doesn't expose the file.
