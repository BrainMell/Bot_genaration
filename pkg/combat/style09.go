package combat

// style09.go - RUNE MONOLITH "The Basalt Codex" (Style 9).
// Ancient carved stone: content ENGRAVED into basalt (dark inset + light
// offset), carved glyph columns down the margins, carved glyph bands as
// separators, recessed panels, ember glow behind key values.
// Recognizable by structure: engraving + glyph margins + carved bands.

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
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	cardstyle.Speckle(dc, 0, 0, W, H, 110, 0x9E11A7, cardstyle.N(0, 0, 0, 255))
	cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2.4, cardstyle.Darken(p.Panel, 40), false)
	cardstyle.FrameHairline(dc, 0, 0, W, H, 16, 1, cardstyle.WithA(p.Accent, 90), false)
	// carved glyph margins
	cardstyle.GlyphCol(dc, 28, 60, H-60, 56, cardstyle.N(255, 255, 255, 14))
	cardstyle.GlyphCol(dc, W-40, 60, H-60, 56, cardstyle.N(255, 255, 255, 14))
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s09Header - engraved title with carved band.
func s09Header(dc *gg.Context, W, y float64, title, sub string, p *cardstyle.Palette) float64 {
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 27, strings.ToUpper(cardstyle.Sanitize(title)), W/2, y, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-130, 14)
	if sub != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(sub)), W/2, y+26, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-130, 8)
	}
	// carved band: grooves + ember ticks
	dc.SetColor(cardstyle.Darken(p.Panel, 56))
	dc.SetLineWidth(3)
	dc.DrawLine(70, y+46, W-70, y+46)
	dc.Stroke()
	dc.SetColor(cardstyle.Lighten(p.Panel, 26))
	dc.SetLineWidth(1)
	dc.DrawLine(70, y+49, W-70, y+49)
	dc.Stroke()
	dc.SetColor(p.Accent)
	for i := 0; i < 5; i++ {
		tx := W/2 - 100 + float64(i)*50
		dc.SetLineWidth(2)
		dc.DrawLine(tx, y+42, tx+6, y+50)
		dc.Stroke()
	}
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

func s09EngravedRow(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette, size int) {
	cardstyle.Engrave(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(label)), x, y, p.Ink, cardstyle.Darken(p.Panel, 60), 0, 0.5, w*0.55, 8)
	cardstyle.DotLeader(dc, x+w*0.58, x+w-cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(value)), 0)-14, y+1, 1, cardstyle.WithA(p.Muted, 100))
	cardstyle.Engrave(dc, cardstyle.FtCinzel, size, strings.ToUpper(cardstyle.Sanitize(value)), x+w, y, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, w*0.4, 8)
}

func s09Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	cardstyle.EmberGlow(dc, 58, H-44, 42, p.Accent)
	cardstyle.SealMini(dc, 58, H-44, 19, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Caption)), 56), 96, H-44, cardstyle.WithA(p.Ink, 210), 0, 0.5, W-160, 9)
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

// RANK - engraved tablet with ember level.
func s09Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 56, styledName(req), "THE STANDING STONE", p)

	// ember level in recessed circle
	s09Recess(dc, W/2-110, y, 220, 190, p)
	cardstyle.EmberGlow(dc, W/2, y+92, 80, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 56, fmt.Sprintf("%d", req.Level), W/2, y+86, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 140, 22)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "LEVEL", W/2, y+152, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 8)
	y += 206
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, rankLine, W/2, y, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-120, 10)
		y += 30
	}

	// xp groove
	pct := float64(req.XPPercent) / 100
	dc.SetColor(cardstyle.Darken(p.Track, 10))
	dc.DrawRoundedRectangle(80, y, W-160, 10, 5)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(80, y, (W-160)*pct, 10, 5)
	dc.Fill()
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "XP "+cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W/2, y+26, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-140, 8)
	y += 56

	s09Recess(dc, 60, y, W-120, 60+float64(len(req.Standing))*40+8, p)
	ry := y + 40.0
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	for _, r := range rows {
		s09EngravedRow(dc, 88, ry, W-176, r.Label, r.Value, p, 13)
		ry += 40
	}
	y = ry + 24
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), 80, y, p.Muted, cardstyle.Darken(p.Panel, 60), 0, 0.5, 320, 9)
	y += 24
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
		s09EngravedRow(dc, 80, y, W-160, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), p, 12)
		dc.SetColor(cardstyle.Darken(p.Track, 10))
		dc.DrawRoundedRectangle(80, y+13, W-160, 7, 3.5)
		dc.Fill()
		fill := p.Accent
		if pr.Done {
			fill = p.Done
		}
		dc.SetColor(fill)
		dc.DrawRoundedRectangle(80, y+13, (W-160)*frac, 7, 3.5)
		dc.Fill()
		y += 44
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - carved allocation tablet.
func s09Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 56, "ALLOCATION", styledName(req), p)

	s09Recess(dc, W/2-120, y, 240, 180, p)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.EmberGlow(dc, W/2, y+86, 80, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 56, fmt.Sprintf("%d", avail), W/2, y+80, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 140, 22)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, "UNSPENT POINTS", W/2, y+144, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 200, 8)
	y += 196
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.Pill)), W/2, y, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-140, 9)
	y += 34

	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	s09Recess(dc, 60, y, W-120, 40+float64(len(rows))*40+52, p)
	ry := y + 34.0
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		s09EngravedRow(dc, 88, ry, W-176, cardstyle.StatFullName(code), gainFromSub(r.Sub), p, 13)
		ry += 40
	}
	ry += 6
	dc.SetColor(cardstyle.Darken(p.Panel, 56))
	dc.SetLineWidth(2)
	dc.DrawLine(88, ry, W-88, ry)
	dc.Stroke()
	ry += 24
	spent := parseLeadingInt(req.SpentNow, 0)
	s09EngravedRow(dc, 88, ry, W-176, "SPENT", fmt.Sprintf("%d PTS", spent), p, 13)

	s09Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - carved rune medallion.
func s09Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 56, "MASTERY RITE", styledName(req), p)

	// rune socket medallion
	s09Recess(dc, W/2-120, y, 240, 220, p)
	cardstyle.Ring(dc, W/2, y+110, 86, cardstyle.WithA(p.Accent, 110), 2)
	cardstyle.Ring(dc, W/2, y+110, 74, cardstyle.WithA(p.Accent, 60), 1)
	// carved runes around the ring
	for i := 0; i < 8; i++ {
		a := float64(i) * 0.7854
		gx, gy := W/2+98*math.Cos(a), y+110+98*math.Sin(a)
		dc.SetColor(cardstyle.WithA(p.Ink, 120))
		dc.SetLineWidth(2)
		dc.DrawLine(gx-4, gy-6, gx+4, gy+6)
		dc.Stroke()
		dc.DrawLine(gx+4, gy-6, gx-4, gy)
		dc.Stroke()
	}
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.EmberGlow(dc, W/2, y+104, 70, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 52, initials, W/2, y+100, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 130, 20)
	y += 240

	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, y, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-110, 12)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(tierLabel), W/2, y+28, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-120, 9)
	y += 60

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
	s09Recess(dc, 60, y, W-120, 130, p)
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "THE TALLY", 88, y+30, p.Muted, cardstyle.Darken(p.Panel, 60), 0, 0.5, 200, 9)
	// rune socket pips
	totalW := float64(maxPips-1) * 40
	sx := W/2 - totalW/2
	for i := 0; i < maxPips; i++ {
		px := sx + float64(i)*40
		if i < cur {
			cardstyle.EmberGlow(dc, px, y+76, 26, p.Accent)
			cardstyle.Ring(dc, px, y+76, 12, p.Accent, 2.4)
			cardstyle.Diamond(dc, px, y+76, 5, p.Accent2)
		} else {
			cardstyle.Ring(dc, px, y+76, 10, cardstyle.WithA(p.Muted, 110), 1.6)
		}
	}
	cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, fmt.Sprintf("%d / %d", cur, maxPips), W/2, y+108, p.Ink, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 140, 9)
	s09Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - carved codex tablet.
func s09Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	title := strings.TrimSpace(req.DocTitle)
	if title == "" {
		title = "THE STONE CODEX"
	}
	y := s09Header(dc, W, 52, title, cardstyle.Sanitize(req.DocQuote), p)
	if req.PageLabel != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-70, 62, p.Accent2, cardstyle.Darken(p.Panel, 60), 1, 0.5, 180, 8)
	}

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		gh := 48 + float64(len(g.Items))*56 + 8
		s09Recess(dc, 52, y, W-104, gh, p)
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(g.Name)), 80, y+26, p.Accent2, cardstyle.Darken(p.Panel, 60), 0, 0.5, 420, 10)
		iy := y + 58
		for _, it := range g.Items {
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("%02d", num), 80, iy-4, cardstyle.WithA(p.Muted, 200), cardstyle.Darken(p.Panel, 60), 0, 0.5, 40, 8)
			cardstyle.Engrave(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(it.Title), 128, iy-6, p.Ink, cardstyle.Darken(p.Panel, 60), 0, 0.5, 430, 11)
			if it.Sub != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Sub)), 62), 128, iy+16, cardstyle.WithA(p.Muted, 210), 0, 0.5, 430, 8)
			}
			if it.Runes != "" {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(it.Runes)), W-86, iy-6, p.Accent2, 1, 2.4)
			}
			if it.Value != "" {
				cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(it.Value)), W-86, iy+16, p.Muted, cardstyle.Darken(p.Panel, 60), 1, 0.5, 200, 8)
			}
			num++
			iy += 56
		}
		y += gh + 14
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - PILLAR COLUMNS rising from a plinth.
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
		// pillar shaft
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
		// capital
		dc.SetColor(cardstyle.Lighten(p.Panel, 24))
		dc.DrawRectangle(cx-46, shaftTop-16, 92, 18)
		dc.Fill()
		// branch name engraved on the capital
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(br.Name), 12)), cx, shaftTop-34, p.Ink, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, colW-14, 9)
		// tier rune sockets down the shaft
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
				// socket
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
				// carve a rune glyph (zig)
				dc.DrawLine(nx-7, sy-9, nx+7, sy-9)
				dc.Stroke()
				dc.DrawLine(nx+7, sy-9, nx-7, sy)
				dc.Stroke()
				dc.DrawLine(nx-7, sy, nx+7, sy+9)
				dc.Stroke()
				cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 12), nx, sy-44, p.Ink, 0.5, 0.5, colW-10, 8)
				cardstyle.Text(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), nx, sy+40, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, 70, 7)
			}
		}
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// EQUIP - relic vault wall of carved niches.
func s09Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 44, "THE RELIC WALL", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

	// hero niche
	s09Recess(dc, 60, y+10, 380, 720, p)
	drawHeroFitted(dc, req, 250, y+340, 320, 340, *p)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, y+600, cardstyle.Lighten(p.Ink, 10), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 340, 12)
	if req.SealText != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", 250, y+636, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 240, 10)
	}
	cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Caption)), 52), 250, y+680, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, 340, 9)

	// 9 niches
	gx, gy, gw, gh, gap := 500.0, y+10, 300.0, 222.0, 20.0
	for i, s := range req.Slots {
		col, row := i%3, i/3
		x := gx + float64(col)*(gw+gap)
		yy := gy + float64(row)*(gh+gap)
		empty := s.Empty
		s09Recess(dc, x, yy, gw, gh, p)
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, slotCode, x+16, yy+26, p.Muted, cardstyle.Darken(p.Panel, 60), 0, 0.5, 110, 8)
		for t := 0; t < s.Tier && t < 5; t++ {
			dc.SetColor(p.Accent)
			dc.SetLineWidth(2.2)
			dc.DrawLine(x+gw-28-float64(t)*18, yy+20, x+gw-22-float64(t)*18, yy+32)
			dc.Stroke()
			dc.DrawLine(x+gw-22-float64(t)*18, yy+32, x+gw-28-float64(t)*18, yy+32)
			dc.Stroke()
		}
		name := "EMPTY NICHE"
		ncol := cardstyle.WithA(p.Muted, 140)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = cardstyle.Lighten(p.Ink, 10)
		}
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 16, cardstyle.TruncateRunes(strings.ToUpper(name), 20), x+16, yy+70, ncol, cardstyle.Darken(p.Panel, 60), 0, 0.5, gw-32, 10)
		if !empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			dc.SetColor(cardstyle.Darken(p.Track, 10))
			dc.DrawRoundedRectangle(x+16, yy+gh-42, gw-32, 8, 4)
			dc.Fill()
			dc.SetColor(p.Accent)
			dc.DrawRoundedRectangle(x+16, yy+gh-42, (gw-32)*frac, 8, 4)
			dc.Fill()
			cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+16, yy+gh-20, cardstyle.WithA(p.Muted, 220), 0, 0.5, gw-32, 8)
		}
	}
	return dc
}

// SHOP - carved market stela.
func s09Shop(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 50, "THE EXCHANGE", styledName(req), p)
	entries := req.Entries
	if len(entries) > 9 {
		entries = entries[:9]
	}
	for _, e := range entries {
		s09EngravedRow(dc, 70, y, W-140, e.Title, e.Value, p, 15)
		extra := e.Sub
		if e.Runes != "" {
			extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
		}
		if extra != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 88), 70, y+20, cardstyle.WithA(p.Muted, 200), 0, 0.5, W-140, 8)
		}
		y += 52
	}
	s09Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - carved chapter stela.
func s09GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s09Base(dc, W, H)

	y := s09Header(dc, W, 46, styledName(req), "GUILD STONE", p)
	if req.Motto != "" {
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, strings.ToUpper("\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 62)+"\""), W/2, y, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, W-120, 9)
	}
	y += 36

	// crest niche
	s09Recess(dc, W/2-110, y, 220, 170, p)
	cardstyle.EmberGlow(dc, W/2, y+82, 70, p.Accent)
	cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 52, firstRuneUpper(styledName(req)), W/2, y+76, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 120, 20)
	y += 192

	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	s09Recess(dc, 52, y, W-104, 36+float64(len(rows))*38+78, p)
	ry := y + 30.0
	for _, r := range rows {
		s09EngravedRow(dc, 80, ry, W-160, r.Label, r.Value, p, 12)
		ry += 38
	}
	ry += 4
	s09EngravedRow(dc, 80, ry, W-160, "LEVEL", fmt.Sprintf("%d", req.Level), p, 14)
	pct := float64(req.XPPercent) / 100
	dc.SetColor(cardstyle.Darken(p.Track, 10))
	dc.DrawRoundedRectangle(80, ry+13, W-160, 7, 3.5)
	dc.Fill()
	dc.SetColor(p.Accent)
	dc.DrawRoundedRectangle(80, ry+13, (W-160)*pct, 7, 3.5)
	dc.Fill()
	ry += 40
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		bx := 80 + float64(i)*((W-160)/3)
		cardstyle.Engrave(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 9), bx+(W-160)/6, ry, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, (W-160)/3-10, 8)
		cardstyle.Text(dc, cardstyle.FtCinzel, 14, fmt.Sprintf("L%d", b.Level), bx+(W-160)/6, ry+20, p.Accent2, 0.5, 0.5, 60, 9)
	}
	s09Footer(dc, W, H, req, p)
	return dc
}
