#!/usr/bin/env bash
# Renders every raster brand asset from the SVG sources in this folder.
# Requires rsvg-convert (librsvg) and ImageMagick: brew install librsvg imagemagick
set -euo pipefail

BRAND="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLIC="$(cd "$BRAND/../frontend/public" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

render() { rsvg-convert -w "$2" -h "$2" "$BRAND/$1" -o "$3"; }

# Browser tabs are 16-32px, so they get the simplified two-contour cut.
cp "$BRAND/app-icon-small.svg" "$PUBLIC/favicon.svg"
render app-icon-small.svg 16 "$TMP/16.png"
render app-icon-small.svg 32 "$TMP/32.png"
render app-icon.svg 48 "$TMP/48.png"
magick "$TMP/16.png" "$TMP/32.png" "$TMP/48.png" "$PUBLIC/favicon.ico"

# Home screen and install icons. iOS and maskable Android icons are full-bleed
# squares because the platform applies its own rounded mask.
render app-icon-square.svg 180 "$PUBLIC/apple-touch-icon.png"
render app-icon.svg 192 "$PUBLIC/icon-192.png"
render app-icon.svg 512 "$PUBLIC/icon-512.png"
render app-icon-square.svg 512 "$PUBLIC/icon-maskable-512.png"

# Repository social preview (upload in GitHub: Settings > General > Social preview).
rsvg-convert -w 1280 -h 640 "$BRAND/social-preview.svg" -o "$BRAND/social-preview.png"
rsvg-convert -w 1280 -h 640 "$BRAND/social-preview-dark.svg" -o "$BRAND/social-preview-dark.png"

# PNGs for places that cannot take SVG (chat apps, slides, docs).
rsvg-convert -h 256 "$BRAND/logo.svg" -o "$BRAND/logo.png"
rsvg-convert -h 256 "$BRAND/logo-dark.svg" -o "$BRAND/logo-dark.png"
render app-icon.svg 1024 "$BRAND/app-icon-1024.png"

for f in "$PUBLIC"/*.png "$BRAND"/*.png; do magick "$f" -strip "$f"; done
echo "brand assets written to $PUBLIC and $BRAND"
