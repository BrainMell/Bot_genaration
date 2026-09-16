package combat

// style04.go - WOODMERE "The Hearth & Bough" (Style 4).
// Tavern carved oak: hanging SIGNBOARD header on iron hooks, content on a
// vellum page with binding stitches down the left edge, carved notch borders,
// branch-and-leaf ornament, amber tally meters.
// Recognizable by structure: hanging sign + stitched vellum page + notches.

import (
	"fmt"
	"image"
	"math"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

func s04Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.Woodmere()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Planks(dc, W, H, 0x5700D3, cardstyle.N(0, 0, 0, 60), cardstyle.N(255, 230, 180, 22))
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s04Signboard - hanging wooden sign with iron hooks + chains.
func s04Signboard(dc *gg.Context, W, y, w, h float64, title, sub string, p *cardstyle.Palette) float64 {
	// hooks + chains
	cardstyle.BevelRect(dc, W/2-w/2-26, y, 26, 18, cardstyle.N(46, 42, 38, 255), cardstyle.N(96, 90, 84, 255), cardstyle.N(16, 14, 12, 255), 1.6)
	cardstyle.BevelRect(dc, W/2+w/2, y, 26, 18, cardstyle.N(46, 42, 38, 255), cardstyle.N(96, 90, 84, 255), cardstyle.N(16, 14, 12, 255), 1.6)
	for _, cx := range []float64{W/2 - w/2 - 13, W/2 + w/2 + 13} {
		for i := 0; i < 3; i++ {
			cardstyle.Ring(dc, cx, y+28+float64(i)*14, 4.4, cardstyle.N(120, 112, 104, 255), 2)
		}
	}
	// board
	bx, by := W/2-w/2, y+30
	dc.SetColor(cardstyle.N(74, 48, 26, 255))
	dc.DrawRoundedRectangle(bx-6, by-6, w+12, h+12, 10)
	dc.Fill()
	cardstyle.BevelRect(dc, bx, by, w, h, cardstyle.N(112, 76, 42, 255), cardstyle.N(168, 122, 72, 255), cardstyle.N(44, 28, 14, 255), 2.6)
	// carved notches along the board edge
	dc.SetColor(cardstyle.N(44, 28, 14, 220))
	dc.SetLineWidth(3)
	for x := bx + 24; x < bx+w-24; x += 24 {
		dc.DrawLine(x, by+7, x+10, by+7)
		dc.Stroke()
		dc.DrawLine(x, by+h-7, x+10, by+h-7)
		dc.Stroke()
	}
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(title)), W/2, by+h/2-7, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, w-36, 13)
	if sub != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 12, cardstyle.Sanitize(sub), W/2, by+h-16, cardstyle.N(216, 170, 104, 230), 0.5, 0.5, w-36, 9)
	}
	return by + h + 22
}

// s04Page - vellum page with binding stitches at the left edge.
func s04Page(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) {
	dc.SetColor(cardstyle.N(36, 22, 10, 255))
	dc.DrawRoundedRectangle(x-10, y-6, w+18, h+12, 8)
	dc.Fill()
	dc.SetColor(p.Panel)
	dc.DrawRoundedRectangle(x, y, w, h, 6)
	dc.Fill()
	dc.SetColor(cardstyle.N(60, 38, 20, 40))
	dc.DrawRectangle(x, y, 16, h)
	dc.Fill()
	cardstyle.Stitches(dc, x+8, y+12, y+h-10, 22, cardstyle.N(120, 84, 46, 255))
	// subtle page grain
	dc.SetColor(cardstyle.N(96, 62, 30, 14))
	dc.SetLineWidth(1)
	for gy := y + 20; gy < y+h-8; gy += 18 {
		dc.DrawLine(x+24, gy, x+w-14, gy+3)
		dc.Stroke()
	}
}

func s04CarvedTitle(dc *gg.Context, x, y float64, text string, p *cardstyle.Palette) {
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(text)), x, y, cardstyle.N(116, 82, 50, 255), 0, 0.5, 320, 10)
	dc.SetColor(cardstyle.N(60, 38, 20, 160))
	dc.SetLineWidth(1.2)
	dc.DrawLine(x, y+12, x+cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(text)), 0)+14, y+12)
	dc.Stroke()
	// little leaf
	cardstyle.Diamond(dc, x+cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(text)), 0)+26, y-1, 4, cardstyle.N(96, 128, 64, 220))
}

func s04Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.SealMini(dc, 52, H-42, 19, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 58), 88, H-42, cardstyle.N(226, 192, 142, 220), 0, 0.5, W-150, 9)
	}
}

func renderStyle04(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s04Rank(req).Image()
	case "ALLOCATE":
		return s04Allocate(req).Image()
	case "SKILLUP":
		return s04Skillup(req).Image()
	case "ABILITIES":
		return s04Abilities(req).Image()
	case "SKILLTREE":
		return s04Skilltree(req).Image()
	case "EQUIP":
		return s04Equip(req).Image()
	case "SHOP":
		return s04Shop(req).Image()
	case "GUILDINFO":
		return s04GuildInfo(req).Image()
	}
	return nil
}

// RANK - signboard + vellum ledger page.
func s04Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	name := styledName(req)
	y := s04Signboard(dc, W, 30, 380, 64, name, "THE TAVERN LEDGER", p)

	// page 1: record
	s04Page(dc, 48, y, W-96, 380, p)
	s04CarvedTitle(dc, 84, y+30, "THE RECORD", p)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 40, fmt.Sprintf("LEVEL %d", req.Level), W/2, y+78, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 300, 20)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, rankLine, W/2, y+116, cardstyle.N(150, 104, 56, 255), 0.5, 0.5, W-140, 10)
	}
	pct := float64(req.XPPercent) / 100
	cardstyle.Text(dc, cardstyle.FtMedieval, 13, "EXPERIENCE  "+cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), 96, y+152, cardstyle.N(52, 34, 20, 230), 0, 0.5, W-200, 9)
	dc.SetColor(cardstyle.N(70, 46, 26, 255))
	dc.DrawRoundedRectangle(96, y+164, W-192, 12, 6)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(96, y+164, (W-192)*pct, 12, 6)
	dc.Fill()
	dc.SetColor(cardstyle.N(255, 214, 140, 110))
	dc.DrawRoundedRectangle(96, y+164, (W-192)*pct, 5, 4)
	dc.Fill()
	// standing as carved notches rows
	rows := req.Standing
	if len(rows) > 4 {
		rows = rows[:4]
	}
	ry := y + 202.0
	for _, r := range rows {
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(r.Label)), 96, ry, cardstyle.N(52, 34, 20, 255), 0, 0.5, 170, 9)
		cardstyle.DotLeader(dc, 250, W-190, ry, 1.1, cardstyle.N(150, 104, 56, 150))
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), W-96, ry, cardstyle.N(52, 34, 20, 255), 1, 0.5, 140, 9)
		ry += 34
	}
	_ = ry

	// page 2: progression
	y2 := y + 396.0
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	ph := 60 + float64(len(prog))*50
	s04Page(dc, 48, y2, W-96, ph, p)
	s04CarvedTitle(dc, 84, y2+30, cardstyle.Sanitize(req.ProgressTitle), p)
	ry = y2 + 62
	for _, pr := range prog {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(pr.Label)), 96, ry, cardstyle.N(52, 34, 20, 255), 0, 0.5, 230, 9)
		cardstyle.Text(dc, cardstyle.FtMedieval, 12, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), W-96, ry, cardstyle.N(150, 104, 56, 255), 1, 0.5, 170, 9)
		cardstyle.NotchMeter(dc, 96, ry+16, W-192, pr.Max, pr.Cur, cardstyle.N(70, 46, 26, 200), p.Accent)
		ry += 50
	}
	s04Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - carved points + notch meters.
func s04Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	y := s04Signboard(dc, W, 30, 360, 62, "ALLOCATION", styledName(req), p)

	s04Page(dc, 48, y, W-96, 250, p)
	s04CarvedTitle(dc, 84, y+30, "THE POINTS BOARD", p)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 58, fmt.Sprintf("%d", avail), W/2, y+104, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 140, 24)
	cardstyle.Text(dc, cardstyle.FtMedieval, 13, "UNSPENT POINTS  ·  "+strings.ToUpper(cardstyle.Sanitize(req.Pill)), W/2, y+156, cardstyle.N(150, 104, 56, 255), 0.5, 0.5, W-140, 10)
	// carve 7 max notches for the unspent pool feel
	cardstyle.NotchMeter(dc, 150, y+196, W-300, 10, avail%10+1, cardstyle.N(70, 46, 26, 170), p.Accent)

	// stat rows page
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	y2 := y + 266.0
	ph2 := 56 + float64(len(rows))*46 + 40
	s04Page(dc, 48, y2, W-96, ph2, p)
	s04CarvedTitle(dc, 84, y2+28, "THE SEVEN TAPS", p)
	ry := y2 + 60
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.StatFullName(code), 96, ry, cardstyle.N(52, 34, 20, 255), 0, 0.5, 170, 9)
		cardstyle.DotLeader(dc, 250, W-206, ry, 1.1, cardstyle.N(150, 104, 56, 150))
		cardstyle.Text(dc, cardstyle.FtCinzelDec, 17, gainFromSub(r.Sub), W-116, ry, cardstyle.N(150, 78, 24, 255), 1, 0.5, 90, 10)
		ry += 46
	}
	spent := parseLeadingInt(req.SpentNow, 0)
	dc.SetColor(cardstyle.N(60, 38, 20, 150))
	dc.SetLineWidth(1.2)
	dc.DrawLine(96, ry, W-96, ry)
	dc.Stroke()
	ry += 26
	cardstyle.Text(dc, cardstyle.FtCinzel, 14, "SPENT", 96, ry, cardstyle.N(52, 34, 20, 230), 0, 0.5, 120, 9)
	cardstyle.DotLeader(dc, 200, W-206, ry, 1.1, cardstyle.N(150, 104, 56, 140))
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("%d pts", spent), W-116, ry, cardstyle.N(150, 78, 24, 255), 1, 0.5, 100, 9)

	// footer CTA
	cardstyle.FooterNote(dc, W/2, y2+ph2+22, ".j allocate <stat> [amount]  ·  spent points are permanent", cardstyle.N(226, 192, 142, 230), cardstyle.FtMedieval, 14, W-90)
	s04Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - wooden medallion tag.
func s04Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	y := s04Signboard(dc, W, 30, 360, 62, "SKILL UP", styledName(req), p)

	// hanging medallion (wooden disc on a chain from the sign)
	for i := 0; i < 3; i++ {
		cardstyle.Ring(dc, W/2, y+6+float64(i)*16, 5, cardstyle.N(120, 112, 104, 255), 2.2)
	}
	cy := y + 96.0
	dc.SetColor(cardstyle.N(46, 28, 14, 255))
	dc.DrawCircle(W/2, cy, 122)
	dc.Fill()
	dc.SetColor(cardstyle.N(112, 76, 42, 255))
	dc.DrawCircle(W/2, cy, 114)
	dc.Fill()
	dc.SetColor(cardstyle.N(44, 28, 14, 220))
	dc.SetLineWidth(3)
	dc.DrawCircle(W/2, cy, 114)
	dc.Stroke()
	dc.SetColor(cardstyle.N(168, 122, 72, 120))
	dc.SetLineWidth(1.2)
	dc.DrawCircle(W/2, cy, 96)
	dc.Stroke()
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 58, initials, W/2, cy-6, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, 140, 22)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 14, tierLabel, W/2, cy+34, cardstyle.N(216, 170, 104, 240), 0.5, 0.5, 140, 9)

	// skill name carved below
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, cy+178, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, W-100, 12)

	// notches page
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
	py := cy + 220.0
	ph := 150.0
	s04Page(dc, 48, py, W-96, ph, p)
	s04CarvedTitle(dc, 84, py+30, "THE HEARTH TALLY", p)
	cardstyle.NotchMeter(dc, 120, py+80, W-240, maxPips, cur, cardstyle.N(70, 46, 26, 170), p.Accent)
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("%d / %d", cur, maxPips), W/2, py+118, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 140, 10)
	s04Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - guestbook ledger.
func s04Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	title := strings.TrimSpace(req.DocTitle)
	if title == "" {
		title = "THE GUESTBOOK"
	}
	y := s04Signboard(dc, W, 26, 520, 60, title, cardstyle.Sanitize(req.PageLabel), p)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 88)+"\"", W/2, y+6, cardstyle.N(216, 170, 104, 230), 0.5, 0.5, W-160, 10)
		y += 32
	}

	// one tall page
	ph := H - y - 74
	s04Page(dc, 44, y, W-88, ph, p)
	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	iy := y + 40.0
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		s04CarvedTitle(dc, 86, iy, g.Name, p)
		iy += 34
		for _, it := range g.Items {
			cardstyle.Text(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("%02d", num), 86, iy, cardstyle.N(150, 104, 56, 255), 0, 0.5, 40, 8)
			cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(it.Title), 130, iy-2, cardstyle.N(52, 34, 20, 255), 0, 0.5, 430, 11)
			if it.Sub != "" {
				cardstyle.Text(dc, cardstyle.FtMedieval, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(it.Sub), 64), 130, iy+18, cardstyle.N(150, 104, 56, 240), 0, 0.5, 430, 8)
			}
			if it.Runes != "" {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(it.Runes), W-96, iy-2, cardstyle.N(150, 78, 24, 255), 1, 2.2)
			}
			if it.Value != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.Sanitize(it.Value), W-96, iy+18, cardstyle.N(52, 34, 20, 220), 1, 0.5, 200, 9)
			}
			num++
			iy += 52
		}
		iy += 10
	}
	s04Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - a literal branching tree with hanging wooden tags.
func s04Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1200.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	y := s04Signboard(dc, W, 24, 460, 58, "THE HEARTH & BOUGH", strings.ToUpper(cardstyle.Sanitize(req.ClassName)), p)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Chip(dc, W-230, 40, 180, 32, ptsText, cardstyle.N(46, 28, 14, 255), cardstyle.N(168, 122, 72, 255), cardstyle.N(240, 202, 130, 255), cardstyle.FtCinzel, 12)

	// ground
	groundY := H - 120.0
	dc.SetColor(cardstyle.N(28, 16, 8, 220))
	dc.DrawRectangle(0, groundY, W, H-groundY)
	dc.Fill()

	// trunk
	dc.SetColor(cardstyle.N(64, 40, 20, 255))
	dc.DrawRoundedRectangle(W/2-26, groundY-160, 52, 160, 10)
	dc.Fill()
	dc.SetColor(cardstyle.N(40, 24, 12, 220))
	dc.SetLineWidth(2)
	for i := 0; i < 6; i++ {
		dc.DrawLine(W/2-20, groundY-140+float64(i)*24, W/2+18, groundY-130+float64(i)*24)
		dc.Stroke()
	}

	// root word on the plinth
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(req.ClassName)), W/2, groundY+46, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, 400, 12)

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
	// limbs: one per branch, spreading upward
	limbTop := y + 20.0
	for bi, br := range req.Branches {
		t := float64(bi) - float64(n-1)/2
		endX := W/2 + t*(W*0.30)
		endY := limbTop + 60 + math.Abs(t)*40
		// carved limb (two curves approximated by thick lines)
		dc.SetColor(cardstyle.N(88, 56, 30, 255))
		dc.SetLineWidth(14)
		dc.DrawLine(W/2, groundY-140, (W/2+endX)/2, (groundY-160+endY)/2)
		dc.Stroke()
		dc.SetLineWidth(9)
		dc.DrawLine((W/2+endX)/2, (groundY-160+endY)/2, endX, endY)
		dc.Stroke()
		// branch tag at the limb tip
		cardstyle.Tag(dc, endX-80, endY-14, 160, 30, cardstyle.N(74, 48, 26, 255), cardstyle.N(168, 122, 72, 255), cardstyle.N(240, 202, 130, 255), strings.ToUpper(cardstyle.Sanitize(br.Name)), cardstyle.FtCinzel, 12)
		// hanging skill tags at successive heights along the limb direction
		order := []int{}
		nodesByTier := map[int][]portraitNode{}
		for _, s := range br.Skills {
			tt := s.Tier
			if tt < 1 {
				tt = 1
			}
			if tt > maxTier {
				tt = maxTier
			}
			if _, ok := nodesByTier[tt]; !ok {
				order = append(order, tt)
			}
			nodesByTier[tt] = append(nodesByTier[tt], s)
		}
		for k, tt := range order {
			list := nodesByTier[tt]
			ty := endY + 74 + float64(k)*92
			// lean tag positions outward with depth
			span := (endX - W/2) * 0.22
			for j, s := range list {
				tx := endX + span + (float64(j)-float64(len(list)-1)/2)*96
				lit := s.State == "learned" || s.State == "maxed"
				base := cardstyle.N(74, 48, 26, 255)
				edge := cardstyle.N(168, 122, 72, 255)
				txt := cardstyle.N(216, 190, 140, 235)
				if lit {
					cardstyle.TorchGlow(dc, tx, ty+16, 46, cardstyle.N(255, 179, 64, 255))
					txt = cardstyle.N(255, 214, 140, 255)
				}
				if s.State == "locked" {
					base = cardstyle.N(52, 34, 18, 255)
				}
				// string from limb tip
				dc.SetColor(cardstyle.N(120, 112, 104, 180))
				dc.SetLineWidth(1.4)
				dc.DrawLine(tx, ty-34, tx, ty-12)
				dc.Stroke()
				cardstyle.Tag(dc, tx-76, ty-12, 152, 40, base, edge, txt, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 14), cardstyle.FtCinzel, 11)
				lvlCol := cardstyle.N(216, 170, 104, 240)
				if s.State == "maxed" {
					lvlCol = cardstyle.N(255, 179, 64, 255)
				}
				cardstyle.Text(dc, cardstyle.FtMedieval, 11, fmt.Sprintf("%d/%d", s.Cur, s.Max), tx, ty+36, lvlCol, 0.5, 0.5, 80, 8)
			}
		}
	}
	s04Footer(dc, W, H, req, p)
	return dc
}

// EQUIP - pegged tool wall.
func s04Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	s04Signboard(dc, W, 24, 460, 58, "THE TOOL WALL", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

	// hero shield plaque
	s04Page(dc, 50, 180, 400, 700, p)
	drawHeroFitted(dc, req, 250, 460, 320, 340, *p)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, 700, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 340, 12)
	if req.SealText != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 14, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", 250, 734, cardstyle.N(150, 104, 56, 255), 0.5, 0.5, 240, 9)
	}
	cardstyle.Text(dc, cardstyle.FtMedieval, 13, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 56), 250, 790, cardstyle.N(116, 82, 50, 240), 0.5, 0.5, 340, 9)

	// 9 pegged tags
	gx, gy, gw, gh, gap := 500.0, 180.0, 300.0, 210.0, 20.0
	for i, s := range req.Slots {
		col, row := i%3, i/3
		x := gx + float64(col)*(gw+gap)
		y := gy + float64(row)*(gh+gap)
		empty := s.Empty
		// peg
		cardstyle.Rivet(dc, x+gw/2, y-10, 5, cardstyle.N(46, 42, 38, 255))
		dc.SetColor(cardstyle.N(120, 112, 104, 170))
		dc.SetLineWidth(1.4)
		dc.DrawLine(x+gw/2, y-8, x+gw/2, y+4)
		dc.Stroke()
		base := cardstyle.N(226, 192, 142, 255)
		if empty {
			base = cardstyle.N(190, 158, 112, 255)
		}
		dc.SetColor(cardstyle.N(36, 22, 10, 255))
		dc.DrawRoundedRectangle(x-4, y, gw+8, gh, 8)
		dc.Fill()
		dc.SetColor(base)
		dc.DrawRoundedRectangle(x, y+4, gw, gh-8, 6)
		dc.Fill()
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 12, slotCode, x+16, y+28, cardstyle.N(150, 104, 56, 255), 0, 0.5, 90, 8)
		for t := 0; t < s.Tier && t < 5; t++ {
			cardstyle.Diamond(dc, x+gw-22-float64(t)*18, y+28, 5, cardstyle.N(150, 78, 24, 255))
		}
		name := "EMPTY PEG"
		ncol := cardstyle.N(120, 90, 60, 220)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = cardstyle.N(52, 34, 20, 255)
		}
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.TruncateRunes(name, 20), x+16, y+70, ncol, 0, 0.5, gw-32, 10)
		if !empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			cardstyle.NotchMeter(dc, x+16, y+gh-44, gw-32, 10, int(frac*10), cardstyle.N(70, 46, 26, 150), p.Accent)
			cardstyle.Text(dc, cardstyle.FtMedieval, 11, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+16, y+gh-20, cardstyle.N(116, 82, 50, 240), 0, 0.5, gw-32, 8)
		}
	}
	return dc
}

// SHOP - market board.
func s04Shop(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	y := s04Signboard(dc, W, 26, 400, 60, "THE MARKET", styledName(req), p)
	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	ph := H - y - 80
	s04Page(dc, 44, y, W-88, ph, p)
	iy := y + 40.0
	for _, e := range entries {
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(e.Title), 84, iy, cardstyle.N(52, 34, 20, 255), 0, 0.5, 380, 11)
		if e.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtMedieval, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 66), 84, iy+22, cardstyle.N(150, 104, 56, 240), 0, 0.5, 400, 8)
		}
		if e.Runes != "" {
			cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, cardstyle.Sanitize(e.Runes), 84, iy+44, cardstyle.N(150, 78, 24, 255), 0, 2)
		}
		cardstyle.DotLeader(dc, 420, W-170, iy, 1.1, cardstyle.N(150, 104, 56, 150))
		cardstyle.Text(dc, cardstyle.FtCinzelDec, 18, cardstyle.Sanitize(e.Value), W-84, iy, cardstyle.N(52, 34, 20, 255), 1, 0.5, 140, 10)
		iy += 68
	}
	s04Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - inn sign crest.
func s04GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s04Base(dc, W, H)

	y := s04Signboard(dc, W, 26, 420, 60, styledName(req), "GUILD HOLD", p)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 64)+"\"", W/2, y+2, cardstyle.N(216, 170, 104, 230), 0.5, 0.5, W-120, 10)
	}

	// crest plaque
	cy := y + 120.0
	s04Page(dc, W/2-150, cy-70, 300, 190, p)
	drawHeroFitted(dc, req, W/2, cy+16, 200, 120, *p)
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 40, firstRuneUpper(styledName(req)), W/2, cy+52, cardstyle.N(150, 78, 24, 255), 0.5, 0.5, 90, 18)

	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	py := cy + 150.0
	ph := 46 + float64(len(rows))*36 + 66
	s04Page(dc, 44, py, W-88, ph, p)
	s04CarvedTitle(dc, 84, py+28, "THE CHARTER", p)
	ry := py + 60
	for _, r := range rows {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(r.Label)), 86, ry, cardstyle.N(52, 34, 20, 255), 0, 0.5, 240, 9)
		cardstyle.DotLeader(dc, 250, W-210, ry, 1, cardstyle.N(150, 104, 56, 140))
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), W-86, ry, cardstyle.N(52, 34, 20, 255), 1, 0.5, 200, 10)
		ry += 36
	}
	// level + buildings
	cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("LEVEL %d", req.Level), W/2, ry+14, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 160, 10)
	pct := float64(req.XPPercent) / 100
	dc.SetColor(cardstyle.N(70, 46, 26, 255))
	dc.DrawRoundedRectangle(W/2-140, ry+28, 280, 8, 4)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(W/2-140, ry+28, 280*pct, 8, 4)
	dc.Fill()
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		bx := W/2 + float64(i-1)*160
		cardstyle.Text(dc, cardstyle.FtMedieval, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 9), bx, ry+70, cardstyle.N(116, 82, 50, 240), 0.5, 0.5, 120, 8)
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("L%d", b.Level), bx, ry+92, cardstyle.N(150, 78, 24, 255), 0.5, 0.5, 60, 9)
	}
	s04Footer(dc, W, H, req, p)
	return dc
}
