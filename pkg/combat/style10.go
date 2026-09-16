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

// RANK - letters patent: crest shield + twin-column court ledger.
func s10Rank(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s10Base(dc, W, H)

        cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, strings.ToUpper(cardstyle.Sanitize(styledName(req))), cardstyle.FtCinzelDec, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "LETTERS PATENT OF THE COURT", W/2, 122, p.Muted, 0.5, 2.6)

        // crest shield with the level
        cardstyle.Shield(dc, W/2, 262, 170, 200, cardstyle.Hex(0x581520), p.Accent, 3.4)
        cardstyle.EmberGlow(dc, W/2, 252, 64, cardstyle.HexA(0xd4a856, 90))
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 54, fmt.Sprintf("%d", req.Level), W/2, 244, p.Accent, 0.5, 0.5, 120, 20)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, "LEVEL", W/2, 296, p.Muted, 0.5, 0.5, 110, 8)
        // filigree flourishes flanking the shield
        for _, sgn := range []float64{-1, 1} {
                fx := W/2 + sgn*130
                dc.SetColor(cardstyle.WithA(p.Accent, 150))
                dc.SetLineWidth(1.3)
                if sgn < 0 {
                        dc.DrawArc(fx+22, 262, 24, math.Pi/2, math.Pi*1.5)
                } else {
                        dc.DrawArc(fx-22, 262, 24, -math.Pi/2, math.Pi/2)
                }
                dc.Stroke()
                cardstyle.Diamond(dc, fx, 306, 4, p.Accent)
                cardstyle.Diamond(dc, fx, 200, 3, cardstyle.WithA(p.Accent, 120))
        }
        rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
        if rankLine == "" && req.RankLetter != "" {
                rankLine = req.RankLetter + "-RANK ADVENTURER"
        }
        cardstyle.Pennant(dc, W/2-140, 386, 280, 34, p.Panel, p.Accent, p.Ink, rankLine, cardstyle.FtCinzel, 12)

        // experience bar spanning the letters
        pct := float64(req.XPPercent) / 100
        cardstyle.Text(dc, cardstyle.FtCinzel, 11, "EXPERIENCE", 56, 448, p.Muted, 0, 0.5, 160, 8)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-56, 448, p.Ink, 1, 0.5, 200, 8)
        dc.SetColor(p.Track)
        dc.DrawRoundedRectangle(56, 460, W-112, 7, 3.5)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.DrawRoundedRectangle(56, 460, (W-112)*pct, 7, 3.5)
        dc.Fill()

        // twin-column court ledger with central gold rail
        colY0, colY1 := 540.0, 880.0
        dc.SetColor(cardstyle.WithA(p.Accent, 150))
        dc.SetLineWidth(1.2)
        dc.DrawLine(W/2, colY0-24, W/2, colY1+8)
        dc.Stroke()
        for yy := colY0 - 8; yy <= colY1; yy += 42 {
                cardstyle.Diamond(dc, W/2, yy, 3.4, cardstyle.WithA(p.Accent, 170))
        }
        // left column: THE RECORD
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE RECORD", 163, colY0, p.Accent, 0.5, 2)
        rows := req.Standing
        if len(rows) > 5 {
                rows = rows[:5]
        }
        n := float64(len(rows))
        if n < 1 {
                n = 1
        }
        step := (colY1 - colY0 - 40) / n
        if step > 62 {
                step = 62
        }
        for i, r := range rows {
                yy := colY0 + 44 + float64(i)*step
                cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(r.Label)), 56, yy, p.Muted, 0, 0.5, 200, 9)
                cardstyle.DotLeader(dc, 56, 270-cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(r.Value)), 0)-8, yy+1, 0.9, cardstyle.WithA(p.Accent, 80))
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(r.Value)), 270, yy, p.Ink, 1, 0.5, 150, 9)
        }
        // right column: THE PATH AHEAD
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE PATH AHEAD", 437, colY0, p.Accent, 0.5, 2)
        prog := req.Progress
        if len(prog) > 3 {
                prog = prog[:3]
        }
        for i, pr := range prog {
                yy := colY0 + 44 + float64(i)*112
                cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(pr.Label)), 330, yy, p.Muted, 0, 0.5, 214, 9)
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
                cardstyle.Text(dc, cardstyle.FtCinzel, 12, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), 544, yy, p.Ink, 1, 0.5, 180, 8)
                dc.SetColor(p.Track)
                dc.DrawRoundedRectangle(330, yy+12, 214, 6, 3)
                dc.Fill()
                dc.SetColor(fill)
                dc.DrawRoundedRectangle(330, yy+12, 214*frac, 6, 3)
                dc.Fill()
        }
        s10Footer(dc, W, H, req, p)
        return dc
}

// ALLOCATE - court ledger: shield of offering + stat/spend twin columns.
func s10Allocate(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s10Base(dc, W, H)

        cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, "ALLOCATION", cardstyle.FtCinzelDec, 24)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE COURT LEDGER", W/2, 122, p.Muted, 0.5, 2.6)

        avail := parseLeadingInt(req.PointsBig, 0)
        cardstyle.Shield(dc, W/2, 256, 160, 190, cardstyle.Hex(0x581520), p.Accent, 3.2)
        cardstyle.EmberGlow(dc, W/2, 246, 60, cardstyle.HexA(0xd4a856, 90))
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 50, fmt.Sprintf("%d", avail), W/2, 238, p.Accent, 0.5, 0.5, 110, 18)
        cardstyle.Text(dc, cardstyle.FtCinzel, 11, "UNSPENT", W/2, 288, p.Muted, 0.5, 0.5, 110, 8)
        classLine := strings.ToUpper(cardstyle.Sanitize(req.Pill))
        if strings.Contains(strings.ToLower(classLine), "starter") && !strings.Contains(strings.ToLower(classLine), "tier") {
                classLine += " TIER"
        }
        cardstyle.Pennant(dc, W/2-120, 376, 240, 32, p.Panel, p.Accent, p.Ink, classLine, cardstyle.FtCinzel, 11)

        // twin columns
        colY0, colY1 := 464.0, 900.0
        dc.SetColor(cardstyle.WithA(p.Accent, 150))
        dc.SetLineWidth(1.2)
        dc.DrawLine(W/2, colY0-20, W/2, colY1)
        dc.Stroke()
        for yy := colY0 - 4; yy <= colY1; yy += 42 {
                cardstyle.Diamond(dc, W/2, yy, 3.4, cardstyle.WithA(p.Accent, 170))
        }
        // left: the seven paths
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE SEVEN PATHS", 163, colY0, p.Accent, 0.5, 2)
        rows := req.Rows
        if len(rows) > 7 {
                rows = rows[:7]
        }
        step := (colY1 - colY0 - 60) / 7
        for i, r := range rows {
                code := strings.ToUpper(cardstyle.Sanitize(r.Label))
                yy := colY0 + 40 + float64(i)*step
                cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(cardstyle.StatFullName(code))), 56, yy, p.Muted, 0, 0.5, 170, 9)
                cardstyle.DotLeader(dc, 56, 268-cardstyle.SpacedWidth(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(gainFromSub(r.Sub))), 0)-8, yy+1, 0.9, cardstyle.WithA(p.Accent, 80))
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(gainFromSub(r.Sub))), 270, yy, p.Ink, 1, 0.5, 120, 9)
        }
        // right: the spending ledger
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE SPENDING", 437, colY0, p.Accent, 0.5, 2)
        spent := parseLeadingInt(req.SpentNow, 0)
        ry := colY0 + 40.0
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, "SPENT", 330, ry, p.Muted, 0, 0.5, 120, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("%d PTS", spent), 544, ry, p.Ink, 1, 0.5, 140, 9)
        ry += 34
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), 330, ry, p.Muted, 0, 0.5, 214, 9)
        ry += 40
        cardstyle.Pennant(dc, 330, ry, 214, 40, p.Track, p.Accent, p.Ink, ".J ALLOCATE <STAT> [N]", cardstyle.FtCinzel, 10)
        ry += 62
        cardstyle.Text(dc, cardstyle.FtCinzel, 10, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.CtaSub), 44)), 330, ry, cardstyle.WithA(p.Ink, 200), 0, 0.5, 214, 7)
        ry += 28
        cardstyle.Text(dc, cardstyle.FtCinzel, 9, "HALF VALUE PER POINT", 330, ry, cardstyle.WithA(p.Muted, 190), 0, 0.5, 214, 7)
        ry += 16
        cardstyle.Text(dc, cardstyle.FtCinzel, 9, "AFTER HEAVY SINGLE-STAT", 330, ry, cardstyle.WithA(p.Muted, 190), 0, 0.5, 214, 7)
        ry += 16
        cardstyle.Text(dc, cardstyle.FtCinzel, 9, "INVESTMENT", 330, ry, cardstyle.WithA(p.Muted, 190), 0, 0.5, 214, 7)
        // candle glow ornament at the column base
        cardstyle.EmberGlow(dc, 437, colY1-40, 40, cardstyle.HexA(0xd4a856, 70))
        cardstyle.Diamond(dc, 437, colY1-40, 6, p.Accent)
        s10Footer(dc, W, H, req, p)
        return dc
}

// SKILLUP - investiture: shield, chevron mastery arc, station pennant.
func s10Skillup(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s10Base(dc, W, H)

        cardstyle.Cartouche(dc, W/2, 74, 440, 58, p.Panel, p.Accent, p.Ink, "INVESTITURE", cardstyle.FtCinzelDec, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 122, p.Muted, 0.5, 2.6)

        // great shield with skill initials
        cardstyle.Shield(dc, W/2, 300, 210, 240, cardstyle.Hex(0x581520), p.Accent, 3.4)
        initials := initialLetters(cardstyle.Sanitize(req.SkillName))
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 56, initials, W/2, 286, p.Accent, 0.5, 0.5, 140, 22)
        for _, sgn := range []float64{-1, 1} {
                fx := W/2 + sgn*155
                dc.SetColor(cardstyle.WithA(p.Accent, 150))
                dc.SetLineWidth(1.4)
                if sgn < 0 {
                        dc.DrawArc(fx+24, 300, 26, math.Pi/2, math.Pi*1.5)
                } else {
                        dc.DrawArc(fx-24, 300, 26, -math.Pi/2, math.Pi/2)
                }
                dc.Stroke()
                cardstyle.Diamond(dc, fx, 352, 4, p.Accent)
        }

        cardstyle.Text(dc, cardstyle.FtCinzelDec, 23, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, 460, p.Ink, 0.5, 0.5, W-100, 12)
        tierLabel := fmt.Sprintf("TIER %d", req.Tier)
        if req.Ascended {
                tierLabel = "ASCENDED"
        }
        cardstyle.Pennant(dc, W/2-110, 486, 220, 32, p.Panel, p.Accent, p.Ink, strings.ToUpper(tierLabel), cardstyle.FtCinzel, 12)

        // chevron mastery arc
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
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 12, "MASTERY", W/2, 580, p.Muted, 0.5, 3)
        arcW := 300.0
        x0 := W/2 - arcW/2 + 24
        for i := 0; i < maxPips; i++ {
                cxp := x0 + (arcW-48)*float64(i)/float64(maxPips-1)
                cyp := 640.0
                if i < cur {
                        cardstyle.EmberGlow(dc, cxp, cyp, 26, cardstyle.HexA(0xd4a856, 80))
                }
                dc.SetLineWidth(3)
                if i < cur {
                        dc.SetColor(p.Accent)
                } else {
                        dc.SetColor(cardstyle.WithA(p.Muted, 120))
                }
                dc.DrawLine(cxp-10, cyp+7, cxp, cyp-7)
                dc.Stroke()
                dc.DrawLine(cxp, cyp-7, cxp+10, cyp+7)
                dc.Stroke()
        }
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, fmt.Sprintf("%d / %d", cur, maxPips), W/2, 692, p.Ink, 0.5, 0.5, 160, 12)

        cardstyle.Pennant(dc, W/2-130, 742, 260, 36, p.Track, p.Accent, p.Ink, "STATION - "+strings.ToUpper(tierLabel), cardstyle.FtCinzel, 11)
        // base filigree ornament
        cardstyle.EmberGlow(dc, W/2, 850, 40, cardstyle.HexA(0xd4a856, 60))
        dc.SetColor(cardstyle.WithA(p.Accent, 140))
        dc.SetLineWidth(1.2)
        dc.DrawLine(W/2-90, 850, W/2-24, 850)
        dc.Stroke()
        dc.DrawLine(W/2+24, 850, W/2+90, 850)
        dc.Stroke()
        cardstyle.Diamond(dc, W/2, 850, 6, p.Accent)
        s10Footer(dc, W, H, req, p)
        return dc
}

// ABILITIES - the court retinue list (cartouche groups).
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
        // short-page filler: court ornament in the remaining field
        if y < H-200 {
                ey := (y + H - 120) / 2
                cardstyle.EmberGlow(dc, W/2, ey, 44, cardstyle.HexA(0xd4a856, 60))
                dc.SetColor(cardstyle.WithA(p.Accent, 140))
                dc.SetLineWidth(1.2)
                dc.DrawLine(W/2-120, ey, W/2-26, ey)
                dc.Stroke()
                dc.DrawLine(W/2+26, ey, W/2+120, ey)
                dc.Stroke()
                cardstyle.Diamond(dc, W/2, ey, 7, p.Accent)
                cardstyle.DiamondOutline(dc, W/2, ey, 16, cardstyle.WithA(p.Accent, 110), 1)
                cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "BY DECREE OF THE COURT", W/2, ey+44, cardstyle.WithA(p.Muted, 170), 0.5, 3)
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

        // hero dais (kept)
        cardstyle.Shield(dc, 250, 420, 330, 560, p.Panel, p.Accent, 3)
        drawHeroFitted(dc, req, 250, 400, 280, 300, *p)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 20, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, 640, p.Ink, 0.5, 0.5, 300, 11)
        if req.SealText != "" {
                cardstyle.Pennant(dc, 175, 672, 150, 34, p.Track, p.Accent, p.Ink, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", cardstyle.FtCinzel, 11)
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 52), 250, 740, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 320, 9)

        // two gold display rails; nine pendant shields hang from drop-rods
        railX0, railX1 := 520.0, 1444.0
        rails := []struct {
                y   float64
                n   int
                off int
                sw  float64
                sh  float64
        }{{236.0, 5, 0, 172.0, 208.0}, {596.0, 4, 5, 188.0, 224.0}}
        for _, rl := range rails {
                dc.SetColor(p.Accent)
                dc.SetLineWidth(3)
                dc.DrawLine(railX0, rl.y, railX1, rl.y)
                dc.Stroke()
                dc.SetLineWidth(1)
                dc.DrawLine(railX0, rl.y+6, railX1, rl.y+6)
                dc.Stroke()
                cardstyle.Diamond(dc, railX0, rl.y, 6, p.Accent)
                cardstyle.Diamond(dc, railX1, rl.y, 6, p.Accent)
                span := railX1 - railX0
                cw := span / float64(rl.n)
                for j := 0; j < rl.n; j++ {
                        idx := rl.off + j
                        if idx >= len(req.Slots) {
                                break
                        }
                        s := req.Slots[idx]
                        cx := railX0 + cw*float64(j) + cw/2
                        dc.SetColor(cardstyle.WithA(p.Accent, 180))
                        dc.SetLineWidth(2.4)
                        dc.DrawLine(cx, rl.y, cx, rl.y+22)
                        dc.Stroke()
                        cardstyle.Ring(dc, cx, rl.y+26, 5, cardstyle.WithA(p.Accent, 200), 1.6)
                        empty := s.Empty
                        fill := p.Panel
                        edge := p.Accent
                        if empty {
                                fill = cardstyle.Darken(p.Panel, 24)
                                edge = cardstyle.WithA(p.Accent, 110)
                        }
                        shieldCy := rl.y + 44 + rl.sh/2
                        cardstyle.Shield(dc, cx, shieldCy, rl.sw, rl.sh, fill, edge, 2.4)
                        slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
                        if len([]rune(slotCode)) > 4 {
                                slotCode = string([]rune(slotCode)[:4])
                        }
                        cardstyle.Text(dc, cardstyle.FtCinzel, 11, slotCode, cx, rl.y+58, p.Muted, 0.5, 0.5, 120, 8)
                        for t := 0; t < s.Tier && t < 5; t++ {
                                cardstyle.Diamond(dc, cx+rl.sw/2-26-float64(t)*16, rl.y+58, 4.4, p.Accent)
                        }
                        name := "EMPTY"
                        ncol := cardstyle.WithA(p.Muted, 150)
                        if !empty {
                                name = cardstyle.Sanitize(s.Name)
                                ncol = p.Ink
                        }
                        cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.TruncateRunes(strings.ToUpper(name), 16), cx, shieldCy+2, ncol, 0.5, 0.5, rl.sw-28, 9)
                        if s.TierLabel != "" {
                                cardstyle.Text(dc, cardstyle.FtCinzel, 10, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), cx, shieldCy+28, p.Accent2, 0.5, 0.5, rl.sw-28, 8)
                        }
                        if !empty && s.DurMax > 0 {
                                frac := s.Dur / s.DurMax
                                if frac > 1 {
                                        frac = 1
                                }
                                dc.SetColor(p.Track)
                                dc.DrawRoundedRectangle(cx-rl.sw/2+26, shieldCy+rl.sh/2-34, rl.sw-52, 6, 3)
                                dc.Fill()
                                dc.SetColor(p.Accent)
                                dc.DrawRoundedRectangle(cx-rl.sw/2+26, shieldCy+rl.sh/2-34, (rl.sw-52)*frac, 6, 3)
                                dc.Fill()
                                cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), cx, shieldCy+rl.sh/2-16, cardstyle.WithA(p.Muted, 220), 0.5, 0.5, rl.sw-40, 8)
                        }
                }
        }
        return dc
}

// SHOP - the court emporium: twin-column ledger of wares.
func s10Shop(req *portraitRequest) *gg.Context {
        entries := req.Entries
        if len(entries) > 8 {
                entries = entries[:8]
        }
        nE := len(entries)
        if nE < 1 {
                nE = 1
        }
        const W = 800.0
        H := 190.0 + float64(nE)*108.0 + 130.0
        dc := gg.NewContext(int(W), int(H))
        p := s10Base(dc, W, H)

        cardstyle.Cartouche(dc, W/2, 70, 460, 56, p.Panel, p.Accent, p.Ink, "THE COURT EMPORIUM", cardstyle.FtCinzelDec, 19)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, 116, p.Muted, 0.5, 2.4)

        if len(entries) == 0 {
                entries = nil
        }
        // central rail
        dc.SetColor(cardstyle.WithA(p.Accent, 150))
        dc.SetLineWidth(1.2)
        dc.DrawLine(W/2, 160, W/2, 950)
        dc.Stroke()
        for yy := 170.0; yy <= 950; yy += 46 {
                cardstyle.Diamond(dc, W/2, yy, 3.4, cardstyle.WithA(p.Accent, 170))
        }
        // split entries into two columns
        half := (len(entries) + 1) / 2
        cols := [][]int{}
        if len(entries) > 0 {
                cols = [][]int{ {0}, {1} }
                cols[0] = []int{}
                cols[1] = []int{}
                for i := range entries {
                        if i < half {
                                cols[0] = append(cols[0], i)
                        } else {
                                cols[1] = append(cols[1], i)
                        }
                }
        }
        colX := []float64{56, W/2 + 24}
        colW := []float64{W/2 - 80, W/2 - 80}
        for ci, idxs := range cols {
                if len(idxs) == 0 {
                        continue
                }
                cx := colX[ci]
                cw := colW[ci]
                n := float64(len(idxs))
                step := 200.0
                if n > 1 {
                        step = 700 / (n - 1)
                        if step > 240 {
                                step = 240
                        }
                } else {
                        step = 0
                }
                y0 := 240.0
                if n == 1 {
                        y0 = 400
                }
                for k, ei := range idxs {
                        e := entries[ei]
                        yy := y0 + float64(k)*step
                        cardstyle.Diamond(dc, cx, yy-2, 4.6, p.Accent)
                        cardstyle.Text(dc, cardstyle.FtCinzel, 15, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(e.Title)), 16), cx+18, yy-2, p.Ink, 0, 0.5, cw-90, 10)
                        cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(e.Value)), cx+cw, yy-2, p.Accent, 1, 0.5, 110, 9)
                        extra := e.Sub
                        if e.Runes != "" {
                                extra = strings.ToUpper(cardstyle.Sanitize(e.Runes)) + "  " + extra
                        }
                        if extra != "" {
                                cardstyle.Text(dc, cardstyle.FtCinzel, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(extra)), 52), cx+18, yy+20, cardstyle.WithA(p.Muted, 200), 0, 0.5, cw-20, 8)
                        }
                        dc.SetColor(cardstyle.WithA(p.Accent, 80))
                        dc.SetLineWidth(0.9)
                        dc.DrawLine(cx, yy+44, cx+cw, yy+44)
                        dc.Stroke()
                }
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(req.Caption)), W/2, 1000, p.Accent, 0.5, 0.5, 400, 9)
        s10Footer(dc, W, H, req, p)
        return dc
}

// GUILDINFO - chapter arms: great shield + twin-column charter.
func s10GuildInfo(req *portraitRequest) *gg.Context {
        const W, H = 800.0, 860.0
        dc := gg.NewContext(int(W), int(H))
        p := s10Base(dc, W, H)

        cardstyle.Cartouche(dc, W/2, 66, 480, 54, p.Panel, p.Accent, p.Ink, "CHAPTER ARMS", cardstyle.FtCinzelDec, 19)

        accent := cardstyle.Hex(0x801c28)
        if req.HexColor != "" {
                hx := utils.ParseHexColor(req.HexColor)
                accent = cardstyle.N(hx.R, hx.G, hx.B, 255)
        }
        cardstyle.Shield(dc, W/2, 226, 180, 210, accent, p.Accent, 4)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 46, firstRuneUpper(styledName(req)), W/2, 216, cardstyle.Hex(0xf4e2ce), 0.5, 0.5, 110, 18)
        name := styledName(req)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(name)), W/2, 360, p.Ink, 0.5, 0.5, W-140, 12)
        if req.Motto != "" {
                cardstyle.Text(dc, cardstyle.FtCinzel, 12, "\""+cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Motto)), 58)+"\"", W/2, 388, p.Muted, 0.5, 0.5, W-120, 9)
        }

        // twin columns
        colY0, colY1 := 448.0, 780.0
        dc.SetColor(cardstyle.WithA(p.Accent, 150))
        dc.SetLineWidth(1.2)
        dc.DrawLine(W/2, colY0-16, W/2, colY1+10)
        dc.Stroke()
        for yy := colY0; yy <= colY1; yy += 40 {
                cardstyle.Diamond(dc, W/2, yy, 3.2, cardstyle.WithA(p.Accent, 160))
        }
        // left: the charter
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE CHARTER", 216, colY0, p.Accent, 0.5, 2)
        rows := req.Rows
        if len(rows) > 4 {
                rows = rows[:4]
        }
        step := (colY1 - colY0 - 60) / 4
        for i, r := range rows {
                yy := colY0 + 44 + float64(i)*step
                cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.Sanitize(r.Label)), 56, yy, p.Muted, 0, 0.5, 150, 9)
                cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.Sanitize(r.Value)), 376, yy, p.Ink, 1, 0.5, 180, 9)
        }
        // right: level + holdings
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "THE HOLDINGS", 584, colY0, p.Accent, 0.5, 2)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, "LEVEL", 424, colY0+44, p.Muted, 0, 0.5, 100, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 15, fmt.Sprintf("%d", req.Level), 744, colY0+44, p.Ink, 1, 0.5, 80, 9)
        pct := float64(req.XPPercent) / 100
        dc.SetColor(p.Track)
        dc.DrawRoundedRectangle(424, colY0+56, 320, 6, 3)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.DrawRoundedRectangle(424, colY0+56, 320*pct, 6, 3)
        dc.Fill()
        bs := req.Buildings
        if len(bs) > 3 {
                bs = bs[:3]
        }
        for i, b := range bs {
                yy := colY0 + 118 + float64(i)*88
                cardstyle.Pennant(dc, 424, yy, 150, 34, p.Panel, cardstyle.WithA(p.Accent, 140), p.Ink, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(b.Name), 10)), cardstyle.FtCinzel, 11)
                cardstyle.Text(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("LVL %d", b.Level), 744, yy+16, p.Accent2, 1, 0.5, 90, 9)
        }
        s10Footer(dc, W, H, req, p)
        return dc
}
