# Resolution & Refresh-Rate Coverage: Gaps and Roadmap

This document captures the state of EDID resolution/refresh-rate coverage in
sunbeams, what the current round of work ships, and what we intentionally
deferred. It exists so that future contributors (and our future selves) can
pick up any of these threads without re-doing the research.

## Design constraints

Three hard constraints bound every decision here:

1. **DTD pixel-clock ceiling is 655.35 MHz.** Any mode above that (e.g., 4K@120,
   3440×1440@144+) cannot be expressed as a Detailed Timing Descriptor and has
   to land in the `add_custom_modes.sh` xrandr overflow script.
2. **CTA extension-block count is effectively capped at 5.** EDID 1.4 permits
   up to 255 extensions, but older AMDGPU firmwares and several HDMI
   passthrough paths (AVRs, KVMs, capture cards) only parse the first 2–3
   reliably. The project targets a **≤5 extension block** ceiling for the
   public v1 — room to breathe without regressing HDMI-passthrough users.
3. **Standard timings are limited to 8 slots.** Already full.

These shape what earns a DTD slot vs. a VIC vs. an xrandr entry.

## Round 1 — what ships now

Scope chosen for the first public release. All items fit within the
5-extension ceiling (verified against the generator: with the expanded VDB
and new Y420CMDB, cta1 holds 4 DTDs; 2 new DTD modes push extensions from
22 → 24 overflow DTDs, still 4 overflow blocks + 1 primary = 5 ext).

### New DTD modes

| Mode           | Pixel clock (CVT-RB) | Target                                  |
|----------------|----------------------|-----------------------------------------|
| 1280×800@90    | ~105 MHz             | Steam Deck OLED native refresh          |
| 1920×1080@240  | ~555 MHz             | Esports / 1080p OLEDs                   |

> `1920×1080@165` was considered but dropped to preserve the ≤5-extension
> ceiling. 165 Hz monitors cleanly accept 144 Hz via the VRR range limits,
> so the loss is minimal. Revisit in Round 2 if we also lift the extension
> budget.

### New xrandr-overflow modes

| Mode           | Pixel clock (CVT-RB) | Target                                  |
|----------------|----------------------|-----------------------------------------|
| 2560×1440@240  | ~965 MHz             | LG 27GR95QE / QHD OLED                  |
| 3440×1440@175  | ~933 MHz             | Alienware AW3423DW native               |
| 5120×1440@120  | ~938 MHz             | Samsung Odyssey G9 / super-ultrawide    |
| 3840×2160@240  | ~2120 MHz            | MSI 321URX / 4K OLED flagship           |

### CTA VIC expansion (9 → 17)

Added: 93, 94, 95, 96, 117 (4K@24/25/30/50/100) and 32, 33, 34
(1080p@24/25/30). Cinematic + TV fallback coverage for near-zero byte cost.
Leaves 14 free slots before hitting the 31-VIC VDB limit.

### YCbCr 4:2:0 capability

New `CTAY420CMDB()` block (extended tag 0x0F) flagging VICs 97, 117, 118
(4K@60/100/120) as 4:2:0-capable. Lets HDMI 2.0 receivers negotiate the
bandwidth-reduced fallback when the 600 MHz TMDS ceiling is exceeded. Chosen
over a full Y420VDB because it reuses existing VDB entries for ~3 bytes
instead of scaling with VIC count.

## Streaming feasibility: why we advertise high refresh rates

The EDID is a *capability declaration*, not a commitment. Advertising
1080p@240 or 4K@240 lets capable clients opt in — it does not force the host
to encode there. Non-capable clients negotiate down to 120/60 automatically.

That said, reaching those rates end-to-end depends on pieces outside
sunbeams' control. Real-world ceilings as of 2026:

**Moonlight clients**

- Moonlight-PC / Qt: 120 FPS default, experimental 240 FPS in settings.
- Moonlight-iOS (ProMotion devices): 120 FPS default, 240 FPS unlockable.
- Moonlight-Android: 120 FPS cap on most devices.
- Moonlight Embedded (Pi / Chromecast): 60 FPS hardware ceiling.

**Sunshine host encoders**

- NVENC (RTX 30/40/50): 4K@240 HEVC sustainable.
- AMF (RDNA3+): similar envelope, slightly lower efficiency than NVENC.
- QuickSync (Arc): competitive at 1440p@120, tighter at 4K@120.
- AV1 encode (RTX 40/50, RDNA3, Arc): better efficiency; decoder support on
  clients is the bottleneck, not the encoder.

**Client decoders (the practical ceiling)**

- Steam Deck / most handhelds: 1080p@120 HEVC.
- Apple TV 4K (2022+): 4K@120 HEVC.
- Flagship phones (A17 Pro, SD8G3+): 1440p@120 HEVC comfortably.
- AV1 decode: iPhone 15 Pro+, Snapdragon 8 Gen 2+, newest Android TVs only.

**Network**

- Wi-Fi 5: 1080p@120 reliable with good signal.
- Wi-Fi 6 / 6E (clean 6 GHz): 1440p@120 and 4K@60 solid; 4K@120 possible.
- Wi-Fi 7: 4K@120 reliable, 4K@240 achievable with strong signal.
- Wired GigE: 4K@240 HEVC fits (~800 Mbps) with minimal jitter headroom.

**Jitter dominates at high FPS.** At 240 Hz a frame is 4.17 ms; a single 10 ms
Wi-Fi stall eats multiple frames. End-to-end latency floors sit around
20–35 ms wired / 35–60 ms Wi-Fi, so perceived smoothness above ~120 Hz is
bound more by network jitter than by refresh rate.

**Takeaway for users:** sunbeams' EDID advertises the full mode range because
declaration is cheap and enables capable setups. **YMMV** — the mode you pick
in Moonlight has to survive the encoder, the network, and the client decoder.
If a mode stutters, drop to a lower refresh; the EDID won't punish you.

## Deferred work

Everything below was consciously held out of Round 1. Order is roughly
ascending effort.

### Medium tier (afternoon – 1-2 days each)

- **Short Audio Descriptors (SAD) + Speaker Allocation Data Block.**
  Sunshine streams its own audio path, so this is cosmetic for the streaming
  case, but some receivers (AVRs, legacy capture cards) refuse HDMI handshake
  without audio descriptors present. Low risk, ~40 lines in
  `internal/edid/cta.go` + byte-level tests + golden refresh.

- **AMD FreeSync VSDB (OUI 00-00-1A).** We already emit CVT-adaptive range
  limits which covers VRR on most Linux paths. Adding the AMD-specific VSDB
  helps Windows Moonlight hosts that sniff the AMD block explicitly. Minor
  data block, byte-level test, golden refresh.

- **Y420VDB for 4:2:0-exclusive VICs.** Round 1 ships Y420CMDB only. Adding
  Y420VDB would let us advertise VICs 119/120 (4K@100/120 with 4:2:0 only) as
  distinct entries — redundant with CMDB for most clients, occasionally
  useful for bandwidth-constrained receivers that key off VIC numbers rather
  than capability maps.

### Big tier (~1 week)

- **DisplayID v2.0 extension block.** The proper fix for the 655 MHz DTD
  ceiling and the 2-DTD base-block limit. DisplayID lets us express
  arbitrary high-refresh modes natively without xrandr overflow scripts.
  Requires a new `internal/displayid/` package, a parallel test suite, and a
  golden-file addition. Would eliminate `add_custom_modes.sh` as a required
  step for most users.

### Known mode gaps not yet slotted

Modes discussed but not chosen for Round 1. Add when a device preset or user
demand justifies the DTD/xrandr slot.

- **1920×1080@165** — considered for Round 1, dropped to keep the ≤5
  extension-block ceiling. 165 Hz monitors accept 144 Hz cleanly via VRR.
  First candidate to re-add if we lift the ceiling or drop a legacy mode.
- **2560×1440@165** — considered for Round 1 and dropped during implementation.
  Pixel clock (641.64 MHz) sits just under the 655 MHz DTD ceiling, so the
  generator classified it as DTD-capable rather than xrandr-only; adding it
  as a DTD would have been the third new DTD this round and pushed the
  extension-block count from 5 to 6. Like `1920×1080@165`, 165 Hz monitors
  fall back to 144 Hz cleanly via the VRR range, so the loss is minimal.
- **1080p@360** — CSGO/Valorant esports tier. Decoder support limited;
  network jitter wastes most of the benefit over 240.
- **2560×1600@240** — Steam Deck OLED form factor gaming laptops
  (ROG Flow Z13, Zephyrus G14 2024+). Wait for device preset demand.
- **5120×2160@60 and @120** — LG 40WP95C-class 5K2K ultrawide. Niche but
  growing; xrandr overflow only.
- **iPhone Pro Max native (2796×1290@120)** — streams fine at 1920×1080@120
  via Moonlight's letterbox/scale. Explicit DTD would avoid scaling but
  costs a slot.
- **Pixel 9 Pro native (2856×1280@120)** — same reasoning as iPhone.
- **Samsung Galaxy S24+/S25 non-Ultra native (2340×1080@120)** — already
  covered as a generic 19.5:9 mode; adding a dedicated preset is cheap if
  user demand warrants it.
- **Legion Go / Claw / Ally X specific overrides** — parent resolutions
  (1920×1080@120, 2560×1600@144) are covered; device-specific presets would
  only add convenience, not capability.

### Structural improvements (orthogonal to modes)

- **Compatibility profile flag.** A `--profile=conservative|standard|greedy`
  CLI flag that toggles the extension-block ceiling: conservative = ≤3 ext,
  standard = ≤5 ext (current default), greedy = ≤7 ext. Useful for users
  with older AVRs or capture cards that truncate extensions.
- **xrandr script consolidation.** The current `add_custom_modes.sh` runs
  every time the user switches displays. Moving it into the systemd user
  service lifecycle would make the overflow transparent.
- **Per-client EDID variants.** Sunshine 0.22+ supports per-app display
  configs. Emitting tailored EDIDs per common client (Deck, Apple TV, M28U)
  would reduce noise on each receiver's end, at the cost of generator
  complexity.

## Extension-block math reference

For contributors who touch the generator and need to re-run the budget:

```
Per CTA extension block:
  128 bytes total
  - 1 byte tag (0x02)
  - 1 byte revision (0x03)
  - 1 byte DTD offset
  - 1 byte feature support
  = 124 bytes after header
  - 1 byte checksum
  = 123 bytes for data blocks + DTDs
  
DTD size: 18 bytes

CTA block 1 (with data blocks):
  Data blocks consume N bytes
  DTDs fit in: floor((123 - N) / 18)

CTA overflow blocks (no data blocks):
  DTDs fit in: floor(123 / 18) = 6
```

Current round data-block sizes:

| Block                      | Bytes |
|----------------------------|-------|
| CTAVideoDataBlock (17 VIC) |    18 |
| CTAHDMIVSDB                |     8 |
| CTAHFVSDB                  |     8 |
| CTAHDRStaticMetadata       |     7 |
| CTAY420CMDB                |     3 |
| CTAColorimetry             |     4 |
| CTAVCDB                    |     3 |
| **Total**                  |    51 |

cta1 DTD capacity = (123 − 51) / 18 = **4 DTDs**

Round 1 counts:

- 32 original modes + 2 new DTD + 4 new xrandr = 38 modes.
- 4 original xrandr + 4 new xrandr = 8 xrandr-only modes.
- 38 − 8 = 30 DTD-capable modes total.
- 2 DTDs in base block → 28 DTDs in extensions.
- cta1 holds 4 → 24 overflow DTDs.
- 24 / 6 = 4.0 → 4 overflow blocks.
- Total extensions = 1 + 4 = **5** ✓

Re-run `make verify-golden` any time these numbers change to keep
`testdata/virtual_display_reference.bin` in sync.
