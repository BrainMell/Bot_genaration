package chess

import (
	"image/color"
	"math"
	"strings"

	"image-service/pkg/utils"
)

// rgb is a float colour (0..255) used while shading so we only clamp once.
type rgb struct{ r, g, b float64 }

func hex(s string) rgb {
	c := utils.ParseHexColor(s)
	return rgb{float64(c.R), float64(c.G), float64(c.B)}
}
func (a rgb) mul(k float64) rgb { return rgb{a.r * k, a.g * k, a.b * k} }
func (a rgb) add(b rgb) rgb     { return rgb{a.r + b.r, a.g + b.g, a.b + b.b} }
func (a rgb) mix(b rgb, t float64) rgb {
	return rgb{lerp(a.r, b.r, t), lerp(a.g, b.g, t), lerp(a.b, b.b, t)}
}
func (a rgb) rgba() color.RGBA {
	cl := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.RGBA{cl(a.r), cl(a.g), cl(a.b), 255}
}

// shadeCtx is what a texture function gets for every board pixel.
type shadeCtx struct {
	x, y   float64 // pixel position inside the 8x8 board (0..size)
	lx, ly float64 // position inside the current square (0..cell)
	cell   float64
	file   int // 0..7 (a..h) in board space, already flipped-independent
	rank   int // 0..7 (8..1)
	light  bool
	seed   uint32
}

// Theme describes one selectable board look.
type Theme struct {
	Key, Label, Blurb string
	Light, Dark       rgb
	Shade             func(t *Theme, s shadeCtx) rgb
	Frame, FrameLine  rgb
	FrameStyle        string // "plain" | "gold" | "studs" | "ember" | "neon" | "bevel"
	CoordColor        rgb
	LastMove          color.NRGBA
	Dot               color.NRGBA
	Vignette          float64 // 0 = none
	UseFantasy        bool    // fantasy font for coordinates
}

func (t *Theme) square(s shadeCtx) rgb {
	if t.Shade != nil {
		return t.Shade(t, s)
	}
	if s.light {
		return t.Light
	}
	return t.Dark
}

func baseOf(t *Theme, s shadeCtx) rgb {
	if s.light {
		return t.Light
	}
	return t.Dark
}

// ── individual shaders ──────────────────────────────────────────────────────

func shadeClassic(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	g := 1.035 - 0.07*(s.ly/s.cell) // soft top-light gradient inside each square
	n := (vnoise(s.x*0.9, s.y*0.9, 7) - 0.5) * 0.035
	return c.mul(g + n)
}

func shadeWalnut(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	ox, oy := float64(s.file)*13.7, float64(s.rank)*9.3
	w := fbm(s.x*0.012+ox, s.y*0.0035+oy, 4, s.seed+3)
	grain := 0.5 + 0.5*math.Sin(s.y*0.22+w*14)
	fine := (vnoise(s.x*0.5, s.y*0.05, s.seed+9) - 0.5) * 0.05
	k := 0.90 + 0.17*grain + fine
	edge := math.Min(math.Min(s.lx, s.cell-s.lx), math.Min(s.ly, s.cell-s.ly))
	if edge < 1.4 {
		k *= 0.86
	}
	return c.mul(k)
}

func shadeMarble(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	ox, oy := float64(s.file)*5.1, float64(s.rank)*7.7
	w := fbm(s.x*0.006+ox, s.y*0.006+oy, 5, s.seed+5)
	vein := math.Abs(math.Sin((s.x*0.010 + s.y*0.014) + w*9))
	v := math.Pow(1-vein, 14)
	vc := rgb{60, 62, 72}
	if !s.light {
		vc = rgb{205, 207, 215}
	}
	soft := (fbm(s.x*0.01+ox, s.y*0.01, 3, s.seed+8) - 0.5) * 0.10
	return c.mul(1+soft).mix(vc, v*0.55)
}

func shadeMidnight(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	g := 1.12 - 0.22*(s.y/(s.cell*8))
	r := c.mul(g)
	if hash2(int(s.x), int(s.y), s.seed+11) > 0.9987 {
		r = r.mix(rgb{190, 230, 255}, 0.9)
	}
	edge := math.Min(math.Min(s.lx, s.cell-s.lx), math.Min(s.ly, s.cell-s.ly))
	if edge < 1.2 {
		r = r.mix(rgb{80, 120, 200}, 0.25)
	}
	return r
}

func shadeStone(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	ox, oy := float64(s.file)*17.3, float64(s.rank)*11.9
	tint := 0.92 + 0.16*hash2(s.file, s.rank, s.seed+1) // every slab is a slightly different stone
	n := fbm(s.x*0.045+ox, s.y*0.045+oy, 5, s.seed+2)
	k := tint * (0.82 + 0.36*n)
	// hairline cracks (ridged noise)
	cr := math.Abs(fbm(s.x*0.02+ox, s.y*0.02+oy, 4, s.seed+4) - 0.5)
	if cr < 0.012 {
		k *= 0.55 + cr*30
	}
	// bevel: light from the top-left, mortar in the joints
	e := math.Min(math.Min(s.lx, s.cell-s.lx), math.Min(s.ly, s.cell-s.ly))
	switch {
	case e < 2.2:
		k *= 0.42
	case s.lx < 6 || s.ly < 6:
		k *= 1.16
	case s.cell-s.lx < 6 || s.cell-s.ly < 6:
		k *= 0.80
	}
	return c.mul(k)
}

func shadeInferno(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	ox, oy := float64(s.file)*9.1, float64(s.rank)*6.3
	n := fbm(s.x*0.04+ox, s.y*0.04+oy, 5, s.seed+21)
	r := c.mul(0.78 + 0.5*n)
	crack := math.Abs(fbm(s.x*0.017+ox, s.y*0.017+oy, 4, s.seed+23) - 0.5)
	thr := 0.020
	if s.light {
		thr = 0.009
	}
	if crack < thr {
		glow := 1 - crack/thr
		r = r.mix(rgb{255, 130, 30}, glow*0.9).add(rgb{glow * 40, glow * 12, 0})
	} else if crack < thr*3 {
		r = r.mix(rgb{200, 70, 20}, (1-(crack-thr)/(thr*2))*0.18)
	}
	e := math.Min(math.Min(s.lx, s.cell-s.lx), math.Min(s.ly, s.cell-s.ly))
	if e < 1.8 {
		r = r.mul(0.55)
	}
	return r
}

func shadeSakura(t *Theme, s shadeCtx) rgb {
	c := baseOf(t, s)
	n := (fbm(s.x*0.03, s.y*0.03, 3, s.seed+31) - 0.5) * 0.10
	g := 1.02 - 0.05*(s.ly/s.cell)
	return c.mul(g + n)
}

var themeOrder = []string{"classic", "walnut", "marble", "midnight", "dungeon", "inferno", "sakura"}

var themes = map[string]*Theme{
	"classic": {
		Key: "classic", Label: "Classic Green", Blurb: "the original, polished",
		Light: hex("#EBECD0"), Dark: hex("#739552"), Shade: shadeClassic,
		Frame: hex("#2C3A27"), FrameLine: hex("#8DAA6B"), FrameStyle: "bevel", CoordColor: hex("#DDE6C6"),
		LastMove: color.NRGBA{255, 235, 70, 120}, Dot: color.NRGBA{20, 40, 10, 95},
	},
	"walnut": {
		Key: "walnut", Label: "Walnut Wood", Blurb: "warm tournament wood",
		Light: hex("#E9CFA4"), Dark: hex("#A9754A"), Shade: shadeWalnut,
		Frame: hex("#3A2415"), FrameLine: hex("#C79A62"), FrameStyle: "bevel", CoordColor: hex("#F0D9B5"),
		LastMove: color.NRGBA{255, 214, 60, 125}, Dot: color.NRGBA{50, 25, 5, 100},
	},
	"marble": {
		Key: "marble", Label: "Royal Marble", Blurb: "white & slate veined stone",
		Light: hex("#F2F1EC"), Dark: hex("#6B6E7A"), Shade: shadeMarble,
		Frame: hex("#17181D"), FrameLine: hex("#D4AF37"), FrameStyle: "gold", CoordColor: hex("#E7D7A0"),
		LastMove: color.NRGBA{232, 185, 55, 130}, Dot: color.NRGBA{20, 20, 30, 95},
	},
	"midnight": {
		Key: "midnight", Label: "Midnight Neon", Blurb: "dark indigo with neon edges",
		Light: hex("#3E4A78"), Dark: hex("#1A2040"), Shade: shadeMidnight,
		Frame: hex("#0A0D1C"), FrameLine: hex("#35E0FF"), FrameStyle: "neon", CoordColor: hex("#8FE9FF"),
		LastMove: color.NRGBA{0, 230, 255, 105}, Dot: color.NRGBA{120, 240, 255, 120},
	},
	"dungeon": {
		Key: "dungeon", Label: "Dungeon Keep", Blurb: "RPG torchlit flagstone",
		Light: hex("#B3A687"), Dark: hex("#4B4659"), Shade: shadeStone,
		Frame: hex("#2B1D12"), FrameLine: hex("#D9A441"), FrameStyle: "studs", CoordColor: hex("#F1CF7A"),
		LastMove: color.NRGBA{255, 176, 40, 135}, Dot: color.NRGBA{255, 220, 120, 120},
		Vignette: 0.42, UseFantasy: true,
	},
	"inferno": {
		Key: "inferno", Label: "Inferno Forge", Blurb: "RPG obsidian & lava cracks",
		Light: hex("#58403B"), Dark: hex("#221412"), Shade: shadeInferno,
		Frame: hex("#150C0A"), FrameLine: hex("#FF7A1F"), FrameStyle: "ember", CoordColor: hex("#FFB066"),
		LastMove: color.NRGBA{255, 110, 20, 130}, Dot: color.NRGBA{255, 150, 60, 130},
		Vignette: 0.5, UseFantasy: true,
	},
	"sakura": {
		Key: "sakura", Label: "Sakura Garden", Blurb: "soft pink blossom",
		Light: hex("#FDF0F3"), Dark: hex("#E4A0B8"), Shade: shadeSakura,
		Frame: hex("#5A2A44"), FrameLine: hex("#F5B8CC"), FrameStyle: "bevel", CoordColor: hex("#FBD3E0"),
		LastMove: color.NRGBA{255, 120, 160, 120}, Dot: color.NRGBA{90, 30, 60, 95},
	},
}

var themeAliases = map[string]string{
	"default": "classic", "green": "classic", "old": "classic", "wood": "walnut", "wooden": "walnut",
	"dark": "midnight", "neon": "midnight", "rpg": "dungeon", "stone": "dungeon", "keep": "dungeon",
	"lava": "inferno", "fire": "inferno", "forge": "inferno", "pink": "sakura", "garden": "sakura",
}

func GetTheme(name string) *Theme {
	k := strings.ToLower(strings.TrimSpace(name))
	if a, ok := themeAliases[k]; ok {
		k = a
	}
	if t, ok := themes[k]; ok {
		return t
	}
	return themes["classic"]
}

// ── piece sets ─────────────────────────────────────────────────────────────

type PieceSet struct{ Key, Label, Dir, Credit string }

var pieceOrder = []string{"classic", "chessnut", "fantasy", "celtic", "spatial", "rhosgfx"}

var pieceSets = map[string]PieceSet{
	"classic":  {"classic", "Classic", "", "bot original"},
	"chessnut": {"chessnut", "Chessnut", "chessnut", "Alexis Luengas, Apache-2.0"},
	"fantasy":  {"fantasy", "Fantasy (RPG)", "fantasy", "Maurizio Monge, MIT"},
	"celtic":   {"celtic", "Celtic", "celtic", "Maurizio Monge, MIT"},
	"spatial":  {"spatial", "Spatial", "spatial", "Maurizio Monge, MIT"},
	"rhosgfx":  {"rhosgfx", "Candy Wood", "rhosgfx", "RhosGFX, CC0"},
}

var pieceAliases = map[string]string{"default": "classic", "rpg": "fantasy", "wood": "rhosgfx", "candy": "rhosgfx", "nut": "chessnut"}

func GetPieceSet(name string) PieceSet {
	k := strings.ToLower(strings.TrimSpace(name))
	if a, ok := pieceAliases[k]; ok {
		k = a
	}
	if p, ok := pieceSets[k]; ok {
		return p
	}
	return pieceSets["classic"]
}
