package economy

// style_craft.go - per-style compositions for the item-creation family
// (CRAFT / BREW / COOK / FORGE / FISH, 1000x600 landscape) and the rank-up
// DECREE card, in the 9 rebuilt design systems. Style 0/7 keeps the baked
// decree-family art. Each style speaks its own visual language here.

import (
	"fmt"
	"image"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

// styledEconomyKinds - transaction types with rebuilt compositions.
func styledEconomyKinds(t string) bool {
	switch t {
	case "CRAFT", "BREW", "COOK", "FORGE", "FISH", "DECREE":
		return true
	}
	return false
}

// renderStyledEconomy returns the styled canvas or nil to fall back.
func renderStyledEconomy(req *TransactionCardRequest) image.Image {
	if req == nil || req.Style <= 0 || req.Style == 7 || req.Style > 10 {
		return nil
	}
	if !styledEconomyKinds(req.Type) {
		return nil
	}
	var img image.Image
	switch req.Style {
	case 1:
		img = econStyle01(req)
	case 2:
		img = econStyle02(req)
	case 3:
		img = econStyle03(req)
	case 4:
		img = econStyle04(req)
	case 5:
		img = econStyle05(req)
	case 6:
		img = econStyle06(req)
	case 8:
		img = econStyle08(req)
	case 9:
		img = econStyle09(req)
	case 10:
		img = econStyle10(req)
	}
	return img
}

func econName(req *TransactionCardRequest) string {
	n := cardstyle.Sanitize(req.Nickname)
	if n == "" {
		n = "Adventurer"
	}
	return n
}

func econItem(req *TransactionCardRequest) string {
	i := cardstyle.Sanitize(req.ItemName)
	if i == "" {
		i = "Unknown Item"
	}
	return i
}

func econTypeTitle(t string) string {
	switch t {
	case "CRAFT":
		return "CRAFTED"
	case "BREW":
		return "BREWED"
	case "COOK":
		return "COOKED"
	case "FORGE":
		return "FORGED"
	case "FISH":
		return "CATCH OF THE DAY"
	case "DECREE":
		return "RANK UP"
	}
	return strings.ToUpper(t)
}

func econCaption(req *TransactionCardRequest, t string) string {
	if c := cardstyle.Sanitize(req.Details); c != "" && t != "DECREE" {
		return c
	}
	if t == "DECREE" {
		return "keep rising - the guild watches"
	}
	return fmt.Sprintf(".j %s %s", strings.ToLower(t), econItem(req))
}

// ── Style 1 STONEKEEP - anvil plaque ─────────────────────────────────

func econStyle01(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.Stonekeep()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Speckle(dc, 0, 0, W, H, 90, 0xC7A11, cardstyle.N(0, 0, 0, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2.4, cardstyle.Darken(p.Panel, 40), false)
	// iron plate title top-left
	cardstyle.BevelRect(dc, 40, 34, 380, 70, p.Accent, cardstyle.Lighten(p.Accent, 40), cardstyle.N(10, 10, 12, 255), 2.6)
	for _, c := range [][2]float64{{52, 46}, {408, 46}, {52, 92}, {408, 92}} {
		cardstyle.Rivet(dc, c[0], c[1], 4.4, p.Track)
	}
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, econTypeTitle(req.Type), 62, 69, p.SealTx, 0, 0.5, 340, 13)
	// maker slab
	cardstyle.BevelRect(dc, 46, 130, 300, 54, p.Panel, cardstyle.Lighten(p.Panel, 42), cardstyle.Darken(p.Panel, 66), 3)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 16, strings.ToUpper(cardstyle.Sanitize(econName(req))), 62, 157, p.Ink, 0, 0.5, 260, 10)
	// item slab center
	cardstyle.BevelRect(dc, 90, 220, 820, 190, p.Panel, cardstyle.Lighten(p.Panel, 42), cardstyle.Darken(p.Panel, 66), 3.4)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 44, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 24), W/2, 300, p.Ink, cardstyle.Lighten(p.Panel, 60), 0.5, 0.5, 700, 20)
	if req.Amount > 1 {
		cardstyle.Rivet(dc, 866, 88, 26, cardstyle.N(255, 179, 64, 255))
		cardstyle.Text(dc, cardstyle.FtCinzel, 20, fmt.Sprintf("X%d", int(req.Amount)), 866, 88, cardstyle.N(30, 20, 8, 255), 0.5, 0.5, 60, 10)
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 17, cardstyle.TruncateRunes(econCaption(req, req.Type), 76), W/2, 490, cardstyle.WithA(p.Ink, 210), 0.5, 0.5, 820, 10)
	cardstyle.SealMini(dc, 70, 552, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	return dc.Image()
}

// ── Style 2 GOLDEN ARCANUM - transmutation circle ────────────────────

func econStyle02(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.GoldenArcanum()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.StarField(dc, 0, 0, W, H, 24, 0xAC2338, cardstyle.N(232, 200, 120, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 12, 1.2, cardstyle.WithA(p.Accent, 170), true)
	// left transmutation circle with item initial
	cardstyle.MagicCircle(dc, 250, 300, 150, p.Accent)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 64, firstRuneOf(econItem(req)), 250, 294, cardstyle.Hex(0xffe9a8), 0.5, 0.5, 120, 26)
	// right ledger
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(econTypeTitle(req.Type)), 620, 170, p.Accent, 0.5, 0.5, 480, 14)
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(econName(req))), 620, 210, p.Muted, 0.5, 0.5, 460, 10)
	cardstyle.DotLeader(dc, 480, 760, 244, 1, cardstyle.WithA(p.Accent, 110))
	cardstyle.Text(dc, cardstyle.FtCinzel, 26, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), W/2+120, 286, p.Ink, 0.5, 0.5, 700, 14)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 18, fmt.Sprintf("X%d", int(req.Amount)), W/2+120, 324, p.Accent2, 0.5, 0.5, 120, 10)
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2+120, 370, p.Muted, 0.5, 0.5, 640, 10)
	cardstyle.SealMini(dc, 620, 470, 22, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 12)
	return dc.Image()
}

// ── Style 3 RETRO COURT - playbill notice ────────────────────────────

func econStyle03(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.RetroCourt()
	cardstyle.PageBase(dc, W, H, p.Bg, p.Bg2, 26)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2, p.Accent, false)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 15, 0.7, cardstyle.WithA(p.Accent, 150), false)
	cardstyle.Text(dc, cardstyle.FtIMFell, 44, strings.ToUpper(econTypeTitle(req.Type)), W/2, 96, p.Ink, 0.5, 0.5, 860, 24)
	dc.SetColor(p.Ink)
	dc.SetLineWidth(1.6)
	dc.DrawLine(70, 140, W-70, 140)
	dc.Stroke()
	dc.SetLineWidth(0.6)
	dc.DrawLine(70, 146, W-70, 146)
	dc.Stroke()
	cardstyle.Fleuron(dc, W/2, 143, 4.4, p.Accent)
	cardstyle.Text(dc, cardstyle.FtIMFellIt, 18, "performed by "+econName(req), W/2, 178, p.Accent, 0.5, 0.5, 700, 11)
	cardstyle.Text(dc, cardstyle.FtIMFell, 40, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 26)), W/2, 268, p.Ink, 0.5, 0.5, 840, 18)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtIMFell, 22, fmt.Sprintf("A QUANTITY OF %d", int(req.Amount)), W/2, 316, p.Accent, 0.5, 0.5, 500, 12)
	}
	cardstyle.DotLeader(dc, 320, 680, 356, 1.2, cardstyle.WithA(p.Ink, 140))
	cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2, 392, p.Muted, 0.5, 0.5, 800, 10)
	cardstyle.SealMini(dc, W/2, 470, 26, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtIMFell, 14)
	return dc.Image()
}

// ── Style 4 WOODMERE - workbench plaque ──────────────────────────────

func econStyle04(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.Woodmere()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Planks(dc, W, H, 0x5700D4, cardstyle.N(0, 0, 0, 60), cardstyle.N(255, 230, 180, 22))
	// hanging sign
	cardstyle.BevelRect(dc, W/2-260, 30, 520, 76, cardstyle.N(112, 76, 42, 255), cardstyle.N(168, 122, 72, 255), cardstyle.N(44, 28, 14, 255), 2.6)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, econTypeTitle(req.Type), W/2, 66, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, 460, 14)
	// vellum result panel
	dc.SetColor(cardstyle.N(36, 22, 10, 255))
	dc.DrawRoundedRectangle(120, 150, 760, 330, 10)
	dc.Fill()
	dc.SetColor(p.Panel)
	dc.DrawRoundedRectangle(132, 162, 736, 306, 8)
	dc.Fill()
	cardstyle.Stitches(dc, 142, 176, 452, 22, cardstyle.N(120, 84, 46, 255))
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, "THE WORK OF", 190, 200, cardstyle.N(116, 82, 50, 255), 0, 0.5, 220, 9)
	cardstyle.Text(dc, cardstyle.FtCinzel, 20, strings.ToUpper(cardstyle.Sanitize(econName(req))), 190, 232, cardstyle.N(52, 34, 20, 255), 0, 0.5, 300, 11)
	cardstyle.DotLeader(dc, 200, 800, 276, 1.2, cardstyle.N(150, 104, 56, 160))
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 40, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 24), W/2, 330, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 660, 18)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 20, fmt.Sprintf("X%d", int(req.Amount)), W/2, 374, cardstyle.N(150, 78, 24, 255), 0.5, 0.5, 120, 12)
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2, 520, cardstyle.N(226, 192, 142, 230), 0.5, 0.5, 800, 10)
	cardstyle.SealMini(dc, 70, 552, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	return dc.Image()
}

// ── Style 5 EMBLEM NOIR - operation report ───────────────────────────

func econStyle05(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.EmblemNoir()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	dc.SetColor(cardstyle.N(255, 255, 255, 6))
	dc.SetLineWidth(1)
	for y := 0.0; y < H; y += 5 {
		dc.DrawLine(0, y, W, y)
		dc.Stroke()
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(econTypeTitle(req.Type)), 60, 62, p.Accent, 0, 0.5, 420, 10)
	cardstyle.Stamp(dc, 830, 62, "VERIFIED", p.Accent)
	dc.SetColor(cardstyle.WithA(p.Accent, 160))
	dc.SetLineWidth(1)
	dc.DrawLine(60, 82, W-60, 82)
	dc.Stroke()
	cardstyle.Text(dc, cardstyle.FtInterSemi, 26, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 30)), 60, 190, p.Ink, 0, 0.5, 640, 14)
	cardstyle.Text(dc, cardstyle.FtInter, 13, "OPERATOR: "+strings.ToUpper(cardstyle.Sanitize(econName(req))), 60, 230, p.Muted, 0, 0.5, 640, 9)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtInter, 13, fmt.Sprintf("QUANTITY: %d", int(req.Amount)), 60, 256, p.Muted, 0, 0.5, 640, 9)
	}
	cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 70)), 60, 300, cardstyle.WithA(p.Muted, 180), 0, 0.5, 700, 8)
	// wax seal
	cardstyle.WaxSeal(dc, 860, 260, 56, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 22)
	// corner registration marks
	for _, c := range [][2]float64{{26, 26}, {W - 26, 26}, {26, H - 26}, {W - 26, H - 26}} {
		dc.SetColor(cardstyle.WithA(p.Muted, 110))
		dc.SetLineWidth(1)
		dc.DrawLine(c[0]-7, c[1], c[0]+7, c[1])
		dc.Stroke()
		dc.DrawLine(c[0], c[1]-7, c[0], c[1]+7)
		dc.Stroke()
	}
	return dc.Image()
}

// ── Style 6 SOUL FORGE - celestial transmutation notice ──────────────

func econStyle06(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.SoulForge()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.StarField(dc, 16, 16, W-16, H-16, 34, 0x50F6CF, cardstyle.N(255, 255, 255, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 11, 1.3, p.PanelEd, false)
	cardstyle.Diamond(dc, W/2, 11, 5.4, p.PanelEd)
	cardstyle.Diamond(dc, W/2, H-11, 5.4, p.PanelEd)
	// title + subtitle + rule
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, strings.ToUpper(econTypeTitle(req.Type)), W/2, 74, p.Accent, 0.5, 0.5, 560, 16)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 102, p.Muted, 0.5, 2.6)
	cardstyle.DotLeader(dc, W/2-110, W/2+110, 120, 1, cardstyle.WithA(p.Accent, 140))
	// magic circle + item initial left
	cardstyle.MagicCircle(dc, 190, 330, 108, p.Accent)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 58, firstRuneOf(econItem(req)), 190, 324, p.Accent2, p.Accent2, 0.5, 0.5, 110, 22)
	// item + caption panel
	cardstyle.PanelRounded(dc, 360, 218, 590, 224, 14, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.2)
	cardstyle.Text(dc, cardstyle.FtMedieval, 27, cardstyle.TruncateRunes(econItem(req), 26), 388, 268, p.Ink, 0, 0.5, 540, 14)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("QUANTITY %d", int(req.Amount)), 388, 304, p.Accent2, 0, 0.5, 300, 11)
	}
	cardstyle.DotLeader(dc, 388, 922, 348, 1.2, cardstyle.WithA(p.Muted, 110))
	cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(econCaption(req, req.Type), 66), 388, 380, p.Muted, 0, 0.5, 540, 10)
	// gold pill CTA
	cardstyle.PillCTA(dc, W/2, 512, 380, 40, cardstyle.TruncateRunes(strings.ToLower(econCaption(req, req.Type)), 40), "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 13)
	cardstyle.FooterNote(dc, W/2, H-26, cardstyle.Sanitize(req.SealText), p.Caption, cardstyle.FtMedieval, 12, 300)
	return dc.Image()
}

// ── Style 8 NEON ARCADE - crafted toast ──────────────────────────────

func econStyle08(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.NeonArcade()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Scanlines(dc, W, H, cardstyle.N(255, 64, 160, 10), 4)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2, cardstyle.HexA(0xff40a0, 200), false)
	// marquee
	dc.SetColor(cardstyle.WithA(p.Panel, 235))
	dc.DrawRoundedRectangle(W/2-330, 34, 660, 70, 6)
	dc.Fill()
	cardstyle.BracketCorners(dc, W/2-330, 34, 660, 70, 16, cardstyle.HexA(0x40e0ff, 220), 2)
	cardstyle.GlowText(dc, cardstyle.FtPS2P, 18, strings.ToUpper(econTypeTitle(req.Type)), W/2, 70, cardstyle.HexA(0xff40a0, 120), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 600, 9)
	// item panel
	dc.SetColor(cardstyle.WithA(p.Panel, 225))
	dc.DrawRoundedRectangle(90, 160, W-180, 230, 4)
	dc.Fill()
	cardstyle.BracketCorners(dc, 90, 160, W-180, 230, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
	cardstyle.GlowText(dc, cardstyle.FtPS2P, 24, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), W/2, 240, cardstyle.HexA(0x40e0ff, 130), cardstyle.Hex(0x40e0ff), 0.5, 0.5, 740, 10)
	cardstyle.Text(dc, cardstyle.FtInter, 12, "OPERATOR: "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 296, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 640, 9)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtPS2P, 11, fmt.Sprintf("X%d", int(req.Amount)), W/2, 330, cardstyle.Hex(0xff9ecb), 0.5, 0.5, 140, 8)
	}
	cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(econCaption(req, req.Type), 74), W/2, 440, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 780, 9)
	cardstyle.Text(dc, cardstyle.FtPS2P, 8, "INSERT COIN TO CONTINUE", W/2, 560, cardstyle.HexA(0xff9ecb, 180), 0.5, 0.5, 320, 7)
	return dc.Image()
}

// ── Style 9 RUNE MONOLITH - carved tablet ────────────────────────────

func econStyle09(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.RuneMonolith()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Speckle(dc, 0, 0, W, H, 100, 0x9E11A8, cardstyle.N(0, 0, 0, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2.2, cardstyle.Darken(p.Panel, 40), false)
	cardstyle.GlyphCol(dc, 26, 50, H-50, 52, cardstyle.N(255, 255, 255, 13))
	cardstyle.GlyphCol(dc, W-38, 50, H-50, 52, cardstyle.N(255, 255, 255, 13))
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 27, strings.ToUpper(econTypeTitle(req.Type)), W/2, 76, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 700, 14)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 108, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 700, 9)
	dc.SetColor(cardstyle.Darken(p.Panel, 56))
	dc.SetLineWidth(3)
	dc.DrawLine(120, 130, W-120, 130)
	dc.Stroke()
	// recessed item niche
	dc.SetColor(cardstyle.Darken(p.Panel, 34))
	dc.DrawRoundedRectangle(150, 170, 700, 220, 8)
	dc.Fill()
	dc.SetColor(cardstyle.N(0, 0, 0, 110))
	dc.SetLineWidth(2)
	dc.DrawLine(150, 390, 150, 170)
	dc.DrawLine(150, 170, 850, 170)
	dc.Stroke()
	dc.SetColor(cardstyle.Lighten(p.Panel, 34))
	dc.SetLineWidth(1.4)
	dc.DrawLine(850, 170, 850, 390)
	dc.DrawLine(150, 390, 850, 390)
	dc.Stroke()
	cardstyle.EmberGlow(dc, W/2, 268, 90, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 40, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 24), W/2, 262, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 620, 18)
	if req.Amount > 1 {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 18, fmt.Sprintf("X%d", int(req.Amount)), W/2, 322, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 12)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.TruncateRunes(strings.ToUpper(econCaption(req, req.Type)), 76), W/2, 470, cardstyle.WithA(p.Ink, 215), 0.5, 0.5, 800, 9)
	cardstyle.EmberGlow(dc, 78, 548, 40, p.Accent)
	cardstyle.SealMini(dc, 78, 548, 19, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	return dc.Image()
}

// ── Style 10 CRIMSON COURT - court commission ────────────────────────

func econStyle10(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.CrimsonCourt()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Damask(dc, W, H, 76, cardstyle.HexA(0xd4a856, 20))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2.4, p.Accent, false)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 16, 1, cardstyle.WithA(p.Accent, 130), false)
	cardstyle.Cartouche(dc, W/2, 72, 520, 56, p.Panel, p.Accent, p.Ink, strings.ToUpper(econTypeTitle(req.Type)), cardstyle.FtCinzelDec, 22)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "COMMISSIONED TO "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 122, p.Muted, 0.5, 2.4)
	// shield with item
	cardstyle.Shield(dc, W/2, 300, 300, 240, p.Panel, p.Accent, 3)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), W/2, 276, p.Ink, 0.5, 0.5, 260, 14)
	if req.Amount > 1 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("X%d", int(req.Amount)), W/2, 318, p.Accent2, 0.5, 0.5, 120, 11)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(econCaption(req, req.Type)), 70), W/2, 470, cardstyle.WithA(p.Ink, 220), 0.5, 0.5, 820, 10)
	cardstyle.WaxSeal(dc, 80, 520, 30, p.Seal, cardstyle.Darken(p.Seal, 44), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 16)
	return dc.Image()
}

// DECREE (rank-up) per style - one composition per system.
func econDecree(req *TransactionCardRequest) image.Image {
	switch req.Style {
	case 6:
		return econDecreeSoulForge(req)
	}
	return nil // other systems: decree keeps the royal bake (it IS a decree)
}

func econDecreeSoulForge(req *TransactionCardRequest) image.Image {
	const W, H = 1000.0, 600.0
	dc := gg.NewContext(int(W), int(H))
	p := cardstyle.SoulForge()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.StarField(dc, 16, 16, W-16, H-16, 34, 0x50F6D0, cardstyle.N(255, 255, 255, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 11, 1.3, p.PanelEd, false)
	cardstyle.Diamond(dc, W/2, 11, 5.4, p.PanelEd)
	cardstyle.Diamond(dc, W/2, H-11, 5.4, p.PanelEd)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, "SOUL ASCENSION", W/2, 76, p.Accent, 0.5, 0.5, 620, 16)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "A RANK DECREE FOR "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 106, p.Muted, 0.5, 2.6)
	cardstyle.DotLeader(dc, W/2-110, W/2+110, 124, 1, cardstyle.WithA(p.Accent, 140))
	// old -> new on the magic circle
	cardstyle.MagicCircle(dc, W/2, 300, 104, p.Accent)
	ledger := cardstyle.Sanitize(req.Details)
	parts := strings.Split(ledger, "->")
	if len(parts) == 2 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 24, strings.ToUpper(cardstyle.Sanitize(strings.TrimSpace(parts[0]))), W/2-150, 296, p.Muted, 0.5, 0.5, 160, 12)
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, "TO", W/2, 262, p.Muted, 0.5, 0.5, 60, 9)
		cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 40, strings.ToUpper(cardstyle.Sanitize(strings.TrimSpace(parts[1]))), W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 200, 16)
	} else {
		cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 36, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 18)), W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 420, 14)
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 15, "the guild watches - keep rising", W/2, 452, p.Muted, 0.5, 0.5, 600, 10)
	cardstyle.Text(dc, cardstyle.FtMedieval, 14, "keep rising - the guild watches", W/2, H-30, p.Caption, 0.5, 0.5, 600, 9)
	return dc.Image()
}

// firstRuneOf - first uppercased rune of s.
func firstRuneOf(s string) string {
	r := []rune(strings.ToUpper(cardstyle.Sanitize(s)))
	if len(r) == 0 {
		return "?"
	}
	return string(r[:1])
}

// QAStyledEconomy exposes the styled economy renderer for the QA harness.
func QAStyledEconomy(req *TransactionCardRequest) image.Image {
	return renderStyledEconomy(req)
}
