// Package cardstyle - shared primitives for the 9 rebuilt card-style
// design systems (2026-09-16 v2 redesign). Owner rule: each style is a
// genuine design system - own structure, motifs, typography and textures -
// NOT a recolor of one shared skeleton. This package holds ONLY style-agnostic
// drawing vocabulary + per-style palettes + font registry; per-card
// compositions live in pkg/combat (portrait kinds) and pkg/economy
// (craft/decree kinds). Style 7 (Royal Decree) never routes here: it keeps
// the canonical baked art.
package cardstyle

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
)

// ResolveFont maps a kit font token to an absolute path on disk.
// v3: probes the known font homes so a missing file can never silently
// fall back to the previously-loaded face (which cascaded into giant
// wrong-font text across every style).
func ResolveFont(token string) string {
	if strings.Contains(token, "/") {
		p := utils.GetAssetPath("rpgasset", "ui", token)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	for _, h := range []string{"craft/" + token, "craft/fonts/" + token, token} {
		p := utils.GetAssetPath("rpgasset", "ui", h)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return utils.GetAssetPath("rpgasset", "ui", "craft/"+token)
}

// Font tokens (resolved through ResolveFont).
const (
	FtCinzel    = "Cinzel.ttf"
	FtCinzelDec = "CinzelDecBold.ttf"
	FtMedieval  = "MedievalSharp.ttf"
	FtIMFell    = "fonts/IMFellEnglish-Regular.ttf"
	FtIMFellIt  = "fonts/IMFellEnglish-Italic.ttf"
	FtPS2P      = "fonts/PressStart2P-Regular.ttf"
	FtInter     = "Inter-Medium.ttf"
	FtInterSemi = "Inter-SemiBold.ttf"
	FtKenney    = "KenneyFuture.ttf"
)

// NRGBA shortcut.
func N(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

func Hex(x uint32) color.NRGBA {
	return N(uint8(x>>16&0xFF), uint8(x>>8&0xFF), uint8(x&0xFF), 255)
}

func HexA(x uint32, a uint8) color.NRGBA {
	return N(uint8(x>>16&0xFF), uint8(x>>8&0xFF), uint8(x&0xFF), a)
}

func Clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func Lighten(c color.NRGBA, amt uint8) color.NRGBA {
	return N(Clamp8(float64(c.R)+float64(amt)), Clamp8(float64(c.G)+float64(amt)), Clamp8(float64(c.B)+float64(amt)), c.A)
}

func Darken(c color.NRGBA, amt uint8) color.NRGBA {
	return N(Clamp8(float64(c.R)-float64(amt)), Clamp8(float64(c.G)-float64(amt)), Clamp8(float64(c.B)-float64(amt)), c.A)
}

func WithA(c color.NRGBA, a uint8) color.NRGBA { return N(c.R, c.G, c.B, a) }

// Palette - the color roles a design system needs. Styles interpret roles
// freely but every field is set so shared helpers never nil-out.
type Palette struct {
	ID      int
	Name    string
	Bg, Bg2 color.NRGBA
	Vign    uint8
	Panel   color.NRGBA
	PanelEd color.NRGBA
	Ink     color.NRGBA
	Muted   color.NRGBA
	Accent  color.NRGBA
	Accent2 color.NRGBA
	Track   color.NRGBA
	Fill    color.NRGBA
	Done    color.NRGBA
	Seal    color.NRGBA
	SealTx  color.NRGBA
	Caption color.NRGBA
}

// ── deterministic RNG (stable renders across runs) ────────────────────
type RNG struct{ s uint32 }

func NewRNG(seed uint32) *RNG { return &RNG{s: seed} }

func (r *RNG) Next() float64 {
	r.s = r.s*1664525 + 1013904223
	return float64(r.s>>8&0xFFFFFF) / float64(0xFFFFFF)
}

func (r *RNG) Range(a, b float64) float64 { return a + r.Next()*(b-a) }

// ── text ──────────────────────────────────────────────────────────────
// Fit reduces the font size until s fits maxW (returns the size used).
func Fit(dc *gg.Context, font string, size int, s string, maxW float64, min int) int {
	for size > min {
		dc.LoadFontFace(ResolveFont(font), float64(size))
		w, _ := dc.MeasureString(s)
		if w <= maxW {
			return size
		}
		size--
	}
	return min
}

// Text loads (with fit), colors and draws an anchored string.
func Text(dc *gg.Context, font string, size int, s string, x, y float64, col color.NRGBA, ax, ay float64, maxW float64, minSize int) {
	if s == "" {
		return
	}
	size = Fit(dc, font, size, s, maxW, minSize)
	dc.SetColor(col)
	dc.DrawStringAnchored(s, x, y, ax, ay)
}

// Spaced draws letter-spaced text centered on ax fraction of its own width.
func Spaced(dc *gg.Context, font string, size int, s string, x, y float64, col color.NRGBA, ax float64, tracking float64) {
	if s == "" {
		return
	}
	dc.LoadFontFace(ResolveFont(font), float64(size))
	total := 0.0
	for _, r := range s {
		w, _ := dc.MeasureString(string(r))
		total += w + tracking
	}
	total -= tracking
	cx := x - total*ax
	dc.SetColor(col)
	for _, r := range s {
		w, _ := dc.MeasureString(string(r))
		dc.DrawStringAnchored(string(r), cx, y, 0, 0.5)
		cx += w + tracking
	}
}

// SpacedWidth measures the letter-spaced width of s.
func SpacedWidth(dc *gg.Context, font string, size int, s string, tracking float64) float64 {
	dc.LoadFontFace(ResolveFont(font), float64(size))
	total := 0.0
	for _, r := range s {
		w, _ := dc.MeasureString(string(r))
		total += w + tracking
	}
	return total - tracking
}

// GlowText paints a soft radial halo behind crisp text.
func GlowText(dc *gg.Context, font string, size int, s string, x, y float64, glow, col color.NRGBA, ax, ay float64, maxW float64, minSize int) {
	if s == "" {
		return
	}
	size = Fit(dc, font, size, s, maxW, minSize)
	dc.LoadFontFace(ResolveFont(font), float64(size))
	tw, th := dc.MeasureString(s)
	cx := x + tw*(ax-0.5)
	cy := y + th*(ay-0.5)
	r := th * 1.05
	g := gg.NewRadialGradient(cx, cy, 0, cx, cy, r)
	g.AddColorStop(0, WithA(glow, 80))
	g.AddColorStop(0.55, WithA(glow, 34))
	g.AddColorStop(1, WithA(glow, 0))
	dc.SetFillStyle(g)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
	dc.SetColor(col)
	dc.DrawStringAnchored(s, x, y, ax, ay)
}

// Star4 - four-point star (vector; fonts lack star glyphs).
func Star4(dc *gg.Context, x, y, r float64, col color.NRGBA) {
	dc.MoveTo(x, y-r)
	dc.QuadraticTo(x+r*0.10, y-r*0.10, x+r, y)
	dc.QuadraticTo(x+r*0.10, y+r*0.10, x, y+r)
	dc.QuadraticTo(x-r*0.10, y+r*0.10, x-r, y)
	dc.QuadraticTo(x-r*0.10, y-r*0.10, x, y-r)
	dc.ClosePath()
	dc.SetColor(col)
	dc.Fill()
}

// Star8 - eight-point star (two rotated Star4 passes).
func Star8(dc *gg.Context, x, y, r float64, col color.NRGBA) {
	Star4(dc, x, y, r, col)
	dc.Push()
	dc.RotateAbout(math.Pi/4, x, y)
	Star4(dc, x, y, r*0.72, col)
	dc.Pop()
}

// Engrave paints letterpress text: dark inset below, highlight above.
func Engrave(dc *gg.Context, font string, size int, s string, x, y float64, ink, hi color.NRGBA, ax, ay float64, maxW float64, minSize int) {
	size = Fit(dc, font, size, s, maxW, minSize)
	dc.SetColor(WithA(Darken(ink, 90), 160))
	dc.DrawStringAnchored(s, x, y+1.6, ax, ay)
	dc.SetColor(WithA(hi, 70))
	dc.DrawStringAnchored(s, x, y-1.2, ax, ay)
	dc.SetColor(ink)
	dc.DrawStringAnchored(s, x, y, ax, ay)
}

// ── structure primitives ──────────────────────────────────────────────
func Diamond(dc *gg.Context, x, y, r float64, col color.NRGBA) {
	dc.MoveTo(x, y-r)
	dc.LineTo(x+r, y)
	dc.LineTo(x, y+r)
	dc.LineTo(x-r, y)
	dc.ClosePath()
	dc.SetColor(col)
	dc.Fill()
}

func DiamondOutline(dc *gg.Context, x, y, r float64, col color.NRGBA, lw float64) {
	dc.MoveTo(x, y-r)
	dc.LineTo(x+r, y)
	dc.LineTo(x, y+r)
	dc.LineTo(x-r, y)
	dc.ClosePath()
	dc.SetColor(col)
	dc.SetLineWidth(lw)
	dc.Stroke()
}

// FrameHairline - thin frame inset; cornerDiamonds adds diamond studs.
func FrameHairline(dc *gg.Context, x, y, w, h, inset, lw float64, col color.NRGBA, cornerDiamonds bool) {
	dc.SetColor(col)
	dc.SetLineWidth(lw)
	dc.DrawRectangle(x+inset, y+inset, w-inset*2, h-inset*2)
	dc.Stroke()
	if cornerDiamonds {
		for _, c := range [][2]float64{{x + inset, y + inset}, {x + w - inset, y + inset}, {x + inset, y + h - inset}, {x + w - inset, y + h - inset}} {
			Diamond(dc, c[0], c[1], 4.5, col)
		}
	}
}

// PanelRounded - filled rounded panel + hairline edge.
func PanelRounded(dc *gg.Context, x, y, w, h, r float64, fill, edge color.NRGBA, lw float64) {
	dc.SetColor(fill)
	dc.DrawRoundedRectangle(x, y, w, h, r)
	dc.Fill()
	if edge.A > 0 {
		dc.SetColor(edge)
		dc.SetLineWidth(lw)
		dc.DrawRoundedRectangle(x, y, w, h, r)
		dc.Stroke()
	}
}

// DotLeader - dotted leader line between two x positions.
func DotLeader(dc *gg.Context, x0, x1, y, r float64, col color.NRGBA) {
	if x1 <= x0 {
		return
	}
	dc.SetColor(col)
	for x := x0; x <= x1; x += r * 3.2 {
		dc.DrawCircle(x, y, r)
		dc.Fill()
	}
}

// Chip - small rounded label chip.
func Chip(dc *gg.Context, x, y, w, h float64, label string, fill, edge, txt color.NRGBA, font string, size int) {
	PanelRounded(dc, x, y, w, h, h*0.28, fill, edge, 1.2)
	size = Fit(dc, font, size, label, w-10, 7)
	dc.SetColor(txt)
	dc.DrawStringAnchored(label, x+w/2, y+h/2-0.5, 0.5, 0.5)
}

// PillCTA - prominent rounded call-to-action pill with double outline.
func PillCTA(dc *gg.Context, cx, cy, w, h float64, label, sub string, fill, edge, txt, subTx color.NRGBA, font string, size int) {
	dc.SetColor(fill)
	dc.DrawRoundedRectangle(cx-w/2, cy-h/2, w, h, h*0.42)
	dc.Fill()
	dc.SetColor(edge)
	dc.SetLineWidth(1.6)
	dc.DrawRoundedRectangle(cx-w/2, cy-h/2, w, h, h*0.42)
	dc.Stroke()
	dc.SetLineWidth(0.8)
	dc.DrawRoundedRectangle(cx-w/2+4, cy-h/2+4, w-8, h-8, h*0.36)
	dc.Stroke()
	if sub != "" {
		size = Fit(dc, font, size, label, w-30, 10)
		dc.SetColor(txt)
		dc.DrawStringAnchored(label, cx, cy-9, 0.5, 0.5)
		ss := Fit(dc, font, 11, sub, w-30, 7)
		dc.LoadFontFace(ResolveFont(font), float64(ss))
		dc.SetColor(subTx)
		dc.DrawStringAnchored(sub, cx, cy+11, 0.5, 0.5)
	} else {
		size = Fit(dc, font, size, label, w-30, 10)
		dc.SetColor(txt)
		dc.DrawStringAnchored(label, cx, cy, 0.5, 0.5)
	}
}

// SegBar - segmented LED bar (Neon Arcade).
func SegBar(dc *gg.Context, x, y, w, h float64, frac float64, segs int, on, off color.NRGBA) {
	gap := 3.0
	sw := (w - gap*float64(segs-1)) / float64(segs)
	lit := int(math.Round(frac * float64(segs)))
	for i := 0; i < segs; i++ {
		c := off
		if i < lit {
			c = on
		}
		dc.SetColor(c)
		dc.DrawRoundedRectangle(x+float64(i)*(sw+gap), y, sw, h, 2)
		dc.Fill()
	}
}

// BevelRect - light top/left + dark bottom/right edges (stone/metal).
func BevelRect(dc *gg.Context, x, y, w, h float64, base, light, dark color.NRGBA, lw float64) {
	dc.SetColor(base)
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()
	dc.SetColor(light)
	dc.SetLineWidth(lw)
	dc.DrawLine(x, y+h, x, y)
	dc.DrawLine(x, y, x+w, y)
	dc.Stroke()
	dc.SetColor(dark)
	dc.DrawLine(x+w, y, x+w, y+h)
	dc.DrawLine(x, y+h, x+w, y+h)
	dc.Stroke()
}

// Rivet - small metal dome.
func Rivet(dc *gg.Context, x, y, r float64, base color.NRGBA) {
	dc.SetColor(Darken(base, 60))
	dc.DrawCircle(x, y, r)
	dc.Fill()
	dc.SetColor(Lighten(base, 70))
	dc.DrawCircle(x-r*0.3, y-r*0.3, r*0.45)
	dc.Fill()
}

// StarField - deterministic stars in a rect.
func StarField(dc *gg.Context, x0, y0, x1, y1 float64, n int, seed uint32, col color.NRGBA) {
	rng := NewRNG(seed)
	for i := 0; i < n; i++ {
		dc.SetColor(WithA(col, uint8(40+rng.Next()*160)))
		dc.DrawCircle(rng.Range(x0, x1), rng.Range(y0, y1), 0.6+rng.Next()*1.2)
		dc.Fill()
	}
}

// MagicCircle - concentric ritual rings with ticks + diamonds.
func MagicCircle(dc *gg.Context, cx, cy, r float64, col color.NRGBA) {
	dc.SetColor(WithA(col, 60))
	dc.SetLineWidth(1.2)
	dc.DrawCircle(cx, cy, r)
	dc.Stroke()
	dc.DrawCircle(cx, cy, r*0.82)
	dc.Stroke()
	dc.SetColor(WithA(col, 40))
	dc.SetLineWidth(0.8)
	dc.DrawCircle(cx, cy, r*0.64)
	dc.Stroke()
	for i := 0; i < 36; i++ {
		a := float64(i) * math.Pi / 18
		l := 5.0
		if i%9 == 0 {
			l = 10
			Diamond(dc, cx+(r+12)*math.Cos(a), cy+(r+12)*math.Sin(a), 4, WithA(col, 110))
		}
		dc.SetColor(WithA(col, 70))
		dc.SetLineWidth(1)
		dc.DrawLine(cx+(r-l)*math.Cos(a), cy+(r-l)*math.Sin(a), cx+r*math.Cos(a), cy+r*math.Sin(a))
		dc.Stroke()
	}
}

// Ring - single circle stroke.
func Ring(dc *gg.Context, cx, cy, r float64, col color.NRGBA, lw float64) {
	dc.SetColor(col)
	dc.SetLineWidth(lw)
	dc.DrawCircle(cx, cy, r)
	dc.Stroke()
}

// Vignette - radial darkening toward edges.
func Vignette(dc *gg.Context, w, h float64, strength uint8) {
	if strength == 0 {
		return
	}
	g := gg.NewRadialGradient(w/2, h/2, math.Min(w, h)*0.25, w/2, h/2, math.Max(w, h)*0.78)
	g.AddColorStop(0, N(0, 0, 0, 0))
	g.AddColorStop(1, N(0, 0, 0, strength))
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, w, h)
	dc.Fill()
}

// VertGrad fills the whole canvas with a vertical two-stop gradient.
func VertGrad(dc *gg.Context, w, h float64, top, bot color.NRGBA) {
	g := gg.NewLinearGradient(0, 0, 0, h)
	g.AddColorStop(0, top)
	g.AddColorStop(1, bot)
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, w, h)
	dc.Fill()
}

// Scanlines - CRT lines (Neon Arcade).
func Scanlines(dc *gg.Context, w, h float64, col color.NRGBA, step float64) {
	dc.SetColor(col)
	dc.SetLineWidth(1)
	for y := 0.0; y < h; y += step {
		dc.DrawLine(0, y, w, y)
		dc.Stroke()
	}
}

// BracketCorners - HUD bracket corners (Neon Arcade).
func BracketCorners(dc *gg.Context, x, y, w, h, l float64, col color.NRGBA, lw float64) {
	dc.SetColor(col)
	dc.SetLineWidth(lw)
	for _, c := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		sx, sy := c[0], c[1]
		dx, dy := 1.0, 1.0
		if sx > x+w/2 {
			dx = -1
		}
		if sy > y+h/2 {
			dy = -1
		}
		dc.DrawLine(sx, sy, sx+dx*l, sy)
		dc.Stroke()
		dc.DrawLine(sx, sy, sx, sy+dy*l)
		dc.Stroke()
	}
}

// Damask - ornate diamond lattice (Crimson Court).
func Damask(dc *gg.Context, w, h float64, step float64, col color.NRGBA) {
	for y := 0.0; y < h+step; y += step {
		for x := 0.0; x < w+step; x += step {
			off := 0.0
			if int(y/step)%2 == 1 {
				off = step / 2
			}
			DiamondOutline(dc, x+off, y, step*0.32, col, 1)
			Diamond(dc, x+off, y, step*0.1, col)
		}
	}
}

// Halftone - print dots (Retro Court).
func Halftone(dc *gg.Context, w, h float64, step float64, col color.NRGBA) {
	dc.SetColor(col)
	for y := step / 2; y < h; y += step {
		for x := step / 2; x < w; x += step {
			dc.DrawCircle(x, y, 1.1)
			dc.Fill()
		}
	}
}

// Speckle - stone noise (Stonekeep/Monolith).
func Speckle(dc *gg.Context, x0, y0, x1, y1 float64, n int, seed uint32, col color.NRGBA) {
	rng := NewRNG(seed)
	for i := 0; i < n; i++ {
		dc.SetColor(WithA(col, uint8(14+rng.Next()*30)))
		dc.DrawCircle(rng.Range(x0, x1), rng.Range(y0, y1), 0.7+rng.Next()*1.6)
		dc.Fill()
	}
}

// Planks - horizontal wood planks + grain (Woodmere).
func Planks(dc *gg.Context, w, h float64, seed uint32, dark, grain color.NRGBA) {
	rng := NewRNG(seed)
	dc.SetColor(dark)
	dc.SetLineWidth(2.4)
	for y := 124.0; y < h; y += 124 {
		dc.DrawLine(0, y, w, y)
		dc.Stroke()
	}
	dc.SetColor(grain)
	dc.SetLineWidth(1)
	for i := 0; i < 60; i++ {
		y := rng.Range(20, h)
		dc.DrawLine(rng.Range(0, w*0.4), y, rng.Range(w*0.6, w), y+rng.Range(-4, 4))
		dc.Stroke()
	}
}

// GlyphCol - carved rune column (Rune Monolith).
func GlyphCol(dc *gg.Context, x, yTop, yBot, step float64, col color.NRGBA) {
	dc.SetColor(col)
	dc.SetLineWidth(2)
	for y := yTop; y < yBot; y += step {
		dc.DrawLine(x, y, x+9, y+13)
		dc.Stroke()
		dc.DrawLine(x+9, y+13, x, y+26)
		dc.Stroke()
		if int(y/step)%2 == 0 {
			dc.DrawLine(x+9, y, x, y+13)
			dc.Stroke()
		}
	}
}

// Fleuron - small print ornament (Retro Court).
func Fleuron(dc *gg.Context, x, y, r float64, col color.NRGBA) {
	Diamond(dc, x, y, r*0.55, col)
	dc.SetColor(col)
	dc.SetLineWidth(1.2)
	dc.DrawLine(x-r*1.6, y, x-r*0.7, y)
	dc.Stroke()
	dc.DrawLine(x+r*0.7, y, x+r*1.6, y)
	dc.Stroke()
	dc.DrawCircle(x-r*1.85, y, r*0.18)
	dc.Fill()
	dc.DrawCircle(x+r*1.85, y, r*0.18)
	dc.Fill()
}

// Stitches - binding stitches down an edge (Woodmere).
func Stitches(dc *gg.Context, x, y0, y1, step float64, col color.NRGBA) {
	dc.SetColor(col)
	dc.SetLineWidth(2)
	for y := y0; y < y1; y += step {
		dc.DrawLine(x-4, y, x+4, y+8)
		dc.Stroke()
	}
}

// WaxSeal - wax circle w/ double ring + text.
func WaxSeal(dc *gg.Context, cx, cy, r float64, wax, rim, tx color.NRGBA, text, font string, size int) {
	dc.SetColor(WithA(wax, 245))
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
	dc.SetColor(rim)
	dc.SetLineWidth(3)
	dc.DrawCircle(cx, cy, r)
	dc.Stroke()
	dc.SetColor(WithA(Lighten(wax, 40), 200))
	dc.SetLineWidth(1.4)
	dc.DrawCircle(cx, cy, r-6)
	dc.Stroke()
	if text != "" {
		size = Fit(dc, font, size, text, r*1.3, 8)
		dc.SetColor(tx)
		dc.DrawStringAnchored(text, cx, cy-1, 0.5, 0.5)
	}
}

// Tag - hanging tag with hole + string.
func Tag(dc *gg.Context, x, y, w, h float64, base, edge, txt color.NRGBA, label, font string, size int) {
	dc.SetColor(WithA(Darken(edge, 40), 140))
	dc.SetLineWidth(1.2)
	dc.DrawLine(x+w/2, y-13, x+w/2, y)
	dc.Stroke()
	dc.SetColor(base)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(edge)
	dc.SetLineWidth(1.4)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Stroke()
	dc.SetColor(Darken(base, 50))
	dc.DrawCircle(x+w/2, y+9, 2.6)
	dc.Fill()
	if label != "" {
		size = Fit(dc, font, size, label, w-10, 7)
		dc.SetColor(txt)
		dc.DrawStringAnchored(label, x+w/2, y+h*0.62, 0.5, 0.5)
	}
}

// Pennant - heraldic pennant (Crimson Court).
func Pennant(dc *gg.Context, x, y, w, h float64, fill, edge, txt color.NRGBA, label, font string, size int) {
	dc.MoveTo(x, y)
	dc.LineTo(x+w, y)
	dc.LineTo(x+w, y+h*0.72)
	dc.LineTo(x+w/2, y+h)
	dc.LineTo(x, y+h*0.72)
	dc.ClosePath()
	dc.SetColor(fill)
	dc.Fill()
	dc.SetColor(edge)
	dc.SetLineWidth(1.6)
	dc.Stroke()
	if label != "" {
		size = Fit(dc, font, size, label, w-8, 7)
		dc.SetColor(txt)
		dc.DrawStringAnchored(label, x+w/2, y+h*0.4, 0.5, 0.5)
	}
}

// Shield - heraldic shield.
func Shield(dc *gg.Context, cx, cy, w, h float64, fill, edge color.NRGBA, lw float64) {
	x0, y0 := cx-w/2, cy-h/2
	dc.MoveTo(x0, y0)
	dc.LineTo(x0+w, y0)
	dc.LineTo(x0+w, y0+h*0.5)
	dc.CubicTo(x0+w, y0+h*0.85, cx+w*0.28, y0+h, cx, y0+h)
	dc.CubicTo(cx-w*0.28, y0+h, x0, y0+h*0.85, x0, y0+h*0.5)
	dc.ClosePath()
	dc.SetColor(fill)
	dc.Fill()
	if edge.A > 0 {
		dc.SetColor(edge)
		dc.SetLineWidth(lw)
		dc.Stroke()
	}
}

// Cartouche - ornate gold header plaque (Crimson Court).
func Cartouche(dc *gg.Context, cx, cy, w, h float64, fill, edge, ink color.NRGBA, title, font string, size int) {
	dc.SetColor(fill)
	dc.DrawRoundedRectangle(cx-w/2, cy-h/2, w, h, h*0.45)
	dc.Fill()
	dc.SetColor(edge)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(cx-w/2, cy-h/2, w, h, h*0.45)
	dc.Stroke()
	Diamond(dc, cx-w/2-14, cy, 5, edge)
	Diamond(dc, cx+w/2+14, cy, 5, edge)
	size = Fit(dc, font, size, title, w-40, 10)
	dc.SetColor(ink)
	dc.DrawStringAnchored(title, cx, cy-0.5, 0.5, 0.5)
}

// Stamp - rotated file stamp (Noir).
func Stamp(dc *gg.Context, x, y float64, text string, col color.NRGBA) {
	dc.Push()
	dc.Translate(x, y)
	dc.Rotate(-0.05)
	dc.SetColor(WithA(col, 200))
	dc.SetLineWidth(1.6)
	dc.DrawRectangle(-88, -16, 176, 32)
	dc.Stroke()
	dc.LoadFontFace(ResolveFont(FtInterSemi), 15)
	dc.DrawStringAnchored(text, 0, 0, 0.5, 0.5)
	dc.Pop()
}

// TorchGlow - warm radial light (Stonekeep).
func TorchGlow(dc *gg.Context, cx, cy, r float64, col color.NRGBA) {
	g := gg.NewRadialGradient(cx, cy, 0, cx, cy, r)
	g.AddColorStop(0, WithA(col, 120))
	g.AddColorStop(1, WithA(col, 0))
	dc.SetFillStyle(g)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
}

// EmberGlow - warm core light (Monolith).
func EmberGlow(dc *gg.Context, cx, cy, r float64, col color.NRGBA) {
	g := gg.NewRadialGradient(cx, cy, 0, cx, cy, r)
	g.AddColorStop(0, WithA(col, 90))
	g.AddColorStop(1, WithA(col, 0))
	dc.SetFillStyle(g)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
}

// MeterBar - generic track+fill bar with hairline edge.
func MeterBar(dc *gg.Context, x, y, w, h, frac float64, track, fill, hi, edge color.NRGBA, radius float64) {
	dc.SetColor(track)
	dc.DrawRoundedRectangle(x, y, w, h, radius)
	dc.Fill()
	fw := w * frac
	if fw > w {
		fw = w
	}
	if fw > radius {
		dc.SetColor(fill)
		dc.DrawRoundedRectangle(x, y, fw, h, radius)
		dc.Fill()
		if hi.A > 0 {
			dc.SetColor(hi)
			dc.DrawRoundedRectangle(x, y, fw, h*0.42, radius)
			dc.Fill()
		}
	}
	if edge.A > 0 {
		dc.SetColor(edge)
		dc.SetLineWidth(1.6)
		dc.DrawRoundedRectangle(x, y, w, h, radius)
		dc.Stroke()
	}
}

// NotchMeter - carved tally notches (Woodmere).
func NotchMeter(dc *gg.Context, x, y, w float64, n, filled int, carve, ember color.NRGBA) {
	if n < 1 {
		n = 1
	}
	step := w / float64(n)
	if step > 18 {
		step = 18
	}
	x0 := x + (w-step*float64(n))/2
	dc.SetLineWidth(3.4)
	for i := 0; i < n; i++ {
		c := carve
		if i < filled {
			c = ember
		}
		dc.SetColor(c)
		dc.DrawLine(x0+float64(i)*step+step/2, y-7, x0+float64(i)*step+step/2, y+7)
		dc.Stroke()
	}
}

// SealMini - small seal dot with text.
func SealMini(dc *gg.Context, cx, cy, r float64, fill, tx color.NRGBA, text, font string, size int) {
	dc.SetColor(fill)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
	dc.SetColor(WithA(Darken(fill, 60), 220))
	dc.SetLineWidth(2)
	dc.DrawCircle(cx, cy, r)
	dc.Stroke()
	if text != "" {
		size = Fit(dc, font, size, text, r*1.4, 7)
		dc.SetColor(tx)
		dc.DrawStringAnchored(text, cx, cy-0.5, 0.5, 0.5)
	}
}

// LedgerRow - thin rule row with label left / value right (Noir).
func LedgerRow(dc *gg.Context, x, y, w float64, label, value string, rule, ink, muted color.NRGBA, fontL, fontV string, size int) {
	dc.SetColor(rule)
	dc.SetLineWidth(0.8)
	dc.DrawLine(x, y+9, x+w, y+9)
	dc.Stroke()
	dc.SetColor(muted)
	dc.LoadFontFace(ResolveFont(fontL), float64(size))
	dc.DrawStringAnchored(label, x, y, 0, 0.5)
	if value != "" {
		dc.SetColor(ink)
		dc.DrawStringAnchored(value, x+w, y, 1, 0.5)
	}
}

// FooterNote - small centered footnote.
func FooterNote(dc *gg.Context, cx, y float64, text string, col color.NRGBA, font string, size int, maxW float64) {
	if text == "" {
		return
	}
	size = Fit(dc, font, size, text, maxW, 8)
	dc.SetColor(col)
	dc.DrawStringAnchored(text, cx, y, 0.5, 0.5)
}

// Sanitize strips control characters (presentation safety).
func Sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(' ')
			continue
		}
		if r < 32 {
			continue
		}
		// 2026-09-16: the theme faces (Cinzel/IM Fell/PressStart/Inter) carry
		// no emoji or misc-symbol glyphs - anything outside the covered
		// ranges renders as tofu boxes. Drop them so cards never show boxes.
		if (r >= 0x2190 && r <= 0x2BFF) || // arrows, math symbols, dingbats
			(r >= 0x1F000) || // emoji planes
			r == 0xFE0F || r == 0x200D || r == 0x20E3 { // VS16, ZWJ, keycap
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// TruncateRunes limits a string to n runes with ellipsis.
func TruncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "..."
}

// StatFullName - the owner's Allocate design displays full stat names.
func StatFullName(code string) string {
	switch code {
	case "HP":
		return "HEALTH"
	case "ATK":
		return "ATTACK"
	case "DEF":
		return "DEFENSE"
	case "MAG":
		return "MAGIC"
	case "SPD":
		return "SPEED"
	case "LUCK":
		return "LUCK"
	case "CRIT":
		return "CRIT"
	}
	return code
}

// FmtF trims a float for display.
func FmtF(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%.1f", v)
}

// PageBase - aged-paper base with halftone print (Retro Court family).
func PageBase(dc *gg.Context, w, h float64, top, bot color.NRGBA, halftoneAlpha uint8) {
	VertGrad(dc, w, h, top, bot)
	Halftone(dc, w, h, 14, WithA(N(120, 96, 70, 255), halftoneAlpha))
}
