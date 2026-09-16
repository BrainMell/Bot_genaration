package combat

// style02.go - GOLDEN ARCANUM "Ritual of Ascension" (Style 2).
// Everything is organized by CONCENTRIC RITUAL CIRCLES: a great circle as
// the structural skeleton, orbital bands for sections, gold hairline doubles,
// glowing sigil numerals, translucent indigo panels, diamond markers.
// Recognizable by structure: circles and orbital bands - never slabs or bills.

import (
	"fmt"
	"image"
	"math"
	"strings"

	"image-service/pkg/cardstyle"
	"image-service/pkg/utils"

	"github.com/fogleman/gg"
)

func s02Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.GoldenArcanum()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.StarField(dc, 0, 0, W, H, 26, 0xAC2337, cardstyle.N(232, 200, 120, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 14, 1.2, cardstyle.WithA(p.Accent, 170), true)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 21, 0.6, cardstyle.WithA(p.Accent, 110), false)
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s02GreatCircle - the structural ritual circle with tick marks.
func s02GreatCircle(dc *gg.Context, cx, cy, r float64, p *cardstyle.Palette) {
	cardstyle.MagicCircle(dc, cx, cy, r, p.Accent)
	cardstyle.Ring(dc, cx, cy, r*0.46, cardstyle.WithA(p.Accent, 60), 0.8)
	for i := 0; i < 12; i++ {
		a := float64(i) * math.Pi / 6
		cardstyle.Diamond(dc, cx+r*0.64*math.Cos(a), cy+r*0.64*math.Sin(a), 3, cardstyle.WithA(p.Accent, 90))
	}
}

// s02Band - translucent orbital band with hairline edges.
func s02Band(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) {
	dc.SetColor(p.Panel)
	dc.DrawRoundedRectangle(x, y, w, h, 10)
	dc.Fill()
	dc.SetColor(cardstyle.WithA(p.Accent, 150))
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x+6, y+6, w-12, h-12, 7)
	dc.Stroke()
	dc.SetLineWidth(0.6)
	dc.DrawRoundedRectangle(x+10, y+10, w-20, h-20, 5)
	dc.Stroke()
}

func s02SectionHead(dc *gg.Context, cx, y float64, text string, p *cardstyle.Palette) {
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(text)), cx, y, p.Accent, 0.5, 3)
	cardstyle.DotLeader(dc, cx-90, cx-14, y, 0.9, cardstyle.WithA(p.Accent, 110))
	cardstyle.DotLeader(dc, cx+14, cx+90, y, 0.9, cardstyle.WithA(p.Accent, 110))
	cardstyle.Diamond(dc, cx, y, 3, p.Accent)
}

func s02Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.SealMini(dc, 52, H-42, 19, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 58), 88, H-42, p.Muted, 0, 0.5, W-150, 9)
	}
}

func renderStyle02(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s02Rank(req).Image()
	case "ALLOCATE":
		return s02Allocate(req).Image()
	case "SKILLUP":
		return s02Skillup(req).Image()
	case "EVOLVE":
		return s02Evolve(req).Image()
	case "ABILITIES":
		return s02Abilities(req).Image()
	case "SKILLTREE":
		return s02Skilltree(req).Image()
	case "EQUIP":
		return s02Equip(req).Image()
	case "SHOP":
		return s02Shop(req).Image()
	case "GUILDINFO":
		return s02GuildInfo(req).Image()
	}
	return nil
}

// RANK - level sigil in the great circle, orbital bands for sections.
// s02Filler - constellation dial ornament to close short pages.
func s02Filler(dc *gg.Context, W float64, y0, y1 float64, p *cardstyle.Palette, label string) {
	if y1-y0 < 70 {
		return
	}
	cy := (y0 + y1) / 2
	// small central dial with orbiting diamonds
	cardstyle.Ring(dc, W/2, cy, 26, cardstyle.WithA(p.Accent, 140), 1.1)
	cardstyle.Ring(dc, W/2, cy, 44, cardstyle.WithA(p.Accent, 70), 0.7)
	cardstyle.Diamond(dc, W/2, cy, 5, p.Accent)
	dc.SetColor(cardstyle.WithA(p.Accent, 90))
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		cardstyle.Diamond(dc, W/2+44*math.Cos(a), cy+44*math.Sin(a), 2.2, cardstyle.WithA(p.Accent, 110))
	}
	// hairlines out to the flanks
	dc.SetLineWidth(0.7)
	dc.DrawLine(W/2-170, cy, W/2-56, cy)
	dc.Stroke()
	dc.DrawLine(W/2+56, cy, W/2+170, cy)
	dc.Stroke()
	if label != "" {
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, strings.ToUpper(cardstyle.Sanitize(label)), W/2, cy+72, cardstyle.WithA(p.Muted, 150), 0.5, 3)
	}
}

// RANK - the astrolabe of ascension: level hub, standing stats as radial
// spokes, XP as an arc around the rim, path ahead in orbital slots.
func s02Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 50, p.Muted, 0.5, 3)

	cx, cy, R := W/2, 292.0, 150.0
	s02GreatCircle(dc, cx, cy, R, p)
	// outer double rim
	cardstyle.Ring(dc, cx, cy, R+14, cardstyle.WithA(p.Accent, 90), 0.8)
	cardstyle.Ring(dc, cx, cy, R+20, cardstyle.WithA(p.Accent, 50), 0.6)

	// XP arc around the rim (from top, clockwise)
	pct := float64(req.XPPercent) / 100
	dc.SetLineWidth(5)
	dc.SetColor(cardstyle.WithA(cardstyle.Hex(0xffe9a8), 200))
	dc.DrawArc(cx, cy, R+14, -math.Pi/2, -math.Pi/2+pct*2*math.Pi)
	dc.Stroke()
	dc.SetLineWidth(1)
	dc.SetColor(cardstyle.WithA(cardstyle.Hex(0xffe9a8), 80))
	dc.DrawArc(cx, cy, R+22, -math.Pi/2, -math.Pi/2+pct*2*math.Pi)
	dc.Stroke()

	// hub: level
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 56, fmt.Sprintf("%d", req.Level), cx, cy-6, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 130, 24)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "LEVEL", cx, cy+34, p.Muted, 0.5, 4)

	// standing stats as radial spokes over the lower arc
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	n := len(rows)
	for i, r := range rows {
		a := math.Pi*0.10 + math.Pi*0.80*float64(i)/math.Max(float64(n-1), 1)
		sx, sy := cx+62*math.Cos(a), cy+62*math.Sin(a)
		ex, ey := cx+R*math.Cos(a), cy+R*math.Sin(a)
		dc.SetColor(cardstyle.WithA(p.Accent, 130))
		dc.SetLineWidth(1)
		dc.DrawLine(sx, sy, ex, ey)
		dc.Stroke()
		cardstyle.Diamond(dc, ex, ey, 4.4, p.Accent)
		// label outside the rim
		lx, ly := cx+(R+44)*math.Cos(a), cy+(R+44)*math.Sin(a)
		if ly > 448 {
			ly = 448
		}
		if ly < cy-10 {
			ly = cy - 10
		}
		align := 0.5
		tx := lx
		if math.Cos(a) > 0.35 {
			align = 0
		} else if math.Cos(a) < -0.35 {
			align = 1
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(r.Label)), tx, ly, p.Muted, align, 0.5, 150, 8)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), tx, ly+20, p.Accent2, align, 0.5, 150, 9)
		// small glow tick just inside the rim
		cardstyle.Diamond(dc, cx+(R-40)*math.Cos(a), cy+(R-40)*math.Sin(a), 2.2, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 130))
	}

	// xp readout + rank chip
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.XPNow))+" / "+strings.ToUpper(cardstyle.Sanitize(req.XPLeft)), cx, 524, p.Muted, 0.5, 0.5, 260, 9)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Chip(dc, W/2-130, 546, 260, 30, rankLine, cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 13)
	}

	// THE PATH AHEAD band (orbital slots)
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	py := 600.0
	if len(prog) > 0 {
		ph := 56 + float64(len(prog))*48 + 6
		s02Band(dc, 52, py, W-104, ph, p)
		s02SectionHead(dc, W/2, py+26, cardstyle.Sanitize(req.ProgressTitle), p)
		ry := py + 56
		for _, pr := range prog {
			frac := 0.0
			if pr.Max > 0 {
				frac = float64(pr.Cur) / float64(pr.Max)
			}
			if pr.Done {
				frac = 1
			}
			fill := p.Accent
			if pr.Done {
				fill = p.Done
			}
			cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(pr.Label)), 74, ry, p.Ink, 0, 0.5, 240, 9)
			cardstyle.Text(dc, cardstyle.FtCinzel, 13, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), W-74, ry, p.Accent2, 1, 0.5, 170, 9)
			cardstyle.MeterBar(dc, 74, ry+12, W-148, 8, frac, p.Track, fill, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 110), cardstyle.WithA(p.Accent, 120), 4)
			ry += 48
		}
		py += 56 + float64(len(prog))*48 + 24
	}
	// base ornament: small orbit dial
	s02Filler(dc, W, py+6, H-90, p, "THE GUILD OBSERVATORY")
	s02Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - the ritual heptagon: unspent in the hub, seven stats as nodes.
func s02Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "RITUAL OF ALLOCATION", W/2, 50, p.Muted, 0.5, 3.4)

	cx, cy, R := W/2, 330.0, 152.0
	s02GreatCircle(dc, cx, cy, R-24, p)
	cardstyle.Ring(dc, cx, cy, R, cardstyle.WithA(p.Accent, 120), 0.9)

	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 56, fmt.Sprintf("%d", avail), cx, cy-6, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 120, 22)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "UNSPENT", cx, cy+34, p.Muted, 0.5, 4)

	// heptagon of the seven stats
	codes := map[string]string{"HP": "HP", "ATK": "AT", "DEF": "DE", "MAG": "MA", "SPD": "SP", "LUCK": "LU", "CRIT": "CR"}
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	pts := [][2]float64{}
	for i := range rows {
		a := -math.Pi/2 + float64(i)*2*math.Pi/7
		pts = append(pts, [2]float64{cx + R*math.Cos(a), cy + R*math.Sin(a)})
	}
	// connective heptagon
	dc.SetColor(cardstyle.WithA(p.Accent, 110))
	dc.SetLineWidth(1)
	for i := range pts {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		dc.DrawLine(a[0], a[1], b[0], b[1])
		dc.Stroke()
	}
	for i, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		nx, ny := pts[i][0], pts[i][1]
		// spoke from hub
		dc.SetColor(cardstyle.WithA(p.Accent, 70))
		dc.SetLineWidth(0.8)
		dc.DrawLine(cx, cy, nx, ny)
		dc.Stroke()
		// node socket
		dc.SetColor(cardstyle.WithA(p.Track, 235))
		dc.DrawCircle(nx, ny, 25)
		dc.Fill()
		cardstyle.Ring(dc, nx, ny, 25, p.Accent, 1.4)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, codes[code], nx, ny-1, p.Ink, 0.5, 0.5, 44, 9)
		// name + gain outside the heptagon
		ox, oy := cx+(R+52)*math.Cos(-math.Pi/2+float64(i)*2*math.Pi/7), cy+(R+52)*math.Sin(-math.Pi/2+float64(i)*2*math.Pi/7)
		align := 0.5
		if math.Cos(-math.Pi/2+float64(i)*2*math.Pi/7) > 0.35 {
			align = 0
		} else if math.Cos(-math.Pi/2+float64(i)*2*math.Pi/7) < -0.35 {
			align = 1
		}
		oy = math.Max(30, math.Min(oy, 590))
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(cardstyle.StatFullName(code))), ox, oy, p.Muted, align, 0.5, 130, 9)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, gainFromSub(r.Sub), ox, oy+20, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 200), align, 0.5, 110, 9)
	}

	// class line + spending ledger band
	classLine := strings.ToUpper(cardstyle.Sanitize(req.Pill))
	if strings.Contains(strings.ToLower(classLine), "starter") && !strings.Contains(strings.ToLower(classLine), "tier") {
		classLine += " TIER"
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, classLine, W/2, 598, p.Ink, 0.5, 3)
	by := 630.0
	s02Band(dc, 52, by, W-104, 170, p)
	spent := parseLeadingInt(req.SpentNow, 0)
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), 84, by+40, p.Muted, 0, 0.5, 280, 9)
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("SPENT %d PTS", spent), W-84, by+40, p.Accent2, 1, 0.5, 220, 10)
	cardstyle.MeterBar(dc, 84, by+58, W-168, 9, 0, p.Track, p.Accent, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 110), cardstyle.WithA(p.Accent, 130), 4)
	if req.CtaLabel != "" {
		cardstyle.Chip(dc, W/2-170, by+92, 340, 34, strings.ToUpper(cardstyle.Sanitize(req.CtaLabel)), cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 12)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 10, "HALF VALUE PER POINT AFTER HEAVY SINGLE-STAT INVESTMENT", 84, by+144, cardstyle.WithA(p.Muted, 180), 0, 0.5, W-168, 8)
	s02Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - the mastery seal: initials in the triangle, mastery as a
// progress arc around the seal, orbital pips, status orbit.
func s02Skillup(req *portraitRequest) *gg.Context {
	skillTitle := strings.ToUpper(cardstyle.Sanitize(req.DocTitle))
	if skillTitle == "" {
		skillTitle = "RITUAL OF MASTERY"
	}

	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, skillTitle, W/2, 50, p.Muted, 0.5, 3.4)

	cx, cy, R := W/2, 292.0, 128.0
	s02GreatCircle(dc, cx, cy, R, p)
	// triangle inscribed
	dc.SetColor(cardstyle.WithA(p.Accent, 110))
	dc.SetLineWidth(1.2)
	for i := 0; i <= 3; i++ {
		a := -math.Pi/2 + float64(i)*2*math.Pi/3
		x, y := cx+96*math.Cos(a), cy+96*math.Sin(a)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 54, initials, cx, cy-8, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 130, 22)

	// mastery ARC around the seal
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
	frac := 0.0
	if maxPips > 0 {
		frac = float64(cur) / float64(maxPips)
	}
	dc.SetLineWidth(6)
	dc.SetColor(cardstyle.WithA(cardstyle.Hex(0xffe9a8), 210))
	dc.DrawArc(cx, cy, R+18, -math.Pi/2, -math.Pi/2+frac*2*math.Pi)
	dc.Stroke()
	dc.SetLineWidth(1)
	dc.SetColor(cardstyle.WithA(p.Accent, 110))
	dc.DrawArc(cx, cy, R+18, -math.Pi/2+frac*2*math.Pi, -math.Pi/2+2*math.Pi)
	dc.Stroke()
	// mastery notches on the arc
	for i := 0; i < maxPips; i++ {
		a := -math.Pi/2 + (float64(i)+0.5)*2*math.Pi/float64(maxPips)
		mx, my := cx+(R+18)*math.Cos(a), cy+(R+18)*math.Sin(a)
		if i < cur {
			cardstyle.Diamond(dc, mx, my, 4, cardstyle.Hex(0xffe9a8))
		} else {
			cardstyle.DiamondOutline(dc, mx, my, 3, cardstyle.WithA(p.Muted, 130), 1)
		}
	}

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 25, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, 476, p.Accent, 0.5, 0.5, W-90, 12)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	if req.SubLabel != "" {
		tierLabel = strings.ToUpper(cardstyle.Sanitize(req.SubLabel))
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, tierLabel, W/2, 508, p.Muted, 0.5, 3)

	// status orbit slots
	s02Band(dc, 52, 570, W-104, 160, p)
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, "MASTERY", 84, 610, p.Muted, 0, 0.5, 200, 9)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 18, fmt.Sprintf("%d / %d", cur, maxPips), W-84, 610, cardstyle.Hex(0xffe9a8), 1, 0.5, 140, 11)
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, "STATUS", 84, 662, p.Muted, 0, 0.5, 200, 9)
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(tierLabel), W-84, 662, p.Accent2, 1, 0.5, 200, 9)
	// orbit dial ornament
	s02Filler(dc, W, 754, H-90, p, "THE BLADE REMEMBERS")
	s02Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - illuminated folio with sigil bullets.
func s02Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	title := strings.TrimSpace(req.DocTitle)
	if title == "" {
		title = "ABILITY CODEX"
	}
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, cardstyle.Sanitize(title), W/2, 60, p.Accent, 0.5, 0.5, W-220, 16)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 78)+"\"", W/2, 94, p.Muted, 0.5, 0.5, W-160, 9)
	}
	if req.PageLabel != "" {
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-56, 44, p.Accent2, 1, 2)
	}

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	y := 136.0
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		gh := 52 + float64(len(g.Items))*56 + 6
		s02Band(dc, 48, y, W-96, gh, p)
		s02SectionHead(dc, W/2, y+26, g.Name, p)
		iy := y + 56
		for _, it := range g.Items {
			cardstyle.Diamond(dc, 78, iy-4, 6, p.Accent)
			cardstyle.Ring(dc, 78, iy-4, 10, cardstyle.WithA(p.Accent, 90), 0.9)
			cardstyle.Text(dc, cardstyle.FtCinzel, 18, cardstyle.Sanitize(it.Title), 104, iy-6, p.Ink, 0, 0.5, 420, 11)
			if it.Sub != "" {
				cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(it.Sub), 62), 104, iy+18, p.Muted, 0, 0.5, 420, 8)
			}
			if it.Runes != "" {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 15, cardstyle.Sanitize(it.Runes), W-88, iy-6, p.Accent2, 1, 2.6)
			}
			if it.Value != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(it.Value), W-88, iy+18, p.Muted, 1, 0.5, 220, 9)
			}
			num++
			iy += 56
		}
		y += gh + 14
	}
	s02Filler(dc, W, y+10, H-90, p, "THE ARCANUM CHARTS")
	s02Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - RADIAL MANDALA (root center, spokes = branches, rings = tiers).
func s02Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1000.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 28, "THE ARCANE CONSTELLATION", W/2, 54, p.Accent, 0.5, 0.5, W-220, 14)
	classLine := strings.ToUpper(cardstyle.Sanitize(req.ClassName))
	if req.Level > 0 {
		classLine = fmt.Sprintf("%s - LV %d", classLine, req.Level)
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, classLine, W/2, 86, p.Muted, 0.5, 2.6)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Chip(dc, W-230, 40, 180, 32, ptsText, cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 12)

	cx, cy := W/2, 545.0
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
	if maxTier > 4 {
		maxTier = 4
	}
	// rings
	for t := 1; t <= maxTier; t++ {
		r := 100 + float64(t-1)*105
		cardstyle.Ring(dc, cx, cy, r, cardstyle.WithA(p.Accent, 55), 1)
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("TIER %d", t), cx, cy-r-10, cardstyle.WithA(p.Muted, 190), 0.5, 1.8)
	}
	// root sigil
	s02GreatCircle(dc, cx, cy, 58, p)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 26, "ROOT", cx, cy, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 200), p.Accent, 0.5, 0.5, 90, 12)

	sector := 2 * math.Pi / float64(n)
	for bi, br := range req.Branches {
		mid := -math.Pi/2 + float64(bi)*sector + sector/2
		// spoke line from root to outer ring
		orr := 100 + float64(maxTier-1)*105
		dc.SetColor(cardstyle.WithA(p.Accent, 90))
		dc.SetLineWidth(1.1)
		dc.DrawLine(cx+58*math.Cos(mid), cy+58*math.Sin(mid), cx+orr*math.Cos(mid), cy+orr*math.Sin(mid))
		dc.Stroke()
		// branch label chip at rim
		lr := orr + 54
		lx, ly := cx+lr*math.Cos(mid), cy+lr*math.Sin(mid)
		cardstyle.Chip(dc, lx-80, ly-14, 160, 28, strings.ToUpper(cardstyle.Sanitize(br.Name)), cardstyle.WithA(p.Panel, 235), p.Accent, p.Ink, cardstyle.FtCinzel, 11)
		// nodes along the spoke
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
		for _, t := range order {
			list := nodesByTier[t]
			r := 100 + float64(t-1)*105
			for j, s := range list {
				off := (float64(j) - float64(len(list)-1)/2) * (sector * 0.26)
				a := mid + off
				x, yy := cx+r*math.Cos(a), cy+r*math.Sin(a)
				s02Node(dc, x, yy, s.State, p)
				// skill name + progress beside the node, flipped near card edges
				lx := x + 38*math.Cos(a)
				ly := yy + 38*math.Sin(a)
				if math.Hypot(lx-cx, ly-cy) > 430 {
					// would collide with the branch-chip ring: flip inward
					lx = x - 44*math.Cos(a)
					ly = yy - 44*math.Sin(a)
				}
				ax := 0.0
				if math.Cos(a) < -0.25 {
					ax = 1
					lx -= 8
				} else if math.Cos(a) > 0.25 {
					lx += 0
				} else {
					ax = 0.5
					lx = x
					if math.Sin(a) < 0 {
						ly = yy - 44
					} else {
						ly = yy + 40
					}
				}
				if lx < 70 {
					lx = 70
				}
				if lx > W-70 {
					lx = W - 70
				}
				if ly < 150 {
					ly = 150
				}
				if ly > H-96 {
					ly = H - 96
				}
				cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 14), lx, ly, p.Ink, ax, 0.5, 130, 8)
				cardstyle.Text(dc, cardstyle.FtInter, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), lx, ly+15, cardstyle.WithA(p.Muted, 220), ax, 0.5, 70, 7)
			}
		}
	}
	// legend
	ly := H - 60.0
	lx := W/2 - 230.0
	for _, l := range []struct {
		label, state string
	}{{"LEARNED", "learned"}, {"MAXED", "maxed"}, {"OPEN", "open"}, {"LOCKED", "locked"}} {
		s02Node(dc, lx, ly, l.state, p)
		cardstyle.Text(dc, cardstyle.FtInter, 11, l.label, lx+16, ly, p.Muted, 0, 0.5, 90, 8)
		lx += 128
	}
	s02Footer(dc, W, H, req, p)
	return dc
}

func s02Node(dc *gg.Context, x, y float64, state string, p *cardstyle.Palette) {
	switch state {
	case "learned":
		cardstyle.Diamond(dc, x, y, 9, p.Accent)
		cardstyle.Ring(dc, x, y, 14, cardstyle.WithA(p.Accent, 120), 1)
		cardstyle.GlowText(dc, cardstyle.FtCinzel, 14, " ", x, y, p.Accent, p.Accent, 0.5, 0.5, 10, 6)
	case "maxed":
		cardstyle.Diamond(dc, x, y, 11, cardstyle.Hex(0xffe9a8))
		cardstyle.Ring(dc, x, y, 16, cardstyle.HexA(0xffe9a8, 150), 1.2)
	case "open":
		cardstyle.DiamondOutline(dc, x, y, 8, cardstyle.WithA(p.Ink, 200), 1.4)
	default:
		cardstyle.DiamondOutline(dc, x, y, 5.5, cardstyle.WithA(p.Muted, 110), 1)
	}
}

// EQUIP - slots on a great ring around the hero.
func s02Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, "THE ARMORY CIRCLE", W/2, 56, p.Accent, 0.5, 0.5, 620, 16)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 90, p.Muted, 0.5, 3)

	// hero in the center circle
	cardstyle.PanelRounded(dc, W/2-190, 170, 380, 640, 14, p.Panel, cardstyle.WithA(p.Accent, 150), 1.2)
	drawHeroFitted(dc, req, W/2, 430, 300, 330, *p)
	cardstyle.DotLeader(dc, W/2-130, W/2+130, 660, 1.1, cardstyle.WithA(p.Accent, 120))
	cardstyle.Text(dc, cardstyle.FtCinzel, 20, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 700, p.Ink, 0.5, 0.5, 320, 12)
	if req.SealText != "" {
		cardstyle.Chip(dc, W/2-80, 736, 160, 30, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 12)
	}

	// 9 slots: 5 on the left orbit column, 4 on the right
	gw := 320.0
	for i, s := range req.Slots {
		var x, y, gh float64
		if i < 5 {
			x, gh = 60.0, 145.0
			y = 170 + float64(i)*160
		} else {
			x, gh = W-60-gw, 185.0
			y = 190 + float64(i-5)*207
		}
		empty := s.Empty
		fill := p.Panel
		if empty {
			fill = cardstyle.WithA(p.Track, 130)
		}
		cardstyle.PanelRounded(dc, x, y, gw, gh, 12, fill, cardstyle.WithA(p.Accent, 120), 1)
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Diamond(dc, x+26, y+28, 7, p.Accent)
		cardstyle.Text(dc, cardstyle.FtInterSemi, 12, slotCode, x+44, y+28, p.Muted, 0, 0.5, 90, 8)
		for t := 0; t < s.Tier && t < 5; t++ {
			cardstyle.Diamond(dc, x+gw-24-float64(t)*18, y+28, 5, cardstyle.Hex(0xffe9a8))
		}
		name := "EMPTY"
		ncol := cardstyle.WithA(p.Muted, 160)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = p.Ink
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.TruncateRunes(name, 20), x+16, y+70, ncol, 0, 0.5, gw-32, 10)
		if s.TierLabel != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), x+16, y+96, p.Muted, 0, 0.5, gw-32, 8)
		}
		if !empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			cardstyle.MeterBar(dc, x+16, y+gh-36, gw-32, 9, frac, p.Track, p.Accent, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 110), cardstyle.WithA(p.Accent, 120), 4.5)
		}
	}
	return dc
}

// SHOP - arcana stall ledger.
func s02Shop(req *portraitRequest) *gg.Context {
	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	nE := len(entries)
	if nE < 1 {
		nE = 1
	}
	const W = 800.0
	const step = 158.0
	H := 148.0 + float64(nE)*step + 110.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, "ARCANA EXCHANGE", W/2, 58, p.Accent, 0.5, 0.5, W-160, 14)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 92, p.Muted, 0.5, 3)
	cardstyle.DotLeader(dc, W/2-120, W/2+120, 112, 1, cardstyle.WithA(p.Accent, 140))

	y := 148.0
	for _, e := range entries {
		eh := 106.0
		s02Band(dc, 48, y, W-96, eh, p)
		cardstyle.Diamond(dc, 70, y+28, 5, p.Accent)
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(e.Title), 88, y+28, p.Ink, 0, 0.5, 400, 11)
		if e.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 68), 88, y+54, p.Muted, 0, 0.5, 400, 8)
		}
		if e.Runes != "" {
			cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(e.Runes), 88, y+80, p.Accent2, 0, 2.2)
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 19, cardstyle.Sanitize(e.Value), W-70, y+28, p.Accent, 1, 0.5, 150, 11)
		y += step
	}
	if H-90-(y+8) > 60 {
		s02Filler(dc, W, y+8, H-90, p, "THE ARCANUM EXCHANGE")
	}
	s02Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - sigil ring crest.
func s02GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "GUILD SIGIL", W/2, 46, p.Muted, 0.5, 3.4)
	name := styledName(req)
	s02GreatCircle(dc, W/2, 190, 100, p)
	accent := cardstyle.Hex(0x801c28)
	if req.HexColor != "" {
		hx := utils.ParseHexColor(req.HexColor)
		accent = cardstyle.N(hx.R, hx.G, hx.B, 255)
	}
	cardstyle.Shield(dc, W/2, 186, 96, 112, accent, p.Accent, 3)
	initial := firstRuneUpper(name)
	if req.EmblemImg != nil {
		dc.Push()
		dc.DrawCircle(W/2, 184, 40)
		dc.Clip()
		cardstyle.FitEmblem(dc, req.EmblemImg, W/2, 184, 70, 70)
		dc.Pop()
		dc.ResetClip() // gg Pop() keeps the clip mask (found 2026-09-16)
	} else {
		cardstyle.Text(dc, cardstyle.FtCinzelDec, 44, initial, W/2, 182, cardstyle.Hex(0xf4dc8c), 0.5, 0.5, 80, 20)
	}
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(cardstyle.Sanitize(name)), W/2, 330, p.Accent, 0.5, 0.5, W-140, 14)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 62)+"\"", W/2, 362, p.Muted, 0.5, 0.5, W-120, 10)
	}

	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	by := 396.0
	bh := 50 + float64(len(rows))*40 + 4
	s02Band(dc, 52, by, W-104, bh, p)
	s02SectionHead(dc, W/2, by+24, "THE CHARTER", p)
	ry := by + 52
	for _, r := range rows {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(r.Label)), 76, ry, p.Muted, 0, 0.5, 240, 9)
		cardstyle.DotLeader(dc, 250, W-210, ry, 1, cardstyle.WithA(p.Muted, 100))
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), W-76, ry, p.Accent2, 1, 0.5, 200, 10)
		ry += 40
	}
	// level + buildings orbit
	cardstyle.Chip(dc, W/2-70, by+bh+22, 140, 32, fmt.Sprintf("LVL %d", req.Level), cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 14)
	pct := float64(req.XPPercent) / 100
	cardstyle.MeterBar(dc, W/2-160, by+bh+66, 320, 9, pct, p.Track, p.Accent, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 110), cardstyle.WithA(p.Accent, 130), 4.5)
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		cxx := W/2 + float64(i-1)*170
		cyy := by + bh + 140.0
		cardstyle.Ring(dc, cxx, cyy, 32, cardstyle.WithA(p.Accent, 140), 1.3)
		cardstyle.Text(dc, cardstyle.FtInterSemi, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 8), cxx, cyy-10, p.Muted, 0.5, 0.5, 60, 7)
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("L%d", b.Level), cxx, cyy+10, p.Accent2, 0.5, 0.5, 50, 9)
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "THE CONCLAVE RECORDS", W/2, by+bh+118, cardstyle.WithA(p.Muted, 150), 0.5, 3)
	s02Footer(dc, W, H, req, p)
	return dc
}

// s02Evolve - class ascension card. Same design language as this style's
// SKILLUP (theme-internal reuse with variation): own base, frame and
// medallion, with the ceremony's own title and the new-class identity.
func s02Evolve(req *portraitRequest) *gg.Context {
	e := *req
	e.DocTitle = "RITUAL OF BECOMING"
	if e.SubLabel == "" {
		e.SubLabel = "EVOLVED"
	}
	return s02Skillup(&e)
}
