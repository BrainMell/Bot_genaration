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
	skillTitle := strings.ToUpper(cardstyle.Sanitize(req.DocTitle))
	if skillTitle == "" {
		skillTitle = "RITUAL OF MASTERY"
	}

	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	s06TitleBlock(dc, W, 58, skillTitle, styledName(req), 30)

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
	if req.SubLabel != "" {
		tierLabel = strings.ToUpper(cardstyle.Sanitize(req.SubLabel))
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
	// constellation filler in the open night field
	fy := (y + H - 80) / 2
	if fy > y+40 && H-80-fy > 40 {
		cardstyle.Ring(dc, W/2, fy, 46, cardstyle.WithA(p.Accent, 90), 0.9)
		cardstyle.Ring(dc, W/2, fy, 74, cardstyle.WithA(p.Accent, 50), 0.6)
		cardstyle.Star4(dc, W/2, fy, 9, cardstyle.Hex(0x9fe8ff))
		cardstyle.GlowText(dc, cardstyle.FtCinzel, 12, "SOUL CODEX", W/2, fy+104, cardstyle.WithA(p.Accent, 50), cardstyle.WithA(p.Muted, 180), 0.5, 0.5, 300, 9)
		for i := 0; i < 10; i++ {
			a := float64(i) * 2.399
			r := 74 + float64(i%3)*10
			cardstyle.Star4(dc, W/2+r*math.Cos(a), fy+r*math.Sin(a), 2.4, cardstyle.WithA(p.Accent, 150))
		}
	}
	s06Footer(dc, W, H, req.Caption)
	return dc
}

// ── SKILLTREE: ORBITAL RINGS (tiers = rings, branches = arc sectors) ──

func s06Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1000.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	cardstyle.Spaced(dc, cardstyle.FtCinzelDec, 28, "SOUL THREADS", W/2, 56, p.Accent, 0.5, 3)
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

	// three ritual bead-threads hanging from gold summoning rings
	n := len(req.Branches)
	if n == 0 {
		n = 1
	}
	slotW := W / float64(n)
	threadBot := 836.0
	for bi, br := range req.Branches {
		tx := slotW*float64(bi) + slotW/2
		// summoning ring the thread hangs from
		cardstyle.MagicCircle(dc, tx, 168, 34, p.Accent)
		cardstyle.Chip(dc, tx-86, 216, 172, 30, strings.ToUpper(cardstyle.Sanitize(br.Name)), cardstyle.WithA(p.Panel, 235), cardstyle.WithA(p.Accent, 170), p.Accent, cardstyle.FtCinzel, 12)
		// the thread (dotted gold hairline)
		for dyy := 252.0; dyy <= threadBot; dyy += 16 {
			cardstyle.Diamond(dc, tx, dyy, 1.6, cardstyle.WithA(p.Accent, 110))
		}
		skills := br.Skills
		if len(skills) > 4 {
			skills = skills[:4]
		}
		step := 108.0
		if len(skills) > 1 {
			step = (threadBot - 300) / float64(len(skills)-1)
			if step > 132 {
				step = 132
			}
		}
		for k, s := range skills {
			sy := 300.0 + float64(k)*step
			if len(skills) == 1 {
				sy = (300.0 + threadBot) / 2
			}
			// tier numeral to the left of the bead
			cardstyle.Text(dc, cardstyle.FtCinzel, 12, tierRoman(s.Tier), tx-40, sy, cardstyle.WithA(p.Muted, 210), 1, 0.5, 40, 8)
			// the bead
			s06StarNode(dc, tx, sy, s.State, p)
			if s.State == "maxed" {
				cardstyle.GlowText(dc, cardstyle.FtCinzel, 13, fmt.Sprintf("%d/%d", s.Cur, s.Max), tx+30, sy+22, p.Accent, p.Accent, 0, 0.5, 90, 8)
			} else {
				cardstyle.Text(dc, cardstyle.FtInter, 11, fmt.Sprintf("%d/%d", s.Cur, s.Max), tx+30, sy+22, cardstyle.WithA(p.Muted, 220), 0, 0.5, 90, 8)
			}
			cardstyle.Text(dc, cardstyle.FtMedieval, 17, cardstyle.TruncateRunes(cardstyle.Sanitize(s.Name), 18), tx+30, sy-8, p.Ink, 0, 0.5, slotW-110, 10)
		}
		// soul flame terminus at the thread's foot
		cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 16, "*", tx, threadBot+22, cardstyle.WithA(p.Accent2, 200), cardstyle.WithA(p.Accent2, 230), 0.5, 0.5, 40, 8)
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

// tierRoman maps 1..4 to roman numerals for tier marks.
func tierRoman(t int) string {
	switch {
	case t >= 4:
		return "IV"
	case t == 3:
		return "III"
	case t == 2:
		return "II"
	default:
		return "I"
	}
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

// ── EQUIP: "RELIC VAULT" (hero panel + three gold relic shelves) ──────

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
	cardstyle.PanelRounded(dc, 60, 150, 380, 720, 16, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.3)
	drawHeroFitted(dc, req, 250, 440, 290, 340, *p)
	cardstyle.DotLeader(dc, 110, 390, 640, 1.1, cardstyle.WithA(p.Accent, 130))
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 17, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, 668, p.Ink, 0.5, 2.4)
	cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 60), 250, 700, p.Muted, 0.5, 0.5, 330, 10)

	// three gold hairline shelves; relics stand ON the lines (no boxes)
	shelfX0, shelfX1 := 500.0, 1436.0
	shelfY := []float64{292.0, 524.0, 756.0}
	cellW := (shelfX1 - shelfX0) / 3
	for k, sy := range shelfY {
		// the shelf line with end diamonds
		cardstyle.DotLeader(dc, shelfX0, shelfX1, sy, 1.4, cardstyle.WithA(p.Accent, 170))
		cardstyle.Diamond(dc, shelfX0, sy, 4, p.Accent)
		cardstyle.Diamond(dc, shelfX1, sy, 4, p.Accent)
		cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, fmt.Sprintf("SHELF %d", k+1), shelfX0, sy-18, cardstyle.WithA(p.Muted, 170), 0, 2.2)
		for j := 0; j < 3; j++ {
			idx := k*3 + j
			if idx >= len(req.Slots) {
				break
			}
			s := req.Slots[idx]
			cx := shelfX0 + cellW*float64(j) + cellW/2
			empty := s.Empty
			slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
			if len([]rune(slotCode)) > 4 {
				slotCode = string([]rune(slotCode)[:4])
			}
			cardstyle.Chip(dc, cx-29, sy-116, 58, 22, slotCode, cardstyle.WithA(p.Track, 235), cardstyle.WithA(p.Accent2, 160), p.Accent2, cardstyle.FtInter, 11)
			for t := 0; t < s.Tier && t < 5; t++ {
				cardstyle.Diamond(dc, cx+78-float64(t)*18, sy-105, 5, p.Accent)
			}
			name := "EMPTY"
			ncol := cardstyle.WithA(p.Muted, 170)
			if !empty {
				name = cardstyle.Sanitize(s.Name)
				ncol = p.Ink
			}
			cardstyle.Text(dc, cardstyle.FtMedieval, 18, cardstyle.TruncateRunes(name, 16), cx, sy-72, ncol, 0.5, 0.5, cellW-36, 10)
			if s.TierLabel != "" {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), cx, sy-46, p.Accent, 0.5, 2)
			}
			// the relic: glowing gem standing on the shelf, or a hollow outline
			if empty {
				cardstyle.Ring(dc, cx, sy-16, 12, cardstyle.WithA(p.Muted, 110), 1.2)
				cardstyle.Diamond(dc, cx, sy-16, 3, cardstyle.WithA(p.Muted, 130))
			} else {
				cardstyle.MagicCircle(dc, cx, sy-16, 15, p.Accent)
				cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 18, "*", cx, sy-17, p.Accent2, p.Accent2, 0.5, 0.5, 40, 8)
			}
			if !empty && s.DurMax > 0 {
				frac := s.Dur / s.DurMax
				if frac > 1 {
					frac = 1
				}
				cardstyle.GlowText(dc, cardstyle.FtCinzel, 15, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), cx, sy+34, p.Accent2, p.Accent2, 0.5, 0.5, cellW-60, 10)
				cardstyle.MeterBar(dc, cx-70, sy+52, 140, 8, frac, p.Track, p.Accent2, cardstyle.WithA(p.Accent2, 90), cardstyle.WithA(p.Accent, 110), 4)
			} else if !empty {
				cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "SOLID", cx, sy+38, cardstyle.WithA(p.Accent2, 200), 0.5, 2.2)
			}
		}
	}
	return dc
}

// ── SHOP: "WARDROBE OF WONDERS" ───────────────────────────────────────

func s06Shop(req *portraitRequest) *gg.Context {
	entries := req.Entries
	if len(entries) > 8 {
		entries = entries[:8]
	}
	nE := len(entries)
	if nE < 1 {
		nE = 1
	}
	const W = 800.0
	const step = 104.0
	H := 186.0 + float64(nE)*step + 118.0
	dc := gg.NewContext(int(W), int(H))
	p := s06Base(dc, W, H)

	y := s06TitleBlock(dc, W, 58, "WARDROBE", "OF WONDERS", 34)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, y, p.Muted, 0.5, 2.6)

	ey := 186.0
	for _, e := range entries {
		eh := 92.0
		cardstyle.PanelRounded(dc, 56, ey, W-112, eh, 12, cardstyle.WithA(p.Panel, 215), cardstyle.WithA(p.PanelEd, 120), 1)
		cardstyle.Text(dc, cardstyle.FtMedieval, 19, cardstyle.Sanitize(e.Title), 76, ey+30, p.Ink, 0, 0.5, 400, 12)
		if e.Sub != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 70), 76, ey+56, p.Muted, 0, 0.5, 420, 9)
		}
		if e.Runes != "" {
			cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(e.Runes), 76, ey+78, p.Accent2, 0, 2.4)
		}
		cardstyle.DotLeader(dc, 420, W-150, ey+30, 1.2, cardstyle.WithA(p.Muted, 100))
		cardstyle.Text(dc, cardstyle.FtCinzel, 20, cardstyle.Sanitize(e.Value), W-76, ey+30, p.Accent, 1, 0.5, 140, 12)
		ey += step
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
	if req.EmblemImg != nil {
		dc.Push()
		dc.DrawCircle(W/2, y+92, 58)
		dc.Clip()
		cardstyle.FitEmblem(dc, req.EmblemImg, W/2, y+92, 100, 100)
		dc.Pop()
		dc.ResetClip() // gg Pop() keeps the clip mask (found 2026-09-16)
	} else {
		cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 52, initial, W/2, y+88, p.Accent2, p.Accent2, 0.5, 0.5, 90, 24)
	}
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
	case "EVOLVE":
		return s06Evolve(req).Image()
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

// s06Evolve - class ascension card. Same design language as this style's
// SKILLUP (theme-internal reuse with variation): own base, frame and
// medallion, with the ceremony's own title and the new-class identity.
func s06Evolve(req *portraitRequest) *gg.Context {
	e := *req
	e.DocTitle = "RITUAL OF REBIRTH"
	if e.SubLabel == "" {
		e.SubLabel = "EVOLVED"
	}
	return s06Skillup(&e)
}
