# Supported Devices and Resolutions

The default configuration in `internal/config/defaults.toml` targets the following devices:

| Device | Resolution | Refresh | HDR |
|---|---|---|---|
| Sony 4K OLED (65") | 3840×2160 | 120 Hz | Yes |
| Sony 4K TV (55") | 3840×2160 | 120 Hz | Yes |
| Samsung Frame (55") | 3840×2160 | 120 Hz | Yes |
| Gigabyte M28U | 3840×2160 | 144 Hz | Yes |
| BenQ EX3415R | 3440×1440 | 144 Hz | Yes |
| MacBook Pro 14" (M3 Max) | 3024×1964 | 120 Hz | Yes |
| iPad Pro 11" (M4/M5) | 2420×1668 | 120 Hz | Yes |
| iPad Pro 11" (M2) | 2388×1668 | 120 Hz | Yes |
| Samsung Tab A9+ | 1920×1200 | 60 Hz | — |
| PS Vita | 960×544 | 60 Hz | — |
| PSP | 480×272 | 60 Hz | — |

The EDID also includes resolutions for common streaming targets that aren't tied to a specific owned device:

| Mode | Use Case |
|---|---|
| 3120×1440 @ 120/60 Hz | Samsung Galaxy S24 Ultra / S-series phones |
| 2880×1920 @ 60 Hz | Pixel Tablet / high-res 3:2 tablets |
| 2560×1600 @ 144/60 Hz | Steam Deck OLED / 16" laptops |
| 2560×1440 @ 144/120/60 Hz | QHD monitors |
| 2340×1080 @ 120/60 Hz | Android phones (19.5:9) |
| 2305×1080 @ 120/60 Hz | Android phones (21.5:9) |
| 1920×1080 @ 144/120/60 Hz | FHD gaming |
| 1280×800 @ 60 Hz | Steam Deck LCD |
| 1280×720 @ 60 Hz | Low-bandwidth fallback |

60 Hz fallbacks are included for every high-refresh resolution. Common gaming resolutions are also included as VIC codes.

To add or edit devices, see [customizing.md](customizing.md).
