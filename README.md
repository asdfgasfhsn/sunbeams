# Sunbeams — Virtual Display Toolkit for Sunshine on Bazzite

[![Site](https://img.shields.io/badge/site-asdfgasfhsn.github.io%2Fsunbeams-blue)](https://asdfgasfhsn.github.io/sunbeams/)
[![CI](https://github.com/asdfgasfhsn/sunbeams/actions/workflows/ci.yml/badge.svg)](https://github.com/asdfgasfhsn/sunbeams/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/asdfgasfhsn/sunbeams?include_prereleases&sort=semver)](https://github.com/asdfgasfhsn/sunbeams/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/asdfgasfhsn/sunbeams)](go.mod)
[![License](https://img.shields.io/github/license/asdfgasfhsn/sunbeams)](LICENSE)

Sunbeams is a single-binary toolkit for running a headless [Sunshine](https://github.com/LizardByte/Sunshine) streaming server with a fully virtual display, aimed at [Bazzite](https://bazzite.gg/) Desktop (KDE Plasma / Wayland). It covers the full lifecycle:

- **EDID generation** — synthesise a custom EDID that exposes every resolution your [Moonlight](https://moonlight-stream.org/) clients need (4K, ultrawide, MacBook, iPad, phones, handhelds) in a single binary, with HDR10/HLG metadata and wide range limits.
- **Display switching** — a drop-in `Do`/`Undo` handler for Sunshine's prep commands that reads `SUNSHINE_CLIENT_*` env vars, snaps the request to the nearest configured mode, and drives `kscreen-doctor` atomically. Emits structured, timestamped logs of every decision and every command dispatched so you can debug a stream without SSHing in.
- **Guided Bazzite install / uninstall** — `sunbeams install` scans `/sys/class/drm` for an unused connector, writes the EDID to `/etc/firmware/`, and injects the required kernel arguments via `rpm-ostree kargs`, accounting for Bazzite's immutable `/usr` and early-KMS constraints. `sunbeams uninstall` reverses it — detecting and removing the kargs, firmware file, and systemd user service it added.
- **Status introspection** — `sunbeams status` reports, per connector, whether the virtual EDID is configured, active this boot, and actually loaded — distinguishing a live injection from one that's only pending a reboot.
- **Config + introspection** — TOML-based configuration for devices/modes with `sunbeams config init`/`show`, and `sunbeams devices`/`modes` commands for inspecting what will be encoded into the EDID.

## The Problem

When you run a headless Sunshine streaming server (e.g. a Bazzite box in a rack), the GPU needs an EDID to know which resolutions are available. The common approach — copying an EDID from a physical display — only gives you that one display's resolutions. If you stream to a 4K TV, an ultrawide monitor, a MacBook, and an iPad, you'd need resolutions that no single real monitor supports.

Even with a good EDID, that's only half the job. Sunshine's per-session prep commands need to disable the physical output, enable the virtual one, pick a concrete mode matching what the client asked for, and tear it all down on disconnect — reliably, under Wayland, with logs you can read after the fact. And before any of that works at all, the EDID has to be loaded by the kernel early enough to be visible to the DRM driver, which on Bazzite means fighting the immutable `/usr` filesystem and early-KMS timing.

## The Solution

Sunbeams is one binary that handles all three pieces:

**1. EDID generation (`sunbeams generate`)** produces a synthetic EDID that packs every target resolution into a single file:

- Standard 16:9 resolutions (1080p, 1440p, 4K) at multiple refresh rates
- Ultrawide resolutions (3440×1440)
- Unusual aspect ratios (MacBook 3024×1964, iPad 2420×1668)
- HDR10/HLG metadata and BT.2020 colorimetry
- Wide range limits and HDMI 2.1 VSDBs so the GPU accepts custom high-bandwidth modes (4K@120Hz via VIC 118)

For modes whose pixel clock exceeds the EDID format's 655 MHz limit (e.g. 4K@144Hz), a companion `xrandr` script is generated alongside the binary — see [docs/architecture.md](docs/architecture.md) for the Wayland caveats.

**2. Runtime display switching (`sunbeams switch on|off`)** is wired into Sunshine as the global `Do`/`Undo` command. It reads the `SUNSHINE_CLIENT_*` environment Sunshine sets per session, resolves the request against the configured mode list (exact / refresh-snapped / closest-overall with deltas logged), and drives `kscreen-doctor` in a single atomic call with a retry-with-delay fallback. Every step is logged to stderr with timestamps; `SUNBEAMS_DEBUG=1` adds the session env and `kscreen-doctor` stdout/stderr for diagnosing silent no-ops.

**3. Guided install (`sudo sunbeams install`)** scans `/sys/class/drm` for a disconnected DRM connector, writes the EDID to `/etc/firmware/` (chosen specifically to avoid Bazzite's immutable `/usr` and early-KMS race conditions), and injects `firmware_class.path`, `drm.edid_firmware`, and `video=` kernel arguments via `rpm-ostree kargs`. Alternatively, follow the manual steps in [docs/installation-bazzite.md](docs/installation-bazzite.md).

See [docs/supported-devices.md](docs/supported-devices.md) for the full list of target devices and streaming resolutions baked into the default config.

## Prerequisites

- A Linux machine to use as your Sunshine streaming server (Bazzite/Fedora Atomic recommended)
- No runtime dependencies — `sunbeams` is a single static binary

### Supported platforms

`sunbeams` targets **current stable Bazzite Desktop running KDE Plasma on Wayland** — the default image and session. Everything it shells out to (`kscreen-doctor`, `rpm-ostree`, `/sys/class/drm`) ships in that image with no extra packages.

**Not supported:**

- **Bazzite Deck / Steam Deck images** — these boot into gamescope / Game Mode, which does not expose `kscreen-doctor`. Desktop Mode on a Deck works, but the Deck images are not a tested target.
- **Game Mode / gamescope sessions on any Bazzite variant** — `sunbeams switch` needs `kscreen-doctor`, which isn't available inside gamescope. Switch to Desktop Mode before streaming.
- **bazzite-gnome and other GNOME-based images** (Aurora, Bluefin, stock Silverblue) — no `kscreen-doctor`; the switcher has no equivalent implementation for GNOME/Mutter.
- **X11-only Plasma sessions** are fine for the `add_custom_modes.sh` helper, but the primary path is Wayland.

Other distros with KDE Plasma 6 + Wayland + `kscreen-doctor` + `rpm-ostree` (other Fedora Atomic KDE spins) will likely work but aren't a tested target. The guided `sunbeams install` assumes `rpm-ostree`; on traditional distros (Arch, Fedora Workstation) the manual install steps in [docs/installation-bazzite.md](docs/installation-bazzite.md) apply with minor path tweaks (see [docs/architecture.md](docs/architecture.md#why-etcfirmware-instead-of-usrlibfirmware)).

## Installation

### Download a release binary (recommended)

Download the latest `sunbeams` binary for your architecture from the [Releases page](https://github.com/asdfgasfhsn/sunbeams/releases/latest), make it executable, and put it on your PATH:

```bash
# Example for linux/amd64 — replace <version> with the actual release tag (e.g. v0.1.0)
curl -L https://github.com/asdfgasfhsn/sunbeams/releases/latest/download/sunbeams_<version>_linux_amd64.tar.gz | tar xz
sudo install -m755 sunbeams /usr/local/bin/sunbeams
```

### Build from source

```bash
go install github.com/asdfgasfhsn/sunbeams/cmd/sunbeams@latest
```

Or clone and build:

```bash
git clone https://github.com/asdfgasfhsn/sunbeams.git
cd sunbeams
go build -o sunbeams ./cmd/sunbeams
```

## Quick Start

End-to-end setup on a Bazzite Desktop box you're going to stream *from*:

```bash
# 1. Generate the EDID + helper scripts
sunbeams generate
#   → virtual_display.bin    (the EDID binary)
#   → add_custom_modes.sh    (xrandr helper — X11 sessions only, see docs/architecture.md)
#   → sunshine_commands.txt  (kscreen-doctor command reference)

# 2. Install: copy the EDID to /etc/firmware, scan DRM connectors,
#    inject kernel args via rpm-ostree, then reboot.
sudo sunbeams install

# 3. After rebooting, confirm the EDID is injected and loaded
sunbeams status         # per-connector: configured / active this boot / EDID loaded

# 4. Inspect what will be exposed to clients
sunbeams devices        # configured target devices
sunbeams modes          # every EDID mode + pixel clock + DTD/xrandr status

# 5. Wire it into Sunshine → Configuration → General → global prep commands:
#      Do command:   sunbeams switch on
#      Undo command: sunbeams switch off
#    (Sunshine passes SUNSHINE_CLIENT_WIDTH/HEIGHT/FPS/HDR automatically.)

# To reverse the install later (removes kargs, firmware, and the user service):
sudo sunbeams uninstall
```

See [docs/installation-bazzite.md](docs/installation-bazzite.md) for the manual install path and [docs/sunshine.md](docs/sunshine.md) for the full Sunshine integration, logging reference, and `SUNBEAMS_DEBUG` usage.

## Subcommands

Run `sunbeams <command> --help` for per-command flags.

| Command | Purpose |
|---|---|
| `sunbeams generate` | Generate the EDID binary and helper scripts from the current configuration. Supports `--output-dir`, `--config`, `--no-scripts`, `--validate`. |
| `sunbeams switch on\|off` | Switch the virtual display on (Sunshine Do command) or off (Undo). Overrides: `--width`, `--height`, `--fps`, `--hdr`/`--no-hdr`. |
| `sunbeams devices` | List all configured client devices with their target resolution and refresh rate. |
| `sunbeams modes` | List all EDID modes with pixel clock and whether they're encoded as DTDs or require xrandr. |
| `sunbeams install` | Guided interactive installer for Bazzite (requires `sudo`). Writes the EDID to `/etc/firmware/`, scans DRM connectors, and injects kernel arguments via `rpm-ostree kargs`. |
| `sunbeams uninstall` | Reverse an install (requires `sudo`). Detects and removes the sunbeams kernel args, the `/etc/firmware/edid.bin` file, and the systemd user service. Scope to one output with `--connector <name>`; skip prompts with `-y`. |
| `sunbeams status` | Read-only (no `sudo`). Report per connector whether the virtual EDID is configured, active this boot, and loaded — distinguishing `active` from `configured — reboot pending`. |
| `sunbeams config init` | Write the default configuration to `~/.config/sunbeams/config.toml` as a starting template for customization. |
| `sunbeams config show` | Print the current effective configuration (defaults merged with any user overrides). |
| `sunbeams version` | Print version, commit, and build date. |

## Documentation

- [Supported devices and resolutions](docs/supported-devices.md)
- [Installing on Bazzite (guided + manual)](docs/installation-bazzite.md)
- [Sunshine configuration and environment variables](docs/sunshine.md)
- [Customizing devices and modes](docs/customizing.md)
- [How it works (EDID internals and `/etc/firmware` rationale)](docs/architecture.md)
- [Validating the EDID](docs/validating.md)
- [Troubleshooting](docs/troubleshooting.md)

## Credits

- Original virtual display EDID injection technique: [iamthenuggetman's gist](https://gist.github.com/iamthenuggetman/6d0884954653940596d463a48b2f459c)
- [Reddit thread](https://www.reddit.com/r/linux_gaming/comments/199ylqz/streaming_with_sunshine_from_virtual_screens/) on virtual screen streaming
- [Sunshine](https://github.com/LizardByte/Sunshine) and [Moonlight](https://moonlight-stream.org/) projects

## License

MIT — see [LICENSE](LICENSE).
