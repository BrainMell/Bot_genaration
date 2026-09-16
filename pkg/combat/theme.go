package combat

// ============================================
// 🎨 CARDSTYLE THEME SYSTEM — 2026-09-16 (phase 8)
// ============================================
// Owner rule: "cardstyle should be a CARD SYSTEM THEME, not merely an
// isolated card recolor". A player's `.j cardstyle <n>` drives EVERY
// general RPG presentation card, not just the profile card.
//
// Ten visual identities, one shared geometry. Each theme owns:
//   palette (bg/panel/ink/accent/seal), frame construction, and a
//   signature background motif. Card KINDS keep their own composition
//   ("same universe, different purpose") — a RANK card and an ALLOCATE
//   card in style 1 look related but are not clones.
//
// Style 7 (Royal Decree) is the canonical baked-art baseline: it keeps
// the bg_*.png art path so the default look is untouched. Styles 1-6
// and 8-10 render programmatically through drawPortraitShell().
//
// NOTE: this is a presentation layer ONLY — it never changes payload
// semantics. New themes must set every color role (the decree palette
// documents the role meanings).

import (
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

type cardTheme struct {
	ID   int
	Name string
	// shell surfaces
	Bg, Bg2      color.NRGBA // page base gradient
	Panel        color.NRGBA // main information panel
	PanelEd      color.NRGBA // panel edge shading
	Plate        color.NRGBA // name plate band
	PlateTx      color.NRGBA // name text on the plate
	Banner       color.NRGBA // banner band fill
	BannerTx     color.NRGBA // banner text
	BannerEdge   color.NRGBA // banner fold/edge
	// ink roles
	Ink, Muted color.NRGBA
	Sub        color.NRGBA // secondary/positive text (xp-now, gains)
	Gold       color.NRGBA // rules, dividers, hairlines
	PillBg     color.NRGBA
	PillTx     color.NRGBA
	Track      color.NRGBA
	Fill       color.NRGBA
	FillHi     color.NRGBA
	Done       color.NRGBA
	// seal + caption
	Seal    color.NRGBA
	SealTx  color.NRGBA
	Caption color.NRGBA
	// construction
	FrameStyle int // index into drawThemeFrame
}

// decreeTheme = the exact ink palette the baked Royal Decree art uses.
// When a card has no theme (style 0/7) the renderer still needs these
// roles for the ink swaps — values are byte-identical to today's literals.
func decreeTheme() cardTheme {
	return cardTheme{
		ID: 7, Name: "Royal Decree",
		Bg: n(24, 16, 10, 255), Bg2: n(24, 16, 10, 255),
		Panel: n(233, 215, 171, 255), PanelEd: n(90, 70, 40, 26),
		Plate: n(62, 44, 28, 255), PlateTx: n(214, 170, 82, 255),
		Banner: n(62, 44, 28, 255), BannerTx: n(244, 214, 140, 255), BannerEdge: n(170, 130, 60, 255),
		Ink: n(52, 32, 16, 255), Muted: n(120, 88, 40, 255),
		Sub:  n(84, 96, 44, 255),
		Gold:   n(170, 130, 60, 255),
		PillBg: n(96, 62, 24, 235), PillTx: n(244, 214, 140, 255),
		Track: n(54, 36, 18, 255), Fill: n(214, 170, 82, 255), FillHi: n(240, 205, 120, 90),
		Done: n(110, 160, 80, 255),
		Seal: n(128, 28, 40, 255), SealTx: n(250, 210, 120, 255),
		Caption: n(120, 88, 40, 255),
	}
}

func n(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

// cardThemes — the ten identities. 7 is omitted (baked baseline).
var cardThemes = map[int]*cardTheme{
	1: { // STONEKEEP — dwarven fortress: granite, iron, chiseled edges
		ID: 1, Name: "Stonekeep", FrameStyle: 1,
		Bg: n(84, 86, 92, 255), Bg2: n(58, 60, 66, 255),
		Panel: n(172, 174, 178, 255), PanelEd: n(30, 32, 36, 70),
		Plate: n(40, 42, 48, 255), PlateTx: n(212, 218, 228, 255),
		Banner: n(46, 48, 54, 255), BannerTx: n(224, 228, 236, 255), BannerEdge: n(140, 146, 156, 255),
		Ink: n(28, 30, 34, 255), Muted: n(66, 70, 78, 255),
		Sub:  n(60, 130, 96, 255),
		Gold:   n(120, 126, 138, 255),
		PillBg: n(40, 42, 48, 240), PillTx: n(220, 226, 236, 255),
		Track:  n(52, 54, 60, 255), Fill: n(148, 156, 170, 255), FillHi: n(198, 206, 218, 90),
		Done:   n(110, 170, 120, 255),
		Seal:   n(40, 42, 48, 255), SealTx: n(220, 226, 236, 255),
		Caption: n(66, 70, 78, 255),
	},
	2: { // GOLDEN ARCANUM — arcane athanæum: indigo, ritual gold
		ID: 2, Name: "Golden Arcanum", FrameStyle: 2,
		Bg: n(22, 18, 44, 255), Bg2: n(36, 28, 66, 255),
		Panel: n(40, 33, 70, 255), PanelEd: n(0, 0, 0, 90),
		Plate: n(56, 44, 26, 255), PlateTx: n(226, 192, 108, 255),
		Banner: n(30, 24, 56, 255), BannerTx: n(232, 200, 120, 255), BannerEdge: n(212, 175, 55, 255),
		Ink: n(240, 230, 190, 255), Muted: n(186, 172, 132, 255),
		Sub:  n(140, 200, 160, 255),
		Gold:   n(212, 175, 55, 255),
		PillBg: n(84, 66, 30, 240), PillTx: n(240, 210, 130, 255),
		Track:  n(20, 16, 38, 255), Fill: n(212, 175, 55, 255), FillHi: n(244, 220, 140, 90),
		Done:   n(120, 200, 140, 255),
		Seal:   n(120, 90, 28, 255), SealTx: n(244, 220, 140, 255),
		Caption: n(186, 172, 132, 255),
	},
	3: { // RETRO COURT — victorian playbill: sepia halftone, burgundy rules
		ID: 3, Name: "Retro Court", FrameStyle: 3,
		Bg: n(206, 188, 152, 255), Bg2: n(222, 206, 170, 255),
		Panel: n(238, 224, 188, 255), PanelEd: n(120, 96, 70, 50),
		Plate: n(122, 44, 58, 255), PlateTx: n(244, 226, 190, 255),
		Banner: n(112, 40, 54, 255), BannerTx: n(246, 230, 196, 255), BannerEdge: n(84, 30, 42, 255),
		Ink: n(58, 44, 32, 255), Muted: n(112, 88, 64, 255),
		Sub:  n(70, 110, 80, 255),
		Gold:   n(122, 44, 58, 255),
		PillBg: n(122, 44, 58, 240), PillTx: n(244, 226, 190, 255),
		Track:  n(90, 72, 54, 255), Fill: n(122, 44, 58, 255), FillHi: n(200, 120, 120, 90),
		Done:   n(90, 140, 90, 255),
		Seal:   n(122, 44, 58, 255), SealTx: n(244, 226, 190, 255),
		Caption: n(112, 88, 64, 255),
	},
	4: { // WOODMERE — tavern hearth: carved oak, warm amber
		ID: 4, Name: "Woodmere", FrameStyle: 9,
		Bg: n(88, 58, 34, 255), Bg2: n(66, 42, 24, 255),
		Panel: n(226, 192, 142, 255), PanelEd: n(60, 38, 20, 80),
		Plate: n(70, 46, 26, 255), PlateTx: n(240, 202, 130, 255),
		Banner: n(70, 46, 26, 255), BannerTx: n(240, 202, 130, 255), BannerEdge: n(150, 104, 56, 255),
		Ink: n(52, 34, 20, 255), Muted: n(116, 82, 50, 255),
		Sub:  n(96, 128, 64, 255),
		Gold:   n(196, 140, 60, 255),
		PillBg: n(96, 62, 32, 240), PillTx: n(244, 210, 140, 255),
		Track:  n(70, 48, 28, 255), Fill: n(214, 156, 74, 255), FillHi: n(244, 200, 120, 90),
		Done:   n(120, 180, 100, 255),
		Seal:   n(112, 70, 34, 255), SealTx: n(244, 210, 140, 255),
		Caption: n(116, 82, 50, 255),
	},
	5: { // EMBLEM NOIR — secret society: matte black, gold mark, red wax
		ID: 5, Name: "Emblem Noir", FrameStyle: 4,
		Bg: n(14, 14, 16, 255), Bg2: n(24, 24, 28, 255),
		Panel: n(30, 30, 34, 255), PanelEd: n(0, 0, 0, 120),
		Plate: n(48, 48, 52, 255), PlateTx: n(238, 236, 230, 255),
		Banner: n(20, 20, 24, 255), BannerTx: n(226, 222, 210, 255), BannerEdge: n(198, 166, 100, 255),
		Ink: n(238, 236, 230, 255), Muted: n(158, 154, 146, 255),
		Sub:  n(140, 190, 150, 255),
		Gold:   n(198, 166, 100, 255),
		PillBg: n(58, 56, 60, 240), PillTx: n(238, 236, 230, 255),
		Track:  n(16, 16, 18, 255), Fill: n(198, 166, 100, 255), FillHi: n(236, 210, 150, 90),
		Done:   n(110, 180, 120, 255),
		Seal:   n(148, 30, 40, 255), SealTx: n(244, 214, 140, 255),
		Caption: n(158, 154, 146, 255),
	},
	6: { // HOLO GACHA — collection vault: pastel holo, soft sparkles
		ID: 6, Name: "Holo Gacha", FrameStyle: 5,
		Bg: n(186, 224, 220, 255), Bg2: n(222, 196, 238, 255),
		Panel: n(250, 247, 252, 255), PanelEd: n(160, 140, 190, 60),
		Plate: n(120, 104, 170, 255), PlateTx: n(255, 255, 255, 255),
		Banner: n(130, 112, 180, 255), BannerTx: n(255, 250, 255, 255), BannerEdge: n(222, 120, 160, 255),
		Ink: n(74, 64, 94, 255), Muted: n(130, 120, 152, 255),
		Sub:  n(96, 170, 130, 255),
		Gold:   n(222, 120, 160, 255),
		PillBg: n(140, 122, 190, 240), PillTx: n(255, 250, 255, 255),
		Track:  n(210, 202, 230, 255), Fill: n(222, 120, 160, 255), FillHi: n(255, 200, 220, 110),
		Done:   n(110, 190, 140, 255),
		Seal:   n(222, 120, 160, 255), SealTx: n(255, 245, 250, 255),
		Caption: n(130, 120, 152, 255),
	},
	8: { // NEON ARCADE — arcade cabinet: night grid, magenta/cyan glow
		ID: 8, Name: "Neon Arcade", FrameStyle: 6,
		Bg: n(10, 12, 28, 255), Bg2: n(18, 22, 48, 255),
		Panel: n(16, 20, 44, 255), PanelEd: n(0, 0, 0, 110),
		Plate: n(30, 26, 58, 255), PlateTx: n(64, 224, 255, 255),
		Banner: n(24, 20, 52, 255), BannerTx: n(255, 64, 160, 255), BannerEdge: n(64, 224, 255, 255),
		Ink: n(224, 242, 255, 255), Muted: n(128, 168, 208, 255),
		Sub:  n(64, 224, 160, 255),
		Gold:   n(64, 224, 255, 255),
		PillBg: n(46, 28, 66, 240), PillTx: n(255, 170, 214, 255),
		Track:  n(8, 10, 24, 255), Fill: n(255, 64, 160, 255), FillHi: n(255, 170, 214, 100),
		Done:   n(64, 224, 160, 255),
		Seal:   n(255, 64, 160, 255), SealTx: n(255, 240, 250, 255),
		Caption: n(128, 168, 208, 255),
	},
	9: { // RUNE MONOLITH — ancient stone: basalt, carved glyphs, ember
		ID: 9, Name: "Rune Monolith", FrameStyle: 7,
		Bg: n(44, 50, 46, 255), Bg2: n(30, 34, 32, 255),
		Panel: n(66, 74, 68, 255), PanelEd: n(0, 0, 0, 100),
		Plate: n(24, 26, 24, 255), PlateTx: n(232, 140, 64, 255),
		Banner: n(28, 32, 30, 255), BannerTx: n(232, 190, 130, 255), BannerEdge: n(232, 140, 64, 255),
		Ink: n(224, 230, 214, 255), Muted: n(156, 166, 152, 255),
		Sub:  n(150, 200, 140, 255),
		Gold:   n(232, 140, 64, 255),
		PillBg: n(30, 34, 32, 240), PillTx: n(240, 200, 140, 255),
		Track:  n(22, 24, 22, 255), Fill: n(232, 140, 64, 255), FillHi: n(250, 190, 120, 100),
		Done:   n(130, 190, 120, 255),
		Seal:   n(180, 70, 26, 255), SealTx: n(250, 214, 150, 255),
		Caption: n(156, 166, 152, 255),
	},
	10: { // CRIMSON COURT — vampire court: velvet damask, ornate gold
		ID: 10, Name: "Crimson Court", FrameStyle: 8,
		Bg: n(66, 12, 22, 255), Bg2: n(44, 8, 16, 255),
		Panel: n(96, 22, 34, 255), PanelEd: n(0, 0, 0, 110),
		Plate: n(40, 8, 14, 255), PlateTx: n(232, 190, 120, 255),
		Banner: n(52, 10, 18, 255), BannerTx: n(240, 210, 150, 255), BannerEdge: n(212, 168, 86, 255),
		Ink: n(244, 226, 206, 255), Muted: n(198, 156, 146, 255),
		Sub:  n(140, 190, 140, 255),
		Gold:   n(212, 168, 86, 255),
		PillBg: n(64, 14, 24, 240), PillTx: n(236, 202, 140, 255),
		Track:  n(34, 8, 14, 255), Fill: n(212, 168, 86, 255), FillHi: n(246, 214, 150, 100),
		Done:   n(130, 190, 130, 255),
		Seal:   n(140, 24, 36, 255), SealTx: n(244, 214, 150, 255),
		Caption: n(198, 156, 146, 255),
	},
}

// resolveTheme — nil for style 0 (unset) and 7 (Royal Decree baked art).
func resolveTheme(style int) *cardTheme {
	if style <= 0 || style == 7 || style > 10 {
		return nil
	}
	return cardThemes[style]
}

// Paints the themed backdrop for the portrait kinds (RANK / ALLOCATE /
// SKILLUP). Geometry contract is identical to the bg_*.png bakes:
//   banner band y18..108, name plate (60,132)-(330,186),
//   panel (55,208)-(545,832), dividers y266/y492,
//   lower label (85,520), seal (70,935), caption (305,936).


func themeBannerFor(kind string) string {
	switch kind {
	case "RANK":
		return "ADVENTURER"
	case "ALLOCATE":
		return "ALLOCATION"
	case "SKILLUP":
		return "SKILL UP"
	}
	return "JOKER RPG"
}

func themeTopLabelFor(kind string) string {
	switch kind {
	case "RANK":
		return "THE RECORD"
	case "ALLOCATE":
		return "THE POINTS"
	}
	return "THE RECORD"
}

func themeBottomLabelFor(kind string) string {
	switch kind {
	case "RANK":
		return "THE STANDING"
	case "ALLOCATE":
		return "THE ALLOCATION"
	}
	return ""
}

// drawPortraitShell — full themed backdrop for the 600x1000 family.
func drawPortraitShell(dc *gg.Context, th *cardTheme, kind string) {
	// 1. page base: vertical gradient
	lg := gg.NewLinearGradient(0, 0, 0, portraitH)
	lg.AddColorStop(0, th.Bg)
	lg.AddColorStop(1, th.Bg2)
	dc.SetFillStyle(lg)
	dc.DrawRectangle(0, 0, portraitW, portraitH)
	dc.Fill()

	// 2. signature motif
	drawThemeMotif(dc, th)

	// 3. banner band (y18..108) with fold notches
	dc.SetColor(th.Banner)
	dc.MoveTo(70, 18)
	dc.LineTo(530, 18)
	dc.LineTo(530, 108)
	dc.LineTo(70, 108)
	dc.LineTo(46, 63)
	dc.ClosePath()
	dc.Fill()
	// mirrored fold
	dc.MoveTo(530, 18)
	dc.LineTo(554, 63)
	dc.LineTo(530, 108)
	dc.ClosePath()
	dc.Fill()
	dc.SetColor(th.BannerEdge)
	dc.SetLineWidth(2)
	dc.MoveTo(70, 18)
	dc.LineTo(530, 18)
	dc.LineTo(530, 108)
	dc.LineTo(70, 108)
	dc.LineTo(46, 63)
	dc.ClosePath()
	dc.Stroke()
	banner := themeBannerFor(kind)
	portraitFitText(dc, portraitAsset("CinzelDecBold.ttf"), 30, banner, 420, 18)
	dc.SetColor(th.BannerTx)
	dc.DrawStringAnchored(banner, 300, 64, 0.5, 0.5)

	// 4. name plate (60,132)-(330,186)
	dc.SetColor(th.Plate)
	dc.DrawRoundedRectangle(60, 132, 270, 54, 10)
	dc.Fill()
	dc.SetColor(th.BannerEdge)
	dc.SetLineWidth(1.5)
	dc.DrawRoundedRectangle(60, 132, 270, 54, 10)
	dc.Stroke()

	// 5. main panel (55,208)-(545,832)
	dc.SetColor(th.Panel)
	dc.DrawRectangle(55, 208, 490, 624)
	dc.Fill()
	dc.SetColor(th.PanelEd)
	dc.DrawRectangle(55, 208, 14, 624)
	dc.Fill()
	dc.DrawRectangle(531, 208, 14, 624)
	dc.Fill()
	drawThemeFrame(dc, th, 55, 208, 490, 624)

	// 6. panel labels + dividers
	topLabel := themeTopLabelFor(kind)
	if topLabel != "" {
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, topLabel, 300, 10)
		dc.SetColor(th.Muted)
		dc.DrawStringAnchored(topLabel, 300, 232, 0.5, 0.5)
	}
	dc.SetColor(th.Gold)
	dc.SetLineWidth(1.5)
	dc.DrawLine(85, 266, 515, 266)
	dc.Stroke()
	dc.SetLineWidth(1)
	dc.DrawLine(85, 271, 515, 271)
	dc.Stroke()
	dc.SetLineWidth(1.5)
	dc.DrawLine(85, 492, 515, 492)
	dc.Stroke()
	bottomLabel := themeBottomLabelFor(kind)
	if bottomLabel != "" {
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, bottomLabel, 300, 10)
		dc.SetColor(th.Muted)
		dc.DrawStringAnchored(bottomLabel, 85, 520, 0, 0.5)
	}

	// 7. footer rule (seal + caption zone)
	dc.SetColor(th.Gold)
	dc.SetLineWidth(1)
	dc.DrawLine(55, 900, 545, 900)
	dc.Stroke()
}

// drawThemeFrame — per-theme frame construction around a rect.
func drawThemeFrame(dc *gg.Context, th *cardTheme, x, y, w, h float64) {
	switch th.FrameStyle {
	case 1: // stone: beveled light/dark edges + corner rivets
		dc.SetColor(lighten(th.Panel, 46))
		dc.SetLineWidth(4)
		dc.DrawLine(x+3, y+h-3, x+3, y+3)
		dc.DrawLine(x+3, y+3, x+w-3, y+3)
		dc.Stroke()
		dc.SetColor(darken(th.Panel, 70))
		dc.DrawLine(x+w-3, y+3, x+w-3, y+h-3)
		dc.DrawLine(x+3, y+h-3, x+w-3, y+h-3)
		dc.Stroke()
		for _, c := range corners(x, y, w, h) {
			dc.SetColor(darken(th.Plate, 20))
			dc.DrawCircle(c[0], c[1], 5)
			dc.Fill()
			dc.SetColor(lighten(th.Panel, 60))
			dc.DrawCircle(c[0]-1, c[1]-1, 2)
			dc.Fill()
		}
	case 2: // arcanum: double gold hairline + corner diamonds
		dc.SetColor(th.Gold)
		dc.SetLineWidth(1.4)
		dc.DrawRectangle(x+8, y+8, w-16, h-16)
		dc.Stroke()
		dc.SetLineWidth(0.7)
		dc.DrawRectangle(x+14, y+14, w-28, h-28)
		dc.Stroke()
		for _, c := range corners(x+8, y+8, w-16, h-16) {
			r6diamond(dc, c[0], c[1], 4.5, th.Gold)
		}
	case 3: // retro: burgundy double rules + dot rows
		dc.SetColor(th.Gold)
		dc.SetLineWidth(2.2)
		dc.DrawRectangle(x+6, y+6, w-12, h-12)
		dc.Stroke()
		dc.SetLineWidth(0.8)
		dc.DrawRectangle(x+12, y+12, w-24, h-24)
		dc.Stroke()
		dc.SetColor(th.Muted)
		for dx := 24.0; dx < w-24; dx += 16 {
			dc.DrawCircle(x+dx, y+19, 1.4)
			dc.Fill()
			dc.DrawCircle(x+dx, y+h-19, 1.4)
			dc.Fill()
		}
	case 4: // noir: single inset gold line + corner ticks
		dc.SetColor(th.Gold)
		dc.SetLineWidth(1)
		dc.DrawRectangle(x+14, y+14, w-28, h-28)
		dc.Stroke()
		dc.SetLineWidth(2.5)
		for _, c := range corners(x+14, y+14, w-28, h-28) {
			dc.DrawLine(c[0], c[1], c[0]+12, c[1])
			dc.Stroke()
			dc.DrawLine(c[0], c[1], c[0], c[1]+12)
			dc.Stroke()
		}
	case 5: // gacha: soft white rounded frame + sparkles
		dc.SetColor(n(255, 255, 255, 220))
		dc.SetLineWidth(6)
		dc.DrawRoundedRectangle(x+8, y+8, w-16, h-16, 18)
		dc.Stroke()
		dc.SetColor(th.BannerEdge)
		for _, p := range [][2]float64{{x + 30, y + 30}, {x + w - 30, y + 30}, {x + 30, y + h - 30}, {x + w - 30, y + h - 30}} {
			drawSparkle(dc, p[0], p[1], 6)
		}
	case 6: // neon: glow stack magenta + inner cyan
		dc.SetColor(n(255, 64, 160, 60))
		dc.SetLineWidth(9)
		dc.DrawRectangle(x+4, y+4, w-8, h-8)
		dc.Stroke()
		dc.SetColor(n(255, 64, 160, 220))
		dc.SetLineWidth(3)
		dc.DrawRectangle(x+6, y+6, w-12, h-12)
		dc.Stroke()
		dc.SetColor(n(64, 224, 255, 200))
		dc.SetLineWidth(1.2)
		dc.DrawRectangle(x+12, y+12, w-24, h-24)
		dc.Stroke()
	case 7: // monolith: carved grooves + ember ticks
		dc.SetColor(darken(th.Panel, 56))
		dc.SetLineWidth(3)
		dc.DrawRectangle(x+7, y+7, w-14, h-14)
		dc.Stroke()
		dc.SetColor(lighten(th.Panel, 30))
		dc.SetLineWidth(1.4)
		dc.DrawRectangle(x+10, y+10, w-20, h-20)
		dc.Stroke()
		dc.SetColor(th.Gold)
		for i := 0; i < 5; i++ {
			tx := x + 26 + float64(i)*((w-52)/4)
			dc.DrawLine(tx, y+10, tx+6, y+18)
			dc.Stroke()
			dc.DrawLine(tx+6, y+h-18, tx, y+h-10)
			dc.Stroke()
		}
	case 8: // crimson: ornate gold double + corner diamonds
		dc.SetColor(th.Gold)
		dc.SetLineWidth(3)
		dc.DrawRectangle(x+6, y+6, w-12, h-12)
		dc.Stroke()
		dc.SetLineWidth(1)
		dc.DrawRectangle(x+13, y+13, w-26, h-26)
		dc.Stroke()
		for _, c := range corners(x+6, y+6, w-12, h-12) {
			r6diamond(dc, c[0], c[1], 7, th.Gold)
			r6diamond(dc, c[0], c[1], 3.5, darken(th.Panel, 40))
		}
	case 9: // woodmere: carved notch border
		dc.SetColor(darken(th.Panel, 90))
		dc.SetLineWidth(5)
		dc.DrawRectangle(x+8, y+8, w-16, h-16)
		dc.Stroke()
		dc.SetColor(lighten(th.Panel, 40))
		dc.SetLineWidth(1.6)
		dc.DrawRectangle(x+15, y+15, w-30, h-30)
		dc.Stroke()
		for _, c := range corners(x+8, y+8, w-16, h-16) {
			dc.SetColor(darken(th.Plate, 30))
			dc.DrawRectangle(c[0]-5, c[1]-5, 10, 10)
			dc.Fill()
		}
	default: // decree: gold hairline rounded
		dc.SetColor(th.Gold)
		dc.SetLineWidth(1.5)
		dc.DrawRoundedRectangle(x+10, y+10, w-20, h-20, 6)
		dc.Stroke()
	}
}

// ── per-theme signature motifs ────────────────────────────────────────
func drawThemeMotif(dc *gg.Context, th *cardTheme) {
	switch th.ID {
	case 1: // stonekeep: block seams + rivet heads
		dc.SetColor(n(0, 0, 0, 40))
		for y := 140.0; y < portraitH; y += 118 {
			dc.DrawLine(0, y, portraitW, y)
			dc.Stroke()
		}
		for x := 150.0; x < portraitW; x += 150 {
			dc.DrawLine(x, 0, x, portraitH)
			dc.Stroke()
		}
		dc.SetColor(n(255, 255, 255, 26))
		for y := 90.0; y < portraitH; y += 118 {
			for x := 60.0; x < portraitW; x += 150 {
				dc.DrawCircle(x, y, 3)
				dc.Fill()
			}
		}
	case 2: // arcanum: ritual circles
		dc.SetColor(n(212, 175, 55, 36))
		dc.SetLineWidth(1.2)
		dc.DrawCircle(300, 500, 200)
		dc.Stroke()
		dc.DrawCircle(300, 500, 158)
		dc.Stroke()
		for i := 0; i < 12; i++ {
			a := float64(i) * math.Pi / 6
			r6diamond(dc, 300+200*math.Cos(a), 500+200*math.Sin(a), 3, n(212, 175, 55, 60))
		}
	case 3: // retro: halftone dots
		dc.SetColor(n(120, 96, 70, 34))
		for y := 22.0; y < portraitH; y += 14 {
			for x := 22.0; x < portraitW; x += 14 {
				dc.DrawCircle(x, y, 1.1)
				dc.Fill()
			}
		}
	case 4: // woodmere: planks + grain
		dc.SetColor(n(0, 0, 0, 60))
		for y := 125.0; y < portraitH; y += 125 {
			dc.SetLineWidth(2.4)
			dc.DrawLine(0, y, portraitW, y)
			dc.Stroke()
		}
		dc.SetColor(n(255, 230, 180, 22))
		for y := 40.0; y < portraitH; y += 26 {
			dc.SetLineWidth(1)
			dc.DrawLine(0, y+float64(int(y)%17), portraitW, y+8)
			dc.Stroke()
		}
	case 5: // noir: carbon crosshatch + center emblem
		dc.SetColor(n(255, 255, 255, 7))
		for y := 0.0; y < portraitH; y += 5 {
			dc.SetLineWidth(1)
			dc.DrawLine(0, y, portraitW, y)
			dc.Stroke()
		}
		dc.SetColor(n(198, 166, 100, 22))
		r6diamond(dc, 300, 520, 210, n(198, 166, 100, 20))
		r6diamond(dc, 300, 520, 160, n(198, 166, 100, 26))
	case 6: // gacha: sparkles
		dc.SetColor(n(255, 255, 255, 130))
		for _, p := range [][2]float64{{70, 120}, {520, 90}, {90, 420}, {540, 380}, {60, 760}, {530, 720}, {300, 950}, {140, 250}, {470, 260}} {
			drawSparkle(dc, p[0], p[1], 5)
		}
	case 8: // neon: horizon grid + scanlines
		dc.SetColor(n(64, 224, 255, 40))
		for i := 0; i < 9; i++ {
			y := 300 + float64(i)*80
			dc.SetLineWidth(1)
			dc.DrawLine(0, y, portraitW, y)
			dc.Stroke()
		}
		for i := -4; i <= 4; i++ {
			x0 := 300 + float64(i)*40
			x1 := 300 + float64(i)*160
			dc.DrawLine(x0, 0, x1, portraitH)
			dc.Stroke()
		}
		dc.SetColor(n(255, 64, 160, 14))
		for y := 0.0; y < portraitH; y += 4 {
			dc.SetLineWidth(1)
			dc.DrawLine(0, y, portraitW, y)
			dc.Stroke()
		}
	case 9: // monolith: carved glyph columns
		dc.SetColor(n(255, 255, 255, 16))
		for x := 60.0; x < portraitW; x += 60 {
			for y := 40.0; y < portraitH; y += 90 {
				dc.SetLineWidth(2)
				dc.DrawLine(x, y, x+10, y+14)
				dc.Stroke()
				dc.DrawLine(x+10, y+14, x, y+28)
				dc.Stroke()
			}
		}
	case 10: // crimson: damask diamonds
		dc.SetColor(n(212, 168, 86, 34))
		for y := 0.0; y < portraitH+80; y += 80 {
			for x := 0.0; x < portraitW+80; x += 80 {
				off := 0.0
				if int(y/80)%2 == 1 {
					off = 40
				}
				r6diamond(dc, x+off, y, 26, n(212, 168, 86, 30))
			}
		}
	}
}

// ── themed seal / caption helpers (mirror the wax-seal contract) ──────
func drawThemedSeal(dc *gg.Context, th *cardTheme, text string, x, y, r float64) {
	dc.SetColor(th.Seal)
	dc.DrawCircle(x, y, r)
	dc.Fill()
	dc.SetColor(darken(th.Seal, 60))
	dc.SetLineWidth(2.5)
	dc.DrawCircle(x, y, r)
	dc.Stroke()
	dc.SetColor(lighten(th.Seal, 40))
	dc.SetLineWidth(1)
	dc.DrawCircle(x, y, r-5)
	dc.Stroke()
	t := portraitSanitize(text)
	if t == "" {
		return
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, t, r*1.5, 8)
	dc.SetColor(th.SealTx)
	dc.DrawStringAnchored(t, x, y-1, 0.5, 0.5)
}

func drawThemedCaption(dc *gg.Context, th *cardTheme, caption string) {
	c := portraitSanitize(caption)
	if c == "" {
		return
	}
	runes := []rune(c)
	if len(runes) > 52 {
		c = string(runes[:49]) + "..."
	}
	portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 18, c, portraitCaptionMax, 11)
	dc.SetColor(th.Caption)
	dc.DrawStringAnchored(c, portraitCaptionX, portraitCaptionY, 0, 0.5)
}

// ── small color helpers ───────────────────────────────────────────────
func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func alphaN(c color.NRGBA, a uint8) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: a} }

func lighten(c color.NRGBA, amt uint8) color.NRGBA {
	return n(clamp8(float64(c.R)+float64(amt)), clamp8(float64(c.G)+float64(amt)), clamp8(float64(c.B)+float64(amt)), c.A)
}

func darken(c color.NRGBA, amt uint8) color.NRGBA {
	return n(clamp8(float64(c.R)-float64(amt)), clamp8(float64(c.G)-float64(amt)), clamp8(float64(c.B)-float64(amt)), c.A)
}

func corners(x, y, w, h float64) [4][2]float64 {
	return [4][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}}
}

func drawSparkle(dc *gg.Context, x, y, r float64) {
	dc.MoveTo(x, y-r)
	dc.LineTo(x+r*0.3, y-r*0.3)
	dc.LineTo(x+r, y)
	dc.LineTo(x+r*0.3, y+r*0.3)
	dc.LineTo(x, y+r)
	dc.LineTo(x-r*0.3, y+r*0.3)
	dc.LineTo(x-r, y)
	dc.LineTo(x-r*0.3, y-r*0.3)
	dc.ClosePath()
	dc.Fill()
}

