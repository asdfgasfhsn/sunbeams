# How It Works

The generated EDID contains:

- **Base EDID 1.4 block** (128 bytes): preferred 4K@60Hz timing, ultrawide timing, monitor name ("VirtStream"), and wide display range limits (up to 1700 MHz pixel clock, 300 Hz refresh)
- **CTA-861 extension blocks** (128 bytes each): VIC codes for standard resolutions, HDR Static Metadata (HDR10, HLG, PQ), BT.2020/DCI-P3 colorimetry, and additional Detailed Timing Descriptors for non-standard resolutions
- **HDMI Vendor-Specific Data Blocks**: an HDMI VSDB (IEEE 00-0C-03) declaring 600 MHz max TMDS clock and deep color support, plus an HDMI Forum VSDB (IEEE C4-5D-D8) declaring HDMI 2.1 FRL at 48 Gbps. Without these, the Linux DRM driver defaults to a 165 MHz TMDS clock limit (HDMI 1.0), which caps selectable modes at ~1920×1200@60Hz
- **xrandr companion script**: for modes whose pixel clock exceeds the DTD format's 655 MHz limit — the EDID's range limits still cover them, but the modes must be added via `xrandr --newmode`

## EDID Pixel Clock Limitation

The EDID Detailed Timing Descriptor stores pixel clock as a 16-bit value in 10 kHz units, capping at 655.35 MHz. Resolutions that exceed this (4K@120Hz ~1024 MHz, 4K@144Hz ~1229 MHz, ultrawide@144Hz ~739 MHz, MacBook native@120Hz ~738 MHz) cannot be encoded as DTDs. The workaround is:

1. The EDID's range limits are set wide enough to cover these modes
2. 4K@120Hz is also included as VIC code 118 (HDMI 2.1 standard), 4K@100Hz as VIC 119 — both work natively under Wayland with no extra setup
3. `add_custom_modes.sh` adds the remaining modes (4K@144, ultrawide@144, MacBook native@120) via xrandr — **X11 only**

> **Wayland limitation:** `xrandr --newmode`/`--addmode` talk to the X server's RandR extension. Plasma on Wayland (the Bazzite Desktop default) ignores that state, so the script has no effect there. XWayland will run the commands without error but kscreen-doctor, KWin, and Sunshine will never see the added modes. Modes without a VIC code and exceeding 655 MHz are only reachable from an X11 Plasma session. This is a limitation of the Wayland display pipeline, not sunbeams — there is currently no userspace tool on Wayland equivalent to `xrandr --newmode`.

## Why `/etc/firmware` instead of `/usr/lib/firmware`?

The Linux kernel normally searches for firmware files in `/lib/firmware/` (which is `/usr/lib/firmware/` on most distros). On Bazzite and other Fedora Atomic systems, `/usr` is an immutable read-only filesystem managed by rpm-ostree — you can't write files there.

The `firmware_class.path=/etc/firmware` kernel parameter tells the kernel firmware loader to look in `/etc/firmware` *before* the default paths. We use `/etc` rather than `/usr/local` because:

- `/etc` is writable and persists across rpm-ostree updates and rebases
- `/etc` is mounted earlier in the boot process than `/usr/local`, which matters because Bazzite's GPU drivers initialize early (early KMS) — if the EDID file isn't available at that point, the virtual display won't be created
- Some users have reported that `/usr/local/lib/firmware` isn't reliably included in the initramfs, causing the EDID to be invisible during early boot

If you're on a traditional (non-atomic) distro like Arch or standard Fedora Workstation, you can place the file in `/usr/lib/firmware/edid/` and use `drm.edid_firmware=HDMI-A-1:edid/edid.bin` without the `firmware_class.path` parameter.

## Display Switching Strategies

Sunbeams supports two display-switching backends, selected at runtime by `internal/switcher/strategy.go`:

**`KScreenStrategy`** — the default for KDE Plasma desktop mode. Calls `kscreen-doctor` to disable the physical connector, enable the virtual one, and apply the requested mode atomically. Falls back to a 3-step sequence with a 2-second delay if the atomic call fails. No root required at runtime.

**`GamescopeStrategy`** — for Bazzite Gaming Mode where gamescope is the compositor and `kscreen-doctor` doesn't run. Two operations:

1. **Mode selection via `~/.config/gamescope/modes.cfg`.** Each line is `<MonitorName>:<W>x<H>@<R>`. The strategy reads `cfg.EDID.MonitorName` (the name baked into our EDID), finds or appends the matching line, and atomically rewrites the file. Gamescope picks up the new mode on the next connector hotplug.

2. **Physical connector disable via DRM debugfs.** A 40-line shell helper installed at `/usr/local/sbin/sunbeams-drm-force` writes `off` (or `on`) to `/sys/kernel/debug/dri/<pci>/<connector>/force` and triggers `udevadm`. The kernel treats this as a fake hotplug. The helper is invoked via `sudo -n` with a NOPASSWD entry scoped to that single binary.

Strategy selection (`Select(name, opts)`):

- `auto` (default): chooses `debugfs` if `$GAMESCOPE_WAYLAND_DISPLAY` is set, else `kscreen`
- explicit `kscreen` or `debugfs` overrides auto
- `$SUNBEAMS_STRATEGY` env (set to `kscreen` or `debugfs`) is consulted between flag and auto-detect

`SwitchOff` for the gamescope strategy does an optional safe-revert: rewrites the modes.cfg line to a low-risk mode (default `1920x1080@60`, configurable via `[gaming].safe_revert_mode`) before re-enabling the physical. This avoids a documented Plasma black-screen-on-return bug if you switch back to desktop while the virtual is at an exotic resolution.
