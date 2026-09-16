package combat

// style09.go - RUNE MONOLITH "The Basalt Stele" (Style 9) - v3 REBUILD.
// EVERY portrait card is a carved stone STELE: a tall tapered monolith with
// a pointed apex standing on the basalt cavern floor. Content lives in
// horizontal REGISTERS carved into the stone, separated by glyph bands.
// Structural signatures (visible in silhouette, not color):
//   - the stele silhouette itself (pointed apex + tapered shoulders)
//   - register bands + glyph separators, engraved (dark inset + light offset)
//   - carved glyph columns inside the stele edges, ember glow in cuts
// Wide formats use the wide tablet-wall variant of the same stele.
// SKILLTREE keeps its pillar colonnade, EQUIP its niche wall (both already
// stele-kin); all other kinds are rebuilt around the monolith.

import (
	"fmt"
	"image"
	"math"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

func s09Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.RuneMonolith()
	cardstyle.VertGrad(dc, W, H, cardstyle.Darken(p.Bg, 26), cardstyle.Darken(p.Bg2, 34))
	cardstyle.Speckle(dc, 0, 0, W, H, 120, 0x9E11A7, cardstyle.N(0, 0, 0, 255))
	return &p
}

// s09Header - engraved title with carved band (wide formats).
func s09Header(dc *gg.Context, W, y float64, title, sub string, p *cardstyle.Palette) float64 {
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 27, strings.ToUpper(cardstyle.Sanitize(title)), W/2, y, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-130, 14)
	if sub != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(sub)), W/2, y+26, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-130, 8)
	}
	s09GlyphBand(dc, 70, W-70, y+46, p)
	return y + 68
}

// s09Recess - recessed carved panel.
func s09Recess(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) {
	dc.SetColor(cardstyle.Darken(p.Panel, 34))
	dc.DrawRoundedRectangle(x, y, w, h, 8)
	dc.Fill()
	dc.SetColor(cardstyle.N(0, 0, 0, 110))
	dc.SetLineWidth(2)
	dc.DrawLine(x, y+h, x, y)
	dc.DrawLine(x, y, x+w, y)
	dc.Stroke()
	dc.SetColor(cardstyle.Lighten(p.Panel, 34))
	dc.SetLineWidth(1.4)
	dc.DrawLine(x+w, y, x+w, y+h)
	dc.DrawLine(x, y+h, x+w, y+h)
	dc.Stroke()
}

// s09StelePath builds the stele silhouette (pointed apex, straight flanks).
func s09StelePath(dc *gg.Context, x, y, w, h, apex float64) {
	cx := x + w/2
	dc.ClearPath()
	dc.MoveTo(x, y+h)
	dc.LineTo(x, y+apex)
	dc.QuadraticTo(x, y+apex*0.30, cx, y)
	dc.QuadraticTo(x+w, y+apex*0.30, x+w, y+apex)
	dc.LineTo(x+w, y+h)
	dc.ClosePath()
}

// s09Stele draws the monolith and returns the inner content rect.
func s09Stele(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) (float64, float64, float64, float64) {
	apex := math.Min(120.0, w*0.22)
	// floor shadow
	dc.SetColor(cardstyle.N(0, 0, 0, 130))
	dc.DrawEllipse(x+w/2, y+h+8, w*0.72, 16)
	dc.Fill()

	s09StelePath(dc, x, y, w, h, apex)
	dc.SetColor(cardstyle.Darken(p.Panel, 22))
	dc.Fill()
	dc.SetColor(cardstyle.N(0, 0, 0, 90))
	dc.SetLineWidth(3)
	dc.Stroke()

	// stone tone variation inside the silhouette
	dc.Push()
	s09StelePath(dc, x, y, w, h, apex)
	dc.Clip()
	// vertical light: top lighter (moonlight), bottom darker
	steps := 28
	for i := 0; i < steps; i++ {
		f0 := float64(i) / float64(steps)
		c := cardstyle.Lighten(p.Panel, uint8(18*(1-f0)))
		dc.SetColor(c)
		dc.DrawRectangle(x, y+h*f0, w, h/float64(steps)+1)
		dc.Fill()
	}
	cardstyle.Speckle(dc, x, y, x+w, y+h, 60, 0x51B2E7, cardstyle.N(0, 0, 0, 220))
	// faint glyph columns inside the flanks
	cardstyle.GlyphCol(dc, x+22, y+apex+30, y+h-30, 54, cardstyle.N(255, 255, 255, 15))
	cardstyle.GlyphCol(dc, x+w-34, y+apex+30, y+h-30, 54, cardstyle.N(255, 255, 255, 15))
	dc.Pop()
	dc.ResetClip() // gg Pop() keeps the clip mask (found 2026-09-16)

	// inner carved groove following the silhouette
	in := 14.0
	s09StelePath(dc, x+in, y+in*0.8, w-in*2, h-in, apex*0.86)
	dc.SetColor(cardstyle.N(0, 0, 0, 120))
	dc.SetLineWidth(2.4)
	dc.Stroke()
	s09StelePath(dc, x+in+5, y+in*0.8+4, w-(in+5)*2, h-in-8, apex*0.82)
	dc.SetColor(cardstyle.Lighten(p.Panel, 26))
	dc.SetLineWidth(1)
	dc.Stroke()

	ix, iy := x+44, y+apex+26
	return ix, iy, w - 88, y + h - iy - 40
}

// s09GlyphBand - carved separator with ember ticks (register divider).
func s09GlyphBand(dc *gg.Context, x0, x1, y float64, p *cardstyle.Palette) {
	dc.SetColor(cardstyle.N(0, 0, 0, 130))
	dc.SetLineWidth(2.6)
	dc.DrawLine(x0, y, x1, y)
	dc.Stroke()
	dc.SetColor(cardstyle.Lighten(p.Panel, 30))
	dc.SetLineWidth(1)
	dc.DrawLine(x0, y+3, x1, y+3)
	dc.Stroke()
	dc.SetColor(p.Accent)
	for i := 0; i < 5; i++ {
		tx := (x0+x1)/2 - 96 + float64(i)*48
		dc.SetLineWidth(2)
		dc.DrawLine(tx, y-4, tx+7, y+4)
		dc.Stroke()
	}
}

// s09CarveRow - engraved ledger row with dot leader.
func s09CarveRow(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette, size int) {
	cardstyle.Engrave(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(label)), x, y, p.Ink, cardstyle.Darken(p.Panel, 60), 0, 0.5, w*0.55, 8)
	cardstyle.DotLeader(dc, x+w*0.58, x+w-cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(value)), 0)-14, y+1, 1, cardstyle.WithA(p.Muted, 100))
	cardstyle.Engrave(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(value)), x+w, y, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, w*0.4, 8)
}

// s09Distribute - evenly spaced positions filling [y0, y1].
func s09Distribute(n, y0, y1 float64) []float64 {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []float64{(y0 + y1) / 2}
	}
	step := (y1 - y0) / (n - 1)
	out := make([]float64, int(n))
	for i := range out {
		out[i] = y0 + step*float64(i)
	}
	return out
}

// s09RegTitle - engraved register heading with tiny ember diamonds.
func s09RegTitle(dc *gg.Context, cx, y float64, title string, p *cardstyle.Palette) {
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(title)), cx, y, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 400, 9)
	cardstyle.Diamond(dc, cx-118, y, 3.4, cardstyle.WithA(p.Accent, 170))
	cardstyle.Diamond(dc, cx+118, y, 3.4, cardstyle.WithA(p.Accent, 170))
}

// s09Notches - carved notch meter (chiselled grooves, ember fill).
func s09Notches(dc *gg.Context, cx, y, w float64, n, filled int, p *cardstyle.Palette) {
	if n < 1 {
		n = 1
	}
	gap := 10.0
	nw := (w - gap*float64(n-1)) / float64(n)
	x := cx - w/2
	for i := 0; i < n; i++ {
		dc.SetColor(cardstyle.N(0, 0, 0, 140))
		dc.DrawRectangle(x, y, nw, 12)
		dc.Fill()
		if i < filled {
			dc.SetColor(p.Accent)
			dc.DrawRectangle(x+1.5, y+1.5, nw-3, 9)
			dc.Fill()
			cardstyle.EmberGlow(dc, x+nw/2, y+6, 18, p.Accent)
		} else {
			dc.SetColor(cardstyle.Lighten(p.Panel, 22))
			dc.DrawRectangle(x, y+12, nw, 1.4)
			dc.Fill()
		}
		x += nw + gap
	}
}

// s09Footer - carved seal at the cavern floor (outside the stele).
func s09Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.EmberGlow(dc, 52, H-38, 38, p.Accent)
	cardstyle.SealMini(dc, 52, H-38, 17, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 10)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Caption)), 54), 86, H-38, cardstyle.WithA(p.Ink, 200), 0, 0.5, W-150, 9)
	}
}

func renderStyle09(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s09Rank(req).Image()
	case "ALLOCATE":
		return s09Allocate(req).Image()
	case "SKILLUP":
		return s09Skillup(req).Image()
	case "EVOLVE":
		return s09Evolve(req).Image()
	case "ABILITIES":
		return s09Abilities(req).Image()
	case "SKILLTREE":
		return s09Skilltree(req).Image()
	case "EQUIP":
		return s09Equip(req).Image()
	case "SHOP":
		return s09Shop(req).Image()
	case "GUILDINFO":
		return s09GuildInfo(req).Image()
	}
	return nil
}

// RANK - the stele of ascension: level carved in the apex register.
func s09Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)
	sx, sy, sw, _ := s09Stele(dc, 66, 46, W-132, H-130, p)

	// apex register: winged glyph + level numeral
	cardstyle.EmberGlow(dc, sx+sw/2, sy+64, 78, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 64, fmt.Sprintf("%d", req.Level), sx+sw/2, sy+58, cardstyle.Lighten(p.Ink, 16), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 130, 26)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "LEVEL", sx+sw/2, sy+104, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 8)
	// wing glyphs flanking the numeral
	dc.SetColor(cardstyle.WithA(p.Accent, 150))
	for sgn := -1.0; sgn <= 1; sgn += 2 {
		wx := sx + sw/2 + sgn*96
		for i := 0; i < 3; i++ {
			dc.SetLineWidth(2 - float64(i)*0.4)
			dc.DrawLine(wx, sy+52-float64(i)*12, wx+sgn*(34-float64(i)*8), sy+58-float64(i)*12)
			dc.Stroke()
		}
	}
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(rankLine, 22), sx+sw/2, sy+136, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 9)
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+156, p)

	// name + xp register (kept clear of the glyph band above)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(styledName(req))), sx+sw/2, sy+188, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-30, 13)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(req.XPNow))+" / "+strings.ToUpper(cardstyle.Sanitize(req.XPLeft)), sx+sw/2, sy+214, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 8)
	s09Notches(dc, sx+sw/2, sy+230, sw-70, 10, int(math.Round(float64(req.XPPercent)/100*10)), p)
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+258, p)

	// record register (distributed)
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	s09RegTitle(dc, sx+sw/2, sy+290, "THE RECORD", p)
	ys := s09Distribute(float64(len(rows)), sy+322, sy+488)
	for i, r := range rows {
		s09CarveRow(dc, sx, ys[i], sw, r.Label, r.Value, p, 14)
	}
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+518, p)

	// path register (anchored to the stele base)
	s09RegTitle(dc, sx+sw/2, sy+548, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), p)
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	ys2 := s09Distribute(float64(len(prog)), sy+574, sy+668)
	for i, pr := range prog {
		frac := 0.0
		if pr.Max > 0 {
			frac = float64(pr.Cur) / float64(pr.Max)
		}
		if pr.Done {
			frac = 1
		}
		s09CarveRow(dc, sx, ys2[i], sw, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), p, 12)
		s09Notches(dc, sx+sw/2, ys2[i]+16, sw-60, 8, int(math.Round(frac*8)), p)
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - the stele of offering: points in apex, seven stat sockets.
func s09Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)
	sx, sy, sw, _ := s09Stele(dc, 66, 46, W-132, H-130, p)

	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.EmberGlow(dc, sx+sw/2, sy+64, 78, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 64, fmt.Sprintf("%d", avail), sx+sw/2, sy+58, cardstyle.Lighten(p.Ink, 16), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 130, 26)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "UNSPENT", sx+sw/2, sy+104, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 8)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.Pill)), sx+sw/2, sy+136, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 9)
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+170, p)

	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), sx+sw/2, sy+202, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-30, 12)
	s09RegTitle(dc, sx+sw/2, sy+236, "THE SEVEN PATHS", p)

	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	ys := s09Distribute(float64(len(rows)), sy+278, sy+560)
	for i, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		s09CarveRow(dc, sx, ys[i], sw, cardstyle.StatFullName(code), gainFromSub(r.Sub), p, 15)
	}
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+592, p)

	spent := parseLeadingInt(req.SpentNow, 0)
	s09CarveRow(dc, sx, sy+628, sw, "SPENT", fmt.Sprintf("%d PTS", spent), p, 14)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), sx+sw/2, sy+660, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 8)
	ctaSafe := strings.NewReplacer("[", "(", "]", ")").Replace(cardstyle.Sanitize(req.CtaLabel))
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(ctaSafe), sx+sw/2, sy+694, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-30, 9)
	if req.CtaSub != "" {
		subSafe := strings.NewReplacer("[", "(", "]", ")").Replace(cardstyle.Sanitize(req.CtaSub))
		cardstyle.Text(dc, cardstyle.FtCinzel, 10, strings.ToUpper(cardstyle.TruncateRunes(subSafe, 52)), sx+sw/2, sy+718, cardstyle.WithA(p.Muted, 200), 0.5, 0.5, sw-30, 8)
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - the stele of the rite: initials in a carved ring, notch mastery.
func s09Skillup(req *portraitRequest) *gg.Context {
	skillTitle := strings.ToUpper(cardstyle.Sanitize(req.DocTitle))
	if skillTitle == "" {
		skillTitle = "MASTERY"
	}

	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)
	sx, sy, sw, _ := s09Stele(dc, 66, 46, W-132, H-130, p)

	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	// carved ring socket in apex register
	rcx, rcy := sx+sw/2, sy+80
	dc.SetColor(cardstyle.N(0, 0, 0, 120))
	dc.DrawCircle(rcx, rcy, 62)
	dc.Fill()
	cardstyle.EmberGlow(dc, rcx, rcy, 60, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 44, initials, rcx, rcy-2, cardstyle.Lighten(p.Ink, 16), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 100, 18)
	dc.SetColor(cardstyle.Lighten(p.Panel, 26))
	cardstyle.Ring(dc, rcx, rcy, 66, cardstyle.Lighten(p.Panel, 26), 1.4)
	cardstyle.Ring(dc, rcx, rcy, 58, cardstyle.N(0, 0, 0, 120), 1.6)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	if req.SubLabel != "" {
		tierLabel = strings.ToUpper(cardstyle.Sanitize(req.SubLabel))
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, tierLabel, rcx, rcy+92, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 160, 9)
	s09GlyphBand(dc, sx+10, sx+sw-10, sy+206, p)

	// skill name register
	words := strings.Split(strings.ToUpper(cardstyle.Sanitize(req.SkillName)), " ")
	wy := sy + 250.0
	for _, w := range words {
		cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 28, w, sx+sw/2, wy, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 15)
		wy += 44
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(styledName(req))), sx+sw/2, wy+2, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 8)
	s09GlyphBand(dc, sx+10, sx+sw-10, wy+34, p)

	// mastery register: carved notches
	maxPips := req.SkillMax
	if maxPips < 1 {
		maxPips = 1
	}
	if maxPips > 7 {
		maxPips = 7
	}
	cur := req.Cur
	if cur > maxPips {
		cur = maxPips
	}
	s09RegTitle(dc, sx+sw/2, wy+76, skillTitle, p)
	s09Notches(dc, sx+sw/2, wy+104, sw-70, maxPips, cur, p)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, fmt.Sprintf("%d / %d", cur, maxPips), sx+sw/2, wy+156, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 160, 13)
	s09GlyphBand(dc, sx+10, sx+sw-10, wy+206, p)

	// status register
	s09CarveRow(dc, sx, wy+244, sw, "STATUS", tierLabel, p, 14)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "THE BLADE REMEMBERS", sx+sw/2, wy+296, cardstyle.WithA(p.Muted, 210), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-40, 8)
	// base ornament: carved rayed diamond
	bcy := sy + 720.0
	cardstyle.EmberGlow(dc, sx+sw/2, bcy, 46, p.Accent)
	dc.SetColor(cardstyle.N(0, 0, 0, 120))
	cardstyle.DiamondOutline(dc, sx+sw/2, bcy, 40, cardstyle.Lighten(p.Panel, 30), 1.4)
	cardstyle.DiamondOutline(dc, sx+sw/2, bcy, 30, cardstyle.N(0, 0, 0, 130), 1.2)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 16, initialLetters(cardstyle.Sanitize(req.SkillName)), sx+sw/2, bcy-1, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 60, 9)
	s09Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - wide tablet wall: stacked carved slabs, registers fill height.
func s09Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	// wide stele (tablet wall)
	sx, sy, sw, _ := s09Stele(dc, 60, 40, W-120, H-100, p)

	cardstyle.EmberGlow(dc, sx+sw/2, sy+30, 64, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 28, strings.ToUpper(cardstyle.Sanitize(req.DocTitle)), sx+sw/2, sy+26, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, sw-160, 15)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, cardstyle.Sanitize(req.PageLabel), sx+sw-10, sy+24, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, 150, 9)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper("\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 96)+"\""), sx+sw/2, sy+54, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-120, 9)
	}
	s09GlyphBand(dc, sx+12, sx+sw-12, sy+84, p)

	groups := req.Groups
	if len(groups) > 6 {
		groups = groups[:6]
	}
	totalRows := 0
	for _, g := range groups {
		totalRows += len(g.Items)
	}
	if totalRows == 0 {
		totalRows = 1
	}
	// slab register per group; rows distributed inside each slab
	y := sy + 116.0
	avail := (H - 170) - y
	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	for _, g := range groups {
		gRows := float64(len(g.Items))
		if gRows < 1 {
			gRows = 1
		}
		// slab height proportional to rows, min 130
		slabH := avail * (gRows / float64(totalRows))
		if slabH < 120 {
			slabH = 120
		}
		// carved slab
		dc.SetColor(cardstyle.Darken(p.Panel, 40))
		dc.DrawRoundedRectangle(sx, y, sw, slabH-14, 8)
		dc.Fill()
		dc.SetColor(cardstyle.N(0, 0, 0, 110))
		dc.SetLineWidth(2)
		dc.DrawLine(sx, y, sx+sw, y)
		dc.DrawLine(sx, y, sx, y+slabH-14)
		dc.Stroke()
		dc.SetColor(cardstyle.Lighten(p.Panel, 30))
		dc.SetLineWidth(1.2)
		dc.DrawLine(sx+sw, y, sx+sw, y+slabH-14)
		dc.Stroke()
		// group title engraved in slab
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(g.Name)), sx+24, y+30, p.Muted, cardstyle.Darken(p.Panel, 60), 0, 0.5, sw*0.5, 9)
		if g.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(g.Sub), 30)), sx+24, y+50, cardstyle.WithA(p.Muted, 170), 0, 0.5, sw*0.5, 8)
		}
		ys := s09Distribute(gRows, y+72, y+slabH-58)
		for ii, it := range g.Items {
			iy := ys[ii]
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("%03d", num), sx+24, iy, p.Accent, cardstyle.Darken(p.Panel, 60), 0, 0.5, 56, 8)
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 16, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Title)), 24), sx+92, iy, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0, 0.5, sw-300, 10)
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Value)), 12), sx+sw-24, iy, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, 130, 9)
			extra := it.Sub
			if it.Runes != "" {
				extra = strings.ToUpper(cardstyle.Sanitize(it.Runes)) + "  " + extra
			}
			if extra != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 80), sx+92, iy+20, cardstyle.WithA(p.Muted, 190), 0, 0.5, sw-130, 8)
			}
			num++
		}
		y += slabH
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - the colonnade of pillars (kept: it is stele-kin and unique).
func s09Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1200.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	s09Header(dc, W, 50, "THE PILLARS OF MASTERY", strings.ToUpper(cardstyle.Sanitize(req.ClassName)), p)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, ptsText, W-90, 66, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, 180, 9)

	groundY := H - 64.0
	// plinth
	dc.SetColor(cardstyle.Darken(p.Panel, 44))
	dc.DrawRectangle(80, groundY, W-160, 34)
	dc.Fill()
	dc.SetColor(cardstyle.N(0, 0, 0, 110))
	dc.SetLineWidth(2)
	dc.DrawLine(80, groundY, W-80, groundY)
	dc.Stroke()

	n := len(req.Branches)
	if n == 0 {
		n = 1
	}
	maxTier := 1
	for _, br := range req.Branches {
		for _, s := range br.Skills {
			if s.Tier > maxTier {
				maxTier = s.Tier
			}
		}
	}
	if maxTier > 5 {
		maxTier = 5
	}
	tierH := 150.0
	colW := (W - 200) / float64(n)
	for bi, br := range req.Branches {
		cx := 100 + float64(bi)*colW + colW/2
		shaftTop := groundY - float64(maxTier)*tierH - 60
		dc.SetColor(cardstyle.Lighten(p.Panel, 8))
		dc.DrawRectangle(cx-34, shaftTop, 68, groundY-shaftTop)
		dc.Fill()
		dc.SetColor(cardstyle.N(0, 0, 0, 110))
		dc.SetLineWidth(2)
		dc.DrawLine(cx-34, shaftTop, cx-34, groundY)
		dc.Stroke()
		dc.DrawLine(cx+34, shaftTop, cx+34, groundY)
		dc.Stroke()
		dc.SetColor(cardstyle.Lighten(p.Panel, 24))
		dc.DrawRectangle(cx-46, shaftTop-16, 92, 18)
		dc.Fill()
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(br.Name), 12)), cx, shaftTop-34, p.Ink, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, colW-14, 9)
		order := []int{}
		nodesByTier := map[int][]portraitNode{}
		for _, s := range br.Skills {
			t := s.Tier
			if t < 1 {
				t = 1
			}
			if t > maxTier {
				t = maxTier
			}
			if _, ok := nodesByTier[t]; !ok {
				order = append(order, t)
			}
			nodesByTier[t] = append(nodesByTier[t], s)
		}
		for k, t := range order {
			list := nodesByTier[t]
			sy := groundY - 60 - float64(k)*tierH
			for j, s := range list {
				nx := cx + (float64(j)-float64(len(list)-1)/2)*74
				lit := s.State == "learned" || s.State == "maxed"
				dc.SetColor(cardstyle.Darken(p.Panel, 50))
				dc.DrawCircle(nx, sy, 26)
				dc.Fill()
				dc.SetColor(cardstyle.N(0, 0, 0, 100))
				dc.SetLineWidth(2)
				dc.DrawCircle(nx, sy, 26)
				dc.Stroke()
				if lit {
					cardstyle.EmberGlow(dc, nx, sy, 44, p.Accent)
				}
				glyphCol := cardstyle.WithA(p.Muted, 130)
				if lit {
					glyphCol = p.Accent2
				}
				if s.State == "maxed" {
					glyphCol = cardstyle.Hex(0xffdca0)
					cardstyle.Ring(dc, nx, sy, 30, cardstyle.HexA(0xffdca0, 150), 1.4)
				}
				dc.SetColor(glyphCol)
				dc.SetLineWidth(2.4)
				dc.DrawLine(nx-7, sy-9, nx+7, sy-9)
				dc.Stroke()
				dc.DrawLine(nx+7, sy-9, nx-7, sy)
				dc.Stroke()
				dc.DrawLine(nx-7, sy, nx+7, sy+9)
				dc.Stroke()
				// 2026-09-17: node labels used maxW=colW-10 (up to ~990px) while
				// adjacent tier-mates sit only 74px apart - names collided into
				// garble ("WAR MARCHSERENADE OF"). Cap each label to the 148px
				// even-slot pitch and stagger odd nodes below the medallion so
				// neighbours can never overlap horizontally or vertically.
				lblY, lblW := sy-44, 140.0
				if j%2 == 1 {
					lblY = sy + 58
				}
				if lblW > colW-10 {
					lblW = colW - 10
				}
				cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 14), nx, lblY, p.Ink, 0.5, 0.5, lblW, 8)
				cardstyle.Text(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), nx, sy+40, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, 70, 7)
			}
		}
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// EQUIP - relic vault wall of carved niches (kept: stele-kin, unique).
func s09Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 44, "THE RELIC ASCENT", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

	floorY := H - 92.0
	// grand plinth floor
	dc.SetColor(cardstyle.Darken(p.Panel, 46))
	dc.DrawRectangle(60, floorY, W-120, 26)
	dc.Fill()
	dc.SetColor(cardstyle.N(0, 0, 0, 110))
	dc.SetLineWidth(2)
	dc.DrawLine(60, floorY, W-60, floorY)
	dc.Stroke()

	// nine carved steps ascending left -> right; relics rest on each tread
	stepW, stepGap := 128.0, 10.0
	x0 := 76.0
	baseTop := floorY - 54.0
	rise := 58.0
	for i, s := range req.Slots {
		cx := x0 + float64(i)*(stepW+stepGap) + stepW/2
		treadY := baseTop - float64(i)*rise
		// riser column down to the floor
		dc.SetColor(cardstyle.Lighten(p.Panel, 6))
		dc.DrawRectangle(cx-stepW/2, treadY, stepW, floorY-treadY)
		dc.Fill()
		dc.SetColor(cardstyle.N(0, 0, 0, 110))
		dc.SetLineWidth(2)
		dc.DrawLine(cx-stepW/2, treadY, cx-stepW/2, floorY)
		dc.Stroke()
		dc.DrawLine(cx+stepW/2, treadY, cx+stepW/2, floorY)
		dc.Stroke()
		// tread highlight
		dc.SetColor(cardstyle.Lighten(p.Panel, 26))
		dc.DrawRectangle(cx-stepW/2, treadY, stepW, 6)
		dc.Fill()
		empty := s.Empty
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, slotCode, cx, treadY-96, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, stepW-6, 7)
		name := "EMPTY"
		ncol := cardstyle.WithA(p.Muted, 140)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = cardstyle.Lighten(p.Ink, 10)
		}
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(name), 14), cx, treadY-70, ncol, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, stepW-2, 8)
		// the relic glyph brazier set into the tread
		if empty {
			cardstyle.Ring(dc, cx, treadY-30, 16, cardstyle.WithA(p.Muted, 110), 1.4)
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "0/0", cx, treadY-4, cardstyle.WithA(p.Muted, 130), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 60, 7)
		} else {
			cardstyle.EmberGlow(dc, cx, treadY-32, 30, p.Accent)
			cardstyle.Ring(dc, cx, treadY-30, 16, cardstyle.WithA(p.Accent, 170), 1.6)
			dc.SetColor(p.Accent2)
			dc.SetLineWidth(2.4)
			dc.DrawLine(cx-6, treadY-37, cx+6, treadY-37)
			dc.Stroke()
			dc.DrawLine(cx+6, treadY-37, cx-6, treadY-30)
			dc.Stroke()
			dc.DrawLine(cx-6, treadY-30, cx+6, treadY-23)
			dc.Stroke()
			if s.DurMax > 0 {
				frac := s.Dur / s.DurMax
				if frac > 1 {
					frac = 1
				}
				cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, cardstyle.FmtF(s.Dur)+"/"+cardstyle.FmtF(s.DurMax), cx, treadY-4, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 76, 7)
				s09Notches(dc, cx, treadY+16, stepW-26, 5, int(math.Round(frac*5)), p)
			}
		}
		// tier ember ticks above the name
		for t := 0; t < s.Tier && t < 5; t++ {
			dc.SetColor(p.Accent)
			dc.SetLineWidth(2)
			dc.DrawLine(cx-20+float64(t)*11, treadY-116, cx-14+float64(t)*11, treadY-108)
			dc.Stroke()
		}
	}

	// summit plateau: the hero statue watches from the top of the ascent
	sumX := x0 + 9*(stepW+stepGap) + 40.0
	s09Recess(dc, sumX, y+30, W-60-sumX, floorY-(y+30), p)
	drawHeroFitted(dc, req, (sumX+W-60)/2, y+300, 230, 280, *p)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 20, strings.ToUpper(cardstyle.Sanitize(styledName(req))), (sumX+W-60)/2, y+610, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-60-sumX-20, 11)
	if req.SealText != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", (sumX+W-60)/2, y+644, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 200, 9)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Caption)), 44), (sumX+W-60)/2, y+686, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, W-60-sumX-24, 8)
	s09Footer(dc, W, H, req, p)
	return dc
}

// SHOP - the exchange stele: registers distributed down the stone.
func s09Shop(req *portraitRequest) *gg.Context {
	entries := req.Entries
	if len(entries) > 9 {
		entries = entries[:9]
	}
	n := len(entries)
	if n < 1 {
		n = 1
	}
	const W = 800.0
	H := 200.0 + float64(n)*96.0 + 150.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)
	sx, sy, sw, _ := s09Stele(dc, 74, 44, W-148, H-116, p)

	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, "THE EXCHANGE", sx+sw/2, sy+40, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, sw-120, 14)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), sx+sw/2, sy+70, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-100, 9)
	s09GlyphBand(dc, sx+12, sx+sw-12, sy+96, p)

	ys := s09Distribute(float64(n), sy+140, H-200)
	for i, e := range entries {
		s09CarveRow(dc, sx, ys[i], sw, e.Title, e.Value, p, 16)
		extra := e.Sub
		if e.Runes != "" {
			extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
		}
		if extra != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 84), sx, ys[i]+22, cardstyle.WithA(p.Muted, 200), 0, 0.5, sw, 8)
		}
		if i < n-1 {
			s09GlyphBand(dc, sx+30, sx+sw-30, ys[i]+58, p)
		}
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - the charter stele: crest register + charter register.
func s09GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 860.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)
	sx, sy, sw, _ := s09Stele(dc, 80, 42, W-160, H-110, p)

	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 30, strings.ToUpper(cardstyle.Sanitize(styledName(req))), sx+sw/2, sy+34, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, sw-90, 16)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "GUILD STONE", sx+sw/2, sy+62, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 200, 9)
	s09GlyphBand(dc, sx+12, sx+sw-12, sy+88, p)

	// crest register
	ccy := sy + 180.0
	dc.SetColor(cardstyle.N(0, 0, 0, 120))
	dc.DrawCircle(sx+sw/2, ccy, 74)
	dc.Fill()
	cardstyle.EmberGlow(dc, sx+sw/2, ccy, 70, p.Accent)
	if req.EmblemImg != nil {
		dc.Push()
		dc.DrawCircle(sx+sw/2, ccy, 60)
		dc.Clip()
		cardstyle.FitEmblem(dc, req.EmblemImg, sx+sw/2, ccy, 106, 106)
		dc.Pop()
		dc.ResetClip() // gg Pop() keeps the clip mask (found 2026-09-16)
	} else {
		cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 56, firstRuneUpper(styledName(req)), sx+sw/2, ccy-2, cardstyle.Lighten(p.Ink, 16), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 120, 22)
	}
	if req.Motto != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "\""+cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Motto)), 56)+"\"", sx+sw/2, ccy+104, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, sw-70, 9)
	}
	s09GlyphBand(dc, sx+12, sx+sw-12, ccy+134, p)

	// charter register
	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	ys := s09Distribute(float64(len(rows)+1), ccy+180, ccy+400)
	for i, r := range rows {
		s09CarveRow(dc, sx, ys[i], sw, r.Label, r.Value, p, 14)
	}
	ly := ys[len(rows)]
	s09CarveRow(dc, sx, ly, sw, "LEVEL", fmt.Sprintf("%d", req.Level), p, 15)
	pct := float64(req.XPPercent) / 100
	s09Notches(dc, sx+sw/2, ly+18, sw-80, 10, int(math.Round(pct*10)), p)
	s09GlyphBand(dc, sx+12, sx+sw-12, ly+42, p)

	// holdings register
	bs := req.Buildings
	if len(bs) > 3 {
		bs = bs[:3]
	}
	ys2 := s09Distribute(float64(len(bs)), ly+70, ly+128)
	for i, b := range bs {
		s09CarveRow(dc, sx, ys2[i], sw, b.Name, fmt.Sprintf("LVL %d", b.Level), p, 13)
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// s09Evolve - class ascension card. Same design language as this style's
// SKILLUP (theme-internal reuse with variation): own base, frame and
// medallion, with the ceremony's own title and the new-class identity.
func s09Evolve(req *portraitRequest) *gg.Context {
	e := *req
	e.DocTitle = "ASCENSION"
	if e.SubLabel == "" {
		e.SubLabel = "EVOLVED"
	}
	return s09Skillup(&e)
}
