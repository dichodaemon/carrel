---
name: display-svg
description: >
  Display an SVG file in the browser with correct viewport scaling
  for the terminal display pipeline. Handles wide SVGs that exceed
  the 1024 px display limit.
  Usage: /display-svg <path>
---

# Display SVG

Display an SVG file as a browser screenshot scaled to fit the
terminal display pipeline. The display system clips images wider
than 1024 px, so wide SVGs (common with `--world-extents` or
`svg_primitives.js` diagrams) must be scaled before capture.

## Trigger

User says "display svg", "show svg", "screenshot svg", or invokes
`/display-svg`.

## Arguments

- Required: path to the SVG file (e.g., `/tmp/diagram.svg`).

## Step 1: Get the SVG native dimensions

```bash
cat <path>.svg | grep -oP '<svg[^>]*width="\K[0-9.]+' | head -1
cat <path>.svg | grep -oP '<svg[^>]*height="\K[0-9.]+' | head -1
```

The `<svg[^>]*` prefix anchors the match to the root `<svg>` tag
(stops at the first `>`). The `[0-9.]+` pattern handles fractional
values. `head -1` takes the first match.

## Step 2: Compute display dimensions

```
displayWidth  = min(nativeWidth, 1024)
displayHeight = ceil(displayWidth * nativeHeight / nativeWidth)
```

If the SVG is already ≤ 1024 px wide, use its native dimensions
as-is and skip the `tab.evaluate` resize in Step 4.

## Step 3: Close all browser tabs, then open a fresh one

```
browser action: close, all: true, kill: true

browser action: open
  name: "svg"
  url: "file://<absolute-path>"
  viewport: { width: <displayWidth>, height: <displayHeight>, scale: 1 }
```

Always close **all** tabs first. Reusing an existing tab causes
clipping even when the viewport dimensions are correct. `scale: 1`
prevents the default `deviceScaleFactor` (often 1.25) from inflating
the capture.

## Step 4: Scale the SVG element and screenshot

```js
await tab.evaluate(() => {
  const svg = document.querySelector('svg');
  svg.setAttribute('width', '<displayWidth>');
  svg.setAttribute('height', '<displayHeight>');
});
await new Promise(r => setTimeout(r, 200));
await tab.screenshot({ fullPage: true });
```

Setting `width`/`height` on the SVG element scales the rendering;
the `viewBox` attribute preserves the full content. The 200 ms delay
lets the browser re-layout before capture. `fullPage: true` captures
the entire rendered page.

If the SVG is already ≤ 1024 px wide (Step 2), skip the
`tab.evaluate` — just take the screenshot directly:

```js
await tab.screenshot({ fullPage: true });
```

## Common mistakes

- **Not closing all tabs before opening.** Stale tabs cause clipping
  even with correct viewport dimensions.
- **Using native SVG dimensions as viewport when width > 1024.**
  The display pipeline clips the image vertically — content is lost,
  not scaled.
- **Forgetting `scale: 1`.** The default `deviceScaleFactor` (often
  1.25) inflates the capture beyond 1024 px, triggering the same
  clipping.
- **Using `selector: 'svg'` instead of `fullPage: true`.** Produces
  different bounding than the native dimensions.
- **Skipping the `tab.evaluate` resize for wide SVGs.** The browser
  renders the SVG at its intrinsic `width`/`height`; setting the
  viewport alone does not scale the content.

## Rules

- **Display-safe width.** Screenshots must be ≤ 1024 px wide. Scale
  wide SVGs via `tab.evaluate` before capture. This only affects the
  preview — committed SVG files keep their native dimensions.
- **Never fabricate image descriptions.** Display the image and let
  the user evaluate it.
