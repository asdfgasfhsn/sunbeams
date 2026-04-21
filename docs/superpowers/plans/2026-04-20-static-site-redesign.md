# Static Site Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current minimal `site/index.html` + `site/style.css` with a retro-futurist CLI landing page that visually demonstrates the "one virtual display → every client resolution" value prop, per `docs/superpowers/specs/2026-04-20-static-site-redesign-design.md` (rev 2, commit `32f2acf`).

**Architecture:** Two static files plus one self-hosted WOFF2. Hero is an inline SVG (asymmetric sun + perspective grid + three-layer sky rays + six delivery beams landing on aspect-ratio-correct device viewports). Below the hero: a flush-mono install line, a "problem" panel that the hero's central beam flows into, three full-width horizontal rows for the three commands, a resolution-proportional device mosaic, a right-edge-bleeding terminal block, and a footer. Palette is four tokens below the fold; the full warm ramp is confined to the hero SVG. Typography is a single self-hosted display mono used for everything.

**Tech Stack:** HTML5, CSS (custom properties, Grid, inline SVG with gradients + filters), vanilla JS for clipboard. One WOFF2 font (CC0 licensed). No build step, no network deps, no framework.

**Important note on mockups:** The mockups under `.superpowers/brainstorm/29149-1776642784/content/hero-v4.html` and `page-wireframe.html` predate rev 2 of the spec. They show sun-stripes, clipart device silhouettes, centered symmetric composition, "01/02/03" feature cards, a Mac-dots terminal, and system-ui fonts — **all of which this plan explicitly replaces**. Do not copy from those mockups where they diverge from the spec or from the code in this plan.

---

## File structure

| File | Purpose |
|---|---|
| `site/index.html` | Full rewrite. Skip link, nav, main with sections (`#hero`, `#install`, `#problem`, `#what`, `#devices`, `#quickstart`), footer. Inline SVG hero. Inline `<script>` for clipboard. |
| `site/style.css` | Full rewrite. `@font-face`, four palette tokens, reset, base typography (all-mono), focus-visible, skip-link, per-section component styles, one 720px responsive breakpoint, `prefers-reduced-motion` branch. |
| `site/fonts/DepartureMono-Regular.woff2` | New. The display monospace. ≤ 120 KB. |
| `site/fonts/LICENSE` | New. Copy of the font's CC0 license file. |

`site/install.sh` is unchanged.

No automated tests. Verification is manual at each task: open `site/index.html` in a browser and walk the described checks. For local preview with relative paths, serve the directory via `python3 -m http.server 8000 --directory site` and visit `http://localhost:8000/`.

---

## Task 1: Install the display font

**Files:**
- Create: `site/fonts/DepartureMono-Regular.woff2`
- Create: `site/fonts/LICENSE`

- [ ] **Step 1: Download Departure Mono**

Visit https://departuremono.com/ (or the GitHub mirror https://github.com/departuretype/DepartureMono) and download the latest WOFF2 and the bundled license/readme.

Run:

```bash
mkdir -p site/fonts
# After downloading the release zip to ~/Downloads/DepartureMono-*.zip:
unzip -j ~/Downloads/DepartureMono-*.zip '*.woff2' -d site/fonts/
unzip -j ~/Downloads/DepartureMono-*.zip 'LICENSE*' -d site/fonts/
ls -la site/fonts/
```

The unzip may place the woff2 with a longer filename (e.g. `DepartureMono-Regular-1.422.woff2`). Rename to `DepartureMono-Regular.woff2`:

```bash
mv site/fonts/DepartureMono-Regular*.woff2 site/fonts/DepartureMono-Regular.woff2
mv site/fonts/LICENSE* site/fonts/LICENSE
```

Expected: `site/fonts/` contains exactly `DepartureMono-Regular.woff2` and `LICENSE`.

- [ ] **Step 2: Verify size and license**

Run:

```bash
ls -la site/fonts/DepartureMono-Regular.woff2
head -3 site/fonts/LICENSE
```

Expected: file size ≤ 120 KB. License starts with `Creative Commons Legal Code` or equivalent CC0 header.

If the chosen font file exceeds 120 KB or its license is not CC0 / SIL OFL / MIT, stop and switch to a fallback: Commit Mono (https://github.com/eigilnikolajsen/commit-mono) or Monaspace Krypton (https://monaspace.githubnext.com/). Repeat Step 1 with the new file, renaming as `DepartureMono-Regular.woff2` to keep the rest of this plan's CSS references stable — a slight lie but it keeps the plan deterministic.

- [ ] **Step 3: Commit**

```bash
git add site/fonts/DepartureMono-Regular.woff2 site/fonts/LICENSE
git commit -m "feat(site): add Departure Mono for the landing page redesign"
```

---

## Task 2: CSS foundation — tokens, reset, base typography

**Files:**
- Modify: `site/style.css` (full rewrite)

- [ ] **Step 1: Replace style.css with the foundation layer**

Replace the entire contents of `site/style.css` with:

```css
/* sunbeams landing page — self-hosted mono, four-token palette, dark only */

@font-face {
  font-family: "Departure Mono";
  src: url("./fonts/DepartureMono-Regular.woff2") format("woff2");
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}

:root {
  --bg: #07050f;
  --surface: #0f0a22;
  --fg: #e8ecff;
  --muted: #9aa3c7;
  --accent: #ffe066;

  --hairline: 1px solid rgba(154, 163, 199, 0.18);
  --radius: 6px;

  --font-mono: "Departure Mono", ui-monospace, "SFMono-Regular",
               "Cascadia Code", Menlo, Consolas, monospace;

  --s-1: 0.35rem;
  --s-2: 0.7rem;
  --s-3: 1rem;
  --s-4: 1.4rem;
  --s-5: 2rem;
  --s-6: 3rem;
  --s-7: 4.5rem;
  --s-8: 6rem;
}

*, *::before, *::after { box-sizing: border-box; }

html { background: var(--bg); color-scheme: dark; }

body {
  margin: 0;
  background: var(--bg);
  color: var(--fg);
  font-family: var(--font-mono);
  font-size: 0.88rem;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
}

a {
  color: var(--fg);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 150ms ease, color 150ms ease;
}

a:hover { border-bottom-color: var(--accent); color: var(--accent); }

button {
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

code, pre {
  font-family: var(--font-mono);
  font-size: inherit;
}

pre { margin: 0; }

:focus { outline: none; }
:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 3px;
  border-radius: 2px;
}

.skip-link {
  position: absolute;
  top: -100px;
  left: 0;
  background: var(--accent);
  color: var(--bg);
  padding: var(--s-2) var(--s-3);
  font-weight: 700;
  z-index: 1000;
  transition: top 150ms ease;
}
.skip-link:focus-visible {
  top: 0;
  outline-offset: 0;
}

.container {
  max-width: 1120px;
  margin-inline: auto;
  padding-inline: var(--s-4);
}

.section {
  padding-block: var(--s-6);
  border-top: var(--hairline);
}

.label {
  display: inline-block;
  font-size: 0.72rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: var(--muted);
}

.kicker {
  font-size: 0.72rem;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  color: var(--accent);
}

h1, h2, h3 { font-weight: 700; margin: 0; letter-spacing: -0.005em; }
h2 { font-size: 1.3rem; }
h3 { font-size: 1rem; }

.visually-hidden {
  position: absolute;
  width: 1px; height: 1px;
  padding: 0; margin: -1px;
  overflow: hidden; clip: rect(0,0,0,0);
  white-space: nowrap; border: 0;
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    transition: none !important;
    animation: none !important;
  }
}
```

- [ ] **Step 2: Verify the font loads**

Make sure `site/index.html` still exists (we'll rewrite it in Task 3, but for now it must link `style.css`). Serve the site and open:

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` in a browser. Open DevTools → Network → Font. Confirm `DepartureMono-Regular.woff2` loads with status 200. The existing page content will render in mono.

Stop the server:

```bash
kill $SERVER_PID
```

Expected: no 404s. The body text appears in the Departure Mono face (or a mono fallback if the font hasn't propagated).

- [ ] **Step 3: Commit**

```bash
git add site/style.css
git commit -m "feat(site): foundation CSS — tokens, all-mono typography, focus-visible, skip link"
```

---

## Task 3: HTML skeleton — nav, main sections, footer, copy script

**Files:**
- Modify: `site/index.html` (full rewrite)

- [ ] **Step 1: Replace index.html with the new skeleton**

Replace the entire contents of `site/index.html` with:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="description" content="Sunbeams generates a synthetic EDID for headless Linux game streaming with Sunshine + Moonlight on Bazzite, so one virtual display speaks every client's native resolution.">
  <title>sunbeams — one virtual display, every client resolution</title>
  <link rel="stylesheet" href="style.css">
</head>
<body>
  <a class="skip-link" href="#install">Skip to install</a>

  <nav class="sb-nav container" aria-label="Primary">
    <a href="#" class="sb-wordmark">sunbeams<span class="sb-dot">.</span></a>
    <ul class="sb-nav-links">
      <li><a href="#install">install</a></li>
      <li><a href="#what">what</a></li>
      <li><a href="#devices">devices</a></li>
      <li><a href="https://github.com/asdfgasfhsn/sunbeams" rel="noopener">github ↗</a></li>
    </ul>
  </nav>

  <main>
    <section id="hero" class="sb-hero" aria-label="Sunbeams fans one virtual display out to every client resolution">
      <!-- Inline SVG added in Task 4 -->
      <div class="sb-hero-placeholder">HERO</div>
    </section>

    <section id="install" class="sb-install section">
      <!-- Install line added in Task 6 -->
      <p class="label">install</p>
    </section>

    <section id="problem" class="sb-problem section">
      <!-- Problem rows added in Task 7 -->
      <p class="label">the problem</p>
    </section>

    <section id="what" class="sb-what section">
      <!-- "What it does" rows added in Task 8 -->
      <p class="label">what sunbeams does</p>
    </section>

    <section id="devices" class="sb-devices section">
      <!-- Device mosaic added in Task 9 -->
      <p class="label">every device, one display</p>
    </section>

    <section id="quickstart" class="sb-quickstart section">
      <!-- Terminal block added in Task 10 -->
      <p class="label">quick start</p>
    </section>
  </main>

  <footer class="sb-footer container">
    <!-- Footer added in Task 11 -->
    <p>footer placeholder</p>
  </footer>

  <script>
    (function () {
      function init() {
        var btn = document.querySelector('[data-copy-target]');
        if (!btn) return;
        var target = document.getElementById(btn.getAttribute('data-copy-target'));
        if (!target) return;
        var label = btn.querySelector('[data-copy-label]');
        btn.addEventListener('click', function () {
          var text = target.textContent.replace(/^\s*\$\s*/, '').trim();
          navigator.clipboard.writeText(text).then(function () {
            if (!label) return;
            var original = label.textContent;
            label.textContent = '[copied]';
            setTimeout(function () { label.textContent = original; }, 2000);
          });
        });
      }
      if (document.readyState !== 'loading') { init(); }
      else { document.addEventListener('DOMContentLoaded', init); }
    })();
  </script>
</body>
</html>
```

- [ ] **Step 2: Add nav and section-placeholder CSS**

Append to `site/style.css`:

```css
/* ---------- nav ---------- */

.sb-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-block: var(--s-3);
}

.sb-wordmark {
  color: var(--fg);
  font-weight: 700;
  letter-spacing: 0.02em;
  border-bottom: none;
}
.sb-wordmark:hover { color: var(--accent); border-bottom: none; }
.sb-wordmark .sb-dot { color: var(--accent); }

.sb-nav-links {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  gap: var(--s-4);
  font-size: 0.8rem;
  text-transform: lowercase;
  letter-spacing: 0.06em;
}
.sb-nav-links a { color: var(--muted); }

/* ---------- hero placeholder ---------- */

.sb-hero {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 40vh;
  background: var(--surface);
  border-block: var(--hairline);
  color: var(--muted);
}

/* ---------- footer ---------- */

.sb-footer {
  padding-block: var(--s-5);
  border-top: var(--hairline);
  color: var(--muted);
  font-size: 0.78rem;
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` and confirm:
- Nav shows `sunbeams.` on the left, four links on the right.
- Section anchors work: clicking `install`/`what`/`devices` scrolls to the labeled placeholder sections.
- Pressing Tab on page load shows the yellow "Skip to install" link sliding in from the top; pressing Enter jumps to the install section.
- Focus rings are yellow (`--accent`) with ≥ 2 px outline.

Stop the server:

```bash
kill $SERVER_PID
```

Expected: all four checks pass.

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): HTML skeleton with nav, anchor sections, skip link, copy script"
```

---

## Task 4: Hero SVG

**Files:**
- Modify: `site/index.html` (replace the `<div class="sb-hero-placeholder">` with the full inline SVG)
- Modify: `site/style.css` (replace the placeholder `.sb-hero` block)

The hero composition, defined in the 1000×520 viewBox:

- **Sun** at `(380, 320)`, radius 140. Clean disc — no stripes. Soft radial halo behind it.
- **Perspective grid floor** below y=320, with all vanishing lines converging at `(620, 320)` (off-center right).
- **Sky rays** fanning from the sun in nine directions across the upper hemisphere (three layers: outer halo / primary / inner core).
- **Six delivery beams** — narrow at the sun (3 px half-width), wide at the device (14 px half-width), rendered in two layers (outer glow + crisp inner).
- **Six device viewports** — plain rectangles at each device's native aspect ratio, filled with a subtle gradient, labeled below with their resolution in mono.
- **Title lockup** in the upper-left quadrant: `SUNBEAMS` at ~72 px with extreme letter-spacing, a `HEADLESS · VIRTUAL · STREAMING` kicker beneath.
- No subtitle sentence.

Device positions and aspect viewports:

| Device | Center | Viewport size | Top y | Resolution label |
|---|---|---|---|---|
| 4K TV | (120, 400) | 96×54 (16:9) | 373 | `3840×2160` |
| Steam Deck | (240, 410) | 80×50 (16:10) | 385 | `1280×800` |
| Phone | (380, 420) | 22×48 (9:20) | 396 | `2400×1080` |
| MacBook | (520, 410) | 90×60 (3:2) | 380 | `3024×1964` |
| iPad | (680, 405) | 64×48 (4:3) | 381 | `2420×1668` |
| Ultrawide | (870, 395) | 120×51 (21:9) | 370 | `3440×1440` |

- [ ] **Step 1: Replace the hero placeholder with the SVG**

In `site/index.html`, replace this block:

```html
    <section id="hero" class="sb-hero" aria-label="Sunbeams fans one virtual display out to every client resolution">
      <!-- Inline SVG added in Task 4 -->
      <div class="sb-hero-placeholder">HERO</div>
    </section>
```

with:

```html
    <section id="hero" class="sb-hero">
      <svg class="sb-hero-svg" viewBox="0 0 1000 520" role="img"
           aria-labelledby="hero-title hero-desc" preserveAspectRatio="xMidYMid slice">
        <title id="hero-title">Sunbeams — one virtual display fans out to every client</title>
        <desc id="hero-desc">Diagram: one virtual display fans out to a 4K TV, an ultrawide, a MacBook, an iPad, a phone, and a Steam Deck — each labeled with its native resolution.</desc>

        <defs>
          <linearGradient id="sb-sky" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#230a44"/>
            <stop offset="55%" stop-color="#6b1b60"/>
            <stop offset="100%" stop-color="#2a0a4a"/>
          </linearGradient>
          <linearGradient id="sb-floor" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#2a0a4a"/>
            <stop offset="100%" stop-color="#04020a"/>
          </linearGradient>
          <linearGradient id="sb-sun" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#fff2a6"/>
            <stop offset="55%" stop-color="#ff7ab0"/>
            <stop offset="100%" stop-color="#ff2f6b"/>
          </linearGradient>
          <radialGradient id="sb-halo" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stop-color="#ffd46a" stop-opacity="0.45"/>
            <stop offset="60%" stop-color="#ff3d7f" stop-opacity="0.18"/>
            <stop offset="100%" stop-color="#ff3d7f" stop-opacity="0"/>
          </radialGradient>
          <linearGradient id="sb-skyray" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="25%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="65%" stop-color="#ff7ab0" stop-opacity="0.55"/>
            <stop offset="100%" stop-color="#ff3d7f" stop-opacity="0"/>
          </linearGradient>

          <linearGradient id="sb-beam-tv"    x1="380" y1="320" x2="120" y2="373" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>
          <linearGradient id="sb-beam-deck"  x1="380" y1="320" x2="240" y2="385" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>
          <linearGradient id="sb-beam-phone" x1="380" y1="320" x2="380" y2="396" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>
          <linearGradient id="sb-beam-mac"   x1="380" y1="320" x2="520" y2="380" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>
          <linearGradient id="sb-beam-ipad"  x1="380" y1="320" x2="680" y2="381" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>
          <linearGradient id="sb-beam-uw"    x1="380" y1="320" x2="870" y2="370" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#fff7c2" stop-opacity="1"/>
            <stop offset="55%" stop-color="#ffe066" stop-opacity="0.95"/>
            <stop offset="90%" stop-color="#ff9a6f" stop-opacity="0.85"/>
            <stop offset="100%" stop-color="#ff7ab0" stop-opacity="0.6"/>
          </linearGradient>

          <linearGradient id="sb-screen" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%"  stop-color="#2a1a55"/>
            <stop offset="100%" stop-color="#160427"/>
          </linearGradient>

          <filter id="sb-glow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="3.2" result="b"/>
            <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
          </filter>
          <filter id="sb-glow-strong" x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="6" result="b"/>
            <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
          </filter>

          <!-- A single upward-pointing ray, re-used at many angles via <use transform="rotate(...)"> -->
          <polygon id="sb-ray-primary" points="370,320 378.5,-380 381.5,-380 390,320" fill="url(#sb-skyray)"/>
          <polygon id="sb-ray-outer"   points="360,320 376,-380 384,-380 400,320" fill="url(#sb-skyray)"/>
          <polygon id="sb-ray-core"    points="377,320 379.3,-380 380.7,-380 383,320" fill="url(#sb-skyray)"/>
        </defs>

        <!-- Sky + floor -->
        <rect width="1000" height="320" fill="url(#sb-sky)"/>
        <rect y="320" width="1000" height="200" fill="url(#sb-floor)"/>

        <!-- Sun halo + disc (no carved stripes) -->
        <circle cx="380" cy="320" r="260" fill="url(#sb-halo)"/>
        <circle cx="380" cy="320" r="140" fill="url(#sb-sun)"/>
        <circle cx="380" cy="320" r="140" fill="none" stroke="#ffe066" stroke-width="1.2" opacity="0.55"/>

        <!-- Perspective grid: vanishing point at (620, 320), off-center right -->
        <g stroke="#ff4d8f" stroke-width="1.4" opacity="0.82">
          <line x1="0" y1="320" x2="1000" y2="320"/>
          <line x1="0" y1="340" x2="1000" y2="340"/>
          <line x1="0" y1="365" x2="1000" y2="365"/>
          <line x1="0" y1="395" x2="1000" y2="395"/>
          <line x1="0" y1="435" x2="1000" y2="435"/>
          <line x1="0" y1="485" x2="1000" y2="485"/>
          <line x1="620" y1="320" x2="-300" y2="520"/>
          <line x1="620" y1="320" x2="0"    y2="520"/>
          <line x1="620" y1="320" x2="200"  y2="520"/>
          <line x1="620" y1="320" x2="360"  y2="520"/>
          <line x1="620" y1="320" x2="480"  y2="520"/>
          <line x1="620" y1="320" x2="560"  y2="520"/>
          <line x1="620" y1="320" x2="620"  y2="520"/>
          <line x1="620" y1="320" x2="700"  y2="520"/>
          <line x1="620" y1="320" x2="800"  y2="520"/>
          <line x1="620" y1="320" x2="930"  y2="520"/>
          <line x1="620" y1="320" x2="1200" y2="520"/>
        </g>

        <!-- Sky rays — outer halo (very blurred, low opacity) -->
        <g filter="url(#sb-glow-strong)" opacity="0.55">
          <use href="#sb-ray-outer" transform="rotate(-90 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(-65 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(-40 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(-15 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(10 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(35 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(60 380 320)"/>
          <use href="#sb-ray-outer" transform="rotate(85 380 320)"/>
        </g>

        <!-- Sky rays — primary (medium blur, full gradient) -->
        <g filter="url(#sb-glow)">
          <use href="#sb-ray-primary" transform="rotate(-90 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(-70 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(-45 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(-20 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(5 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(25 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(50 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(70 380 320)"/>
          <use href="#sb-ray-primary" transform="rotate(90 380 320)"/>
        </g>

        <!-- Sky rays — inner crisp core (no blur, highlighted) -->
        <g opacity="0.95">
          <use href="#sb-ray-core" transform="rotate(-85 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(-60 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(-35 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(-10 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(15 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(40 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(65 380 320)"/>
          <use href="#sb-ray-core" transform="rotate(85 380 320)"/>
        </g>

        <!-- Delivery beams — outer glow layer (W_sun=3, W_device=14) -->
        <g filter="url(#sb-glow-strong)" opacity="0.75">
          <polygon points="379.40,317.06 117.20,359.28 122.80,386.72 380.60,322.94" fill="url(#sb-beam-tv)"/>
          <polygon points="378.74,317.28 234.11,372.30 245.89,397.70 381.26,322.72" fill="url(#sb-beam-deck)"/>
          <polygon points="377.00,320.00 366.00,396.00 394.00,396.00 383.00,320.00" fill="url(#sb-beam-phone)"/>
          <polygon points="378.82,322.76 514.48,392.87 525.52,367.13 381.18,317.24" fill="url(#sb-beam-mac)"/>
          <polygon points="379.40,322.94 677.21,394.72 682.79,367.28 380.60,317.06" fill="url(#sb-beam-ipad)"/>
          <polygon points="379.70,322.98 868.58,383.93 871.42,356.07 380.30,317.02" fill="url(#sb-beam-uw)"/>
        </g>

        <!-- Delivery beams — crisp inner core (W_sun=2, W_device=10) -->
        <g filter="url(#sb-glow)">
          <polygon points="379.60,318.04 118.00,363.20 122.00,382.80 380.40,321.96" fill="url(#sb-beam-tv)"/>
          <polygon points="379.16,318.19 235.79,375.93 244.21,394.07 380.84,321.81" fill="url(#sb-beam-deck)"/>
          <polygon points="378.00,320.00 370.00,396.00 390.00,396.00 382.00,320.00" fill="url(#sb-beam-phone)"/>
          <polygon points="379.21,321.84 516.06,389.19 523.94,370.81 380.79,318.16" fill="url(#sb-beam-mac)"/>
          <polygon points="379.60,321.96 678.01,390.80 681.99,371.20 380.40,318.04" fill="url(#sb-beam-ipad)"/>
          <polygon points="379.80,321.95 868.99,379.95 871.01,360.05 380.20,318.05" fill="url(#sb-beam-uw)"/>
        </g>

        <!-- Device viewports (plain rectangles at native aspect ratios) -->
        <g font-family="Departure Mono, ui-monospace, monospace" font-size="9" fill="#ffe066" letter-spacing="1">
          <!-- 4K TV 96x54 -->
          <g transform="translate(120,400)">
            <rect x="-48" y="-27" width="96" height="54" rx="1.5" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="44" text-anchor="middle">3840×2160</text>
          </g>
          <!-- Steam Deck 80x50 -->
          <g transform="translate(240,410)">
            <rect x="-40" y="-25" width="80" height="50" rx="1.5" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="41" text-anchor="middle">1280×800</text>
          </g>
          <!-- Phone 22x48 (vertical) -->
          <g transform="translate(380,420)">
            <rect x="-11" y="-24" width="22" height="48" rx="2.5" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="40" text-anchor="middle">2400×1080</text>
          </g>
          <!-- MacBook 90x60 -->
          <g transform="translate(520,410)">
            <rect x="-45" y="-30" width="90" height="60" rx="1.5" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="46" text-anchor="middle">3024×1964</text>
          </g>
          <!-- iPad 64x48 -->
          <g transform="translate(680,405)">
            <rect x="-32" y="-24" width="64" height="48" rx="2" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="40" text-anchor="middle">2420×1668</text>
          </g>
          <!-- Ultrawide 120x51 -->
          <g transform="translate(870,395)">
            <rect x="-60" y="-25.5" width="120" height="51" rx="1.5" fill="url(#sb-screen)" stroke="#ffe066" stroke-width="1.3"/>
            <text x="0" y="41" text-anchor="middle">3440×1440</text>
          </g>
        </g>

        <!-- Title lockup (upper-left) -->
        <text x="60" y="108" fill="#fff3b8" font-family="Departure Mono, ui-monospace, monospace"
              font-size="72" font-weight="900" letter-spacing="8"
              style="paint-order:stroke;stroke:#ff3d7f;stroke-width:1.5">SUNBEAMS</text>
        <text x="64" y="138" fill="#ffd7ea" font-family="Departure Mono, ui-monospace, monospace"
              font-size="12" letter-spacing="6">HEADLESS · VIRTUAL · STREAMING</text>
      </svg>
    </section>
```

- [ ] **Step 2: Replace the placeholder hero styles**

In `site/style.css`, replace the entire `/* ---------- hero placeholder ---------- */` block with:

```css
/* ---------- hero ---------- */

.sb-hero {
  position: relative;
  width: 100%;
  aspect-ratio: 1000 / 520;
  overflow: hidden;
  background: #07050f;
  border-block: var(--hairline);
}

.sb-hero-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/`. Confirm:
- The hero fills the width of the viewport and maintains its 1000:520 aspect.
- Sun is visibly off-center to the **left** (around 38% of the width).
- Perspective grid lines converge to a point **right** of center (around 62%).
- No horizontal stripes carved into the sun.
- Nine sky rays fan upward across the purple sky.
- Six delivery beams reach from the sun to six device viewports on the grid floor, narrow where they leave the sun and wide where they land.
- Each device is a plain rectangle with a yellow resolution label (`3840×2160`, `1280×800`, `2400×1080`, `3024×1964`, `2420×1668`, `3440×1440`).
- `SUNBEAMS` title sits in the upper-left, tracked out, with a pink outline.
- Kicker `HEADLESS · VIRTUAL · STREAMING` sits beneath the title.
- No subtitle sentence below the kicker.

Also open DevTools and:
- Run `document.querySelector('.sb-hero-svg').getAttribute('aria-labelledby')` — expected `"hero-title hero-desc"`.
- Narrow the viewport to ~400 px wide. The hero crops cleanly without stretching. The grid and outermost rays may clip — acceptable.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): hero SVG — asymmetric sun, perspective grid, sky rays, delivery beams, device viewports"
```

---

## Task 5: Hero-to-Problem continuing beam

**Files:**
- Modify: `site/index.html` (add a small decorative SVG to the top of the Problem section)
- Modify: `site/style.css` (position the decoration)

Goal: visually continue the central (phone) delivery beam out of the hero's viewBox into the top of the Problem section, so there is no visual break between them. The hero's phone beam exits at roughly `x = 380 / 1000 = 38%` of the hero's width.

- [ ] **Step 1: Add a decorative SVG to the Problem section**

In `site/index.html`, replace:

```html
    <section id="problem" class="sb-problem section">
      <!-- Problem rows added in Task 7 -->
      <p class="label">the problem</p>
    </section>
```

with:

```html
    <section id="problem" class="sb-problem section">
      <svg class="sb-beam-continuation" viewBox="0 0 100 60" aria-hidden="true"
           preserveAspectRatio="none">
        <defs>
          <linearGradient id="sb-cont" x1="0" y1="0" x2="0" y2="1" gradientUnits="userSpaceOnUse">
            <stop offset="0%"  stop-color="#ff9a6f" stop-opacity="0.9"/>
            <stop offset="60%" stop-color="#ff7ab0" stop-opacity="0.45"/>
            <stop offset="100%" stop-color="#ff3d7f" stop-opacity="0"/>
          </linearGradient>
          <filter id="sb-cont-glow" x="-50%" y="-20%" width="200%" height="140%">
            <feGaussianBlur stdDeviation="2" result="b"/>
            <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
          </filter>
        </defs>
        <polygon filter="url(#sb-cont-glow)"
                 points="45,0 42,60 58,60 55,0" fill="url(#sb-cont)"/>
      </svg>
      <div class="container">
        <p class="label">the problem</p>
      </div>
    </section>
```

- [ ] **Step 2: Style the continuation and position it**

In `site/style.css`, replace the `.section` declaration and add the new rules. Find the existing `.section` rule and replace with:

```css
.section {
  padding-block: var(--s-6);
  border-top: var(--hairline);
}
.sb-problem {
  position: relative;
  border-top: none;
  padding-top: 0;
}
.sb-beam-continuation {
  position: absolute;
  top: 0;
  left: 38%;
  transform: translateX(-50%);
  width: 140px;
  height: 60px;
  pointer-events: none;
}
.sb-problem > .container {
  padding-top: var(--s-6);
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` and scroll down past the hero. Confirm:
- A glowing peach-to-magenta vertical beam spills out of the bottom of the hero at ~38% of the width, continues downward into the top of the Problem section, and fades to nothing within ~60 px.
- The beam visually aligns with the phone-delivery beam in the hero (they're both at x ≈ 38%).
- No horizontal rule appears between the hero and the Problem section.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): continue central hero beam into the problem section"
```

---

## Task 6: Install line

**Files:**
- Modify: `site/index.html` (install section content)
- Modify: `site/style.css` (install component styles)

- [ ] **Step 1: Replace the install placeholder with the mono line**

In `site/index.html`, replace:

```html
    <section id="install" class="sb-install section">
      <!-- Install line added in Task 6 -->
      <p class="label">install</p>
    </section>
```

with:

```html
    <section id="install" class="sb-install section">
      <div class="container">
        <p class="label">install</p>
        <div class="sb-install-line">
          <code id="install-cmd"><span class="sb-prompt">$</span> curl -sSL https://asdfgasfhsn.github.io/sunbeams/install.sh | sh</code>
          <button class="sb-copy" type="button" data-copy-target="install-cmd"
                  aria-label="Copy install command to clipboard">
            <span data-copy-label aria-live="polite">[copy]</span>
          </button>
        </div>
        <p class="sb-install-hint">
          Installs to <code>/usr/local/bin/sunbeams</code>. One virtual display, every client resolution — Bazzite Desktop, KDE Plasma on Wayland.
        </p>
      </div>
    </section>
```

- [ ] **Step 2: Add install line styles**

Append to `site/style.css`:

```css
/* ---------- install line ---------- */

.sb-install-line {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  margin-top: var(--s-3);
  padding-block: var(--s-2);
  font-size: 0.9rem;
  overflow-x: auto;
  border-bottom: var(--hairline);
  padding-bottom: var(--s-3);
}

.sb-install-line code {
  flex: 1;
  white-space: nowrap;
  color: var(--fg);
}

.sb-prompt {
  color: var(--accent);
  margin-right: 0.6ch;
}

.sb-copy {
  color: var(--muted);
  font-size: 0.78rem;
  letter-spacing: 0.04em;
  transition: color 150ms ease;
  flex-shrink: 0;
}
.sb-copy:hover { color: var(--accent); }

.sb-install-hint {
  margin: var(--s-3) 0 0;
  color: var(--muted);
  font-size: 0.8rem;
}
.sb-install-hint code {
  color: var(--accent);
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` and confirm:
- Below the hero, a flush monospace line starts with a yellow `$`, shows the full curl command, and ends with `[copy]` on the right.
- No pill, no rounded container background — just a line with a hairline under it.
- Clicking `[copy]` swaps the label to `[copied]` for 2 s, then back.
- Pasting the clipboard contents somewhere confirms the copied text is `curl -sSL https://asdfgasfhsn.github.io/sunbeams/install.sh | sh` (the leading `$ ` stripped by the script).
- A muted hint appears below with `/usr/local/bin/sunbeams` rendered in yellow.

Focus test:
- Tab onto the Copy button. Confirm a yellow outline.
- Press Space/Enter. The label flips to `[copied]` and a screen reader (if enabled) announces it via `aria-live`.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): flush monospace install line with accessible copy affordance"
```

---

## Task 7: Problem section

**Files:**
- Modify: `site/index.html` (problem rows)
- Modify: `site/style.css` (problem component styles)

- [ ] **Step 1: Fill in the problem rows**

In `site/index.html`, replace the container contents of `#problem` (leave the `.sb-beam-continuation` SVG alone). Find:

```html
      <div class="container">
        <p class="label">the problem</p>
      </div>
```

and replace with:

```html
      <div class="container">
        <p class="label">the problem</p>
        <h2 class="sb-problem-heading">Headless streaming breaks on three things.</h2>
        <ol class="sb-problem-rows">
          <li>
            <span class="sb-problem-num" aria-hidden="true">(1)</span>
            <div>
              <h3>No display, no EDID.</h3>
              <p>The GPU reports zero modes without a monitor plugged in. Sunshine has nothing to stream.</p>
            </div>
          </li>
          <li>
            <span class="sb-problem-num" aria-hidden="true">(2)</span>
            <div>
              <h3>One EDID, one resolution.</h3>
              <p>Copying a real monitor's EDID gives you <em>that</em> monitor — not a 4K TV, an ultrawide, a MacBook, <em>and</em> an iPad.</p>
            </div>
          </li>
          <li>
            <span class="sb-problem-num" aria-hidden="true">(3)</span>
            <div>
              <h3>Bazzite fights back.</h3>
              <p>Immutable <code>/usr</code>, early-KMS timing, Wayland session switching — all have to agree before the stream comes up.</p>
            </div>
          </li>
        </ol>
      </div>
```

- [ ] **Step 2: Add problem row styles**

Append to `site/style.css`:

```css
/* ---------- problem rows ---------- */

.sb-problem-heading {
  margin: var(--s-3) 0 var(--s-5);
  max-width: 28ch;
  font-size: 1.4rem;
  line-height: 1.35;
}

.sb-problem-rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: var(--s-3);
}

.sb-problem-rows li {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--s-4);
  padding-block: var(--s-3);
  border-top: var(--hairline);
}
.sb-problem-rows li:last-child { border-bottom: var(--hairline); }

.sb-problem-num {
  color: var(--accent);
  font-size: 0.9rem;
  padding-top: 0.15rem;
  min-width: 2.4ch;
}

.sb-problem-rows h3 {
  color: var(--fg);
  margin-bottom: 0.2rem;
  font-size: 1rem;
}

.sb-problem-rows p {
  color: var(--muted);
  margin: 0;
}

.sb-problem-rows em {
  color: var(--fg);
  font-style: normal;
  letter-spacing: 0.02em;
}

.sb-problem-rows code {
  color: var(--accent);
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` and scroll to the Problem section. Confirm:
- Heading reads `Headless streaming breaks on three things.`
- Three rows are stacked vertically (not side-by-side cards), separated by hairline rules, each starting with `(1)`, `(2)`, `(3)` in yellow.
- Bold/emphasised words (`that`, `and`) render in primary text color, not italic — the CSS strips the italic.
- `/usr` inside row (3) renders in yellow.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): problem section — three stacked rows with bracketed numbers"
```

---

## Task 8: What-it-does rows

**Files:**
- Modify: `site/index.html` (three full-width rows)
- Modify: `site/style.css` (row component styles)

- [ ] **Step 1: Fill in the what-it-does rows**

In `site/index.html`, replace:

```html
    <section id="what" class="sb-what section">
      <!-- "What it does" rows added in Task 8 -->
      <p class="label">what sunbeams does</p>
    </section>
```

with:

```html
    <section id="what" class="sb-what section">
      <div class="container">
        <p class="label">what sunbeams does</p>
        <h2 class="sb-what-heading">One binary, three jobs.</h2>
        <div class="sb-rows">
          <article class="sb-row">
            <div class="sb-row-verb">
              <span class="sb-bracket">[generate]</span>
              <p>Synthesises the EDID</p>
            </div>
            <div class="sb-row-body">
              <code class="sb-row-cmd"><span class="sb-prompt">$</span> sunbeams generate</code>
              <p>Packs every target resolution — 4K, ultrawide, MacBook, iPad, phones, handhelds — into a single EDID binary with HDR10 metadata and wide range limits. Outputs helper scripts for modes that exceed the EDID pixel-clock ceiling.</p>
            </div>
          </article>
          <article class="sb-row">
            <div class="sb-row-verb">
              <span class="sb-bracket">[switch]</span>
              <p>Switches the display</p>
            </div>
            <div class="sb-row-body">
              <code class="sb-row-cmd"><span class="sb-prompt">$</span> sunbeams switch on  /  off</code>
              <p>Wires into Sunshine's Do/Undo prep commands. Reads <code>SUNSHINE_CLIENT_*</code>, snaps the request to the nearest configured mode, drives <code>kscreen-doctor</code> in one atomic call, and logs every decision to stderr.</p>
            </div>
          </article>
          <article class="sb-row">
            <div class="sb-row-verb">
              <span class="sb-bracket">[install]</span>
              <p>Installs on Bazzite</p>
            </div>
            <div class="sb-row-body">
              <code class="sb-row-cmd"><span class="sb-prompt">$</span> sudo sunbeams install</code>
              <p>Scans <code>/sys/class/drm</code> for a free connector, writes the EDID to <code>/etc/firmware/</code> (bypassing the immutable <code>/usr</code>), and injects <code>firmware_class.path</code>, <code>drm.edid_firmware</code>, and <code>video=</code> kernel arguments via <code>rpm-ostree kargs</code>.</p>
            </div>
          </article>
        </div>
      </div>
    </section>
```

- [ ] **Step 2: Add the row styles**

Append to `site/style.css`:

```css
/* ---------- what-it-does rows ---------- */

.sb-what-heading {
  margin: var(--s-3) 0 var(--s-5);
  font-size: 1.4rem;
  line-height: 1.35;
}

.sb-rows {
  display: grid;
  gap: 0;
  border-top: var(--hairline);
}

.sb-row {
  display: grid;
  grid-template-columns: 18rem 1fr;
  gap: var(--s-5);
  padding-block: var(--s-5);
  border-bottom: var(--hairline);
}

.sb-row-verb {
  display: flex;
  flex-direction: column;
  gap: var(--s-2);
}

.sb-bracket {
  color: var(--accent);
  font-size: 0.82rem;
  letter-spacing: 0.06em;
}

.sb-row-verb p {
  margin: 0;
  color: var(--fg);
  font-size: 1.05rem;
  font-weight: 700;
}

.sb-row-body {
  display: flex;
  flex-direction: column;
  gap: var(--s-3);
}

.sb-row-cmd {
  font-size: 0.9rem;
  color: var(--fg);
  border-left: 2px solid var(--accent);
  padding-left: var(--s-3);
}

.sb-row-body p {
  color: var(--muted);
  margin: 0;
  max-width: 60ch;
  font-size: 0.88rem;
}

.sb-row-body code {
  color: var(--accent);
}
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Confirm:
- Heading: `One binary, three jobs.`
- Three full-width rows stacked vertically, each separated by hairlines.
- Each row's left column shows the bracketed label (`[generate]`, `[switch]`, `[install]`) in yellow above a short verb phrase in bold white.
- Each row's right column shows a monospace command prefixed with a yellow `$` and a 2-sentence description in muted grey. The command has a yellow left border.
- No numbered `01 / 02 / 03` chips anywhere.
- Inline `code` references (`/sys/class/drm`, `/etc/firmware/`, `rpm-ostree kargs`, etc.) render in yellow.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): what-it-does as three full-width rows with bracketed verbs"
```

---

## Task 9: Device mosaic

**Files:**
- Modify: `site/index.html` (mosaic tiles)
- Modify: `site/style.css` (proportional CSS Grid)

CSS Grid spans are per-tile; bigger resolution → visibly bigger tile. The grid is 12 columns wide. Spans:

| Tile | Columns | Rows |
|---|---|---|
| 4K OLED | 6 | 2 |
| Ultrawide | 8 | 1 |
| MacBook Pro 14" | 5 | 2 |
| 1440p | 4 | 1 |
| iPad Pro 11" | 4 | 2 |
| Android phone | 3 | 2 |
| 1080p | 4 | 1 |
| Steam Deck | 3 | 1 |
| PS Vita | 2 | 1 |
| PSP | 2 | 1 |
| + more | 3 | 1 |

The grid auto-places these in rows; gaps are filled by empty implicit cells. The eye sees a visibly uneven mosaic — which is the point.

- [ ] **Step 1: Fill in the mosaic**

In `site/index.html`, replace:

```html
    <section id="devices" class="sb-devices section">
      <!-- Device mosaic added in Task 9 -->
      <p class="label">every device, one display</p>
    </section>
```

with:

```html
    <section id="devices" class="sb-devices section">
      <div class="container">
        <p class="label">every device, one display</p>
        <h2 class="sb-devices-heading">Baked into the default EDID. Sized by pixel count.</h2>
        <ul class="sb-mosaic">
          <li class="sb-tile sb-tile--xl" style="--c:6;--r:2;">
            <span class="sb-tile-name">4K OLED</span>
            <span class="sb-tile-res">3840 × 2160</span>
            <span class="sb-tile-tag">@60 · HDR10</span>
          </li>
          <li class="sb-tile sb-tile--wide" style="--c:8;--r:1;">
            <span class="sb-tile-name">Ultrawide</span>
            <span class="sb-tile-res">3440 × 1440</span>
            <span class="sb-tile-tag">@100</span>
          </li>
          <li class="sb-tile sb-tile--l" style="--c:5;--r:2;">
            <span class="sb-tile-name">MacBook Pro 14"</span>
            <span class="sb-tile-res">3024 × 1964</span>
            <span class="sb-tile-tag">@120</span>
          </li>
          <li class="sb-tile sb-tile--m" style="--c:4;--r:1;">
            <span class="sb-tile-name">1440p</span>
            <span class="sb-tile-res">2560 × 1440</span>
            <span class="sb-tile-tag">@144</span>
          </li>
          <li class="sb-tile sb-tile--m" style="--c:4;--r:2;">
            <span class="sb-tile-name">iPad Pro 11"</span>
            <span class="sb-tile-res">2420 × 1668</span>
            <span class="sb-tile-tag">@120</span>
          </li>
          <li class="sb-tile sb-tile--tall" style="--c:3;--r:2;">
            <span class="sb-tile-name">Android phone</span>
            <span class="sb-tile-res">2400 × 1080</span>
            <span class="sb-tile-tag">@60</span>
          </li>
          <li class="sb-tile sb-tile--m" style="--c:4;--r:1;">
            <span class="sb-tile-name">1080p</span>
            <span class="sb-tile-res">1920 × 1080</span>
            <span class="sb-tile-tag">@120</span>
          </li>
          <li class="sb-tile sb-tile--s" style="--c:3;--r:1;">
            <span class="sb-tile-name">Steam Deck</span>
            <span class="sb-tile-res">1280 × 800</span>
            <span class="sb-tile-tag">@60</span>
          </li>
          <li class="sb-tile sb-tile--xs" style="--c:2;--r:1;">
            <span class="sb-tile-name">PS Vita</span>
            <span class="sb-tile-res">960 × 544</span>
            <span class="sb-tile-tag">@60</span>
          </li>
          <li class="sb-tile sb-tile--xs" style="--c:2;--r:1;">
            <span class="sb-tile-name">PSP</span>
            <span class="sb-tile-res">480 × 272</span>
            <span class="sb-tile-tag">@60</span>
          </li>
          <li class="sb-tile sb-tile--more" style="--c:3;--r:1;">
            <a href="https://github.com/asdfgasfhsn/sunbeams#supported-devices-default">
              + more · <code>sunbeams devices</code>
            </a>
          </li>
        </ul>
      </div>
    </section>
```

- [ ] **Step 2: Add the mosaic styles**

Append to `site/style.css`:

```css
/* ---------- device mosaic ---------- */

.sb-devices-heading {
  margin: var(--s-3) 0 var(--s-5);
  font-size: 1.4rem;
  line-height: 1.35;
}

.sb-mosaic {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  grid-auto-rows: 90px;
  gap: 4px;
}

.sb-tile {
  grid-column: span var(--c);
  grid-row: span var(--r);
  background: var(--surface);
  border: 1px solid rgba(255, 224, 102, 0.18);
  padding: var(--s-3);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  transition: border-color 150ms ease, background 150ms ease;
  position: relative;
  overflow: hidden;
}
.sb-tile:hover { border-color: var(--accent); }

.sb-tile-name {
  color: var(--fg);
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.03em;
}

.sb-tile-res {
  color: var(--muted);
  font-size: 0.78rem;
  letter-spacing: 0.04em;
}

.sb-tile-tag {
  color: var(--accent);
  font-size: 0.7rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sb-tile--xl .sb-tile-name { font-size: 1.3rem; }
.sb-tile--xl .sb-tile-res  { font-size: 1rem; }

.sb-tile--l .sb-tile-name  { font-size: 1.1rem; }
.sb-tile--l .sb-tile-res   { font-size: 0.9rem; }

.sb-tile--wide .sb-tile-name { font-size: 1.1rem; }
.sb-tile--wide .sb-tile-res  { font-size: 0.9rem; }

.sb-tile--more a {
  color: var(--accent);
  display: flex;
  align-items: center;
  height: 100%;
  font-size: 0.8rem;
  border-bottom: none;
}
.sb-tile--more code { color: var(--fg); }
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Confirm, at a desktop width ≥ 1024 px:
- The mosaic has **visibly uneven tile sizes**. The 4K OLED tile is the largest; PSP and PS Vita are the smallest.
- Each tile shows the device name in primary text, resolution in muted grey, and an accent-yellow tag in the lower area.
- Tile borders are near-invisible (low-opacity yellow) until hover, when they become solid yellow.
- The `+ more` tile contains a link to the repo's supported-devices anchor.
- There is no uniform grid feel.

The mosaic may wrap into 3–4 rows of varying heights. Empty gaps between tiles are OK — that's the natural result of auto-placed spans.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): device mosaic with tiles sized proportional to pixel count"
```

---

## Task 10: Quick-start terminal

**Files:**
- Modify: `site/index.html` (terminal block)
- Modify: `site/style.css` (terminal component styles + right-edge bleed)

- [ ] **Step 1: Fill in the terminal block**

In `site/index.html`, replace:

```html
    <section id="quickstart" class="sb-quickstart section">
      <!-- Terminal block added in Task 10 -->
      <p class="label">quick start</p>
    </section>
```

with:

```html
    <section id="quickstart" class="sb-quickstart section">
      <div class="container">
        <p class="label">quick start</p>
        <h2 class="sb-quickstart-heading">End-to-end on a Bazzite box.</h2>
      </div>
      <div class="sb-term-wrap">
        <div class="sb-term">
          <div class="sb-term-head">
            <span class="sb-term-path">~/bazzite</span>
            <span class="sb-term-prompt">$</span>
          </div>
<pre class="sb-term-body"><span class="c">#  1. generate the EDID + helper scripts</span>
<span class="p">$</span> sunbeams generate

<span class="c">#  2. install: EDID → /etc/firmware, kernel args via rpm-ostree, then reboot</span>
<span class="p">$</span> sudo sunbeams install

<span class="c">#  3. inspect what the EDID exposes</span>
<span class="p">$</span> sunbeams devices
<span class="p">$</span> sunbeams modes

<span class="c">#  4. wire into Sunshine → General → prep commands:</span>
<span class="c">#     Do:   sunbeams switch on</span>
<span class="c">#     Undo: sunbeams switch off</span></pre>
        </div>
      </div>
    </section>
```

- [ ] **Step 2: Add terminal styles + right-edge bleed**

Append to `site/style.css`:

```css
/* ---------- quick-start terminal ---------- */

.sb-quickstart-heading {
  margin: var(--s-3) 0 var(--s-5);
  font-size: 1.4rem;
  line-height: 1.35;
}

.sb-term-wrap {
  margin-top: var(--s-4);
  padding-left: calc((100vw - 1120px) / 2 + var(--s-4));
  padding-right: 0;
}

@media (max-width: 1152px) {
  .sb-term-wrap {
    padding-left: var(--s-4);
  }
}

.sb-term {
  background: #060310;
  border: 1px solid rgba(255, 224, 102, 0.22);
  border-right: none;
  overflow: hidden;
}

.sb-term-head {
  display: flex;
  align-items: center;
  gap: var(--s-3);
  padding: var(--s-2) var(--s-3);
  background: rgba(255, 224, 102, 0.04);
  border-bottom: 1px solid rgba(255, 224, 102, 0.18);
  font-size: 0.78rem;
}

.sb-term-path {
  color: var(--muted);
  letter-spacing: 0.04em;
}

.sb-term-prompt {
  color: var(--accent);
  font-weight: 700;
}

.sb-term-body {
  padding: var(--s-4) var(--s-4);
  font-size: 0.88rem;
  line-height: 1.8;
  color: var(--fg);
  overflow-x: auto;
}

.sb-term-body .c { color: var(--muted); }
.sb-term-body .p { color: var(--accent); }
```

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Confirm at desktop width ≥ 1200 px:
- Terminal block sits below its heading.
- A tinted header strip at the top reads `~/bazzite` in muted grey + `$` in yellow. **No Mac traffic-light dots.**
- Body is a monospace pre-block showing the four-step flow. Comment lines are muted grey, `$` prompts are yellow, command text is primary white.
- The terminal's **right edge runs off the screen** — it is flush against the viewport's right edge (bleed), no container padding on the right.
- At widths < 1152 px, the left padding collapses to the standard container padding and the bleed continues on the right.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): quick-start terminal block with right-edge viewport bleed"
```

---

## Task 11: Footer

**Files:**
- Modify: `site/index.html` (footer content)
- Modify: `site/style.css` (footer styles already mostly in place — add link styling)

- [ ] **Step 1: Fill in the footer**

In `site/index.html`, replace:

```html
  <footer class="sb-footer container">
    <!-- Footer added in Task 11 -->
    <p>footer placeholder</p>
  </footer>
```

with:

```html
  <footer class="sb-footer container">
    <p class="sb-footer-meta">MIT · sunbeams · headless streaming, honestly</p>
    <nav class="sb-footer-links" aria-label="Project links">
      <a href="https://github.com/asdfgasfhsn/sunbeams">GitHub</a>
      <a href="https://github.com/asdfgasfhsn/sunbeams/releases">Releases</a>
      <a href="https://github.com/asdfgasfhsn/sunbeams#readme">README</a>
      <a href="https://github.com/asdfgasfhsn/sunbeams/tree/main/docs">Docs</a>
    </nav>
  </footer>
```

- [ ] **Step 2: Add footer layout styles**

Append to `site/style.css`:

```css
/* ---------- footer ---------- */

.sb-footer {
  display: flex;
  flex-wrap: wrap;
  gap: var(--s-3);
  justify-content: space-between;
  align-items: center;
}

.sb-footer-meta {
  margin: 0;
}

.sb-footer-links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--s-3);
  font-size: 0.78rem;
}

.sb-footer-links a {
  color: var(--muted);
}
.sb-footer-links a:hover { color: var(--accent); }
```

Note: there's already a `.sb-footer` rule from Task 3 with padding/border-top/color/font-size. This step extends it with the flex layout; the earlier rule stays in place.

- [ ] **Step 3: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Confirm:
- Footer runs as a single horizontal row on desktop: meta on the left, four links on the right.
- Hovering any footer link turns it yellow.
- At narrow widths (< 480 px), the flex row wraps — acceptable.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 4: Commit**

```bash
git add site/index.html site/style.css
git commit -m "feat(site): footer with project links"
```

---

## Task 12: Responsive collapse at 720 px

**Files:**
- Modify: `site/style.css` (single `@media` block)

- [ ] **Step 1: Add the responsive rules**

Append to `site/style.css`:

```css
/* ---------- responsive ---------- */

@media (max-width: 720px) {
  :root {
    --s-6: 2.2rem;
    --s-7: 3rem;
  }

  .sb-nav-links { gap: var(--s-3); }

  .sb-install-line {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--s-2);
  }
  .sb-install-line code { white-space: normal; word-break: break-all; }

  .sb-row {
    grid-template-columns: 1fr;
    gap: var(--s-3);
  }

  .sb-mosaic {
    grid-template-columns: repeat(2, 1fr);
    grid-auto-rows: auto;
  }
  .sb-tile {
    grid-column: span 1 !important;
    grid-row: span 1 !important;
    min-height: 84px;
  }
  .sb-tile--xl, .sb-tile--wide {
    grid-column: span 2 !important;
  }

  .sb-term-wrap {
    padding-left: var(--s-4);
    padding-right: var(--s-4);
  }
  .sb-term { border-right: 1px solid rgba(255, 224, 102, 0.22); }

  .sb-footer { flex-direction: column; align-items: flex-start; }
}
```

- [ ] **Step 2: Verify in browser**

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

Open `http://localhost:8000/` in DevTools with responsive mode at widths 400, 600, 720, 800, and 1200. Confirm:
- At ≤ 720 px: the mosaic is a uniform 2-column grid (4K and Ultrawide span both columns). The proportional sizing is gone.
- At ≤ 720 px: the what-it-does rows stack (label row above body row).
- At ≤ 720 px: the install line wraps the curl command onto multiple lines; `[copy]` sits beneath the code.
- At ≤ 720 px: the terminal block regains its right padding and has a right border — no more viewport bleed.
- At ≤ 720 px: the footer stacks meta above links.
- At > 720 px: all of the above return to their desktop layouts.
- At ≥ 1200 px: the terminal block bleeds right again.

Stop the server:

```bash
kill $SERVER_PID
```

- [ ] **Step 3: Commit**

```bash
git add site/style.css
git commit -m "feat(site): single responsive breakpoint at 720px"
```

---

## Task 13: Accessibility verification

**Files:** none modified in this task — verification only, with small fixes as needed.

- [ ] **Step 1: Skip link behaviour**

Serve and open the page:

```bash
python3 -m http.server 8000 --directory site &
SERVER_PID=$!
sleep 1
```

In the browser:
- Reload. Press Tab once. Confirm the yellow **"Skip to install"** pill slides in at the top-left.
- Press Enter. Confirm the viewport jumps to the Install section (URL changes to `#install`, the focus moves there).

- [ ] **Step 2: Keyboard traversal**

Reload the page. Tab through every interactive element in order, verifying each shows a yellow focus ring with ≥ 2 px outline and ≥ 3 px offset. Expected order:

1. Skip link (first stop, revealed)
2. `sunbeams.` wordmark
3. `install` nav link
4. `what` nav link
5. `devices` nav link
6. `github ↗` nav link
7. `[copy]` copy button
8. `+ more` mosaic link
9. GitHub footer link
10. Releases footer link
11. README footer link
12. Docs footer link

If any element has no visible focus ring, inspect it and ensure the global `:focus-visible` rule (`outline: 2px solid var(--accent); outline-offset: 3px;`) applies. If a descendant overrides `outline: none`, remove that override.

- [ ] **Step 3: Copy button announcement**

Focus the `[copy]` button and press Space. Confirm:
- Label swaps to `[copied]` for 2 s and back.
- In DevTools → Accessibility → Accessibility Tree, confirm the inner `<span data-copy-label>` has `aria-live="polite"` applied.
- If a screen reader is available (VoiceOver on macOS: `Cmd+F5`), confirm it announces the new text.

- [ ] **Step 4: Hero `<desc>` is read first**

Navigate to the hero SVG with a screen reader. Confirm the announced text matches: *"Diagram: one virtual display fans out to a 4K TV, an ultrawide, a MacBook, an iPad, a phone, and a Steam Deck — each labeled with its native resolution."*

- [ ] **Step 5: Contrast verification**

Open DevTools → Elements → select the body. Run:

```js
getComputedStyle(document.body).color
getComputedStyle(document.body).backgroundColor
```

Expected: `rgb(232, 236, 255)` on `rgb(7, 5, 15)` — ratio ≈ 18:1 (AAA).

Select a `.sb-tile` element. Inspect its background (`--surface: #0f0a22`). Pass its foreground text (`--fg: #e8ecff`) through a contrast checker (DevTools has one built in, or use `https://webaim.org/resources/contrastchecker/`). Expected: ≥ 15:1.

Select a `.sb-tile-tag` element. Its text color is `--accent: #ffe066` on `--surface: #0f0a22`. Expected contrast: ≥ 12:1 (AAA).

- [ ] **Step 6: Reduced-motion check**

In DevTools → Rendering → "Emulate CSS media feature prefers-reduced-motion" → `reduce`. Reload. Hover the footer links and the copy button. Confirm there is no color-transition animation — the color change is instant.

- [ ] **Step 7: Font fallback check**

Temporarily rename the font to simulate a load failure:

```bash
mv site/fonts/DepartureMono-Regular.woff2 site/fonts/DepartureMono-Regular.woff2.disabled
```

Reload the page. Confirm:
- The layout still holds — nothing breaks.
- Text renders in the OS monospace fallback (SF Mono / Cascadia Code / Consolas / Menlo, depending on platform).
- The page is still readable.

Restore:

```bash
mv site/fonts/DepartureMono-Regular.woff2.disabled site/fonts/DepartureMono-Regular.woff2
```

- [ ] **Step 8: HTML validity (one-off)**

Paste the rendered HTML into `https://validator.w3.org/nu/#textarea` or run `tidy`:

```bash
if command -v tidy >/dev/null 2>&1; then
  tidy -q -e site/index.html 2>&1 | head -30
else
  echo "tidy not installed; use https://validator.w3.org/ instead"
fi
```

Fix any errors (not warnings). Typical warnings like "trimming empty <code>" or "moving <pre> content" are fine. Commit any fixes as part of Step 9.

- [ ] **Step 9: Stop server and commit any fixes**

```bash
kill $SERVER_PID

# If any fixes were needed in Steps 1–8, stage and commit them:
git status
# If there are changes:
# git add site/...
# git commit -m "fix(site): accessibility adjustments from Task 13 verification"
```

If no fixes were required, skip the commit — the plan is complete.

---

## Self-review

**Spec coverage**

- Nav (spec §1) → Task 3.
- Hero illustration (spec §2) → Task 4.
- Install line (spec §3) → Task 6.
- Problem section + continuing beam (spec §4) → Tasks 5 + 7.
- What-it-does rows (spec §5) → Task 8.
- Device mosaic (spec §6) → Task 9.
- Quick-start terminal (spec §7) → Task 10.
- Footer (spec §8) → Task 11.
- Typography (one self-hosted mono) → Tasks 1 + 2.
- Four-token palette (below the fold) → Task 2.
- Hero-only warm ramp (inside SVG) → Task 4.
- Accessibility (skip link, focus-visible, aria-live, role/labelledby/desc, contrast) → Tasks 2, 3, 4, 6, 13.
- Responsive collapse at 720 px → Task 12.
- `prefers-reduced-motion` branch → Task 2.
- Font fallback stack and load-failure resilience → Task 13 (step 7).

**Placeholder / stale-reference scan**

- No "TBD" / "TODO" / "similar to Task N" in any task. Each task self-contained with full code.
- `install-cmd` element ID, `data-copy-target="install-cmd"`, `data-copy-label` attribute, `#install` anchor, `.sb-hero`, `.sb-problem` — all cross-task IDs/class names match across the plan.
- `SUNSHINE_CLIENT_*` and `kscreen-doctor` spelled consistently.

**Type / naming consistency**

- CSS custom properties (`--bg`, `--surface`, `--fg`, `--muted`, `--accent`, `--hairline`, `--radius`, `--font-mono`, `--s-1` … `--s-8`) introduced in Task 2 and used unchanged through subsequent tasks.
- Section IDs (`#hero`, `#install`, `#problem`, `#what`, `#devices`, `#quickstart`) introduced in Task 3 match the nav links and are not renamed later.
- Component class prefixes (`.sb-nav`, `.sb-hero`, `.sb-install`, `.sb-problem`, `.sb-rows`, `.sb-mosaic`, `.sb-tile`, `.sb-term`, `.sb-footer`) are consistent.
- SVG gradient / filter / polygon IDs are all `sb-*` prefixed in Task 4 and referenced only inside that SVG.

No issues found; plan is ready for execution.
