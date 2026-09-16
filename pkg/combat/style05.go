package combat

// style05.go - EMBLEM NOIR "Classified Dossier" (Style 5).
// Austere secret-society file: matte black, huge negative space, ONE gold
// hairline emblem, wide-tracked data rows over thin rules, file stamps,
// corner registration marks, red wax seal as the only color.
// Recognizable by structure: file stamps + ruled ledger + wax.

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

func s05Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.EmblemNoir()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	// carbon crosshatch
	dc.SetColor(cardstyle.N(255, 255, 255, 6))
	dc.SetLineWidth(1)
	for y := 0.0; y < H; y += 5 {
		dc.DrawLine(0, y, W, y)
		dc.Stroke()
	}
	// registration marks (the dossier's structural signature)
	rm := 26.0
	for _, c := range [][2]float64{{rm, rm}, {W - rm, rm}, {rm, H - rm}, {W - rm, H - rm}} {
		dc.SetColor(cardstyle.WithA(p.Muted, 120))
		dc.SetLineWidth(1)
		dc.DrawLine(c[0]-7, c[1], c[0]+7, c[1])
		dc.Stroke()
		dc.DrawLine(c[0], c[1]-7, c[0], c[1]+7)
		dc.Stroke()
		dc.DrawCircle(c[0], c[1], 3.4)
		dc.Stroke()
	}
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s05FileHead - stamp block + emblem hairline diamond.
func s05FileHead(dc *gg.Context, W float64, title, ref string, p *cardstyle.Palette) float64 {
	cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(title)), 52, 56, p.Accent, 0, 0.5, 420, 9)
	if ref != "" {
		cardstyle.Stamp(dc, W-150, 56, strings.ToUpper(cardstyle.Sanitize(ref)), p.Accent)
	}
	dc.SetColor(cardstyle.WithA(p.Accent, 160))
	dc.SetLineWidth(1)
	dc.DrawLine(52, 74, W-52, 74)
	dc.Stroke()
	return 96
}

func s05Row(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette, size int) {
	cardstyle.LedgerRow(dc, x, y, w, strings.ToUpper(cardstyle.Sanitize(label)), strings.ToUpper(cardstyle.Sanitize(value)), cardstyle.WithA(p.Muted, 90), p.Ink, p.Muted, cardstyle.FtInter, cardstyle.FtInterSemi, size)
}

func s05Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.WaxSeal(dc, 58, H-52, 24, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 12)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 64), 100, H-52, p.Muted, 0, 0.5, W-170, 9)
	}
}

func renderStyle05(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s05Rank(req).Image()
	case "ALLOCATE":
		return s05Allocate(req).Image()
	case "SKILLUP":
		return s05Skillup(req).Image()
	case "ABILITIES":
		return s05Abilities(req).Image()
	case "SKILLTREE":
		return s05Skilltree(req).Image()
	case "EQUIP":
		return s05Equip(req).Image()
	case "SHOP":
		return s05Shop(req).Image()
	case "GUILDINFO":
		return s05GuildInfo(req).Image()
	}
	return nil
}

// RANK - dossier with ghost level numeral.
func s05Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "ADVENTURER FILE", "REF "+styledName(req), p)

	// ghost level numeral backdrop
	cardstyle.Text(dc, cardstyle.FtInterSemi, 190, fmt.Sprintf("%d", req.Level), W-40, 260, cardstyle.WithA(p.Ink, 30), 1, 0.5, 300, 120)

	cardstyle.Text(dc, cardstyle.FtInterSemi, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 52, y+16, p.Ink, 0, 0.5, 380, 12)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 12, rankLine, 52, y+42, p.Accent, 0, 0.5, 380, 9)
	}
	y += 66

	// level row
	s05Row(dc, 52, y, W-104, "LEVEL", fmt.Sprintf("%d", req.Level), p, 15)
	y += 34
	// experience
	cardstyle.Text(dc, cardstyle.FtInter, 13, "EXPERIENCE", 52, y, p.Muted, 0, 0.5, 200, 9)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 13, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-52, y, p.Ink, 1, 0.5, 240, 9)
	pct := float64(req.XPPercent) / 100
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(52, y+12, W-104, 4, 2)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(52, y+12, (W-104)*pct, 4, 2)
	dc.Fill()
	y += 44

	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	for _, r := range rows {
		s05Row(dc, 52, y, W-104, r.Label, r.Value, p, 13)
		y += 34
	}
	y += 12
	cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), 52, y, p.Accent, 0, 0.5, 320, 8)
	y += 22
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	for _, pr := range prog {
		frac := 0.0
		if pr.Max > 0 {
			frac = float64(pr.Cur) / float64(pr.Max)
		}
		if pr.Done {
			frac = 1
		}
		s05Row(dc, 52, y, W-104, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), p, 12)
		dc.SetColor(p.Track)
		dc.DrawRoundedRectangle(52, y+13, W-104, 3, 1.5)
		dc.Fill()
		fill := p.Accent
		if pr.Done {
			fill = p.Done
		}
		dc.SetColor(fill)
		dc.DrawRoundedRectangle(52, y+13, (W-104)*frac, 3, 1.5)
		dc.Fill()
		y += 38
	}
	// the society emblem fills the lower dossier field
	cardstyle.DiamondOutline(dc, W/2, 760, 96, cardstyle.WithA(p.Accent, 150), 1.2)
	cardstyle.DiamondOutline(dc, W/2, 760, 78, cardstyle.WithA(p.Accent, 100), 0.7)
	cardstyle.Diamond(dc, W/2, 760, 12, cardstyle.WithA(p.Accent, 160))
	cardstyle.Spaced(dc, cardstyle.FtInter, 10, "ADVENTURERS GUILD - INTERNAL", W/2, 890, cardstyle.WithA(p.Muted, 130), 0.5, 3)
	s05Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - allocation ledger sheet.
func s05Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "ALLOCATION SHEET", "FORM 7-A", p)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 150, fmt.Sprintf("%d", avail), W-40, 220, cardstyle.WithA(p.Accent, 60), 1, 0.5, 260, 90)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 20, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 52, y+14, p.Ink, 0, 0.5, 340, 11)
	cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(req.Pill)), 52, y+38, p.Muted, 0, 0.5, 340, 8)
	y += 62
	cardstyle.Text(dc, cardstyle.FtInter, 12, "UNSPENT POINTS - RIGHT COLUMN", 52, y, p.Muted, 0, 0.5, 320, 8)
	y += 22

	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		s05Row(dc, 52, y, W-104, cardstyle.StatFullName(code), gainFromSub(r.Sub), p, 14)
		y += 36
	}
	y += 10
	s05Row(dc, 52, y, W-104, "SPENT", fmt.Sprintf("%d PTS", parseLeadingInt(req.SpentNow, 0)), p, 15)
	y += 36
	cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), 52, y, p.Muted, 0, 0.5, 320, 8)
	y += 26
	cardstyle.Text(dc, cardstyle.FtInter, 12, ".J ALLOCATE <STAT> [AMOUNT]", 52, y, p.Accent, 0, 0.5, 360, 9)
	y += 22
	cardstyle.Text(dc, cardstyle.FtInter, 10, "HALF VALUE PER POINT AFTER HEAVY SINGLE-STAT INVESTMENT", 52, y, cardstyle.WithA(p.Muted, 170), 0, 0.5, W-104, 8)
	s05Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - promotion memo.
func s05Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "PROMOTION MEMO", styledName(req), p)

	// gold emblem: hairline diamond + initials
	cardstyle.DiamondOutline(dc, W/2, 268, 96, p.Accent, 1.2)
	cardstyle.DiamondOutline(dc, W/2, 268, 82, cardstyle.WithA(p.Accent, 120), 0.7)
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.Text(dc, cardstyle.FtInterSemi, 44, initials, W/2, 264, p.Ink, 0.5, 0.5, 120, 20)

	y += 30
	cardstyle.Text(dc, cardstyle.FtInterSemi, 20, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), 52, y+30, p.Ink, 0, 0.5, W-104, 11)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Text(dc, cardstyle.FtInter, 12, strings.ToUpper(tierLabel), 52, y+56, p.Accent, 0, 0.5, W-104, 9)
	y += 88

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
	s05Row(dc, 52, y, W-104, "MASTERY", fmt.Sprintf("%d / %d", cur, maxPips), p, 14)
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(52, y+13, W-104, 4, 2)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(52, y+13, (W-104)*frac, 4, 2)
	dc.Fill()
	y += 44
	s05Row(dc, 52, y, W-104, "STATUS", tierLabel, p, 13)
	s05Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - index list.
func s05Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, cardstyle.Sanitize(req.DocTitle), cardstyle.Sanitize(req.PageLabel), p)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 12, strings.ToUpper("\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 92)+"\""), 52, y+8, p.Muted, 0, 0.5, W-104, 9)
		y += 30
	}

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(g.Name)), 52, y, p.Accent, 0, 0.5, 400, 8)
		y += 24
		for _, it := range g.Items {
			cardstyle.Text(dc, cardstyle.FtInter, 12, fmt.Sprintf("%03d", num), 52, y, cardstyle.WithA(p.Muted, 160), 0, 0.5, 60, 8)
			s05Row(dc, 130, y, W-200, it.Title, it.Value, p, 14)
			extra := it.Sub
			if it.Runes != "" {
				extra = strings.ToUpper(cardstyle.Sanitize(it.Runes)) + "  " + extra
			}
			if extra != "" {
				cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(extra), 90)), 130, y+18, p.Muted, 0, 0.5, W-200, 8)
			}
			num++
			y += 48
		}
		y += 12
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - diamond lattice (rotated square grid).
func s05Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1100.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05FileHead(dc, W, "MASTERY LATTICE", strings.ToUpper(cardstyle.Sanitize(req.ClassName)), p)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Stamp(dc, W-150, 100, ptsText, p.Accent)

	// diamond lattice: rotated square grid; tiers go up-left to down-right diagonals
	cx := W / 2
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
	// diagonal grid lines
	g := 66.0
	dc.SetColor(cardstyle.WithA(p.Muted, 40))
	dc.SetLineWidth(0.8)
	for d := -12; d <= 12; d++ {
		off := float64(d) * g
		dc.DrawLine(cx-g*14+off, 0, cx+g*14+off, H)
		dc.Stroke()
		dc.DrawLine(cx-g*14-off, 0, cx+g*14-off, H)
		dc.Stroke()
	}
	// apex wax seal at top node
	cardstyle.WaxSeal(dc, cx, 210, 26, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, "MAX", cardstyle.FtInter, 11)

	// branch columns along the horizontal axis
	totalW := float64(n) * 230
	x0 := cx - totalW/2 + 115
	for bi, br := range req.Branches {
		bx := x0 + float64(bi)*230
		// branch label plate
		cardstyle.Text(dc, cardstyle.FtInterSemi, 13, strings.ToUpper(cardstyle.Sanitize(br.Name)), bx, 300, p.Ink, 0.5, 0.5, 210, 9)
		cardstyle.DiamondOutline(dc, bx, 322, 10, p.Accent, 1.2)
		// vertical tier chain below the label
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
			ny := 396 + float64(k)*150
			// connective diagonal to previous node
			if k > 0 {
				dc.SetColor(cardstyle.WithA(p.Muted, 90))
				dc.SetLineWidth(1.1)
				dc.DrawLine(bx, 396+float64(k-1)*150+26, bx, ny-26)
				dc.Stroke()
			}
			for j, s := range list {
				nx := bx + (float64(j)-float64(len(list)-1)/2)*70
				s05Node(dc, nx, ny, s, p)
			}
			_ = t
		}
	}
	// legend
	ly := H - 66.0
	lx := cx - 240.0
	for _, l := range []struct {
		label, state string
	}{{"LEARNED", "learned"}, {"MAXED", "maxed"}, {"OPEN", "open"}, {"LOCKED", "locked"}} {
		cardstyle.Diamond(dc, lx, ly, 8, s05NodeCol(l.state, p))
		cardstyle.Text(dc, cardstyle.FtInter, 11, l.label, lx+16, ly, p.Muted, 0, 0.5, 100, 8)
		lx += 128
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

func s05NodeCol(state string, p *cardstyle.Palette) color.NRGBA {
	switch state {
	case "learned":
		return p.Accent
	case "maxed":
		return cardstyle.Hex(0xe8c968)
	case "open":
		return p.Ink
	default:
		return cardstyle.WithA(p.Muted, 110)
	}
}

func s05Node(dc *gg.Context, x, y float64, s portraitNode, p *cardstyle.Palette) {
	col := s05NodeCol(s.State, p)
	lit := s.State == "learned" || s.State == "maxed"
	r := 15.0
	if lit {
		cardstyle.Diamond(dc, x, y, r, col)
		cardstyle.DiamondOutline(dc, x, y, r+6, cardstyle.WithA(col, 110), 1)
	} else {
		cardstyle.DiamondOutline(dc, x, y, r-4, col, 1.4)
	}
	cardstyle.Text(dc, cardstyle.FtInterSemi, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 12), x, y+r+16, p.Ink, 0.5, 0.5, 130, 8)
	cardstyle.Text(dc, cardstyle.FtInter, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), x, y+r+32, p.Muted, 0.5, 0.5, 60, 7)
}

// EQUIP - equipment manifest.
func s05Equip(req *portraitRequest) *gg.Context {
	const W, H = 1200.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "EQUIPMENT MANIFEST", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

	// left: hero file photo frame
	cardstyle.FrameHairline(dc, 60, y+20, 300, 400, 10, 1, cardstyle.WithA(p.Accent, 140), false)
	drawHeroFitted(dc, req, 210, y+220, 240, 260, *p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 16, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 210, y+460, p.Ink, 0.5, 0.5, 280, 10)
	if req.SealText != "" {
		cardstyle.WaxSeal(dc, 330, y+400, 22, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 11)
	}

	// right: manifest rows
	bx, bw := 420.0, W-480.0
	ry := y + 40
	for i, s := range req.Slots {
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		var label, value string
		if s.Empty {
			label = slotCode
			value = "EMPTY"
		} else {
			label = slotCode + " - " + cardstyle.Sanitize(s.Name)
			value = strings.ToUpper(cardstyle.Sanitize(s.TierLabel))
			if s.DurMax > 0 {
				value += fmt.Sprintf("  %s/%s", cardstyle.FmtF(s.Dur), cardstyle.FmtF(s.DurMax))
			}
		}
		s05Row(dc, bx, ry, bw, label, value, p, 13)
		if !s.Empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			dc.SetColor(p.Track)
			dc.DrawRoundedRectangle(bx, ry+13, bw*0.5, 3, 1.5)
			dc.Fill()
			dc.SetColor(p.Accent)
			dc.DrawRoundedRectangle(bx, ry+13, bw*0.5*frac, 3, 1.5)
			dc.Fill()
		}
		_ = i
		ry += 44
		if ry > H-160 {
			break
		}
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

// SHOP - procurement list.
func s05Shop(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "PROCUREMENT LIST", styledName(req), p)
	entries := req.Entries
	if len(entries) > 9 {
		entries = entries[:9]
	}
	for _, e := range entries {
		s05Row(dc, 52, y, W-104, e.Title, e.Value, p, 14)
		extra := e.Sub
		if e.Runes != "" {
			extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
		}
		if extra != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(extra), 88)), 52, y+18, p.Muted, 0, 0.5, W-104, 8)
		}
		y += 48
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - chapter charter.
func s05GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	y := s05FileHead(dc, W, "CHAPTER CHARTER", "CHAPTER 01", p)

	// emblem
	cardstyle.DiamondOutline(dc, W/2, y+90, 84, p.Accent, 1.2)
	cardstyle.DiamondOutline(dc, W/2, y+90, 70, cardstyle.WithA(p.Accent, 110), 0.7)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 40, firstRuneUpper(styledName(req)), W/2, y+86, p.Ink, 0.5, 0.5, 90, 18)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, y+210, p.Ink, 0.5, 0.5, W-140, 12)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 12, strings.ToUpper("\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 70)+"\""), W/2, y+238, p.Muted, 0.5, 0.5, W-120, 9)
	}
	y += 270

	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	for _, r := range rows {
		s05Row(dc, 52, y, W-104, r.Label, r.Value, p, 13)
		y += 34
	}
	y += 8
	s05Row(dc, 52, y, W-104, "LEVEL", fmt.Sprintf("%d", req.Level), p, 14)
	pct := float64(req.XPPercent) / 100
	dc.SetColor(p.Track)
	dc.DrawRoundedRectangle(52, y+13, W-104, 3, 1.5)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(52, y+13, (W-104)*pct, 3, 1.5)
	dc.Fill()
	y += 40
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		s05Row(dc, 52, y, W-104, b.Name, fmt.Sprintf("LVL %d", b.Level), p, 12)
		y += 30
	}
	s05Footer(dc, W, H, req, p)
	return dc
}
