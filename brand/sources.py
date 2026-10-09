"""Generates the Repo Scout brand SVG sources in this folder.

The SVGs are committed and are the source of truth; run this only to change
the geometry or the wordmark. Text is converted to outlines so the files need
no fonts. Requires: pip install fonttools brotli uharfbuzz, and the frontend's
node_modules (for the bundled Barlow fonts).

    python brand/sources.py
"""
import math, os
import uharfbuzz as hb
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen

OUT = os.path.dirname(os.path.abspath(__file__))
FONTS = os.path.join(OUT, '..', 'frontend', 'node_modules', '@fontsource')
SEMI = f'{FONTS}/barlow-semi-condensed/files/barlow-semi-condensed-latin-600-normal.woff2'
REG = f'{FONTS}/barlow/files/barlow-latin-400-normal.woff2'

INK, PAPER, ULTRA = '#1B2420', '#F2F4F1', '#3346D3'
INK_D, ULTRA_D, MUTED = '#E1E8E3', '#8E9CFF', '#58655E'

# --- the mark: contour lines stepping north-east toward a summit ----------
ANG, ROT, SQ = math.radians(-45), -35, 1.1
FULL = dict(sw=3.2, rings=[(25, 0), (15.5, 4.0), (7.2, 3.0)], dot_r=3.0, dot_step=1.4)
SMALL = dict(sw=5.2, rings=[(24.5, 0), (11.5, 6.5)], dot_r=5.0, dot_step=2.6)

def f(n):
    return f'{n:.2f}'.rstrip('0').rstrip('.')

def mark(ring, dot, sw, rings, dot_r, dot_step, cx=31, cy=33, indent='  '):
    ux, uy = math.cos(ANG), math.sin(ANG)
    x, y, out = cx, cy, []
    for r, step in rings:
        x += ux * step; y += uy * step
        out.append(f'{indent}<ellipse cx="{f(x)}" cy="{f(y)}" rx="{f(r*SQ)}" ry="{f(r)}" transform="rotate({ROT} {f(x)} {f(y)})"/>')
    x += ux * dot_step; y += uy * dot_step
    rings_g = f'{indent[:-2]}<g fill="none" stroke="{ring}" stroke-width="{sw}">\n' + '\n'.join(out) + f'\n{indent[:-2]}</g>'
    return rings_g + f'\n{indent[:-2]}<circle cx="{f(x)}" cy="{f(y)}" r="{f(dot_r)}" fill="{dot}"/>'

def svg(w, h, body, title):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {f(w)} {f(h)}" width="{f(w)}" height="{f(h)}" role="img" aria-label="{title}">\n'
            f'  <title>{title}</title>\n{body}\n</svg>\n')

def framed(inner, scale):
    return f'<g transform="translate(32 32) scale({scale}) translate(-32 -32)">\n{inner}\n  </g>'

# --- outlined text ------------------------------------------------------------
def text_path(font_path, text, size, x, y, tracking=0.0):
    """Shapes text with HarfBuzz (kerning included) and returns (path d, width)."""
    tt = TTFont(font_path)
    face = hb.Face(hb.Blob(_sfnt(tt)))
    font = hb.Font(face)
    buf = hb.Buffer(); buf.add_str(text); buf.guess_segment_properties()
    hb.shape(font, buf, {'kern': True, 'liga': True})
    upem = tt['head'].unitsPerEm
    s = size / upem
    glyphs = tt.getGlyphSet()
    order = tt.getGlyphOrder()
    pen = SVGPathPen(glyphs, ntos=lambda v: f'{v:.1f}'.rstrip('0').rstrip('.'))
    cursor = 0
    for info, pos in zip(buf.glyph_infos, buf.glyph_positions):
        name = order[info.codepoint]
        tp = TransformPen(pen, (s, 0, 0, -s, x + (cursor + pos.x_offset) * s, y - pos.y_offset * s))
        glyphs[name].draw(tp)
        cursor += pos.x_advance + tracking * upem
    return pen.getCommands(), (cursor - tracking * upem) * s

def _sfnt(tt):
    import io
    tt.flavor = None
    b = io.BytesIO(); tt.save(b); return b.getvalue()

def write(name, content):
    open(f'{OUT}/{name}', 'w').write(content)

# Marks
write('mark.svg', svg(64, 64, mark(INK, ULTRA, **FULL, indent='    '), 'Repo Scout'))
write('mark-dark.svg', svg(64, 64, mark(INK_D, ULTRA_D, **FULL, indent='    '), 'Repo Scout'))
write('mark-small.svg', svg(64, 64, mark(INK, ULTRA, **SMALL, indent='    '), 'Repo Scout'))

# App icon (rounded ink tile) and its small-size cut
TILE = '  <rect width="64" height="64" rx="14" fill="%s"/>\n  ' % INK
write('app-icon.svg', svg(64, 64, TILE + framed(mark(PAPER, ULTRA_D, **{**FULL, 'sw': 3.6}, indent='      '), 0.74), 'Repo Scout'))
write('app-icon-small.svg', svg(64, 64, TILE + framed(mark(PAPER, ULTRA_D, **SMALL, indent='      '), 0.8), 'Repo Scout'))
# Full-bleed square for platforms that apply their own mask (iOS, Android maskable)
SQUARE = '  <rect width="64" height="64" fill="%s"/>\n  ' % INK
write('app-icon-square.svg', svg(64, 64, SQUARE + framed(mark(PAPER, ULTRA_D, **{**FULL, 'sw': 3.6}, indent='      '), 0.62), 'Repo Scout'))

# Horizontal lockup: mark + wordmark, caps centered on the mark
CAP, GAP = 25, 14
LOCKUP = {**FULL, 'sw': 3.7}
size = CAP / 0.7
d, width = text_path(SEMI, 'Repo Scout', size, 64 + GAP, 32 + CAP / 2, tracking=-0.005)
W = 64 + GAP + width
for suffix, ring, dot, ink in [('', INK, ULTRA, INK), ('-dark', INK_D, ULTRA_D, INK_D)]:
    body = mark(ring, dot, **LOCKUP, indent='    ') + f'\n  <path fill="{ink}" d="{d}"/>'
    write(f'logo{suffix}.svg', svg(math.ceil(W), 64, body, 'Repo Scout'))

# Social preview (GitHub recommends 1280x640)
SW, SH = 1280, 640
k = 2.4
lw, lh = W * k, 64 * k
block = lh + 40 + 34  # lockup, gap, tagline cap
lx, ly = (SW - lw) / 2, (SH - block) / 2 - 10
tag = 'Local-first analytics for any Git repository'
td, tw = text_path(REG, tag, 34, 0, 0)
tx = (SW - tw) / 2
td, _ = text_path(REG, tag, 34, tx, ly + lh + 64)
for suffix, bg, ring, dot, ink, muted in [('', PAPER, INK, ULTRA, INK, MUTED), ('-dark', '#111714', INK_D, ULTRA_D, INK_D, '#95A39A')]:
    body = (f'  <rect width="{SW}" height="{SH}" fill="{bg}"/>\n'
            f'  <g transform="translate({f(lx)} {f(ly)}) scale({k})">\n'
            + mark(ring, dot, **LOCKUP, indent='      ') + f'\n    <path fill="{ink}" d="{d}"/>\n  </g>\n'
            f'  <path fill="{muted}" d="{td}"/>')
    write(f'social-preview{suffix}.svg', svg(SW, SH, body, 'Repo Scout: local-first analytics for any Git repository'))
