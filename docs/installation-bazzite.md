# Installing on Bazzite

These instructions assume a current stable **Bazzite Desktop** image running **KDE Plasma on Wayland** (the default session). Check the [Supported platforms](../README.md#supported-platforms) section before proceeding.

> **Not supported — do not follow this guide if you are on:**
>
> - **Bazzite Deck** or any Steam Deck image booting into **Game Mode / gamescope**. `kscreen-doctor` is not available inside gamescope, so `sunbeams switch` cannot drive the display. If you are on a Deck image, boot into Desktop Mode first — and note that the Deck images are not a tested target.
> - **bazzite-gnome** / Aurora / Bluefin / Silverblue or any other GNOME-based image. The switcher is KDE/`kscreen-doctor` specific and has no GNOME equivalent.
>
> Switching from Game Mode to Desktop Mode on a Deck does not make sunbeams supported on that image — file an issue describing your setup before relying on it.

You can use the guided installer or do it manually.

## Guided

```bash
sudo sunbeams install
```

This writes the EDID to `/etc/firmware/`, scans DRM connectors, and injects the required kernel arguments via `rpm-ostree kargs`.

## Manual

**1. Generate the EDID binary:**

```bash
sunbeams generate --output-dir /tmp/sunbeams-build
```

**2. Copy the EDID to the firmware directory:**

```bash
sudo mkdir -p /etc/firmware
sudo cp /tmp/sunbeams-build/virtual_display.bin /etc/firmware/edid.bin
```

We use `/etc/firmware` instead of `/usr/lib/firmware` because `/usr` is immutable on Bazzite (Fedora Atomic). The `/etc` directory is writable, persists across rpm-ostree updates, and is available early in the boot process — avoiding race conditions with early KMS that can occur with `/usr/local`. See [architecture.md](architecture.md#why-etcfirmware-instead-of-usrlibfirmware) for the full rationale.

**3. Find an unused video output on your GPU:**

```bash
for p in /sys/class/drm/*/status; do
  con=${p%/status}; echo -n "${con#*/card?-}: "; cat $p
done
```

Look for a `disconnected` HDMI or DP port (e.g. `HDMI-A-1`).

**4. Inject the EDID via kernel args:**

```bash
sudo rpm-ostree kargs --append-if-missing=\
  "firmware_class.path=/etc/firmware \
   drm.edid_firmware=HDMI-A-1:edid.bin \
   video=HDMI-A-1:e"
```

Replace `HDMI-A-1` with your chosen port.

**5. Reboot:**

```bash
systemctl reboot
```

**6. Verify:**

```bash
kscreen-doctor -o
```

You should see all the DTD-based resolutions listed under the virtual display.

**7. (Optional, X11 only) Add high-bandwidth custom modes:**

> **⚠ Wayland caveat:** `add_custom_modes.sh` uses `xrandr` and only works in an **X11 Plasma session**. Bazzite Desktop defaults to Plasma/Wayland, where this script has no effect — `kscreen-doctor`, System Settings, and Sunshine will not see the added modes. The script now exits with a warning if a Wayland session is detected.
>
> 4K@120Hz is already covered by EDID VIC code 118 and works natively under Wayland without this script. Only modes that exceed the 655 MHz DTD limit *and* have no VIC (4K@144Hz, ultrawide@144Hz, MacBook native@120Hz) require it. If you don't need those, skip this step.
>
> To use the script, log out and choose **Plasma (X11)** at the SDDM login screen, then run:

```bash
./add_custom_modes.sh HDMI-A-1
```

**8. Configure Sunshine:** see [sunshine.md](sunshine.md).
