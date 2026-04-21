# EDID Coverage Round 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand the shipped EDID so tech-savvy gamers hit common resolutions and refresh rates without hand-rolled configs, while staying inside a 5-extension-block compatibility budget.

**Architecture:** Extend the existing TOML-driven generator with: (a) 8 new CTA VICs for cinematic/TV-standard rates, (b) 2 new DTD modes (Steam Deck OLED 90 Hz and 1080p@240), (c) 5 new xrandr-overflow modes for above-655-MHz monitors, (d) a new `CTAY420CMDB` data block advertising YCbCr 4:2:0 fallback for 4K@60/100/120. Round 1 refreshes the golden fixture at the end; byte-level unit tests guard individual primitives.

**Tech Stack:** Go 1.22, `github.com/BurntSushi/toml`, `github.com/stretchr/testify`, `encoding/binary`, `go:embed`. Existing `internal/edid/`, `internal/config/`, `internal/generate/` packages. No new dependencies.

**Spec:** `docs/resolution_gaps.md`.

---

## File Structure

### Files modified

- `internal/edid/cta.go` — add `CTAY420CMDB(vdbVICs, y420VICs []int) []byte` (new function, ~20 lines).
- `internal/edid/cta_test.go` — add `TestCTAY420CMDB` byte-level test.
- `internal/config/config.go` — add `Y420VICs []int` field to `CTAConfig`.
- `internal/config/defaults.toml` — expand `vic_codes`, add `y420_vics`, add 2 new DTD modes, add 5 new xrandr-overflow modes.
- `internal/config/config_test.go` — update asserted VIC list, `Modes` length, add `Y420VICs` assertion.
- `internal/generate/generate.go` — insert `CTAY420CMDB(...)` call into `dataBlocks` slice between `CTAHDRStaticMetadata()` and `CTAColorimetry()`.
- `testdata/virtual_display_reference.bin` — regenerated in final task.
- `testdata/README.md` — update size/SHA and note the new modes.

### Files unchanged

- `cmd/sunbeams/main.go` — no new subcommand needed.
- `internal/switcher/`, `internal/installer/`, `internal/platform/`, `internal/userconfig/` — untouched.

---

## Key Design Decisions (from brainstorming)

1. **Extension-block budget: ≤5.** Every byte-level decision below was sized to keep `ext_count == 5`. Drop or swap modes only if the budget changes.
2. **VIC order matters.** The Y420CMDB bitmap indexes into the VDB by position. VIC list order below is the order emitted into the VDB — do not reorder without updating `y420_vics` tests.
3. **Y420CMDB over Y420VDB.** Capability map (extended tag 0x0F, ~3 bytes) flags existing VDB VICs as 4:2:0-capable. No Y420VDB (extended tag 0x0E) this round.
4. **Golden refresh is one commit at the end.** The golden test (`TestGoldenEDID`) will intentionally fail from Task 4 onward. This is expected and resolved in Task 7.

---

## Task 1: Add byte-level test for `CTAY420CMDB`

**Files:**
- Modify: `internal/edid/cta_test.go`

- [ ] **Step 1: Append the failing test**

Add the following function to the end of `internal/edid/cta_test.go`:

```go
func TestCTAY420CMDB(t *testing.T) {
	// VDB order matches defaults.toml after Round 1.
	vdb := []int{97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1}
	y420 := []int{97, 118, 117}
	got := CTAY420CMDB(vdb, y420)

	// Extended-tag block: header (tag 7), extended tag 0x0F, 1-byte bitmap.
	// Bit positions: 97 -> 0, 118 -> 1, 117 -> 2 → 0b00000111 = 0x07.
	// Header = (0x07 << 5) | payload_length (payload includes ext tag + bitmap = 2).
	assert.Equal(t, byte(0xE2), got[0], "header byte")
	assert.Equal(t, byte(0x0F), got[1], "extended tag byte")
	assert.Equal(t, byte(0x07), got[2], "bitmap byte")
	assert.Len(t, got, 3, "block length")
}

func TestCTAY420CMDB_SkipsMissingVICs(t *testing.T) {
	// If a y420 VIC isn't present in the VDB, skip silently.
	vdb := []int{97, 16}
	y420 := []int{97, 117, 118} // 117, 118 not in VDB
	got := CTAY420CMDB(vdb, y420)

	// Only position 0 (VIC 97) gets flagged.
	assert.Equal(t, byte(0xE2), got[0])
	assert.Equal(t, byte(0x0F), got[1])
	assert.Equal(t, byte(0x01), got[2])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/edid/ -run TestCTAY420CMDB -v`

Expected: FAIL with `undefined: CTAY420CMDB`.

- [ ] **Step 3: Commit the test**

```bash
git add internal/edid/cta_test.go
git commit -m "test(edid): add failing byte-level test for CTAY420CMDB"
```

---

## Task 2: Implement `CTAY420CMDB`

**Files:**
- Modify: `internal/edid/cta.go`

- [ ] **Step 1: Append the implementation**

Add the following function to the end of `internal/edid/cta.go`:

```go
// CTAY420CMDB builds the YCbCr 4:2:0 Capability Map Data Block (extended tag 0x0F).
// The bitmap flags which VICs in the preceding VDB also support YCbCr 4:2:0 sampling.
// VICs listed in y420VICs that are not present in vdbVICs are silently skipped.
func CTAY420CMDB(vdbVICs []int, y420VICs []int) []byte {
	position := make(map[int]int, len(vdbVICs))
	for i, v := range vdbVICs {
		position[v] = i
	}
	// Compute bitmap width: enough bytes to cover the largest flagged position.
	maxBit := -1
	for _, v := range y420VICs {
		if p, ok := position[v]; ok && p > maxBit {
			maxBit = p
		}
	}
	if maxBit < 0 {
		// No matching VICs; emit an empty bitmap (payload = ext tag only).
		header := byte((0x07 << 5) | 1)
		return []byte{header, 0x0F}
	}
	bitmap := make([]byte, (maxBit/8)+1)
	for _, v := range y420VICs {
		if p, ok := position[v]; ok {
			bitmap[p/8] |= 1 << (p % 8)
		}
	}
	payloadLen := 1 + len(bitmap)
	header := byte((0x07 << 5) | (payloadLen & 0x1F))
	out := make([]byte, 0, 1+payloadLen)
	out = append(out, header, 0x0F)
	out = append(out, bitmap...)
	return out
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/edid/ -run TestCTAY420CMDB -v`

Expected: PASS for both `TestCTAY420CMDB` and `TestCTAY420CMDB_SkipsMissingVICs`.

- [ ] **Step 3: Run the full edid package tests**

Run: `go test ./internal/edid/ -v`

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/edid/cta.go
git commit -m "feat(edid): add CTAY420CMDB for YCbCr 4:2:0 capability advertisement"
```

---

## Task 3: Expand CTA VIC list + add `y420_vics` to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/defaults.toml`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Update the `CTAConfig` struct**

In `internal/config/config.go`, replace the `CTAConfig` type:

```go
type CTAConfig struct {
	VICCodes  []int `toml:"vic_codes"`
	Y420VICs  []int `toml:"y420_vics"`
}
```

- [ ] **Step 2: Update `defaults.toml`**

In `internal/config/defaults.toml`, replace the `[cta]` section (lines 16-17) with:

```toml
[cta]
# Order matters: Y420CMDB bitmap indexes into this list by position.
# Keep 4K entries grouped at the head so the bitmap fits in one byte.
vic_codes = [97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1]
# VICs above that also advertise YCbCr 4:2:0 support (fallback for HDMI 2.0
# 600 MHz TMDS ceiling at 4K@60/100/120).
y420_vics = [97, 118, 117]
```

- [ ] **Step 3: Update the existing VIC-list test**

In `internal/config/config_test.go`, replace the VIC assertion line inside `TestLoadDefaults`:

```go
	assert.Equal(t,
		[]int{97, 118, 117, 96, 95, 94, 93, 16, 63, 34, 33, 32, 31, 4, 47, 19, 1},
		cfg.CTA.VICCodes)
	assert.Equal(t, []int{97, 118, 117}, cfg.CTA.Y420VICs)
```

- [ ] **Step 4: Run config tests**

Run: `go test ./internal/config/ -v`

Expected: `TestLoadDefaults` passes with the new assertions.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/defaults.toml internal/config/config_test.go
git commit -m "feat(config): expand CTA VIC list to 17 codes and add y420_vics"
```

---

## Task 4: Wire `CTAY420CMDB` into the generator

**Files:**
- Modify: `internal/generate/generate.go`

- [ ] **Step 1: Insert the Y420CMDB block**

In `internal/generate/generate.go`, replace the `dataBlocks` slice (around lines 119-126):

```go
	// 5. CTA data blocks in fixed tag order.
	// Y420CMDB must follow the VDB so its bitmap indexes align with VDB VIC positions.
	dataBlocks := [][]byte{
		edid.CTAVideoDataBlock(cfg.CTA.VICCodes),
		edid.CTAHDMIVSDB(cfg.EDID.MaxTMDSMHz),
		edid.CTAHFVSDB(cfg.EDID.MaxFRLRate),
		edid.CTAHDRStaticMetadata(),
		edid.CTAY420CMDB(cfg.CTA.VICCodes, cfg.CTA.Y420VICs),
		edid.CTAColorimetry(),
		edid.CTAVCDB(),
	}
```

- [ ] **Step 2: Run generate package tests**

Run: `go test ./internal/generate/ -v`

Expected: `TestGoldenEDID` FAILS (golden fixture is stale — expected from here until Task 7). `TestAllBlockChecksumsValid` PASSES (new block still produces a valid 768-byte output that sums to 0 mod 256 per block).

- [ ] **Step 3: Commit (golden still stale, intentional)**

```bash
git add internal/generate/generate.go
git commit -m "feat(generate): emit Y420CMDB block advertising 4K@60/100/120 4:2:0"
```

---

## Task 5: Add new DTD modes (Steam Deck OLED@90, 1080p@240)

**Files:**
- Modify: `internal/config/defaults.toml`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Append the two new DTD modes**

In `internal/config/defaults.toml`, insert these `[[modes]]` entries immediately before the `[[modes]]` block for `1280 x 800 @ 60` (currently lines 301-305). Put the 1080p@240 near the other 1080p entries for grouping:

Find the existing block:

```toml
[[modes]]
width = 1920
height = 1080
refresh = 144
description = "FHD 144Hz — high-refresh gaming"
```

Insert **immediately before** it:

```toml
[[modes]]
width = 1920
height = 1080
refresh = 240
description = "FHD 240Hz — esports / 1080p OLED"

```

Find the existing block:

```toml
[[modes]]
width = 1280
height = 800
refresh = 60
description = "Steam Deck LCD / WXGA"
```

Insert **immediately before** it:

```toml
[[modes]]
width = 1280
height = 800
refresh = 90
description = "Steam Deck OLED native 90Hz"

```

- [ ] **Step 2: Update the mode-count assertion**

In `internal/config/config_test.go`, change the line:

```go
	assert.Len(t, cfg.Modes, 32)
```

to:

```go
	assert.Len(t, cfg.Modes, 34)
```

- [ ] **Step 3: Run config tests**

Run: `go test ./internal/config/ -v`

Expected: PASS.

- [ ] **Step 4: Sanity-run the generator and confirm block count**

Run: `go run ./cmd/sunbeams generate -o /tmp/edid_check_task5`

Expected output contains: `768 bytes, 6 blocks` (unchanged: 2 new DTDs fit in existing overflow headroom).

If the output says `896 bytes, 7 blocks`, stop — the budget math failed. Investigate before proceeding.

- [ ] **Step 5: Commit**

```bash
git add internal/config/defaults.toml internal/config/config_test.go
git commit -m "feat(config): add 1280x800@90 (Steam Deck OLED) and 1920x1080@240 DTD modes"
```

---

## Task 6: Add new xrandr-overflow modes

**Files:**
- Modify: `internal/config/defaults.toml`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Append the five new xrandr-overflow modes**

In `internal/config/defaults.toml`, insert the following `[[modes]]` entries. Group them near existing entries at the same resolution:

Immediately before:

```toml
[[modes]]
width = 3440
height = 1440
refresh = 144
description = "Ultrawide 144Hz — BenQ EX3415R"
```

insert **both**:

```toml
[[modes]]
width = 3840
height = 2160
refresh = 240
description = "4K 240Hz — MSI 321URX / 4K OLED flagship"

[[modes]]
width = 5120
height = 1440
refresh = 120
description = "Super-ultrawide 120Hz — Samsung Odyssey G9"

[[modes]]
width = 3440
height = 1440
refresh = 175
description = "Ultrawide 175Hz — Alienware AW3423DW native"

```

Immediately before:

```toml
[[modes]]
width = 2560
height = 1440
refresh = 144
description = "QHD 144Hz"
```

insert **both**:

```toml
[[modes]]
width = 2560
height = 1440
refresh = 240
description = "QHD 240Hz — LG 27GR95QE / QHD OLED"

[[modes]]
width = 2560
height = 1440
refresh = 165
description = "QHD 165Hz — common gaming monitor tier"

```

- [ ] **Step 2: Update the mode-count assertion**

In `internal/config/config_test.go`, change the line:

```go
	assert.Len(t, cfg.Modes, 34)
```

to:

```go
	assert.Len(t, cfg.Modes, 39)
```

- [ ] **Step 3: Run config tests**

Run: `go test ./internal/config/ -v`

Expected: PASS.

- [ ] **Step 4: Sanity-run the generator and confirm xrandr count + block count**

Run: `go run ./cmd/sunbeams generate -o /tmp/edid_check_task6`

Expected output contains: `768 bytes, 6 blocks` AND `add_custom_modes.sh (9 modes)` (4 original xrandr + 5 new).

If the block count jumps to 7+, stop and investigate — one of the intended-xrandr modes slipped under the 655 MHz ceiling and became a DTD, pushing past the budget.

- [ ] **Step 5: Commit**

```bash
git add internal/config/defaults.toml internal/config/config_test.go
git commit -m "feat(config): add 5 xrandr-overflow modes for high-refresh gaming monitors"
```

---

## Task 7: Regenerate the golden fixture

**Files:**
- Modify: `testdata/virtual_display_reference.bin`

- [ ] **Step 1: Build the binary**

Run: `make build`

Expected: `sunbeams` binary produced, no errors.

- [ ] **Step 2: Regenerate the fixture**

Run:

```bash
./sunbeams generate --output-dir testdata
mv testdata/virtual_display.bin testdata/virtual_display_reference.bin
rm -f testdata/add_custom_modes.sh testdata/sunshine_commands.txt
```

- [ ] **Step 3: Verify block count and size**

Run: `wc -c testdata/virtual_display_reference.bin`

Expected: `768 testdata/virtual_display_reference.bin`.

If the size is not 768, stop and investigate. A size of 896 = 7 blocks = ≤5 ext budget violated.

- [ ] **Step 4: Verify the golden test now passes**

Run: `make verify-golden`

Expected: all checks pass, diff is empty.

- [ ] **Step 5: Compute new SHA-256**

Run: `shasum -a 256 testdata/virtual_display_reference.bin`

Copy the hash — it goes into the commit message and `testdata/README.md` update in Task 8.

- [ ] **Step 6: Commit the new fixture**

```bash
git add testdata/virtual_display_reference.bin
git commit -m "chore(testdata): refresh golden EDID fixture for Round 1 coverage expansion

New modes: 1280x800@90, 1920x1080@240 (DTD); 2560x1440@165/240,
3440x1440@175, 5120x1440@120, 3840x2160@240 (xrandr).
New VICs: 93, 94, 95, 96, 117 (4K cinematic); 32, 33, 34 (1080p cinematic).
New data block: CTAY420CMDB flagging VICs 97, 117, 118 as 4:2:0-capable.
Extension-block count: 5 (unchanged)."
```

---

## Task 8: Update `testdata/README.md`

**Files:**
- Modify: `testdata/README.md`

- [ ] **Step 1: Update size, SHA, and mode notes**

In `testdata/README.md`, replace the bullet list under `## virtual_display_reference.bin` with:

```markdown
- **Size:** 768 bytes (6 blocks: 1 base + 5 CTA extensions)
- **SHA-256:** `<paste the hash from Task 7 Step 5>`
- **Generator:** produced by `sunbeams generate` against `internal/config/defaults.toml`.
- **Coverage:** 17 CTA VICs, 30 DTD-capable modes, 8 xrandr-overflow modes, YCbCr 4:2:0 capability map for 4K@60/100/120. See `docs/resolution_gaps.md` for the full coverage rationale.
```

- [ ] **Step 2: Verify the doc renders cleanly**

Run: `cat testdata/README.md`

Expected: no truncation, no markdown syntax errors.

- [ ] **Step 3: Commit**

```bash
git add testdata/README.md
git commit -m "docs(testdata): update golden fixture SHA and coverage notes"
```

---

## Task 9: Full verification pass

**Files:** none modified

- [ ] **Step 1: Run the full check suite**

Run: `make check`

Expected: fmt clean, lint clean, all tests pass, golden verified. No warnings.

- [ ] **Step 2: Run with the race detector**

Run: `make test-race`

Expected: all tests pass, no data races.

- [ ] **Step 3: Smoke-test the generate command output**

Run:

```bash
./sunbeams generate -o /tmp/edid_final
wc -c /tmp/edid_final/virtual_display.bin
grep -c "xrandr --newmode" /tmp/edid_final/add_custom_modes.sh
```

Expected:
- `768 /tmp/edid_final/virtual_display.bin`
- `9` (number of xrandr modelines)

- [ ] **Step 4: If `edid-decode` is on PATH (nix develop shell), validate externally**

Run: `command -v edid-decode && ./sunbeams generate --validate -o /tmp/edid_final || echo "edid-decode not available — skipping"`

Expected on Linux with nix shell: no warnings more severe than "edid-decode: this is a normal EDID."

Expected on bare macOS: the echo message.

- [ ] **Step 5: No commit (verification-only task)**

---

## Self-review checklist

Before handing off:

- [ ] **Spec coverage:** VIC expansion (Round 1 scope #1), DTD modes (#2), xrandr modes (#2), Y420CMDB (#3-4), golden refresh (#5). All covered.
- [ ] **Type consistency:** `CTAY420CMDB(vdbVICs, y420VICs []int)` signature matches across Task 1 test and Task 2 implementation. Field name `Y420VICs` matches in config struct, TOML tag `y420_vics`, and generate wiring.
- [ ] **Extension budget:** verified via generator smoke tests in Tasks 5, 6, 7. Math recorded in `docs/resolution_gaps.md` Round 1 section.
- [ ] **Placeholder scan:** no TBDs, all code blocks complete, all commands exact.
- [ ] **TDD discipline:** red-green structure for `CTAY420CMDB` (Tasks 1 and 2). Config and TOML changes verified by the existing `TestLoadDefaults` and `TestGoldenEDID` chain.
