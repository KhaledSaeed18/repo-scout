# Repo Scout brand

The mark is three contour lines tightening toward a summit, as on a survey
map: the steep side is where the lines bunch up, and the dot is the high
point. It matches the app's "survey map" interface and its contour colors.

## Files

| File | Use it for |
| --- | --- |
| `logo.svg`, `logo-dark.svg` | Mark and wordmark side by side. READMEs, docs, slides. Pick by background. |
| `logo.png`, `logo-dark.png` | Same, 256px tall, for places that do not take SVG. |
| `mark.svg`, `mark-dark.svg` | The mark alone, 33px and larger, on a plain background. |
| `mark-small.svg` | The two-contour cut for 32px and smaller, where three lines blur. |
| `app-icon.svg`, `app-icon-1024.png` | Rounded ink tile: app icon, avatars, store listings. |
| `app-icon-small.svg` | Tile with the small cut. Favicons. |
| `app-icon-square.svg` | Full-bleed tile for platforms that apply their own mask (iOS, maskable Android). |
| `social-preview.png`, `social-preview-dark.png` | 1280×640 link preview. Upload in GitHub under Settings › General › Social preview. |

Web icons generated from these live in `frontend/public`: `favicon.ico`
(16, 32, 48), `favicon.svg`, `apple-touch-icon.png`, `icon-192.png`,
`icon-512.png`, `icon-maskable-512.png` and `site.webmanifest`. Inside the app
the mark is drawn by `frontend/src/components/BrandMark.tsx` in theme colors.

## Colors

| Name | Light | Dark | Role |
| --- | --- | --- | --- |
| Ink | `#1B2420` | `#E1E8E3` | Contour lines, wordmark |
| Ultramarine | `#3346D3` | `#8E9CFF` | The summit |
| Paper | `#F2F4F1` | `#111714` | Backgrounds |

The app icon is an ink tile with paper contours and the dark-theme summit.

## Type

The wordmark is Barlow Semi Condensed SemiBold, converted to outlines. Body
text that sits next to the logo uses Barlow, as in the app.

## Use

- Keep clear space around the logo equal to the height of the summit ring
  (about a quarter of the mark's height).
- Use the mark alone at 33px and up, and `mark-small.svg` below that.
- Do not recolor, outline, add effects, stretch, or rotate the mark, and do not
  set the wordmark in another typeface.

## Regenerating

The SVGs are the source of truth. After editing them, render every PNG and
icon with:

```sh
./brand/build.sh   # needs: brew install librsvg imagemagick
```

`sources.py` rebuilds the SVGs themselves (geometry and outlined wordmark) if
the design changes; its header lists what it needs.
