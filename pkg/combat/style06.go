package combat

// style06.go - SOUL FORGE "Celestial Sanctum" (Style 6, owner anchor).
// The owner-provided Allocate card defines this system: deep indigo night
// sky + faint star field, antique-gold hairline frame with center diamonds,
// centered gold display title + letterspaced subtitle + thin rule, faint
// magic circles behind key numbers, glowing cyan numerals, stat chips with
// dotted leaders, rounded indigo panels, gold pill CTA, script caption.
// Recognizable by structure: everything orbits centered gold headings and
// dotted-leader rows inside rounded panels - never banner ribbons or plates.

import (
	"fmt"
	"image"
	"math"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

// ── Soul Forge base + language ────────────────────────────────────────

func s06Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.SoulForge()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	// faint diagonal light streaks (owner card)
	dc.Push()
	dc.Translate(W*0.32, H*0.5)
	dc.Rotate(-0.5)
	dc.SetColor(cardstyle.N(255, 255, 255, 7))
	dc.DrawRectangle(-30, -H, 60, H*2)
	dc.Fill()
	dc.Translate(W*0.42, 0)
	dc.DrawRectangle(-16, -H, 32, H*2)
	dc.Fill()
	dc.Pop()
	cardstyle.StarField(dc, 20, 20, W-20, H-20, 46, 0x50F6CE, cardstyle.N(255, 255, 255, 255))
	// gold hairline frame + top/bottom center diamonds
	cardstyle.FrameHairline(dc, 0, 0, W, H, 12, 1.4, p.PanelEd, false)
	cardstyle.Diamond(dc, W/2, 12, 6, p.PanelEd)
	cardstyle.Diamond(dc, W/2, H-12, 6, p.PanelEd)
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s06TitleBlock - gold display title + letterspaced subtitle + rule.
func s06TitleBlock(dc *gg.Context, W float64, y float64, title, subtitle string, big float64) float64 {
	p := cardstyle.SoulForge()
	cardstyle.Text(dc, cardstyle.FtCinzelDec, int(big), cardstyle.Sanitize(title), W/2, y, p.Accent, 0.5, 0.5, W-90, 18)
	y += big * 0.78
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(subtitle)), W/2, y, p.Muted, 0.5, 5)
	y += 16
	cardstyle.DotLeader(dc, W/2-120, W/2+120, y, 1.1, cardstyle.WithA(p.Accent, 150))
	cardstyle.Diamond(dc, W/2, y, 3.4, p.Accent)
	return y + 22
}

// s06PanelTitle - letterspaced panel heading with short underline.
func s06PanelTitle(dc *gg.Context, cx, y float64, text string) {
	p := cardstyle.SoulForge()
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 16, strings.ToUpper(cardstyle.Sanitize(text)), cx, y, p.Accent, 0.5, 3)
	cardstyle.DotLeader(dc, cx-70, cx+70, y+15, 1, cardstyle.WithA(p.Accent, 120))
}

// s06Footer - script caption + tiny diamond.
func s06Footer(dc *gg.Context, W, H float64, caption string) {
	p := cardstyle.SoulForge()
	if caption == "" {
		return
	}
	c := cardstyle.TruncateRunes(cardstyle.Sanitize(caption), 60)
	cardstyle.FooterNote(dc, W/2, H-32, c, p.Caption, cardstyle.FtMedieval, 15, W-110)
}

// ── ALLOCATE: faithful reproduction of the owner card ─────────────────

func s06Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	y := s06TitleBlock(dc, W, 62, "SOUL FORGE", "ATTRIBUTE ALLOCATION", 40)

	// class line ("SCOUT · STARTER TIER")
	classLine := strings.ToUpper(cardstyle.Sanitize(req.Pill))
	if strings.Contains(strings.ToLower(classLine), "starter") && !strings.Contains(strings.ToLower(classLine), "tier") {
		classLine += " TIER"
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, classLine, W/2, y+6, p.Muted, 0.5, 2.4)

	// magic circle + glowing unspent number
	cardstyle.MagicCircle(dc, W/2, 300, 88, p.Accent)
	avail := parseLeadingInt(req.PointsBig, 0)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 68, fmt.Sprintf("%d", avail), W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 160, 34)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, "UNSPENT POINTS", W/2, 212, p.Ink, 0.5, 3.4)

	// THE SEVEN PATHS panel
	px, py, pw := 56.0, 402.0, W-112
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	ph := 74 + float64(len(rows))*44 + 40
	cardstyle.PanelRounded(dc, px, py, pw, ph, 14, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.2)
	s06PanelTitle(dc, W/2, py+26, "THE SEVEN PATHS")

	ry := py + 62
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		cardstyle.Chip(dc, px+18, ry-11, 58, 22, code, cardstyle.WithA(p.Track, 235), cardstyle.WithA(p.Accent2, 170), p.Accent2, cardstyle.FtInter, 11)
		cardstyle.Text(dc, cardstyle.FtMedieval, 19, cardstyle.StatFullName(code), px+78, ry, p.Ink, 0, 0.5, 180, 12)
		cardstyle.DotLeader(dc, px+262, px+pw-76, ry, 1.2, cardstyle.WithA(p.Muted, 110))
		cardstyle.GlowText(dc, cardstyle.FtCinzel, 21, gainFromSub(r.Sub), px+pw-20, ry, p.Accent2, p.Accent2, 1, 0.5, 92, 13)
		ry += 44
	}
	// SPENT row
	spent := parseLeadingInt(req.SpentNow, 0)
	cardstyle.DotLeader(dc, px+22, px+pw-96, ry+4, 1, cardstyle.WithA(p.Muted, 70))
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, "SPENT", px+70, ry+4, p.Muted, 0.5, 2.6)
	cardstyle.Text(dc, cardstyle.FtCinzel, 17, fmt.Sprintf("%d pts", spent), px+pw-20, ry+4, p.Accent, 1, 0.5, 110, 11)

	// footnote + CTA pill + caption
	note := "Half value per point after heavy single-stat investment"
	cardstyle.FooterNote(dc, W/2, py+ph+24, note, p.Muted, cardstyle.FtMedieval, 13, pw-20)
	cta, ctaSub := req.CtaLabel, req.CtaSub
	if cta == "" {
		cta = ".j allocate <stat> [amount]"
	}
	if ctaSub == "" {
		ctaSub = "e.g. .j allocate ATK 5  ·  .j allocate HP 3"
	}
	cardstyle.PillCTA(dc, W/2, py+ph+78, 400, 46, cta, ctaSub, cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 200), cardstyle.N(42, 35, 24, 255), cardstyle.N(58, 48, 30, 255), cardstyle.FtCinzel, 16)
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── RANK: "ASCENSION RECORD" ──────────────────────────────────────────

func s06Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)
	name := styledName(req)

	s06TitleBlock(dc, W, 58, name, "ASCENSION RECORD", 30)

	// magic circle + glowing level
	cardstyle.MagicCircle(dc, W/2, 268, 82, p.Accent)
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 62, fmt.Sprintf("%d", req.Level), W/2, 264, p.Accent2, p.Accent2, 0.5, 0.5, 150, 30)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "LEVEL", W/2, 348, p.Muted, 0.5, 4)

	// rank pill
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK ADVENTURER"
	}
	if rankLine != "" {
		cardstyle.PillCTA(dc, W/2, 392, 300, 36, rankLine, "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 14)
	}

	// EXPERIENCE dotted-leader + bar
	ey := 442.0
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "EXPERIENCE", 84, ey, p.Muted, 0, 2.6)
	xpTxt := fmt.Sprintf("%s / %s", cardstyle.Sanitize(req.XPNow), cardstyle.Sanitize(req.XPLeft))
	cardstyle.GlowText(dc, cardstyle.FtCinzel, 15, xpTxt, W-84, ey, p.Accent2, p.Accent2, 1, 0.5, 200, 10)
	pct := float64(req.XPPercent) / 100
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	cardstyle.MeterBar(dc, 84, ey+14, W-168, 12, pct, p.Track, p.Accent2, cardstyle.WithA(p.Accent2, 90), cardstyle.WithA(p.Accent, 130), 6)

	// THE RECORD panel (standing rows as chip + dotted leader)
	ry := 496.0
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	ph := 48 + float64(len(rows))*40 + 14
	cardstyle.PanelRounded(dc, 56, ry, W-112, ph, 14, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.2)
	s06PanelTitle(dc, W/2, ry+24, "THE RECORD")
	ry += 52
	for _, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		if len([]rune(code)) > 4 {
			code = string([]rune(code)[:4])
		}
		cardstyle.Chip(dc, 74, ry-11, 48, 22, code, cardstyle.WithA(p.Track, 235), cardstyle.WithA(p.Accent2, 170), p.Accent2, cardstyle.FtInter, 11)
		cardstyle.Text(dc, cardstyle.FtMedieval, 17, cardstyle.StatFullName(strings.ToUpper(cardstyle.Sanitize(r.Label))), 134, ry, p.Ink, 0, 0.5, 170, 11)
		cardstyle.DotLeader(dc, 300, W-104, ry, 1.2, cardstyle.WithA(p.Muted, 110))
		cardstyle.Text(dc, cardstyle.FtCinzel, 17, cardstyle.Sanitize(r.Value), W-84, ry, p.Accent2, 1, 0.5, 150, 11)
		ry += 40
	}

	// THE PATH AHEAD (progress bars)
	py := ry + 22
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	if len(prog) > 0 {
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), 84, py, p.Accent, 0, 2.6)
		py += 22
		for _, pr := range prog {
			frac := 0.0
			if pr.Max > 0 {
				frac = float64(pr.Cur) / float64(pr.Max)
			}
			if pr.Done {
				frac = 1
			}
			cardstyle.Text(dc, cardstyle.FtInter, 13, cardstyle.Sanitize(pr.Label), 84, py+8, p.Ink, 0, 0.5, 210, 10)
			cardstyle.Text(dc, cardstyle.FtCinzel, 13, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), W-84, py+8, p.Accent2, 1, 0.5, 150, 10)
			fill := p.Accent2
			if pr.Done {
				fill = p.Done
			}
			cardstyle.MeterBar(dc, 84, py+18, W-168, 9, frac, p.Track, fill, cardstyle.WithA(fill, 90), cardstyle.WithA(p.Accent, 110), 4.5)
			py += 46
		}
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── SKILLUP: "RITUAL OF MASTERY" ──────────────────────────────────────

func s06Skillup(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	s06TitleBlock(dc, W, 58, "RITUAL OF MASTERY", styledName(req), 30)

	// sigil circle + glowing initials
	cardstyle.MagicCircle(dc, W/2, 300, 92, p.Accent)
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 66, initials, W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 150, 30)
	// orbit pips (skill level)
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
	totalW := float64(maxPips-1) * 26
	sx := W/2 - totalW/2
	for i := 0; i < maxPips; i++ {
		if i < cur {
			cardstyle.Diamond(dc, sx+float64(i)*26, 414, 6.5, p.Accent2)
		} else {
			cardstyle.DiamondOutline(dc, sx+float64(i)*26, 414, 5.5, cardstyle.WithA(p.Muted, 150), 1.2)
		}
	}

	// skill name + tier
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, 470, p.Accent, 0.5, 0.5, W-90, 14)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, tierLabel, W/2, 502, p.Muted, 0.5, 3)

	// mastery meter panel
	py, ph := 546.0, 148.0
	cardstyle.PanelRounded(dc, 56, py, W-112, ph, 14, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.2)
	s06PanelTitle(dc, W/2, py+26, "MASTERY")
	frac := 0.0
	if maxPips > 0 {
		frac = float64(cur) / float64(maxPips)
	}
	cardstyle.MeterBar(dc, 96, py+62, W-192, 12, frac, p.Track, p.Accent2, cardstyle.WithA(p.Accent2, 90), cardstyle.WithA(p.Accent, 130), 6)
	cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("%d / %d", cur, maxPips), W/2, py+96, p.Ink, 0.5, 0.5, 140, 10)

	// CTA
	cta, ctaSub := req.CtaLabel, req.CtaSub
	if cta == "" {
		cta = ".j skill up"
	}
	if ctaSub == "" {
		ctaSub = "grow the ritual - tier by tier"
	}
	cardstyle.PillCTA(dc, W/2, 760, 340, 44, cta, ctaSub, cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), cardstyle.N(58, 48, 30, 255), cardstyle.FtCinzel, 15)
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── ABILITIES: "CODEX OF STARS" ───────────────────────────────────────

func s06Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	// header: doc title + quote + page label
	title := strings.TrimSpace(req.DocTitle)
	if title == "" {
		title = "CODEX OF STARS"
	}
	cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, cardstyle.Sanitize(title), W/2, 62, p.Accent, 0.5, 0.5, W-200, 18)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 16, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 74)+"\"", W/2, 96, p.Muted, 0.5, 0.5, W-160, 10)
	}
	if req.PageLabel != "" {
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-46, 40, p.Accent2, 1, 2)
	}
	cardstyle.DotLeader(dc, W/2-140, W/2+140, 122, 1.1, cardstyle.WithA(p.Accent, 150))

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	y := 158.0
	for gi, g := range req.Groups {
		if gi >= 6 {
			break
		}
		gh := 40 + float64(len(g.Items))*56 + 8
		cardstyle.PanelRounded(dc, 56, y, W-112, gh, 12, cardstyle.WithA(p.Panel, 210), cardstyle.WithA(p.PanelEd, 120), 1)
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 15, strings.ToUpper(cardstyle.Sanitize(g.Name)), W/2, y+22, p.Accent, 0.5, 2.6)
		iy := y + 52
		for _, it := range g.Items {
			cardstyle.Chip(dc, 76, iy-12, 40, 24, fmt.Sprintf("%02d", num), cardstyle.WithA(p.Track, 235), cardstyle.WithA(p.Accent2, 160), p.Accent2, cardstyle.FtInter, 11)
			cardstyle.Text(dc, cardstyle.FtMedieval, 19, cardstyle.Sanitize(it.Title), 132, iy-6, p.Ink, 0, 0.5, 430, 12)
			if it.Sub != "" {
				cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(it.Sub), 60), 132, iy+16, p.Muted, 0, 0.5, 430, 9)
			}
			if it.Runes != "" {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 16, cardstyle.Sanitize(it.Runes), W-96, iy-6, p.Accent2, 1, 3)
			}
			if it.Value != "" {
				cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(it.Value), W-96, iy+16, p.Muted, 1, 0.5, 240, 9)
			}
			num++
			iy += 56
		}
		y += gh + 12
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── SKILLTREE: ORBITAL RINGS (tiers = rings, branches = arc sectors) ──

func s06Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1000.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzelDec, 28, "SOUL CONSTELLATION", W/2, 56, p.Accent, 0.5, 3)
	classLine := strings.ToUpper(cardstyle.Sanitize(req.ClassName))
	if req.Level > 0 {
		classLine = fmt.Sprintf("%s - LV %d", classLine, req.Level)
	}
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, classLine, W/2, 90, p.Muted, 0.5, 2.6)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.PillCTA(dc, W-130, 62, 190, 34, ptsText, "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 12)

	// orbital field
	cx, cy := W/2, 566.0
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
	ringStep := 92.0
	r0 := 92.0
	for t := 1; t <= maxTier; t++ {
		r := r0 + float64(t-1)*ringStep
		cardstyle.Ring(dc, cx, cy, r, cardstyle.WithA(p.Accent, 55), 1.1)
		la := -math.Pi / 4
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("TIER %d", t), cx+(r+16)*math.Cos(la), cy+(r+16)*math.Sin(la), cardstyle.WithA(p.Muted, 200), 0.5, 2)
	}
	// core star
	cardstyle.MagicCircle(dc, cx, cy, 56, p.Accent)
	cardstyle.Star8(dc, cx, cy, 20, p.Accent2)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "CORE", cx, cy+34, p.Muted, 0.5, 2)

	sector := 2 * math.Pi / float64(n)
	for bi, br := range req.Branches {
		mid := -math.Pi/2 + float64(bi)*sector + sector/2 // offset: no label straight up
		// branch label chip at outer edge of its sector
		lr := r0 + float64(maxTier-1)*ringStep + 48
		lx, ly := cx+lr*math.Cos(mid), cy+lr*math.Sin(mid)
		if ly > H-120 {
			ly = H - 120
		}
		if ly < 140 {
			ly = 140
		}
		cardstyle.Chip(dc, lx-84, ly-14, 168, 28, strings.ToUpper(cardstyle.Sanitize(br.Name)), cardstyle.WithA(p.Panel, 235), cardstyle.WithA(p.Accent, 170), p.Accent, cardstyle.FtCinzel, 12)
		// faint arc spine from core ring to outer ring at the sector mid
		ir := 62.0
		dc.SetColor(cardstyle.WithA(p.Accent, 70))
		dc.SetLineWidth(1)
		dc.DrawLine(cx+ir*math.Cos(mid), cy+ir*math.Sin(mid), cx+(lr-30)*math.Cos(mid), cy+(lr-30)*math.Sin(mid))
		dc.Stroke()
		// nodes on rings
		order := []int{}
		nodesByTier := map[int][]portraitNode{}
		for _, s := range br.Skills {
			tier := s.Tier
			if tier < 1 {
				tier = 1
			}
			if tier > maxTier {
				tier = maxTier
			}
			if _, ok := nodesByTier[tier]; !ok {
				order = append(order, tier)
			}
			nodesByTier[tier] = append(nodesByTier[tier], s)
		}
		for _, tier := range order {
			list := nodesByTier[tier]
			r := r0 + float64(tier-1)*ringStep
			for j, s := range list {
				off := (float64(j) - float64(len(list)-1)/2) * (sector * 0.30)
				a := mid + off
				x, yy := cx+r*math.Cos(a), cy+r*math.Sin(a)
				s06StarNode(dc, x, yy, s.State, p)
			}
		}
	}

	// legend
	ly := H - 64.0
	leg := []struct {
		label string
		state string
	}{{"LEARNED", "learned"}, {"MAXED", "maxed"}, {"OPEN", "open"}, {"LOCKED", "locked"}}
	lx := W/2 - 240.0
	for _, l := range leg {
		s06StarNode(dc, lx, ly, l.state, p)
		cardstyle.Text(dc, cardstyle.FtInter, 12, l.label, lx+18, ly, p.Muted, 0, 0.5, 90, 9)
		lx += 130
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

func s06StarNode(dc *gg.Context, x, y float64, state string, p *cardstyle.Palette) {
	switch state {
	case "learned":
		cardstyle.Star4(dc, x, y, 13, p.Accent2)
		cardstyle.Ring(dc, x, y, 17, cardstyle.WithA(p.Accent2, 110), 1)
	case "maxed":
		cardstyle.Star8(dc, x, y, 15, p.Accent)
		cardstyle.Ring(dc, x, y, 20, cardstyle.WithA(p.Accent, 150), 1.2)
	case "open":
		cardstyle.Star4(dc, x, y, 9, cardstyle.WithA(p.Ink, 190))
	default:
		cardstyle.Ring(dc, x, y, 8, cardstyle.WithA(p.Muted, 110), 1.2)
		cardstyle.Diamond(dc, x, y, 2.6, cardstyle.WithA(p.Muted, 130))
	}
}

// ── EQUIP: "RELIC VAULT" (hero panel + 9 relic panels) ────────────────

func s06Equip(req *portraitRequest) *gg.Context {
	const W, H = 1500.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	cardstyle.Text(dc, cardstyle.FtCinzelDec, 34, "RELIC VAULT", W/2, 58, p.Accent, 0.5, 0.5, 520, 18)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 92, p.Muted, 0.5, 3)
	if req.SealText != "" {
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", W-60, 46, p.Accent2, 1, 2)
	}

	// hero panel (left)
	cardstyle.PanelRounded(dc, 60, 150, 400, 720, 16, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.3)
	drawHeroFitted(dc, req, 260, 440, 300, 340, *p)
	cardstyle.DotLeader(dc, 120, 400, 640, 1.1, cardstyle.WithA(p.Accent, 130))
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 17, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 260, 668, p.Ink, 0.5, 2.4)
	cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 60), 260, 700, p.Muted, 0.5, 0.5, 340, 10)

	// 9 relic panels (3x3)
	gx, gy, gw, gh, gap := 500.0, 150.0, 300.0, 222.0, 20.0
	for i, s := range req.Slots {
		col, row := i%3, i/3
		x := gx + float64(col)*(gw+gap)
		y := gy + float64(row)*(gh+gap)
		empty := s.Empty
		fill := p.Panel
		if empty {
			fill = cardstyle.WithA(p.Track, 140)
		}
		cardstyle.PanelRounded(dc, x, y, gw, gh, 14, fill, cardstyle.WithA(p.PanelEd, cardstyleIfByte(empty, 90, 150)), 1.2)
		slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
		if len([]rune(slotCode)) > 4 {
			slotCode = string([]rune(slotCode)[:4])
		}
		cardstyle.Chip(dc, x+14, y+14, 56, 24, slotCode, cardstyle.WithA(p.Track, 235), cardstyle.WithA(p.Accent2, 160), p.Accent2, cardstyle.FtInter, 11)
		// tier diamonds
		for t := 0; t < s.Tier && t < 5; t++ {
			cardstyle.Diamond(dc, x+gw-24-float64(t)*20, y+26, 6, p.Accent)
		}
		if s.TierLabel != "" {
			cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), x+gw-16, y+48, p.Muted, 1, 0.5, 120, 8)
		}
		name := "EMPTY"
		ncol := cardstyle.WithA(p.Muted, 170)
		if !empty {
			name = cardstyle.Sanitize(s.Name)
			ncol = p.Ink
		}
		cardstyle.Text(dc, cardstyle.FtMedieval, 18, name, x+16, y+70, ncol, 0, 0.5, gw-32, 11)
		if !empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			cardstyle.MeterBar(dc, x+16, y+gh-40, gw-32, 10, frac, p.Track, p.Accent2, cardstyle.WithA(p.Accent2, 90), cardstyle.WithA(p.Accent, 110), 5)
			cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+16, y+gh-16, p.Muted, 0, 0.5, gw-32, 8)
		} else if !empty {
			cardstyle.Text(dc, cardstyle.FtInter, 12, "SOLID", x+16, y+gh-24, cardstyle.WithA(p.Accent2, 200), 0, 0.5, gw-32, 9)
		}
	}
	return dc
}

// ── SHOP: "WARDROBE OF WONDERS" ───────────────────────────────────────

func s06Shop(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	y := s06TitleBlock(dc, W, 58, "WARDROBE", "OF WONDERS", 34)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, y, p.Muted, 0.5, 2.6)

	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	ey := 190.0
	for _, e := range entries {
		eh := 92.0
		cardstyle.PanelRounded(dc, 56, ey, W-112, eh, 12, cardstyle.WithA(p.Panel, 215), cardstyle.WithA(p.PanelEd, 120), 1)
		cardstyle.Text(dc, cardstyle.FtMedieval, 19, cardstyle.Sanitize(e.Title), 76, ey+30, p.Ink, 0, 0.5, 400, 12)
		if e.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 70), 76, ey+56, p.Muted, 0, 0.5, 420, 9)
		}
		if e.Runes != "" {
			cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(e.Runes), 76, ey+76, p.Accent2, 0, 2.4)
		}
		cardstyle.DotLeader(dc, 420, W-150, ey+30, 1.2, cardstyle.WithA(p.Muted, 100))
		cardstyle.Text(dc, cardstyle.FtCinzel, 20, cardstyle.Sanitize(e.Value), W-76, ey+30, p.Accent, 1, 0.5, 140, 12)
		ey += eh + 14
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── GUILDINFO: "SANCTUM CHARTER" ──────────────────────────────────────

func s06GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 800.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	y := s06TitleBlock(dc, W, 48, "SANCTUM CHARTER", "GUILD RECORD", 26)

	// sigil ring crest with guild initial
	name := styledName(req)
	cardstyle.MagicCircle(dc, W/2, y+92, 74, p.Accent)
	initial := initialLetters(name)
	if initial == "" {
		initial = "G"
	}
	initial = string([]rune(initial)[:1])
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 52, initial, W/2, y+88, p.Accent2, p.Accent2, 0.5, 0.5, 90, 24)
	cardstyle.Text(dc, cardstyle.FtCinzel, 24, name, W/2, y+196, p.Accent, 0.5, 0.5, W-140, 14)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 64)+"\"", W/2, y+226, p.Muted, 0.5, 0.5, W-120, 10)
	}

	// charter rows
	ry := y + 262.0
	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	for _, r := range rows {
		cardstyle.Text(dc, cardstyle.FtInter, 14, cardstyle.Sanitize(r.Label), 90, ry, p.Muted, 0, 0.5, 240, 10)
		cardstyle.DotLeader(dc, 250, W-220, ry, 1.2, cardstyle.WithA(p.Muted, 100))
		cardstyle.Text(dc, cardstyle.FtCinzel, 15, cardstyle.Sanitize(r.Value), W-90, ry, p.Accent2, 1, 0.5, 200, 10)
		ry += 40
	}

	// level + xp
	cardstyle.PillCTA(dc, 130, ry+24, 110, 34, fmt.Sprintf("LVL %d", req.Level), "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 14)
	pct := float64(req.XPPercent) / 100
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	cardstyle.MeterBar(dc, 210, ry+24, W-300, 12, pct, p.Track, p.Accent2, cardstyle.WithA(p.Accent2, 90), cardstyle.WithA(p.Accent, 130), 6)

	// buildings as sigil circles
	bx := W / 2
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		cxx := bx + float64(i-1)*170
		cardstyle.Ring(dc, cxx, ry+96, 36, cardstyle.WithA(p.Accent, 150), 1.4)
		cardstyle.Ring(dc, cxx, ry+96, 28, cardstyle.WithA(p.Accent, 70), 0.8)
		cardstyle.Text(dc, cardstyle.FtCinzel, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 8), cxx, ry+82, p.Muted, 0.5, 0.5, 60, 7)
		cardstyle.GlowText(dc, cardstyle.FtCinzel, 20, fmt.Sprintf("L%d", b.Level), cxx, ry+102, p.Accent2, p.Accent2, 0.5, 0.5, 54, 10)
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// renderStyle06 dispatches Soul Forge compositions.
func renderStyle06(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s06Rank(req).Image()
	case "ALLOCATE":
		return s06Allocate(req).Image()
	case "SKILLUP":
		return s06Skillup(req).Image()
	case "ABILITIES":
		return s06Abilities(req).Image()
	case "SKILLTREE":
		return s06Skilltree(req).Image()
	case "EQUIP":
		return s06Equip(req).Image()
	case "SHOP":
		return s06Shop(req).Image()
	case "GUILDINFO":
		return s06GuildInfo(req).Image()
	}
	return nil
}

// cardstyleIfByte returns a or b by condition (tiny helper for readability).
func cardstyleIfByte(cond bool, a, b uint8) uint8 {
	if cond {
		return a
	}
	return b
}
