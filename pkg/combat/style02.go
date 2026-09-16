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
func s02Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 52, p.Muted, 0.5, 3)
	s02GreatCircle(dc, W/2, 262, 118, p)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 58, fmt.Sprintf("%d", req.Level), W/2, 250, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 140, 26)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "LEVEL", W/2, 320, p.Muted, 0.5, 4)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Chip(dc, W/2-130, 396, 260, 30, rankLine, cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 13)
	}

	// experience band
	s02Band(dc, 52, 446, W-104, 84, p)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "EXPERIENCE", 74, 476, p.Muted, 0, 2.2)
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-74, 476, p.Accent2, 1, 0.5, 220, 9)
	pct := float64(req.XPPercent) / 100
	cardstyle.MeterBar(dc, 74, 494, W-148, 10, pct, p.Track, p.Accent, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 120), cardstyle.WithA(p.Accent, 140), 5)

	// THE RECORD band
	rows := req.Standing
	if len(rows) > 4 {
		rows = rows[:4]
	}
	by := 552.0
	bh := 56 + float64(len(rows))*40 + 6
	s02Band(dc, 52, by, W-104, bh, p)
	s02SectionHead(dc, W/2, by+26, "THE RECORD", p)
	ry := by + 56
	for _, r := range rows {
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(r.Label)), 74, ry, p.Ink, 0, 0.5, 200, 9)
		cardstyle.DotLeader(dc, 230, W-190, ry, 1.1, cardstyle.WithA(p.Muted, 110))
		cardstyle.Diamond(dc, W-176, ry, 2.6, p.Accent)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), W-74, ry, p.Accent2, 1, 0.5, 140, 9)
		ry += 40
	}

	// THE PATH band
	py := by + bh + 18.0
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	if len(prog) > 0 {
		ph := 56 + float64(len(prog))*48 + 6
		s02Band(dc, 52, py, W-104, ph, p)
		s02SectionHead(dc, W/2, py+26, cardstyle.Sanitize(req.ProgressTitle), p)
		ry = py + 56
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
			dc.SetColor(fill)
			cardstyle.MeterBar(dc, 74, ry+12, W-148, 8, frac, p.Track, fill, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 110), cardstyle.WithA(p.Accent, 120), 4)
			ry += 48
		}
	}
	s02Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - unspent points in the circle; seven nodes on the rim.
func s02Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "RITUAL OF ALLOCATION", W/2, 50, p.Muted, 0.5, 3.4)
	s02GreatCircle(dc, W/2, 250, 112, p)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 60, fmt.Sprintf("%d", avail), W/2, 238, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 140, 26)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "UNSPENT POINTS", W/2, 308, p.Muted, 0.5, 3)
	classLine := strings.ToUpper(cardstyle.Sanitize(req.Pill))
	if strings.Contains(strings.ToLower(classLine), "starter") && !strings.Contains(strings.ToLower(classLine), "tier") {
		classLine += " TIER"
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, classLine, W/2, 382, p.Ink, 0.5, 0.5, 320, 9)

	// seven nodes on a rim arc below the circle
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	ry := 424.0
	for i, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		// node diamond on the left, like a point on a ritual rim
		a := math.Pi*0.15 + float64(i)*math.Pi*0.7/6
		cardstyle.Diamond(dc, 66+10*math.Cos(a), ry+8*math.Sin(a), 5, p.Accent)
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, cardstyle.StatFullName(code), 92, ry, p.Ink, 0, 0.5, 180, 10)
		cardstyle.DotLeader(dc, 268, W-176, ry, 1.1, cardstyle.WithA(p.Muted, 110))
		cardstyle.GlowText(dc, cardstyle.FtCinzel, 17, gainFromSub(r.Sub), W-74, ry, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 130), p.Accent2, 1, 0.5, 90, 10)
		ry += 42
	}

	// spent + CTA band
	s02Band(dc, 52, ry+8, W-104, 118, p)
	spent := parseLeadingInt(req.SpentNow, 0)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "SPENT", W/2-70, ry+38, p.Muted, 0.5, 2.6)
	cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("%d PTS", spent), W/2+70, ry+38, p.Accent, 0.5, 0.5, 120, 10)
	cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.Sanitize(req.SpentLeft), W/2, ry+62, p.Muted, 0.5, 0.5, 260, 8)
	cardstyle.PillCTA(dc, W/2, ry+94, 360, 34, ".j allocate <stat> [amount]", "", cardstyle.WithA(p.Accent, 230), cardstyle.HexA(0xffe9a8, 150), cardstyle.N(30, 22, 10, 255), p.Muted, cardstyle.FtCinzel, 13)
	cardstyle.FooterNote(dc, W/2, ry+150, "Half value per point after heavy single-stat investment", p.Muted, cardstyle.FtMedieval, 12, W-100)
	s02Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - sigil in triangle-in-circle.
func s02Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "RITUAL OF MASTERY", W/2, 50, p.Muted, 0.5, 3.4)
	s02GreatCircle(dc, W/2, 280, 120, p)
	// triangle inscribed
	dc.SetColor(cardstyle.WithA(p.Accent, 110))
	dc.SetLineWidth(1.2)
	for i := 0; i <= 3; i++ {
		a := -math.Pi/2 + float64(i)*2*math.Pi/3
		x, y := W/2+96*math.Cos(a), 280+96*math.Sin(a)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.Stroke()
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 54, initials, W/2, 272, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 130, 22)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 25, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, 452, p.Accent, 0.5, 0.5, W-90, 12)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, tierLabel, W/2, 484, p.Muted, 0.5, 3)

	// orbital pips around a small ring
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
	for i := 0; i < maxPips; i++ {
		a := -math.Pi/2 + float64(i)*2*math.Pi/float64(maxPips)
		x, y := W/2+70*math.Cos(a), 590+70*math.Sin(a)
		if i < cur {
			cardstyle.Diamond(dc, x, y, 7, p.Accent)
			cardstyle.Ring(dc, x, y, 11, cardstyle.WithA(p.Accent, 110), 1)
		} else {
			cardstyle.DiamondOutline(dc, x, y, 5.5, cardstyle.WithA(p.Muted, 140), 1.2)
		}
	}

	s02Band(dc, 52, 690, W-104, 120, p)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "MASTERY", W/2, 720, p.Accent, 0.5, 3)
	frac := 0.0
	if maxPips > 0 {
		frac = float64(cur) / float64(maxPips)
	}
	cardstyle.MeterBar(dc, 84, 742, W-168, 10, frac, p.Track, p.Accent, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 120), cardstyle.WithA(p.Accent, 140), 5)
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, fmt.Sprintf("%d / %d", cur, maxPips), W/2, 780, p.Ink, 0.5, 0.5, 120, 9)
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
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s02Base(dc, W, H)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, "ARCANA EXCHANGE", W/2, 58, p.Accent, 0.5, 0.5, W-160, 14)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 92, p.Muted, 0.5, 3)
	cardstyle.DotLeader(dc, W/2-120, W/2+120, 112, 1, cardstyle.WithA(p.Accent, 140))

	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	y := 140.0
	for _, e := range entries {
		eh := 100.0
		s02Band(dc, 48, y, W-96, eh, p)
		cardstyle.Diamond(dc, 70, y+28, 5, p.Accent)
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(e.Title), 88, y+28, p.Ink, 0, 0.5, 400, 11)
		if e.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 68), 88, y+54, p.Muted, 0, 0.5, 400, 8)
		}
		if e.Runes != "" {
			cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(e.Runes), 88, y+78, p.Accent2, 0, 2.2)
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 19, cardstyle.Sanitize(e.Value), W-70, y+28, p.Accent, 1, 0.5, 150, 11)
		y += eh + 14
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
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 44, initial, W/2, 182, cardstyle.Hex(0xf4dc8c), 0.5, 0.5, 80, 20)
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
	s02Footer(dc, W, H, req, p)
	return dc
}
