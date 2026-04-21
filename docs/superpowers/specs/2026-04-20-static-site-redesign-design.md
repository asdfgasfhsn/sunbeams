# Static site redesign — sunbeam fan-out

**Status:** spec (rev 2, incorporates frontend-design critique)
**Date:** 2026-04-20
**Affected files:** `site/index.html`, `site/style.css`, one self-hosted font file at `site/fonts/`. `site/install.sh` is untouched.

## Goal

Redesign the GitHub Pages landing page at `site/` so it visually demonstrates what Sunbeams provides. The current page is minimal text. The replacement leads with an illustration where a sun on the horizon fires bright sunbeams outward across the sky and onto a row of client-device viewports standing on a perspective grid — making the "one virtual display → every client" value prop instantly legible. Below the hero, a tightened structure walks the visitor through the problem, the three jobs Sunbeams does, the baked-in device list, and a quick-start terminal block.

The aesthetic direction is "retro-futurist CLI" — the sun-over-horizon silhouette is still the backbone, but the execution deliberately avoids every stock-synthwave move (no stripes carved into the sun, no centered symmetry, no gradient-bleed over the whole page, no system-ui headline, no traffic-light-dotted terminal window). The page should read as *designed* for this tool, not as a generic retrowave template.

## Non-goals

- No changes to `install.sh`, CLI behavior, or any Go code.
- No new build step, no CSS/JS tooling, no bundler. The site stays two files (`index.html` + `style.css`) plus the existing `install.sh`, plus one self-hosted font file.
- No external network dependencies at page load — no Google Fonts, no CDN icons, no JS libraries. Self-hosting one WOFF2 is fine; fetching from a CDN is not.
- No light-mode variant. The retro-CLI treatment is dark-only on purpose; we drop the existing `@media (prefers-color-scheme: light)` branch.
- No animation on first load. Hover/focus transitions on interactive elements are allowed (≤ 150 ms). `prefers-reduced-motion` disables them.

## User experience

A visitor landing on `https://asdfgasfhsn.github.io/sunbeams/` sees, in order:

1. **Top nav.** `sunbeams.` wordmark left, three in-page links (`install`, `what`, `devices`) and a GitHub link right. Everything in the display mono, uppercase, tracked-out. Muted until hover. Section IDs match the link targets.

2. **Hero illustration.** Full-bleed, ~16:7 aspect. Composition is deliberately **asymmetric**: the sun sits around **x ≈ 38%** of the frame (off-center left), and the vanishing point of the perspective grid is at **x ≈ 62%** (off-center right). The sun is a clean disc with a crisp stroked edge — **no carved horizontal stripes**. A perspective grid floor recedes to its offset vanishing point. Long glowing sunbeams fan upward and outward across the sky (three layers: soft halo, primary, crisp core), all originating at the sun. Six delivery beams — narrow at the sun, widening to ~28 px where they land — strike a row of client-device viewports standing on the grid.

   The client devices are drawn as **simple framed rectangles at each device's native aspect ratio**, filled with a subtle gradient, and **labeled with their resolution in the display mono** (e.g. `3840×2160`, `3440×1440`, `3024×1964`). No keyboard bars, no home buttons, no control sticks, no decorative details — the frame and the number are the whole rendering. The diversity of aspect ratios *is* the content.

   Title lockup lives in the **upper-left** quadrant (not centered): `SUNBEAMS` set in the display mono at a large size with extreme letter-spacing (~0.2 em), rendered with a warm-core / stroked outer treatment so it reads as a poster rather than body type. Beneath it, a single monospace kicker: `HEADLESS · VIRTUAL · STREAMING`. **No third subtitle line.** The illustration carries the value prop; a secondary sentence dilutes it and belongs in the install hint below.

3. **Install line.** No glowing pill. A single monospace line flush to the content column, prefixed with `$` in the warm accent color, the command in primary text color, and a small `[copy]` affordance at the end of the line. On copy, swap to `[copied]` for 2 s with an `aria-live="polite"` announcement. Below the line, a one-sentence hint in muted mono: install path, platform scope, one-line value prop (the sentence we cut from the hero).

4. **The problem — flowing out of the hero.** No horizontal rule or section reset. One of the hero's six sunbeams is authored to extend *below* the hero viewBox, rendered as a decorative positioned SVG element in the first problem panel, so the sun visually reaches down into this section. The panel contains three short pain-points laid out as three stacked rows rather than three cards side-by-side:
   - **`(1)` No display, no EDID.** The GPU reports zero modes without a monitor plugged in. Sunshine has nothing to stream.
   - **`(2)` One EDID, one resolution.** Copying a real monitor's EDID gives you *that* monitor — not a 4K TV, an ultrawide, a MacBook, *and* an iPad.
   - **`(3)` Bazzite fights back.** Immutable `/usr`, early-KMS timing, Wayland session switching — all have to agree before the stream comes up.

   Numbering is bracketed monospace, not the generic "01 / 02 / 03" chip treatment.

5. **What sunbeams does.** Three **full-width horizontal rows** (not three equal-width cards). Each row has a two-column layout: left column = bracketed label + verb (`[generate]`, `[switch]`, `[install]`); right column = one-line shell command in the warm accent, followed by two sentences explaining what it does. Rows are separated by a hairline rule, not cards. Reads as reference documentation, not marketing bullets.

6. **Every device, one display — as a proportional mosaic.** The device list renders as a CSS Grid mosaic where **each tile's size is proportional to its pixel count**, so a 4K tile is visibly huge and a PSP tile is tiny. The visual *is* the information: the diversity of form factors is made tangible. Each tile contains only:
   - The device name in the display mono (`4K OLED`, `ULTRAWIDE`, `MACBOOK PRO 14"`, etc.).
   - Resolution in muted mono (`3840 × 2160`).
   - Refresh rate as a small inline tag when it's notable (`@ 60`, `@ 120`, `@ 144`).

   Tile borders are hairline in the accent; backgrounds are near-black. HDR and any other single-word tags appear as a small monospace sigil in the corner. Eleven tiles total:

   | Label | Pixels | Rough span |
   |---|---|---|
   | 4K OLED | 3840 × 2160 | huge |
   | Ultrawide | 3440 × 1440 | wide |
   | MacBook Pro 14" | 3024 × 1964 | large |
   | 1440p | 2560 × 1440 | medium |
   | iPad Pro 11" | 2420 × 1668 | medium |
   | Android phone | 2400 × 1080 | medium-tall |
   | 1080p | 1920 × 1080 | medium |
   | Steam Deck | 1280 × 800 | small |
   | PS Vita | 960 × 544 | small |
   | PSP | 480 × 272 | tiny |
   | `+ more — sunbeams devices` | link | small |

   Exact grid spans and row heights are tuned during implementation; the rule is "bigger resolution → bigger tile, visibly." These are hand-picked representatives of the default config; they are not auto-generated from `defaults.toml` (intentionally — keeps build complexity at zero).

7. **Quick start.** A terminal block without the Mac traffic-light dots. Instead: a tinted header strip containing `~/bazzite` and a blinking-free prompt indicator (`$`) in the accent color. The body is a monospace `<pre>` showing the four-step end-to-end flow (generate → install → inspect → wire into Sunshine). The block **bleeds to the right edge of the viewport** — no container padding on the right — so the line continues off-page. Small grid-break, high-impact.

8. **Footer.** One-line flex row: `MIT · v0.1.x · sunbeams` left; GitHub / Releases / README / Docs right, all muted mono.

## Typography

One self-hosted display mono used for **everything** — headings, body, labels, kickers, tiles, terminal, footer. No system-ui, no second sans, no separate body face. Rationale: the tool is a CLI; the audience is a CLI audience; a single characterful mono unifies the page and drops the "two-font SaaS landing page" template.

Font candidates, in preference order:
1. **Departure Mono** (CC0, free) — pixel-inflected, retro-CLI without being cheesy. Strongest match.
2. **Commit Mono** (SIL OFL, free) — modern, characterful, good fallback.
3. **Monaspace Krypton** (OFL, free) — geometric with personality.

Implementer picks one at the start of implementation, commits the WOFF2 to `site/fonts/`, and wires it via a single `@font-face` rule. File must be ≤ 120 KB to keep page weight trivial.

Fallback stack: `"DisplayMono", ui-monospace, "SFMono-Regular", "Cascadia Code", Menlo, Consolas, monospace`. If the font file fails to load, the fallback is still all-mono and the layout holds.

Font sizes: `0.78 rem` labels, `0.88 rem` body and code, `1 rem` row headings, `1.3 rem` section headings, `clamp(3 rem, 7 vw, 6 rem)` for the `SUNBEAMS` wordmark. Line-height 1.6 on body for readability in an all-mono stack.

## Palette

Four tokens outside the hero, full ramp inside.

**Global CSS variables:**
- `--bg: #07050f` — near-black page background
- `--surface: #0f0a22` — panel / tile surface
- `--fg: #e8ecff` — primary text
- `--muted: #9aa3c7` — secondary text, hairlines
- `--accent: #ffe066` — warm highlight (prompt `$`, copy affordance, tile borders on hover, kickers, focus rings). **One** warm accent below the fold.

**Hero-only palette** (lives inside the inline SVG's `<defs>`, does not leak into component styles):
- Sky gradient: `#230a44` → `#6b1b60` → `#2a0a4a`
- Floor gradient: `#2a0a4a` → `#04020a`
- Sun gradient: `#fff2a6` → `#ff7ab0` → `#ff2f6b`
- Ray stops: `#fff7c2` (core) → `#ffe066` (mid) → `#ff9a6f` / `#ff7ab0` (fade) → `#ff3d7f` (tail)
- Device viewport fills: muted indigo-to-magenta gradients per tile

Below the hero, the warm ramp is **deliberately absent**. Surfaces are near-black, text is the two grey-whites, and the only warm color appearing is a single `--accent` used sparingly (prompt, focus, kicker, hairline-on-hover). This gives the hero its full punch and prevents the whole page from reading as a sunset wash.

## Architecture

Two files plus one font.

### `site/index.html`

Single static HTML document. Semantic structure:
- `<a class="skip-link" href="#install">Skip to install</a>` — hidden until focused, first tab stop, jumps past the hero for keyboard users.
- `<nav>` with the four links.
- `<main>` wrapping all content sections. Sections are `<section>` with `id` attributes matching nav targets (`#install`, `#what`, `#devices`). The hero is a `<section>` too, so the skip link can legitimately bypass it.
- `<footer>`.

The hero `<svg>` is inline (we need crisp text, multiple gradients, and filters; CSS `background-image` loses fidelity). It has `role="img"`, `aria-labelledby="hero-title hero-desc"`, and contains a `<title id="hero-title">` plus a `<desc id="hero-desc">` that reads: *"Diagram: one virtual display fans out to 4K TV, ultrawide, MacBook, iPad, phone, Steam Deck — each labeled with its native resolution."* (Function first, decoration second.)

The copy-install button is a real `<button>` with `aria-live="polite"` on its text content so screen readers announce the `copied` state change. A small inline `<script>` handles the clipboard write (`navigator.clipboard.writeText`) and the transient label swap — same no-external-JS pattern as today.

### `site/style.css`

Full rewrite, ~400–500 lines. Organized top-to-bottom:

1. `@font-face` for the display mono (points at `./fonts/...woff2`, `font-display: swap`).
2. `:root` custom properties — palette tokens above, spacing scale (`--s-1 … --s-8`), one radius token, one hairline token, font stack.
3. Reset + base typography. Everything inherits the mono stack.
4. `.skip-link` — visually hidden until `:focus`, then absolutely positioned top-left with high contrast.
5. Global `:focus-visible` treatment: `outline: 2px solid var(--accent); outline-offset: 3px;`. Required for keyboard navigation on a dark palette where browser defaults vanish.
6. Layout primitives: `.container` (max-width 1120, centered); section vertical-rhythm utility.
7. Components:
   - `.sb-nav` — flex row.
   - `.sb-hero` — wraps inline SVG; aspect-ratio pinned via `viewBox` + `preserveAspectRatio="xMidYMid slice"`.
   - `.sb-install-line` — flex row, no pill, copy affordance on the right.
   - `.sb-problem` — stacked rows with hairline separators; no card chrome.
   - `.sb-rows` — three full-width "what it does" rows; two-column grid inside each.
   - `.sb-mosaic` — CSS Grid with explicit per-tile `grid-column` / `grid-row` spans for the proportional device mosaic.
   - `.sb-terminal` — tinted header strip + `<pre>` body; right edge bleeds via negative right margin inside its wrapper.
   - `.sb-footer` — flex row.
8. Responsive: single `@media (max-width: 720 px)`. Below that breakpoint the mosaic collapses to a simple `repeat(auto-fill, minmax(140 px, 1fr))` grid (proportional sizing is desktop-only), the two-column "what it does" rows collapse to stacked, and the terminal no longer bleeds (regains padding). The hero SVG scales via its viewBox and keeps working; at very narrow widths the floor and outer rays crop — acceptable.
9. `@media (prefers-reduced-motion: reduce)` disables the hover/focus transitions.

The hero SVG uses `viewBox="0 0 1000 520"` and `preserveAspectRatio="xMidYMid slice"`. Rays and delivery beams are `<polygon>` elements with `fill="url(#gradient-id)"` and `filter="url(#glow-id)"`. Three layers per ray set: long soft halo (strong blur, low opacity), primary (medium blur, full gradient), crisp inner core (no blur, high opacity). Delivery beams use per-beam `linearGradient` with `gradientUnits="userSpaceOnUse"` so each beam's bright-to-warm gradient runs along the sun→device axis consistently regardless of bounding box. One delivery beam extends past the viewBox's bottom edge into the following section.

### Accessibility checklist

- Skip link as the first tab stop.
- Explicit `:focus-visible` treatment using `--accent`, ≥ 2 px outline, ≥ 3 px offset.
- Copy button: `aria-live="polite"` announcement on the `copied` state swap.
- Hero SVG: `role="img"`, `aria-labelledby`, function-first `<desc>`.
- All nav / link / button elements keep native semantics.
- Contrast: `#e8ecff` on `#07050f` ≈ 18:1 (AAA). `#9aa3c7` on `#07050f` ≈ 8.5:1 (AAA small text) but verify on `#0f0a22` (~7.5:1, still AA normal). `--accent #ffe066` on `#07050f` ≈ 14:1 — safe for all uses including body-size text.
- Motion: none on page load. Hover/focus transitions ≤ 150 ms, disabled under `prefers-reduced-motion`.

### Assets

- One WOFF2 font file at `site/fonts/<name>.woff2`, ≤ 120 KB.
- No image files. All other decoration is inline SVG or CSS.
- No favicon work in this pass (out-of-scope follow-up).

## Data flow

None. The page is static. The copy button reads a fixed install command string from its sibling `<code>` element and writes it to the clipboard via `navigator.clipboard.writeText`.

## Testing

- **Manual visual:** open `site/index.html` in a browser and verify:
  - Hero renders at ≥ 1200, ~800, and ~400 widths. Asymmetric composition holds at all three; the mosaic collapses cleanly at the 720 px breakpoint.
  - Install line copies to clipboard and the affordance swaps to `[copied]` then back.
  - Nav anchor scrolls land on the correct sections.
  - The decorative hero beam that extends into the Problem section aligns visually across the section boundary.
- **Keyboard:** tab from page load. First stop is the skip link (appears). Enter jumps to `#install`. Tab through: nav links → copy button → "what it does" row link (if any) → footer links. Every focused element shows the warm accent ring.
- **Screen reader (one-off smoke test):** VoiceOver or Orca reads the hero's `<desc>` first, announces the `copied` state change on button activation, and reads tiles in order with their resolution.
- **Contrast:** verify each color pair above against `--bg` AND `--surface` (tiles sit on surface, not bg). Use a contrast tool on the built page.
- **HTML validity:** one-off paste into `https://validator.w3.org/` or local `tidy`. Not a CI gate.
- **Font loading:** confirm the fallback mono stack is readable if the WOFF2 fails to load. Simulate by renaming the font file temporarily.
- No automated tests. The existing Go test suite is untouched.

Mockups from brainstorming (`.superpowers/brainstorm/29149-1776642784/content/hero-v4.html`, `.../page-wireframe.html`) are the starting visual reference, but **they predate this rev**: they show the older symmetric hero with sun-stripes, clipart device silhouettes, card-based "what it does," uniform device grid, Mac-dot terminal, and system-ui fonts. Implementation must follow this spec where they diverge.

## Risks and tradeoffs

- **Dark-only is a deliberate loss** for visitors who strongly prefer light. Accepted — the retro-CLI treatment doesn't translate to a light surface, and the audience is technical Linux users comfortable with dark tooling.
- **All-mono is a committed choice.** Body copy in mono is heavier than a sans; line-height 1.6 and a 0.88 rem body size are needed to keep readability. Accepted because the voice (CLI / terminal-adjacent) matches.
- **Proportional device mosaic is only desktop.** Below 720 px it collapses to a uniform grid — the proportional concept doesn't survive narrow widths. Accepted.
- **One large inline SVG** makes `index.html` bigger (~10–14 KB uncompressed). Still trivially small.
- **No JS framework, no build step.** Hand-written HTML/CSS will drift from any future refactor. Mitigation: keep the file small and well-sectioned so changes stay local.
- **Hero SVG carries most of the visual payload.** If a future change wants motion or re-theming, the SVG will need to be broken into reusable pieces. Out of scope.
- **Font-hosting note:** the font chosen must have a license that permits bundling and redistribution with an MIT-licensed repo (CC0, OFL, or similar). Verify at font-selection time and commit a `LICENSE` alongside the WOFF2.

## Out-of-scope follow-ups

- Hero motion (subtle ray shimmer, `prefers-reduced-motion`-respectful).
- Moonlight-client-screenshot section showing what the stream looks like on each device.
- Favicon derived from the hero sun.
- Auto-generated device mosaic from `internal/config/defaults.toml` at build time (would require a build step).
- Light-mode variant (would require a re-skin of the hero SVG; not a 1:1 translation).
