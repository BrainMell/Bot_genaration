package combat

// style10.go - CRIMSON COURT "Court of Crimson" (Style 10).
// Vampire court heraldry: velvet damask field, ornate gold cartouches,
// heraldic SHIELDS for key values, pennant rows, filigree connectors,
// candlelight vignette.
// Recognizable by structure: cartouches + shields + pennants + damask.

import (
	"fmt"
	"image"
	"math"
	"strings"

	"image-service/pkg/cardstyle"
	"image-service/pkg/utils"

	"github.com/fogleman/gg"
)

func s10Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.CrimsonCourt()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Damask(dc, W, H, 76, cardstyle.HexA(0xd4a856, 22))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2.6, p.Accent, false)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 17, 1, cardstyle.WithA(p.Accent, 130), false)
	// corner filigree curls
	for _, c := range [][2]float64{{22, 22}, {W - 22, 22}, {22, H - 22}, {W - 22, H - 22}} {
		sx, sy := c[0], c[1]
		dx, dy := 1.0, 1.0
		if sx > W/2 {
			dx = -1
		}
		if sy > H/2 {
			dy = -1
		}
		dc.SetColor(cardstyle.WithA(p.Accent, 190))
		dc.SetLineWidth(1.6)
		a0 := 0.0
		if dx < 0 {
			a0 = math.Pi
		}
		a1 := a0 + float64(dy)*math.Pi/2
		dc.DrawArc(sx, sy, 12, a0, a1)
		dc.Stroke()
		dc.Stroke()
		cardstyle.Diamond(dc, sx+float64(dx)*10, sy+float64(dy)*10, 4, p.Accent)
	}
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

func s10Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.WaxSeal(dc, 56, H-48, 24, p.Seal, cardstyle.Darken(p.Seal, 44), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 12)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 58), 98, H-48, cardstyle.WithA(p.Ink, 215), 0, 0.5, W-170, 9)
	}
}

func s10Row(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette) {
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(label)), x, y, p.Muted, 0, 0.5, w*0.5, 10)
	cardstyle.DotLeader(dc, x+w*0.52, x+w-cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(value)), 0)-16, y, 1.1, cardstyle.WithA(p.Accent, 90))
	cardstyle.Diamond(dc, x+w*0.52-8, y, 2.6, cardstyle.WithA(p.Accent, 140))
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(value)), x+w, y, p.Ink, 1, 0.5, w*0.45, 10)
}

func renderStyle10(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s10Rank(req).Image()
	case "ALLOCATE":
		return s10Allocate(req).Image()
	case "SKILLUP":
		return s10Skillup(req).Image()
	case "ABILITIES":
		return s10Abilities(req).Image()
	case "SKILLTREE":
		return s10Skilltree(req).Image()
	case "EQUIP":
		return s10Equip(req).Image()
	case "SHOP":
		return s10Shop(req).Image()
	case "GUILDINFO":
		return s10GuildInfo(req).Image()
	}
	return nil
}

// RANK - letters patent.
func s10Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, strings.ToUpper(cardstyle.Sanitize(styledName(req))), cardstyle.FtCinzelDec, 22)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "LETTERS PATENT OF THE COURT", W/2, 122, p.Muted, 0.5, 2.6)
	cardstyle.DotLeader(dc, W/2-110, W/2+110, 138, 1, cardstyle.WithA(p.Accent, 120))

	// heraldic shield with the level
	cardstyle.Shield(dc, W/2, 268, 190, 220, p.Track, p.Accent, 3.4)
	cardstyle.EmberGlow(dc, W/2, 258, 70, cardstyle.HexA(0xd4a856, 90))
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 58, fmt.Sprintf("%d", req.Level), W/2, 250, p.Accent, 0.5, 0.5, 130, 22)
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, "LEVEL", W/2, 306, p.Muted, 0.5, 0.5, 110, 8)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Pennant(dc, W/2-130, 392, 260, 34, p.Panel, p.Accent, p.Ink, rankLine, cardstyle.FtCinzel, 12)
	}

	// experience pennant row
	y := 452.0
	pct := float64(req.XPPercent) / 100
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, "EXPERIENCE", 56, y, p.Muted, 0, 0.5, 180, 9)
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-56, y, p.Ink, 1, 0.5, 220, 9)
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(56, y+12, W-112, 8, 4)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(56, y+12, (W-112)*pct, 8, 4)
	dc.Fill()
	y += 44

	// standing rows
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	s10Row(dc, 56, y, W-112, "THE RECORD", "", p)
	y += 26
	for _, r := range rows {
		s10Row(dc, 56, y, W-112, r.Label, r.Value, p)
		y += 32
	}
	y += 8
	// progress
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	if len(prog) > 0 {
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), 56, y, p.Accent, 0, 0.5, 320, 9)
		y += 24
		for _, pr := range prog {
			frac := 0.0
			if pr.Max > 0 {
				frac = float64(pr.Cur) / float64(pr.Max)
			}
			if pr.Done {
				frac = 1
			}
			s10Row(dc, 56, y, W-112, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), p)
			dc.SetColor(p.Track)
			dc.DrawRoundedRectangle(56, y+12, W-112, 5, 2.5)
			dc.Fill()
			fill := p.Accent
			if pr.Done {
				fill = p.Done
			}
			dc.SetColor(fill)
			dc.DrawRoundedRectangle(56, y+12, (W-112)*frac, 5, 2.5)
			dc.Fill()
			y += 40
		}
	}
	s10Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - court ledger with wax.
func s10Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, "ALLOCATION", cardstyle.FtCinzelDec, 24)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE COURT LEDGER", W/2, 122, p.Muted, 0.5, 2.6)

	// shield with unspent count
	cardstyle.Shield(dc, W/2, 260, 180, 210, p.Track, p.Accent, 3.2)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.EmberGlow(dc, W/2, 248, 66, cardstyle.HexA(0xd4a856, 90))
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 54, fmt.Sprintf("%d", avail), W/2, 240, p.Accent, 0.5, 0.5, 120, 20)
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, "UNSPENT", W/2, 292, p.Muted, 0.5, 0.5, 120, 8)
	classLine := strings.ToUpper(cardstyle.Sanitize(req.Pill))
	if strings.Contains(strings.ToLower(classLine), "starter") && !strings.Contains(strings.ToLower(classLine), "tier") {
		classLine += " TIER"
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, classLine, W/2, 386, p.Ink, 0.5, 0.5, W-120, 10)

	y := 420.0
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		s10Row(dc, 56, y, W-112, cardstyle.StatFullName(code), gainFromSub(r.Sub), p)
		y += 34
	}
	y += 6
	dc.SetColor(cardstyle.WithA(p.Accent, 140))
	dc.SetLineWidth(1.2)
	dc.DrawLine(56, y, W-56, y)
	dc.Stroke()
	y += 22
	spent := parseLeadingInt(req.SpentNow, 0)
	s10Row(dc, 56, y, W-112, "SPENT", fmt.Sprintf("%d PTS", spent), p)
	y += 38
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, ".J ALLOCATE <STAT> [AMOUNT]", W/2, y, p.Accent, 0.5, 0.5, W-120, 9)
	y += 20
	cardstyle.Text(dc, cardstyle.FtCinzel, 10, "HALF VALUE PER POINT AFTER HEAVY SINGLE-STAT INVESTMENT", W/2, y, cardstyle.WithA(p.Muted, 190), 0.5, 0.5, W-110, 8)
	s10Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - investiture of a skill.
func s10Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, "INVESTITURE", cardstyle.FtCinzelDec, 22)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 122, p.Muted, 0.5, 2.6)

	// shield with skill initials
	cardstyle.Shield(dc, W/2, 286, 210, 240, p.Track, p.Accent, 3.4)
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 56, initials, W/2, 272, p.Accent, 0.5, 0.5, 140, 22)
	// filigree curls flanking the shield
	for _, sx := range []float64{W/2 - 150, W/2 + 150} {
		dc.SetColor(cardstyle.WithA(p.Accent, 150))
		dc.SetLineWidth(1.4)
		if sx < W/2 {
			dc.DrawArc(sx+20, 290, 22, math.Pi/2, math.Pi*1.5)
		} else {
			dc.DrawArc(sx-20, 290, 22, -math.Pi/2, math.Pi/2)
		}
		dc.Stroke()
		cardstyle.Diamond(dc, sx, 330, 4, p.Accent)
	}

	y := 436.0
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 23, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, y, p.Ink, 0.5, 0.5, W-100, 12)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Pennant(dc, W/2-110, y+24, 220, 32, p.Panel, p.Accent, p.Ink, strings.ToUpper(tierLabel), cardstyle.FtCinzel, 12)

	y += 92
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
	s10Row(dc, 56, y, W-112, "MASTERY", fmt.Sprintf("%d / %d", cur, maxPips), p)
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(56, y+12, W-112, 7, 3.5)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(56, y+12, (W-112)*frac, 7, 3.5)
	dc.Fill()
	y += 44
	s10Row(dc, 56, y, W-112, "STATION", tierLabel, p)
	s10Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - the court retinue list.
func s10Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	title := strings.TrimSpace(req.DocTitle)
	if title == "" {
		title = "THE RETINUE"
	}
	cardstyle.Cartouche(dc, W/2, 64, 640, 54, p.Panel, p.Accent, p.Ink, strings.ToUpper(cardstyle.Sanitize(title)), cardstyle.FtCinzelDec, 20)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, "\""+cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.DocQuote)), 84)+"\"", W/2, 112, p.Muted, 0.5, 0.5, W-160, 9)
	}
	if req.PageLabel != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-56, 50, p.Accent2, 1, 0.5, 170, 8)
	}

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	y := 148.0
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		cardstyle.Cartouche(dc, W/2, y+16, 420, 34, cardstyle.WithA(p.Panel, 210), cardstyle.WithA(p.Accent, 140), p.Accent, strings.ToUpper(cardstyle.Sanitize(g.Name)), cardstyle.FtCinzel, 13)
		y += 52
		for _, it := range g.Items {
			cardstyle.Diamond(dc, 66, y-2, 5, p.Accent)
			cardstyle.Text(dc, cardstyle.FtCinzel, 13, fmt.Sprintf("%02d", num), 86, y-2, cardstyle.WithA(p.Muted, 190), 0, 0.5, 44, 8)
			s10Row(dc, 140, y-2, W-230, it.Title, it.Value, p)
			extra := it.Sub
			if it.Runes != "" {
				extra = strings.ToUpper(cardstyle.Sanitize(it.Runes)) + "  " + extra
			}
			if extra != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 84), 140, y+18, cardstyle.WithA(p.Muted, 200), 0, 0.5, W-220, 8)
			}
			num++
			y += 50
		}
		y += 12
		cardstyle.DotLeader(dc, W/2-70, W/2+70, y, 1, cardstyle.WithA(p.Accent, 90))
		cardstyle.Diamond(dc, W/2, y, 3, p.Accent)
		y += 16
	}
	s10Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - LINEAGE TREE: heraldic family tree with shield nodes.
func s10Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1200.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 64, 520, 54, p.Panel, p.Accent, p.Ink, "THE LINEAGE OF MASTERY", cardstyle.FtCinzelDec, 19)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.ClassName)), W/2, 110, p.Muted, 0.5, 2.6)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Pennant(dc, W-250, 44, 190, 36, p.Panel, p.Accent, p.Ink, ptsText, cardstyle.FtCinzel, 12)

	// dynastic root shield at top center
	rootY := 190.0
	cardstyle.Shield(dc, W/2, rootY, 130, 150, p.Track, p.Accent, 3)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.ClassName), 10)), W/2, rootY-14, p.Ink, 0.5, 0.5, 130, 10)
	cardstyle.Text(dc, cardstyle.FtCinzel, 11, "DYNASTY", W/2, rootY+22, p.Muted, 0.5, 0.5, 110, 8)

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
	// filigree connectors from root to each branch column
	colW := (W - 160) / float64(n)
	for bi := range req.Branches {
		cx := 80 + float64(bi)*colW + colW/2
		dc.SetColor(cardstyle.WithA(p.Accent, 120))
		dc.SetLineWidth(1.4)
		dc.DrawLine(W/2, rootY+78, cx, rootY+160)
		dc.Stroke()
		cardstyle.Diamond(dc, cx, rootY+160, 4, p.Accent)
	}

	// branch shields down the columns
	for bi, br := range req.Branches {
		cx := 80 + float64(bi)*colW + colW/2
		by := rootY + 210.0
		cardstyle.Shield(dc, cx, by, 150, 120, p.Panel, p.Accent, 2.4)
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(br.Name), 12)), cx, by-14, p.Ink, 0.5, 0.5, colW-16, 9)
		// nodes as small shields below with filigree links
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
		prevY := by + 62.0
		for k, t := range order {
			list := nodesByTier[t]
			ny := by + 130 + float64(k)*132
			_ = t
			// filigree link
			dc.SetColor(cardstyle.WithA(p.Accent, 110))
			dc.SetLineWidth(1.2)
			dc.DrawLine(cx, prevY, cx, ny-42)
			dc.Stroke()
			cardstyle.Diamond(dc, cx, ny-52, 3.4, cardstyle.WithA(p.Accent, 150))
			for j, s := range list {
				nx := cx + (float64(j)-float64(len(list)-1)/2)*150
				lit := s.State == "learned" || s.State == "maxed"
				fill := p.Track
				if lit {
					fill = p.Panel
					cardstyle.EmberGlow(dc, nx, ny, 52, cardstyle.HexA(0xd4a856, 80))
				}
				maxed := s.State == "maxed"
				if s.State == "locked" {
					fill = cardstyle.Darken(p.Track, 16)
				}
				edge := p.Accent
				if maxed {
					edge = cardstyle.Hex(0xf4d696)
				}
				cardstyle.Shield(dc, nx, ny, 96, 84, fill, edge, 2.4)
				cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 12), nx, ny-16, p.Ink, 0.5, 0.5, 90, 8)
				valCol := p.Muted
				if s.State == "maxed" {
					valCol = cardstyle.Hex(0xf4d696)
				}
				cardstyle.Text(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), nx, ny+8, valCol, 0.5, 0.5, 70, 7)
			}
			prevY = by + 130 + float64(k)*132 + 42
		}
	}
	s10Footer(dc, W, H, req, p)
	return dc
}

// EQUIP - the regalia display.
func s10Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 66, 560, 56, p.Panel, p.Accent, p.Ink, "THE REGALIA DISPLAY", cardstyle.FtCinzelDec, 21)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 112, p.Muted, 0.5, 2.6)

	// hero dais
	cardstyle.Shield(dc, 250, 420, 330, 560, p.Panel, p.Accent, 3)
	drawHeroFitted(dc, req, 250, 400, 280, 300, *p)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 20, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, 640, p.Ink, 0.5, 0.5, 300, 11)
	if req.SealText != "" {
		cardstyle.Pennant(dc, 175, 672, 150, 34, p.Track, p.Accent, p.Ink, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", cardstyle.FtCinzel, 11)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 52), 250, 740, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 320, 9)

	// 9 regalia shields
	gx, gy, gw, gh, gap := 500.0, 160.0, 300.0, 250.0, 20.0
	for i, s := range req.Slots {
		col, row := i%3, i/3
		x := gx + float64(col)*(gw+gap)
		y := gy + float64(row)*(gh+gap+16)
		empty := s.Empty
		cardstyle.Shield(dc, x+gw/2, y+gh/2-20, gw, gh, p.Panel, p.Accent, 2.4)
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, slotCode, x+gw/2, y+18, p.Muted, 0.5, 0.5, 120, 8)
		for t := 0; t < s.Tier && t < 5; t++ {
			cardstyle.Diamond(dc, x+gw/2+50-float64(t)*18, y+18, 5, p.Accent)
		}
		name := "EMPTY"
		ncol := cardstyle.WithA(p.Muted, 150)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = p.Ink
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, cardstyle.TruncateRunes(strings.ToUpper(name), 18), x+gw/2, y+58, ncol, 0.5, 0.5, gw-30, 10)
		if s.TierLabel != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), x+gw/2, y+82, p.Accent2, 0.5, 0.5, gw-30, 8)
		}
		if !empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			dc.SetColor(p.Track)
			dc.DrawRoundedRectangle(x+30, y+gh-58, gw-60, 7, 3.5)
			dc.Fill()
			dc.SetColor(p.Accent)
			dc.DrawRoundedRectangle(x+30, y+gh-58, (gw-60)*frac, 7, 3.5)
			dc.Fill()
			cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+gw/2, y+gh-40, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, gw-40, 8)
		}
	}
	return dc
}

// SHOP - the court emporium.
func s10Shop(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 70, 460, 56, p.Panel, p.Accent, p.Ink, "THE COURT EMPORIUM", cardstyle.FtCinzelDec, 19)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 116, p.Muted, 0.5, 2.4)
	y := 152.0
	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	for _, e := range entries {
		cardstyle.Diamond(dc, 62, y-2, 4.6, p.Accent)
		s10Row(dc, 84, y-2, W-150, e.Title, e.Value, p)
		extra := e.Sub
		if e.Runes != "" {
			extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
		}
		if extra != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 84), 84, y+18, cardstyle.WithA(p.Muted, 200), 0, 0.5, W-160, 8)
		}
		y += 56
		cardstyle.DotLeader(dc, W/2-60, W/2+60, y-14, 0.9, cardstyle.WithA(p.Accent, 70))
	}
	s10Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - chapter arms.
func s10GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s10Base(dc, W, H)

	cardstyle.Cartouche(dc, W/2, 66, 480, 54, p.Panel, p.Accent, p.Ink, "CHAPTER ARMS", cardstyle.FtCinzelDec, 19)

	// great shield with guild color
	accent := cardstyle.Hex(0x801c28)
	if req.HexColor != "" {
		hx := utils.ParseHexColor(req.HexColor)
		accent = cardstyle.N(hx.R, hx.G, hx.B, 255)
	}
	cardstyle.Shield(dc, W/2, 230, 220, 250, accent, p.Accent, 4)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 54, firstRuneUpper(styledName(req)), W/2, 220, cardstyle.Hex(0xf4e2ce), 0.5, 0.5, 120, 22)
	name := styledName(req)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(name)), W/2, 392, p.Ink, 0.5, 0.5, W-140, 13)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, "\""+cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Motto)), 62)+"\"", W/2, 422, p.Muted, 0.5, 0.5, W-120, 9)
	}

	y := 452.0
	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	for _, r := range rows {
		s10Row(dc, 56, y, W-112, r.Label, r.Value, p)
		y += 34
	}
	y += 4
	s10Row(dc, 56, y, W-112, "LEVEL", fmt.Sprintf("%d", req.Level), p)
	pct := float64(req.XPPercent) / 100
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(56, y+12, W-112, 6, 3)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(56, y+12, (W-112)*pct, 6, 3)
	dc.Fill()
	y += 44
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		bx := 56 + float64(i)*((W-112)/3)
		cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 9), bx+(W-112)/6, y, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, (W-112)/3-12, 8)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, fmt.Sprintf("L%d", b.Level), bx+(W-112)/6, y+20, p.Accent2, 0.5, 0.5, 60, 9)
	}
	s10Footer(dc, W, H, req, p)
	return dc
}
