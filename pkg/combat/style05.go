package combat

// style05.go - EMBLEM NOIR "Deco Register" (Style 5) - v3 REBUILD.
// Art-deco case file. Structural signatures (visible in SILHOUETTE, not color):
//   - ziggurat stepped header block across the top
//   - LEFT: full-height chartered band (monogram + gold sunburst fan + wax seal)
//   - RIGHT: ledger rows distributed to fill the full column height
//   - deco double-rule headers with center diamond, stepped chevron dividers
//   - diamond pips instead of meters, registration marks in corners
// Recognizable by structure: the band+column split and the ziggurat crown.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"image-service/pkg/cardstyle"

	"github.com/fogleman/gg"
)

// ── base ─────────────────────────────────────────────────────────────

func s05Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
	p := cardstyle.EmblemNoir()
	cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
	// carbon weave
	dc.SetColor(cardstyle.N(255, 255, 255, 5))
	dc.SetLineWidth(1)
	for y := 0.0; y < H; y += 5 {
		dc.DrawLine(0, y, W, y)
		dc.Stroke()
	}
	for x := 0.0; x < W; x += 9 {
		dc.SetColor(cardstyle.N(255, 255, 255, 3))
		dc.DrawLine(x, 0, x, H)
		dc.Stroke()
	}
	// registration marks
	rm := 24.0
	for _, c := range [][2]float64{{rm, rm}, {W - rm, rm}, {rm, H - rm}, {W - rm, H - rm}} {
		dc.SetColor(cardstyle.WithA(p.Muted, 130))
		dc.SetLineWidth(1)
		dc.DrawLine(c[0]-7, c[1], c[0]+7, c[1])
		dc.Stroke()
		dc.DrawLine(c[0], c[1]-7, c[0], c[1]+7)
		dc.Stroke()
		dc.DrawCircle(c[0], c[1], 3.2)
		dc.Stroke()
	}
	cardstyle.Vignette(dc, W, H, p.Vign)
	return &p
}

// s05Ziggurat - stepped art-deco crown across the card top.
func s05Ziggurat(dc *gg.Context, W, cy float64, p *cardstyle.Palette) {
	// three stacked steps, widest at bottom, with gold keylines
	steps := []struct {
		w, h float64
	}{{W * 0.30, 8}, {W * 0.52, 8}, {W * 0.74, 8}}
	y := cy
	dc.SetColor(cardstyle.WithA(p.Accent, 200))
	for i, s := range steps {
		dc.DrawRectangle(W/2-s.w/2, y, s.w, s.h)
		dc.Fill()
		if i == 0 {
			// topmost step carries the diamond finial
			cardstyle.Diamond(dc, W/2, y-8, 5, p.Accent)
		}
		y += s.h + 4
	}
}

// s05DecoHead - tracked caps between double rules with center diamond.
func s05DecoHead(dc *gg.Context, x, y, w float64, title string, p *cardstyle.Palette) {
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(title)), x+w/2, y, p.Ink, 0.5, 4)
	dc.SetColor(cardstyle.WithA(p.Accent, 190))
	dc.SetLineWidth(1.2)
	dc.DrawLine(x, y+12, x+w/2-16, y+12)
	dc.Stroke()
	dc.DrawLine(x+w/2+16, y+12, x+w, y+12)
	dc.Stroke()
	dc.SetLineWidth(0.6)
	dc.DrawLine(x+14, y+17, x+w-14, y+17)
	dc.Stroke()
	cardstyle.Diamond(dc, x+w/2, y+12, 4, p.Accent)
	// stepped end ticks
	dc.SetColor(cardstyle.WithA(p.Accent, 190))
	dc.DrawRectangle(x, y+9, 6, 6)
	dc.Fill()
	dc.DrawRectangle(x+w-6, y+9, 6, 6)
	dc.Fill()
}

// s05Chevrons - stepped deco divider.
func s05Chevrons(dc *gg.Context, cx, y, w float64, p *cardstyle.Palette) {
	dc.SetColor(cardstyle.WithA(p.Accent, 170))
	dc.SetLineWidth(1.1)
	n := 5
	step := w / float64(n*2+1)
	for i := 0; i < n; i++ {
		x := cx - w/2 + step*float64(i*2+1)
		dc.MoveTo(x, y-4)
		dc.LineTo(x+step*0.6, y)
		dc.LineTo(x, y+4)
		dc.Stroke()
		xr := cx + w/2 - step*float64(i*2+1)
		dc.MoveTo(xr, y-4)
		dc.LineTo(xr-step*0.6, y)
		dc.LineTo(xr, y+4)
		dc.Stroke()
	}
	cardstyle.Diamond(dc, cx, y, 3.4, p.Accent)
}

// s05Row - ledger row with anchored rule and end ticks (never floats).
func s05Row(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette, size int) {
	if size < 12 {
		size = 12
	}
	cardstyle.Text(dc, cardstyle.FtInter, size, strings.ToUpper(cardstyle.Sanitize(label)), x, y, p.Muted, 0, 0.5, w*0.55, size-4)
	cardstyle.Text(dc, cardstyle.FtInterSemi, size, strings.ToUpper(cardstyle.Sanitize(value)), x+w, y, p.Ink, 1, 0.5, w*0.45, size-4)
	dc.SetColor(cardstyle.WithA(p.Muted, 80))
	dc.SetLineWidth(0.9)
	dc.DrawLine(x, y+12, x+w, y+12)
	dc.Stroke()
	dc.SetColor(cardstyle.WithA(p.Accent, 150))
	dc.DrawRectangle(x-2, y+9, 4, 4)
	dc.Fill()
	dc.DrawRectangle(x+w-2, y+9, 4, 4)
	dc.Fill()
}

// s05Band - the full-height chartered band. Returns its inner rect.
func s05Band(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) {
	// charcoal panel with double gold keyline + stepped corners
	dc.SetColor(cardstyle.N(16, 16, 18, 255))
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()
	dc.SetColor(cardstyle.WithA(p.Accent, 210))
	dc.SetLineWidth(1.4)
	dc.DrawRectangle(x, y, w, h)
	dc.Stroke()
	dc.SetLineWidth(0.7)
	dc.DrawRectangle(x+5, y+5, w-10, h-10)
	dc.Stroke()
	// stepped corner caps
	dc.SetColor(cardstyle.WithA(p.Accent, 210))
	for _, c := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		dc.DrawRectangle(c[0]-4, c[1]-4, 8, 8)
		dc.Fill()
	}
}

// s05Sunburst - deco fan of gold rays inside the band bottom half.
func s05Sunburst(dc *gg.Context, cx, cy, rad, yMin float64, p *cardstyle.Palette) {
	dc.SetLineWidth(0.8)
	for i := 0; i <= 14; i++ {
		a := math.Pi + math.Pi*float64(i)/14 // upper half fan
		x2 := cx + rad*math.Cos(a)
		y2 := cy + rad*math.Sin(a)*0.9
		if y2 < yMin {
			continue
		}
		dc.SetColor(cardstyle.WithA(p.Accent, uint8(150-intensity(i, 14)*90)))
		dc.DrawLine(cx, cy, x2, y2)
		dc.Stroke()
	}
	cardstyle.Ring(dc, cx, cy, 7, cardstyle.WithA(p.Accent, 200), 1.2)
	cardstyle.Diamond(dc, cx, cy, 3.5, p.Ink)
}

func intensity(i, n int) float64 {
	d := float64(i) / float64(n)
	if d > 0.5 {
		d = 1 - d
	}
	return d * 2
}

// s05Monogram - huge gold figure inside the band top.
func s05Monogram(dc *gg.Context, cx, cy float64, text string, size int, p *cardstyle.Palette) {
	cardstyle.GlowText(dc, cardstyle.FtCinzelDec, size, text, cx, cy, cardstyle.WithA(p.Accent, 70), cardstyle.WithA(p.Ink, 240), 0.5, 0.5, 300, 40)
	// side flourish ticks (kept inside the band)
	dc.SetColor(cardstyle.WithA(p.Accent, 170))
	dc.SetLineWidth(1)
	dc.DrawLine(cx-62, cy, cx-30, cy)
	dc.Stroke()
	dc.DrawLine(cx+30, cy, cx+62, cy)
	dc.Stroke()
}

// s05Pips - deco diamond pips (mastery/progress).
func s05Pips(dc *gg.Context, cx, y float64, n, filled int, p *cardstyle.Palette) {
	step := 34.0
	x0 := cx - step*float64(n-1)/2
	for i := 0; i < n; i++ {
		x := x0 + step*float64(i)
		if i < filled {
			cardstyle.Diamond(dc, x, y, 9, p.Accent)
			cardstyle.DiamondOutline(dc, x, y, 14, cardstyle.WithA(p.Accent, 120), 0.9)
		} else {
			cardstyle.DiamondOutline(dc, x, y, 7, cardstyle.WithA(p.Muted, 130), 1.2)
		}
	}
}

// s05Distribute - even row positions filling [y0, y1].
func s05Distribute(n, y0, y1 float64) []float64 {
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

func s05Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
	dc.SetColor(cardstyle.WithA(p.Accent, 130))
	dc.SetLineWidth(0.8)
	dc.DrawLine(40, H-46, W-40, H-46)
	dc.Stroke()
	cardstyle.WaxSeal(dc, 58, H-26, 15, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 9)
	if req.Caption != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 66)), W/2+14, H-26, p.Muted, 0.5, 0.5, W-190, 8)
	}
	// corner squares
	dc.SetColor(cardstyle.WithA(p.Accent, 170))
	dc.DrawRectangle(40, H-52, 5, 5)
	dc.Fill()
	dc.DrawRectangle(W-45, H-52, 5, 5)
	dc.Fill()
}

func renderStyle05(kind string, req *portraitRequest) image.Image {
	switch kind {
	case "RANK":
		return s05Rank(req).Image()
	case "ALLOCATE":
		return s05Allocate(req).Image()
	case "SKILLUP":
		return s05Skillup(req).Image()
	case "EVOLVE":
		return s05Evolve(req).Image()
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

// RANK - level monogram band + full-height standing ledger.
func s05Rank(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 34, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 11, "ADVENTURER FILE", W/2, 96, p.Muted, 0.5, 5)

	// left band
	bx, bw := 40.0, 176.0
	by, bh := 122.0, H-122-84
	s05Band(dc, bx, by, bw, bh, p)
	s05Monogram(dc, bx+bw/2, by+120, fmt.Sprintf("%d", req.Level), 74, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 9, "LEVEL", bx+bw/2, by+178, p.Muted, 0.5, 5)
	rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
	if rankLine == "" && req.RankLetter != "" {
		rankLine = req.RankLetter + "-RANK"
	}
	rlSize := 12
	rlMax := 14
	if len([]rune(rankLine)) > 15 {
		rlSize = 10
		rlMax = 24
	}
	cardstyle.Text(dc, cardstyle.FtInterSemi, rlSize, cardstyle.TruncateRunes(rankLine, rlMax), bx+bw/2, by+210, p.Accent, 0.5, 0.5, bw-12, 8)
	s05Sunburst(dc, bx+bw/2, by+bh-96, 80, by+250, p)
	// xp arc meter at band top under monogram
	pct := float64(req.XPPercent) / 100
	cardstyle.NotchMeter(dc, bx+24, by+236, bw-48, 10, int(math.Round(pct*10)), p.Muted, p.Accent)
	cardstyle.Text(dc, cardstyle.FtInter, 9, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), bx+bw/2, by+254, p.Muted, 0.5, 0.5, bw-16, 8)

	// right column
	cx0 := bx + bw + 30
	cw := W - cx0 - 40
	cardstyle.Text(dc, cardstyle.FtInterSemi, 21, strings.ToUpper(cardstyle.Sanitize(styledName(req))), cx0, 140, p.Ink, 0, 0.5, cw, 12)
	s05DecoHead(dc, cx0, 168, cw, "THE RECORD", p)
	rows := req.Standing
	if len(rows) > 5 {
		rows = rows[:5]
	}
	ys := s05Distribute(float64(len(rows)), 236, 420)
	for i, r := range rows {
		s05Row(dc, cx0, ys[i], cw, r.Label, r.Value, p, 14)
	}

	s05Chevrons(dc, cx0+cw/2, 466, cw*0.7, p)
	cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(req.ProgressTitle)), cx0, 496, p.Accent, 0, 0.5, cw, 8)
	prog := req.Progress
	if len(prog) > 3 {
		prog = prog[:3]
	}
	ys2 := s05Distribute(float64(len(prog)), 540, 700)
	for i, pr := range prog {
		frac := 0.0
		if pr.Max > 0 {
			frac = float64(pr.Cur) / float64(pr.Max)
		}
		if pr.Done {
			frac = 1
		}
		s05Row(dc, cx0, ys2[i], cw, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), p, 13)
		segs := 12
		cardstyle.SegBar(dc, cx0, ys2[i]+16, cw, 5, float64(frac), segs, p.Accent, cardstyle.WithA(p.Muted, 70))
	}
	// bottom emblem fill
	cardstyle.DiamondOutline(dc, cx0+cw/2, 800, 56, cardstyle.WithA(p.Accent, 140), 1.1)
	cardstyle.DiamondOutline(dc, cx0+cw/2, 800, 44, cardstyle.WithA(p.Accent, 90), 0.7)
	cardstyle.Spaced(dc, cardstyle.FtInter, 9, "GUILD - INTERNAL", cx0+cw/2, 880, cardstyle.WithA(p.Muted, 140), 0.5, 4)
	s05Footer(dc, W, H, req, p)
	return dc
}

// ALLOCATE - points monogram + stat ledger, distributed.
func s05Allocate(req *portraitRequest) *gg.Context {
	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 34, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 11, "ALLOCATION REGISTER", W/2, 96, p.Muted, 0.5, 5)

	bx, bw := 40.0, 176.0
	by, bh := 122.0, H-122-84
	s05Band(dc, bx, by, bw, bh, p)
	avail := parseLeadingInt(req.PointsBig, 0)
	s05Monogram(dc, bx+bw/2, by+120, fmt.Sprintf("%d", avail), 74, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 9, "UNSPENT", bx+bw/2, by+178, p.Muted, 0.5, 5)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 11, strings.ToUpper(cardstyle.Sanitize(req.Pill)), bx+bw/2, by+210, p.Accent, 0.5, 0.5, bw-16, 8)
	s05Sunburst(dc, bx+bw/2, by+bh-96, 80, by+250, p)
	spent := parseLeadingInt(req.SpentNow, 0)
	total := avail + spent
	cardstyle.Text(dc, cardstyle.FtInter, 10, fmt.Sprintf("SPENT %d / %d", spent, total), bx+bw/2, by+bh-24, p.Muted, 0.5, 0.5, bw-16, 8)

	cx0 := bx + bw + 30
	cw := W - cx0 - 40
	cardstyle.Text(dc, cardstyle.FtInterSemi, 21, strings.ToUpper(cardstyle.Sanitize(styledName(req))), cx0, 140, p.Ink, 0, 0.5, cw, 12)
	s05DecoHead(dc, cx0, 168, cw, "THE SEVEN PATHS", p)
	rows := req.Rows
	if len(rows) > 7 {
		rows = rows[:7]
	}
	ys := s05Distribute(float64(len(rows)), 244, 636)
	for i, r := range rows {
		code := strings.ToUpper(cardstyle.Sanitize(r.Label))
		s05Row(dc, cx0, ys[i], cw, cardstyle.StatFullName(code), gainFromSub(r.Sub), p, 14)
	}
	s05Chevrons(dc, cx0+cw/2, 676, cw*0.7, p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 13, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), cx0, 708, p.Accent, 0, 0.5, cw, 9)
	cardstyle.Text(dc, cardstyle.FtInter, 12, strings.ToUpper(cardstyle.Sanitize(req.CtaLabel)), cx0, 740, p.Ink, 0, 0.5, cw, 9)
	if req.CtaSub != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.CtaSub), 60)), cx0, 762, cardstyle.WithA(p.Muted, 170), 0, 0.5, cw, 8)
	}
	cardstyle.Text(dc, cardstyle.FtInter, 9, "HALF VALUE PER POINT AFTER HEAVY SINGLE-STAT INVESTMENT", cx0, 796, cardstyle.WithA(p.Muted, 150), 0, 0.5, cw, 8)
	s05Footer(dc, W, H, req, p)
	return dc
}

// SKILLUP - initials band + diamond mastery pips.
func s05Skillup(req *portraitRequest) *gg.Context {
	skillTitle := strings.ToUpper(cardstyle.Sanitize(req.DocTitle))
	if skillTitle == "" {
		skillTitle = "MASTERY MEMO"
	}

	const W, H = 600.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 34, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 11, skillTitle, W/2, 96, p.Muted, 0.5, 5)

	bx, bw := 40.0, 176.0
	by, bh := 122.0, H-122-84
	s05Band(dc, bx, by, bw, bh, p)
	initials := initialLetters(cardstyle.Sanitize(req.SkillName))
	s05Monogram(dc, bx+bw/2, by+120, initials, 58, p)
	cardstyle.Spaced(dc, cardstyle.FtInter, 9, "SUBJECT", bx+bw/2, by+172, p.Muted, 0.5, 5)
	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	if req.SubLabel != "" {
		tierLabel = strings.ToUpper(cardstyle.Sanitize(req.SubLabel))
	}
	cardstyle.Text(dc, cardstyle.FtInterSemi, 12, tierLabel, bx+bw/2, by+204, p.Accent, 0.5, 0.5, bw-16, 9)
	s05Sunburst(dc, bx+bw/2, by+bh-96, 80, by+250, p)
	cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.Sanitize(styledName(req))), bx+bw/2, by+bh-24, p.Muted, 0.5, 0.5, bw-16, 8)

	cx0 := bx + bw + 30
	cw := W - cx0 - 40
	s05DecoHead(dc, cx0, 150, cw, "PROMOTION", p)
	// skill name centered in column
	words := strings.Split(strings.ToUpper(cardstyle.Sanitize(req.SkillName)), " ")
	ly := 226.0
	for _, w := range words {
		cardstyle.Text(dc, cardstyle.FtCinzel, 26, w, cx0+cw/2, ly, p.Ink, 0.5, 0.5, cw, 14)
		ly += 38
	}
	s05Chevrons(dc, cx0+cw/2, ly+8, cw*0.6, p)

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
	cardstyle.Text(dc, cardstyle.FtInter, 12, "MASTERY", cx0+cw/2, 560, p.Muted, 0.5, 0.5, cw, 8)
	s05Pips(dc, cx0+cw/2, 620, maxPips, cur, p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 16, fmt.Sprintf("%d / %d", cur, maxPips), cx0+cw/2, 668, p.Ink, 0.5, 0.5, cw, 10)
	s05Row(dc, cx0, 740, cw, "STATUS", tierLabel, p, 13)
	// deco emblem fill
	cardstyle.DiamondOutline(dc, cx0+cw/2, 830, 46, cardstyle.WithA(p.Accent, 130), 1)
	cardstyle.Diamond(dc, cx0+cw/2, 830, 8, cardstyle.WithA(p.Accent, 160))
	s05Footer(dc, W, H, req, p)
	return dc
}

// ABILITIES - wide deco register, groups in columns.
func s05Abilities(req *portraitRequest) *gg.Context {
	W := 1000.0
	H := float64(abilitiesH(req))
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 30, p)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 20, strings.ToUpper(cardstyle.Sanitize(req.DocTitle)), W/2, 86, p.Ink, 0.5, 5)
	if strings.TrimSpace(req.DocQuote) != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper("\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 100)+"\""), W/2, 112, cardstyle.WithA(p.Muted, 190), 0.5, 0.5, W-200, 8)
	}
	cardstyle.Spaced(dc, cardstyle.FtInter, 10, cardstyle.Sanitize(req.PageLabel), W-70, 86, p.Accent, 0.5, 3)

	groups := req.Groups
	if len(groups) > 6 {
		groups = groups[:6]
	}
	nG := len(groups)
	if nG == 0 {
		nG = 1
	}
	cols := 1
	if nG > 1 {
		cols = 2
	}
	rowsG := (nG + cols - 1) / cols
	gx0, gy0 := 52.0, 160.0
	gw := (W - 104 - 36*float64(cols-1)) / float64(cols)
	gh := (H - gy0 - 120) / float64(rowsG)

	num := req.StartNumber
	if num < 1 {
		num = 1
	}
	for gi, g := range groups {
		gi2 := gi
		col := gi2 % cols
		row := gi2 / cols
		gx := gx0 + float64(col)*(gw+36)
		gy := gy0 + float64(row)*gh
		s05DecoHead(dc, gx, gy, gw, g.Name, p)
		items := g.Items
		// adaptive step: fill the group band, clamped so rows never crowd/spread
		n := len(items)
		step := 62.0
		iy0 := gy + 54.0
		if gh > 0 {
			if n > 1 {
				step = (gh - 110) / float64(n-1)
				if step > 88 {
					step = 88
				}
				if step < 54 {
					step = 54
				}
			} else {
				iy0 = gy + gh/2 - 40
				if iy0 < gy+54 {
					iy0 = gy + 54
				}
			}
		}
		iy := iy0
		for _, it := range items {
			cardstyle.Text(dc, cardstyle.FtInterSemi, 13, fmt.Sprintf("%03d", num), gx, iy, p.Accent, 0, 0.5, 48, 9)
			cardstyle.Text(dc, cardstyle.FtInterSemi, 15, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Title)), 22), gx+58, iy, p.Ink, 0, 0.5, gw-190, 10)
			cardstyle.Text(dc, cardstyle.FtInterSemi, 13, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Value)), 12), gx+gw, iy, p.Accent, 1, 0.5, 110, 9)
			extra := it.Sub
			if it.Runes != "" {
				extra = strings.ToUpper(cardstyle.Sanitize(it.Runes)) + "  " + extra
			}
			if extra != "" {
				cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(extra), 80)), gx+58, iy+20, p.Muted, 0, 0.5, gw-70, 8)
			}
			dc.SetColor(cardstyle.WithA(p.Muted, 70))
			dc.SetLineWidth(0.8)
			dc.DrawLine(gx+58, iy+34, gx+gw, iy+34)
			dc.Stroke()
			num++
			iy += step
			// close the group with a chevron divider
		}
		// close the group with a chevron divider
		s05Chevrons(dc, gx+gw/2, iy+26, gw*0.5, p)
	}
	// short-page filler: deco emblem anchored in the remaining field
	if gy0+float64(rowsG)*gh < H-190 {
		ey := (gy0 + float64(rowsG)*gh + H - 110) / 2
		cardstyle.DiamondOutline(dc, W/2, ey, 52, cardstyle.WithA(p.Accent, 130), 1)
		cardstyle.DiamondOutline(dc, W/2, ey, 40, cardstyle.WithA(p.Accent, 90), 0.7)
		cardstyle.Spaced(dc, cardstyle.FtInter, 9, "GUILD ARCHIVE", W/2, ey+84, cardstyle.WithA(p.Muted, 140), 0.5, 5)
	}
	// bottom deco rule + ornament
	s05Chevrons(dc, W/2, H-70, W*0.5, p)
	s05Footer(dc, W, H, req, p)
	return dc
}

// SKILLTREE - deco diamond lattice.
func s05Skilltree(req *portraitRequest) *gg.Context {
	const W, H = 1100.0, 1100.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 30, p)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 20, "MASTERY LATTICE", W/2, 92, p.Ink, 0.5, 5)
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	cardstyle.Stamp(dc, W-140, 96, ptsText, p.Accent)

	cx := W / 2
	// deco double diamond frame around the field
	cardstyle.DiamondOutline(dc, cx, 590, 470, cardstyle.WithA(p.Accent, 90), 1)
	cardstyle.DiamondOutline(dc, cx, 590, 440, cardstyle.WithA(p.Accent, 60), 0.7)

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
	// diagonal lattice lines
	g := 72.0
	dc.SetColor(cardstyle.WithA(p.Muted, 34))
	dc.SetLineWidth(0.8)
	for d := -14; d <= 14; d++ {
		off := float64(d) * g
		dc.DrawLine(cx-g*15+off, 140, cx+g*15+off, H-120)
		dc.Stroke()
		dc.DrawLine(cx-g*15-off, 140, cx+g*15-off, H-120)
		dc.Stroke()
	}
	// apex node
	cardstyle.WaxSeal(dc, cx, 300, 30, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, "MAX", cardstyle.FtInter, 11)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 15, strings.ToUpper(cardstyle.Sanitize(req.ClassName)), cx, 356, p.Ink, 0.5, 0.5, 240, 10)

	totalW := float64(n) * 250
	x0 := cx - totalW/2 + 125
	for bi, br := range req.Branches {
		bx := x0 + float64(bi)*250
		cardstyle.Text(dc, cardstyle.FtInterSemi, 14, strings.ToUpper(cardstyle.Sanitize(br.Name)), bx, 420, p.Accent, 0.5, 0.5, 220, 9)
		cardstyle.DiamondOutline(dc, bx, 446, 11, p.Accent, 1.3)
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
			ny := 530 + float64(k)*150
			if k > 0 {
				dc.SetColor(cardstyle.WithA(p.Muted, 100))
				dc.SetLineWidth(1.1)
				dc.DrawLine(bx, 530+float64(k-1)*150+26, bx, ny-26)
				dc.Stroke()
			}
			for j, s := range list {
				nx := bx + (float64(j)-float64(len(list)-1)/2)*76
				s05Node(dc, nx, ny, s, p)
			}
			_ = t
		}
	}
	// legend in deco frame
	ly := H - 84.0
	cardstyle.Chip(dc, cx-330, ly-16, 660, 34, "", cardstyle.N(14, 14, 16, 220), cardstyle.WithA(p.Accent, 140), p.Muted, cardstyle.FtInter, 11)
	lx := cx - 300.0
	for _, l := range []struct {
		label, state string
	}{{"LEARNED", "learned"}, {"MAXED", "maxed"}, {"OPEN", "open"}, {"LOCKED", "locked"}} {
		cardstyle.Diamond(dc, lx, ly+1, 8, s05NodeCol(l.state, p))
		cardstyle.Text(dc, cardstyle.FtInter, 11, l.label, lx+16, ly+1, p.Muted, 0, 0.5, 100, 8)
		lx += 160
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
	r := 16.0
	if lit {
		cardstyle.Diamond(dc, x, y, r, col)
		cardstyle.DiamondOutline(dc, x, y, r+6, cardstyle.WithA(col, 110), 1)
	} else {
		cardstyle.DiamondOutline(dc, x, y, r-4, col, 1.4)
	}
	cardstyle.Text(dc, cardstyle.FtInterSemi, 11, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 12), x, y+r+16, p.Ink, 0.5, 0.5, 140, 8)
	cardstyle.Text(dc, cardstyle.FtInter, 10, fmt.Sprintf("%d/%d", s.Cur, s.Max), x, y+r+33, p.Muted, 0.5, 0.5, 60, 7)
}

// EQUIP - file photo band + manifest column.
func s05Equip(req *portraitRequest) *gg.Context {
	const W, H = 1200.0, 1000.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 30, p)
	s05DecoHead(dc, 52, 88, W-104, "EQUIPMENT MANIFEST", p)

	// left: file photo band
	bx, bw := 60.0, 320.0
	by, bh := 140.0, H-140-90
	s05Band(dc, bx, by, bw, bh, p)
	// stepped photo frame
	cardstyle.FrameHairline(dc, bx+24, by+24, bw-48, bh-120, 8, 1.1, cardstyle.WithA(p.Accent, 170), false)
	drawHeroFitted(dc, req, bx+bw/2, by+(bh-120)/2+14, bw-90, bh-180, *p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 17, strings.ToUpper(cardstyle.Sanitize(styledName(req))), bx+bw/2, by+bh-66, p.Ink, 0.5, 0.5, bw-30, 10)
	if req.SealText != "" {
		cardstyle.WaxSeal(dc, bx+bw-34, by+bh-28, 17, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 10)
	}

	// right: manifest rows distributed
	cx0 := bx + bw + 44
	cw := W - cx0 - 60
	slots := req.Slots
	if len(slots) > 9 {
		slots = slots[:9]
	}
	ys := s05Distribute(float64(len(slots)), 190, H-190)
	for i, s := range slots {
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
		s05Row(dc, cx0, ys[i], cw, label, value, p, 15)
		if !s.Empty && s.DurMax > 0 {
			frac := s.Dur / s.DurMax
			if frac > 1 {
				frac = 1
			}
			cardstyle.SegBar(dc, cx0, ys[i]+18, cw*0.55, 4, frac, 10, p.Accent, cardstyle.WithA(p.Muted, 70))
		}
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

// SHOP - deco procurement register with fan filler.
func s05Shop(req *portraitRequest) *gg.Context {
	entries := req.Entries
	if len(entries) > 9 {
		entries = entries[:9]
	}
	n := len(entries)
	if n < 1 {
		n = 1
	}
	const W = 800.0
	H := 180.0 + float64(n)*74 + 250.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 34, p)
	cardstyle.Spaced(dc, cardstyle.FtCinzel, 18, "PROCUREMENT LIST", W/2, 96, p.Ink, 0.5, 4)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 13, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 124, p.Muted, 0.5, 0.5, 300, 9)

	top := 180.0
	bottom := H - 240.0
	ys := s05Distribute(float64(n), top, bottom)
	for i, e := range entries {
		y := ys[i]
		extra := e.Sub
		if e.Runes != "" {
			extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
		}
		cardstyle.Diamond(dc, 56, y, 5, p.Accent)
		cardstyle.Text(dc, cardstyle.FtInterSemi, 17, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(e.Title)), 24), 78, y-4, p.Ink, 0, 0.5, W-240, 11)
		cardstyle.Text(dc, cardstyle.FtInterSemi, 16, strings.ToUpper(cardstyle.Sanitize(e.Value)), W-56, y-4, p.Accent, 1, 0.5, 150, 10)
		if extra != "" {
			cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(extra), 80)), 78, y+18, p.Muted, 0, 0.5, W-150, 8)
		}
		dc.SetColor(cardstyle.WithA(p.Muted, 80))
		dc.SetLineWidth(0.9)
		dc.DrawLine(56, y+40, W-56, y+40)
		dc.Stroke()
	}
	// deco fan filler under the list
	s05Sunburst(dc, W/2, H-150, 100, H-246, p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 12, strings.ToUpper(cardstyle.Sanitize(req.Caption)), W/2, H-104, p.Accent, 0.5, 0.5, 400, 9)
	s05Footer(dc, W, H, req, p)
	return dc
}

// GUILDINFO - monogram band + charter ledger.
func s05GuildInfo(req *portraitRequest) *gg.Context {
	const W, H = 800.0, 860.0
	dc := gg.NewContext(int(W), int(H))
	p := s05Base(dc, W, H)

	s05Ziggurat(dc, W, 30, p)
	s05DecoHead(dc, 52, 84, W-104, "CHAPTER CHARTER", p)

	bx, bw := 60.0, 210.0
	by, bh := 132.0, H-132-84
	s05Band(dc, bx, by, bw, bh, p)
	if req.EmblemImg != nil {
		cardstyle.FitEmblem(dc, req.EmblemImg, bx+bw/2, by+104, bw-50, 104)
	} else {
		s05Monogram(dc, bx+bw/2, by+110, firstRuneUpper(styledName(req)), 56, p)
	}
	cardstyle.Text(dc, cardstyle.FtInterSemi, 14, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(styledName(req))), 16), bx+bw/2, by+160, p.Ink, 0.5, 0.5, bw-20, 9)
	if req.Motto != "" {
		cardstyle.Text(dc, cardstyle.FtInter, 10, "\""+cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Motto)), 56)+"\"", bx+bw/2, by+186, p.Muted, 0.5, 0.5, bw-24, 8)
	}
	s05Sunburst(dc, bx+bw/2, by+bh-80, 78, by+240, p)
	cardstyle.Text(dc, cardstyle.FtInterSemi, 13, fmt.Sprintf("LEVEL %d", req.Level), bx+bw/2, by+bh-20, p.Accent, 0.5, 0.5, bw-24, 9)

	cx0 := bx + bw + 36
	cw := W - cx0 - 60
	rows := req.Rows
	if len(rows) > 4 {
		rows = rows[:4]
	}
	ys := s05Distribute(float64(len(rows)+1), 180, 420)
	for i, r := range rows {
		s05Row(dc, cx0, ys[i], cw, r.Label, r.Value, p, 14)
	}
	// level with segmented bar
	ly := ys[len(rows)]
	s05Row(dc, cx0, ly, cw, "LEVEL", fmt.Sprintf("%d", req.Level), p, 14)
	pct := float64(req.XPPercent) / 100
	cardstyle.SegBar(dc, cx0, ly+18, cw, 5, pct, 12, p.Accent, cardstyle.WithA(p.Muted, 70))

	s05Chevrons(dc, cx0+cw/2, ly+70, cw*0.6, p)
	cardstyle.Text(dc, cardstyle.FtInter, 11, "CHAPTER HOLDINGS", cx0, ly+96, p.Accent, 0, 0.5, cw, 8)
	bs := req.Buildings
	if len(bs) > 3 {
		bs = bs[:3]
	}
	ys2 := s05Distribute(float64(len(bs)), ly+126, H-140)
	for i, b := range bs {
		s05Row(dc, cx0, ys2[i], cw, b.Name, fmt.Sprintf("LVL %d", b.Level), p, 13)
	}
	s05Footer(dc, W, H, req, p)
	return dc
}

// s05Evolve - class ascension card. Same design language as this style's
// SKILLUP (theme-internal reuse with variation): own base, frame and
// medallion, with the ceremony's own title and the new-class identity.
func s05Evolve(req *portraitRequest) *gg.Context {
	e := *req
	e.DocTitle = "REBRANDING MEMO"
	if e.SubLabel == "" {
		e.SubLabel = "EVOLVED"
	}
	return s05Skillup(&e)
}
