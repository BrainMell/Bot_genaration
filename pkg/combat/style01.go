package combat

// style01.go - STONEKEEP "The Bastion Ledger" (Style 1).
// The card IS a fortress wall: stacked carved stone slabs with real mortar
// gaps (never one continuous panel), iron header plate top-LEFT with rivets,
// stamped text, iron tag values, torch-lit accents, speckled granite.
// Recognizable by structure: slab courses + rivets + iron plates.

import (
        "math"
        "fmt"
        "image"
        "strings"

        "image-service/pkg/cardstyle"
        "image-service/pkg/utils"

        "github.com/fogleman/gg"
)

func s01Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
        p := cardstyle.Stonekeep()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        // masonry seams behind everything
        dc.SetColor(cardstyle.N(0, 0, 0, 46))
        for y := 138.0; y < H; y += 138 {
                dc.SetLineWidth(2.2)
                dc.DrawLine(0, y, W, y)
                dc.Stroke()
        }
        for x := 150.0; x < W; x += 150 {
                dc.SetLineWidth(2)
                dc.DrawLine(x, 0, x, H)
                dc.Stroke()
        }
        cardstyle.Speckle(dc, 0, 0, W, H, 90, 0xA11CE, cardstyle.N(0, 0, 0, 255))
        return &p
}

// s01Slab - carved stone slab section with bevel + corner rivets.
func s01Slab(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette) {
        cardstyle.BevelRect(dc, x, y, w, h, p.Panel, cardstyle.Lighten(p.Panel, 42), cardstyle.Darken(p.Panel, 66), 3)
        cardstyle.Speckle(dc, x, y, x+w, y+h, int(w*h/2600), 0xBEEF7, cardstyle.N(0, 0, 0, 255))
        for _, c := range [][2]float64{{x + 10, y + 10}, {x + w - 10, y + 10}, {x + 10, y + h - 10}, {x + w - 10, y + h - 10}} {
                cardstyle.Rivet(dc, c[0], c[1], 4.2, p.Track)
        }
}

// s01IronPlate - riveted iron header plate top-left with stamped title.
func s01IronPlate(dc *gg.Context, W float64, title, sub string, p *cardstyle.Palette) {
        x, y, w, h := 40.0, 34.0, 420.0, 78.0
        cardstyle.BevelRect(dc, x, y, w, h, p.Accent, cardstyle.Lighten(p.Accent, 40), cardstyle.N(10, 10, 12, 255), 2.6)
        for _, c := range [][2]float64{{x + 9, y + 9}, {x + w - 9, y + 9}, {x + 9, y + h - 9}, {x + w - 9, y + h - 9}} {
                cardstyle.Rivet(dc, c[0], c[1], 4.6, p.Track)
        }
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, strings.ToUpper(cardstyle.Sanitize(title)), x+18, y+30, p.SealTx, 0, 0.5, w-40, 13)
        if sub != "" {
                cardstyle.Spaced(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.Sanitize(sub)), x+18, y+h-16, cardstyle.WithA(p.SealTx, 170), 0, 1.8)
        }
        // keystone date-stamp block right
        kx, kw := W-150, 110.0
        cardstyle.BevelRect(dc, kx, y, kw, h, cardstyle.Darken(p.Accent, 8), cardstyle.Lighten(p.Accent, 30), cardstyle.N(10, 10, 12, 255), 2.2)
        cardstyle.Rivet(dc, kx+9, y+9, 3.6, p.Track)
        cardstyle.Rivet(dc, kx+kw-9, y+9, 3.6, p.Track)
        // chisel-mark stamp: anvil glyph over a tick rule (never an empty plate)
        ax, ay := kx+kw/2, y+h/2-8
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 9, "EST. J", kx+kw/2, ay-6, p.SealTx, cardstyle.N(0, 0, 0, 255), 0.5, 0.5, 80, 6)
        dc.SetColor(cardstyle.WithA(p.SealTx, 130))
        dc.DrawRectangle(ax-16, ay+4, 32, 3)
        dc.Fill()
        dc.DrawRectangle(ax-10, ay+7, 20, 4)
        dc.Fill()
        dc.DrawRectangle(ax-16, ay+11, 32, 2)
        dc.Fill()
        cardstyle.Diamond(dc, ax, ay-18, 3, cardstyle.WithA(p.SealTx, 170))
}

func s01SectionTitle(dc *gg.Context, x, y float64, text string, p *cardstyle.Palette) {
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 16, strings.ToUpper(cardstyle.Sanitize(text)), x, y, p.Ink, cardstyle.Lighten(p.Panel, 60), 0, 0.5, 320, 10)
        dc.SetColor(cardstyle.Darken(p.Panel, 50))
        dc.SetLineWidth(1.4)
        dc.DrawLine(x, y+12, x+180, y+12)
        dc.Stroke()
}

// s01TagRow - iron tag value row (label engraved, value on a hanging tag).
func s01TagRow(dc *gg.Context, x, y, w float64, label, value string, p *cardstyle.Palette) {
        cardstyle.Engrave(dc, cardstyle.FtInterSemi, 15, strings.ToUpper(cardstyle.Sanitize(label)), x, y, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 210, 9)
        cardstyle.Tag(dc, x+w-92, y-16, 84, 30, p.Track, cardstyle.Lighten(p.Track, 46), p.SealTx, strings.ToUpper(cardstyle.Sanitize(value)), cardstyle.FtInter, 12)
}

func s01Meter(dc *gg.Context, x, y, w, h, frac float64, p *cardstyle.Palette) {
        dc.SetColor(cardstyle.Darken(p.Track, 20))
        dc.DrawRoundedRectangle(x, y, w, h, 4)
        dc.Fill()
        fw := w * frac
        if fw > 4 {
                dc.SetColor(p.Fill)
                dc.DrawRoundedRectangle(x, y, fw, h, 4)
                dc.Fill()
                dc.SetColor(cardstyle.Lighten(p.Fill, 50))
                dc.SetLineWidth(1.2)
                dc.DrawLine(x+3, y+2.4, x+fw-3, y+2.4)
                dc.Stroke()
        }
        dc.SetColor(cardstyle.N(10, 10, 12, 200))
        dc.SetLineWidth(1.6)
        dc.DrawRoundedRectangle(x, y, w, h, 4)
        dc.Stroke()
}

func s01Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
        cardstyle.TorchGlow(dc, 56, H-44, 60, cardstyle.N(255, 179, 64, 255))
        cardstyle.SealMini(dc, 56, H-44, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 12)
        if req.Caption != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 56), 96, H-44, cardstyle.WithA(p.Ink, 200), 0, 0.5, W-150, 10)
        }
}

func renderStyle01(kind string, req *portraitRequest) image.Image {
        switch kind {
        case "RANK":
                return s01Rank(req).Image()
        case "ALLOCATE":
                return s01Allocate(req).Image()
        case "SKILLUP":
                return s01Skillup(req).Image()
        case "ABILITIES":
                return s01Abilities(req).Image()
        case "SKILLTREE":
                return s01Skilltree(req).Image()
        case "EQUIP":
                return s01Equip(req).Image()
        case "SHOP":
                return s01Shop(req).Image()
        case "GUILDINFO":
                return s01GuildInfo(req).Image()
        }
        return nil
}

// RANK - three stacked slabs.
func s01Rank(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)
        name := styledName(req)

        s01IronPlate(dc, W, "THE BASTION RECORD", name, p)

        // slab 1: keystone level
        s01Slab(dc, 40, 132, W-80, 240, p)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 54, fmt.Sprintf("LEVEL %d", req.Level), W/2, 196, p.Ink, cardstyle.Lighten(p.Panel, 60), 0.5, 0.5, 380, 24)
        rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
        if rankLine == "" && req.RankLetter != "" {
                rankLine = req.RankLetter + "-RANK ADVENTURER"
        }
        if rankLine != "" {
                cardstyle.Chip(dc, W/2-120, 240, 240, 30, rankLine, p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 13)
        }
        // xp channel
        pct := float64(req.XPPercent) / 100
        cardstyle.Spaced(dc, cardstyle.FtInter, 11, "EXPERIENCE", 66, 306, cardstyle.WithA(p.Ink, 210), 0, 1.8)
        cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-66, 306, cardstyle.WithA(p.Ink, 230), 1, 0.5, 220, 8)
        s01Meter(dc, 66, 318, W-132, 16, pct, p)
        cardstyle.Text(dc, cardstyle.FtInter, 11, fmt.Sprintf("%d%% TO NEXT LEVEL", req.XPPercent), W/2, 352, cardstyle.WithA(p.Ink, 190), 0.5, 0.5, 260, 8)

        // slab 2: standing tags
        rows := req.Standing
        if len(rows) > 4 {
                rows = rows[:4]
        }
        sh := 64 + float64(len(rows))*44 + 8
        s01Slab(dc, 40, 396, W-80, sh, p)
        s01SectionTitle(dc, 64, 428, "THE STANDING", p)
        ry := 462.0
        for _, r := range rows {
                s01TagRow(dc, 64, ry, W-128, r.Label, r.Value, p)
                ry += 44
        }

        // slab 3: progression channels
        py := 396 + sh + 24.0
        prog := req.Progress
        if len(prog) > 3 {
                prog = prog[:3]
        }
        ph := 64 + float64(len(prog))*52 + 8
        s01Slab(dc, 40, py, W-80, ph, p)
        s01SectionTitle(dc, 64, py+32, cardstyle.Sanitize(req.ProgressTitle), p)
        ry = py + 66
        for _, pr := range prog {
                frac := 0.0
                if pr.Max > 0 {
                        frac = float64(pr.Cur) / float64(pr.Max)
                }
                if pr.Done {
                        frac = 1
                }
                cardstyle.Text(dc, cardstyle.FtInterSemi, 13, strings.ToUpper(cardstyle.Sanitize(pr.Label)), 64, ry, p.Ink, 0, 0.5, 230, 9)
                cardstyle.Text(dc, cardstyle.FtInter, 12, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), W-64, ry, cardstyle.WithA(p.Ink, 220), 1, 0.5, 160, 9)
                fill := p.Fill
                if pr.Done {
                        fill = p.Done
                }
                dc.SetColor(fill)
                s01Meter(dc, 64, ry+12, W-128, 12, frac, p)
                ry += 52
        }
        s01Footer(dc, W, H, req, p)
        return dc
}

// ALLOCATE - keystone number slab + iron tag rows.
func s01Allocate(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "STONE ALLOCATION", styledName(req), p)

        // keystone slab with stamped unspent count
        s01Slab(dc, 40, 132, W-80, 210, p)
        avail := parseLeadingInt(req.PointsBig, 0)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 15, "UNSPENT POINTS", W/2, 166, p.Ink, cardstyle.Lighten(p.Panel, 60), 0.5, 0.5, 240, 9)
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 64, fmt.Sprintf("%d", avail), W/2, 232, cardstyle.N(255, 179, 64, 255), cardstyle.N(255, 190, 90, 255), 0.5, 0.5, 160, 30)
        cardstyle.Text(dc, cardstyle.FtInter, 12, strings.ToUpper(cardstyle.Sanitize(req.Pill)), W/2, 310, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 300, 9)

        // stat tag slabs (two columns of iron tags like a smithy board)
        rows := req.Rows
        if len(rows) > 7 {
                rows = rows[:7]
        }
        cols, cw, ch, gap := 2, 244.0, 74.0, 18.0
        gridX, gridY := 40.0, 366.0
        for i, r := range rows {
                col, row := i%cols, i/cols
                x := gridX + float64(col)*(cw+gap)
                y := gridY + float64(row)*(ch+gap)
                s01Slab(dc, x, y, cw, ch, p)
                code := strings.ToUpper(cardstyle.Sanitize(r.Label))
                cardstyle.Chip(dc, x+12, y+12, 44, 20, code, p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtInter, 10)
                cardstyle.Engrave(dc, cardstyle.FtInterSemi, 12, cardstyle.StatFullName(code), x+12, y+50, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 130, 8)
                cardstyle.Text(dc, cardstyle.FtCinzel, 21, gainFromSub(r.Sub), x+cw-14, y+ch/2, cardstyle.N(255, 179, 64, 255), 1, 0.5, 80, 11)
        }

        // spent channel slab
        sy := gridY + 4*(ch+gap) + 6.0
        if len(rows) <= 4 {
                sy = gridY + 2*(ch+gap) + 6.0
        }
        s01Slab(dc, 40, sy, W-80, 96, p)
        spent := parseLeadingInt(req.SpentNow, 0)
        s01SectionTitle(dc, 64, sy+34, "THE LEDGER", p)
        cardstyle.Engrave(dc, cardstyle.FtInterSemi, 14, "SPENT", 64, sy+66, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 120, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 17, fmt.Sprintf("%d PTS - %s", spent, cardstyle.Sanitize(req.SpentLeft)), W-64, sy+66, cardstyle.N(255, 179, 64, 255), 1, 0.5, 360, 10)

        s01Footer(dc, W, H, req, p)
        return dc
}

// SKILLUP - shield-mount slab.
func s01Skillup(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "FORGED SKILL", styledName(req), p)

        // shield slab with engraved initials
        s01Slab(dc, 150, 150, 300, 300, p)
        cardstyle.Shield(dc, W/2, 296, 190, 220, cardstyle.Darken(p.Track, 16), cardstyle.Lighten(p.Track, 44), 3)
        initials := initialLetters(cardstyle.Sanitize(req.SkillName))
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 64, initials, W/2, 292, cardstyle.Lighten(p.Panel, 66), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 150, 24)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, 496, p.Ink, 0.5, 0.5, W-100, 12)
        tierLabel := fmt.Sprintf("TIER %d", req.Tier)
        if req.Ascended {
                tierLabel = "ASCENDED"
        }
        cardstyle.Chip(dc, W/2-80, 522, 160, 28, tierLabel, p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 13)

        // rivet pips
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
        totalW := float64(maxPips-1) * 34
        sx := W/2 - totalW/2
        for i := 0; i < maxPips; i++ {
                if i < cur {
                        cardstyle.TorchGlow(dc, sx+float64(i)*34, 606, 22, cardstyle.N(255, 179, 64, 255))
                        cardstyle.Rivet(dc, sx+float64(i)*34, 606, 8, cardstyle.N(255, 179, 64, 255))
                } else {
                        cardstyle.Rivet(dc, sx+float64(i)*34, 606, 7, p.Track)
                }
        }

        // forge meter slab
        s01Slab(dc, 40, 660, W-80, 150, p)
        s01SectionTitle(dc, 64, 694, "MASTERY FORGE", p)
        frac := 0.0
        if maxPips > 0 {
                frac = float64(cur) / float64(maxPips)
        }
        dc.SetColor(cardstyle.N(255, 179, 64, 255))
        s01Meter(dc, 64, 716, W-128, 16, frac, p)
        cardstyle.Text(dc, cardstyle.FtInter, 13, fmt.Sprintf("%d / %d", cur, maxPips), W/2, 758, p.Ink, 0.5, 0.5, 140, 9)

        s01Footer(dc, W, H, req, p)
        return dc
}

// ABILITIES - two-column chiseled ledger.
func s01Abilities(req *portraitRequest) *gg.Context {
        W := 1000.0
        H := float64(abilitiesH(req))
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, cardstyle.Sanitize(req.DocTitle), "STONE CODEX", p)
        if req.PageLabel != "" {
                cardstyle.Chip(dc, W-190, 44, 150, 30, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 12)
        }
        if strings.TrimSpace(req.DocQuote) != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 86)+"\"", W/2, 134, cardstyle.WithA(p.Ink, 210), 0.5, 0.5, W-160, 10)
        }

        num := req.StartNumber
        if num < 1 {
                num = 1
        }
        y := 168.0
        for gi, g := range req.Groups {
                if gi >= 6 {
                        break
                }
                gh := 56 + float64(len(g.Items))*56 + 8
                s01Slab(dc, 44, y, W-88, gh, p)
                cardstyle.Engrave(dc, cardstyle.FtCinzel, 17, strings.ToUpper(cardstyle.Sanitize(g.Name)), 68, y+30, p.Ink, cardstyle.Lighten(p.Panel, 60), 0, 0.5, 420, 11)
                iy := y + 64
                for _, it := range g.Items {
                        cardstyle.Chip(dc, 68, iy-12, 42, 24, fmt.Sprintf("%02d", num), p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtInter, 11)
                        cardstyle.Engrave(dc, cardstyle.FtInterSemi, 17, cardstyle.Sanitize(it.Title), 124, iy-4, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 420, 11)
                        if it.Sub != "" {
                                cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(it.Sub), 60), 124, iy+18, cardstyle.WithA(p.Ink, 190), 0, 0.5, 420, 8)
                        }
                        if it.Runes != "" {
                                cardstyle.Spaced(dc, cardstyle.FtCinzel, 15, cardstyle.Sanitize(it.Runes), W-140, iy-4, cardstyle.N(255, 179, 64, 255), 1, 2.6)
                        }
                        if it.Value != "" {
                                cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.Sanitize(it.Value), W-68, iy+18, cardstyle.WithA(p.Ink, 220), 1, 0.5, 220, 9)
                        }
                        num++
                        iy += 56
                }
                y += gh + 16
        }
        // carved rune ring in the free stone field
        fy := (y + H - 80) / 2
        if fy > y+40 && H-80-fy > 40 {
                cardstyle.Ring(dc, W/2, fy, 52, cardstyle.Darken(p.Panel, 44), 3)
                cardstyle.Ring(dc, W/2, fy-1.5, 52, cardstyle.Lighten(p.Panel, 34), 1)
                cardstyle.Ring(dc, W/2, fy, 34, cardstyle.Darken(p.Panel, 50), 1.6)
                cardstyle.TorchGlow(dc, W/2, fy, 40, cardstyle.HexA(0xffb45e, 45))
                dc.SetColor(cardstyle.Lighten(p.Panel, 30))
                for i := 0; i < 8; i++ {
                        a := float64(i) * math.Pi / 4
                        cardstyle.Diamond(dc, W/2+52*math.Cos(a), fy+52*math.Sin(a), 2.6, cardstyle.Lighten(p.Panel, 30))
                }
        }
        s01Footer(dc, W, H, req, p)
        return dc
}

// SKILLTREE - masonry grid (courses = tiers, columns = branches).
func s01Skilltree(req *portraitRequest) *gg.Context {
        const W, H = 1200.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "THE HALL OF MASTERY", cardstyle.Sanitize(req.ClassName), p)
        pts := req.SkillPoints
        ptsText := "NO POINTS"
        if pts == 1 {
                ptsText = "1 POINT"
        } else if pts > 1 {
                ptsText = fmt.Sprintf("%d POINTS", pts)
        }
        cardstyle.Chip(dc, W-250, 44, 200, 34, ptsText, p.Track, cardstyle.Lighten(p.Track, 44), p.SealTx, cardstyle.FtCinzel, 14)

        // masonry courses
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
        cw, chh, gap := 200.0, 140.0, 22.0
        totalW := float64(n)*cw + float64(n-1)*gap
        x0 := (W - totalW) / 2
        y0 := 210.0
        // tier course labels on the left wall
        for t := 1; t <= maxTier; t++ {
                yy := y0 + float64(t-1)*(chh+gap)
                cardstyle.Chip(dc, 36, yy+chh/2-16, 62, 32, fmt.Sprintf("T%d", t), p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 14)
        }
        for bi, br := range req.Branches {
                bx := x0 + float64(bi)*(cw+gap)
                // branch lintel
                cardstyle.BevelRect(dc, bx, y0-58, cw, 44, p.Accent, cardstyle.Lighten(p.Accent, 36), cardstyle.N(10, 10, 12, 255), 2.2)
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(cardstyle.Sanitize(br.Name)), bx+cw/2, y0-36, p.SealTx, 0.5, 0.5, cw-16, 9)
                // blocks by tier
                byTier := map[int][]portraitNode{}
                order := []int{}
                for _, s := range br.Skills {
                        t := s.Tier
                        if t < 1 {
                                t = 1
                        }
                        if t > maxTier {
                                t = maxTier
                        }
                        if _, ok := byTier[t]; !ok {
                                order = append(order, t)
                        }
                        byTier[t] = append(byTier[t], s)
                }
                for _, t := range order {
                        list := byTier[t]
                        yy := y0 + float64(t-1)*(chh+gap)
                        for j, s := range list {
                                // multi-skill tiers share the course (offset chips inside the block)
                                bxx := bx
                                _ = j
                                lit := s.State == "learned" || s.State == "maxed"
                                if lit {
                                        cardstyle.TorchGlow(dc, bxx+cw/2, yy+chh/2, 66, cardstyle.N(255, 179, 64, 255))
                                }
                                base := p.Panel
                                if s.State == "locked" {
                                        base = cardstyle.Darken(p.Panel, 40)
                                } else if lit {
                                        base = cardstyle.Lighten(p.Panel, 24)
                                }
                                cardstyle.BevelRect(dc, bxx, yy, cw, chh, base, cardstyle.Lighten(base, 40), cardstyle.Darken(base, 66), 3)
                                if lit {
                                        dc.SetColor(cardstyle.N(255, 179, 64, 34))
                                        dc.DrawRectangle(bxx, yy, cw, chh)
                                        dc.Fill()
                                }
                                for _, c := range [][2]float64{{bxx + 8, yy + 8}, {bxx + cw - 8, yy + 8}, {bxx + 8, yy + chh - 8}, {bxx + cw - 8, yy + chh - 8}} {
                                        cardstyle.Rivet(dc, c[0], c[1], 3.6, p.Track)
                                }
                                cardstyle.Text(dc, cardstyle.FtInterSemi, 14, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 16), bxx+cw/2, yy+chh/2-12, p.Ink, 0.5, 0.5, cw-20, 9)
                                lvl := fmt.Sprintf("%d/%d", s.Cur, s.Max)
                                col := cardstyle.WithA(p.Ink, 200)
                                if s.State == "maxed" {
                                        col = cardstyle.N(140, 84, 18, 255)
                                }
                                cardstyle.Text(dc, cardstyle.FtInter, 12, lvl, bxx+cw/2, yy+chh/2+16, col, 0.5, 0.5, 80, 8)
                        }
                }
        }
        // mortar links between lit blocks (vertical glow seams)
        cardstyle.Text(dc, cardstyle.FtInter, 13, "TORCH-LIT BLOCKS = LEARNED  ·  AMBER RIVETS = MAXED", W/2, H-40, cardstyle.WithA(p.Ink, 235), 0.5, 0.5, 620, 9)
        cardstyle.SealMini(dc, 60, H-40, 18, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
        if req.Caption != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 60), W/2+60, H-66, cardstyle.WithA(p.Ink, 180), 0.5, 0.5, 500, 9)
        }
        return dc
}

// EQUIP - armory wall with stone racks.
func s01Equip(req *portraitRequest) *gg.Context {
        const W, H = 1500.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "THE ARMORY WALL", cardstyle.Sanitize(styledName(req)), p)

        // hero plinth
        s01Slab(dc, 50, 140, 400, 740, p)
        drawHeroFitted(dc, req, 250, 430, 320, 360, *p)
        dc.SetColor(cardstyle.Darken(p.Panel, 50))
        dc.SetLineWidth(2)
        dc.DrawLine(90, 640, 410, 640)
        dc.Stroke()
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 22, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 250, 680, p.Ink, cardstyle.Lighten(p.Panel, 60), 0.5, 0.5, 320, 12)
        if req.SealText != "" {
                cardstyle.Chip(dc, 175, 716, 150, 32, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 13)
        }
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 56), 250, 790, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 340, 9)

        // 9 rack slabs
        gx, gy, gw, gh, gap := 500.0, 140.0, 300.0, 222.0, 20.0
        for i, s := range req.Slots {
                col, row := i%3, i/3
                x := gx + float64(col)*(gw+gap)
                y := gy + float64(row)*(gh+gap)
                empty := s.Empty
                base := p.Panel
                if empty {
                        base = cardstyle.Darken(p.Panel, 34)
                }
                cardstyle.BevelRect(dc, x, y, gw, gh, base, cardstyle.Lighten(base, 40), cardstyle.Darken(base, 66), 3)
                for _, c := range [][2]float64{{x + 9, y + 9}, {x + gw - 9, y + 9}, {x + 9, y + gh - 9}, {x + gw - 9, y + gh - 9}} {
                        cardstyle.Rivet(dc, c[0], c[1], 4, p.Track)
                }
                slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
                if len([]rune(slotCode)) > 4 {
                        slotCode = string([]rune(slotCode)[:4])
                }
                cardstyle.Chip(dc, x+14, y+14, 56, 24, slotCode, p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtInter, 11)
                for t := 0; t < s.Tier && t < 5; t++ {
                        cardstyle.Rivet(dc, x+gw-26-float64(t)*20, y+26, 5.4, cardstyle.N(255, 179, 64, 255))
                }
                name := "EMPTY RACK"
                ncol := cardstyle.WithA(p.Ink, 140)
                if !empty {
                        name = cardstyle.Sanitize(s.Name)
                        ncol = p.Ink
                }
                cardstyle.Engrave(dc, cardstyle.FtInterSemi, 16, cardstyle.TruncateRunes(name, 22), x+16, y+72, ncol, cardstyle.Lighten(base, 55), 0, 0.5, gw-32, 10)
                if !empty && s.DurMax > 0 {
                        frac := s.Dur / s.DurMax
                        if frac > 1 {
                                frac = 1
                        }
                        s01Meter(dc, x+16, y+gh-44, gw-32, 12, frac, p)
                        cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+16, y+gh-16, cardstyle.WithA(p.Ink, 210), 0, 0.5, gw-32, 8)
                }
        }
        return dc
}

// SHOP - stone market counters.
func s01Shop(req *portraitRequest) *gg.Context {
        entries := req.Entries
        if len(entries) > 8 {
                entries = entries[:8]
        }
        nE := len(entries)
        if nE < 1 {
                nE = 1
        }
        const W = 800.0
        const step = 168.0
        H := 152.0 + float64(nE)*step + 96.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "MARKET HALL", cardstyle.Sanitize(styledName(req)), p)
        // distribute slabs across the hold wall
        y := 152.0
        for _, e := range entries {
                eh := 116.0
                s01Slab(dc, 44, y, W-88, eh, p)
                cardstyle.Engrave(dc, cardstyle.FtInterSemi, 17, cardstyle.Sanitize(e.Title), 66, y+36, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 420, 11)
                if e.Sub != "" {
                        cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 70), 66, y+64, cardstyle.WithA(p.Ink, 190), 0, 0.5, 430, 8)
                }
                if e.Runes != "" {
                        cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, cardstyle.Sanitize(e.Runes), 66, y+90, cardstyle.N(255, 179, 64, 255), 0, 2.2)
                }
                cardstyle.Tag(dc, W-140, y+28, 84, 34, p.Track, cardstyle.Lighten(p.Track, 46), p.SealTx, cardstyle.Sanitize(e.Value), cardstyle.FtCinzel, 13)
                y += step
        }
        // carved anvil sigil in the remaining wall
        fy := (y + H - 70) / 2
        if fy > y+80 && H-70-fy > 60 {
                cardstyle.TorchGlow(dc, W/2, fy, 54, cardstyle.HexA(0xffb45e, 70))
                dc.SetColor(cardstyle.Darken(p.Panel, 46))
                dc.DrawRectangle(W/2-34, fy-8, 68, 16)
                dc.Fill()
                dc.DrawRectangle(W/2-24, fy+8, 48, 10)
                dc.Fill()
                dc.SetColor(cardstyle.Lighten(p.Panel, 40))
                dc.DrawRectangle(W/2-34, fy-8, 68, 2)
                dc.Fill()
        }
        s01Footer(dc, W, H, req, p)
        return dc
}

// GUILDINFO - fortress crest slab.
func s01GuildInfo(req *portraitRequest) *gg.Context {
        const W, H = 800.0, 800.0
        dc := gg.NewContext(int(W), int(H))
        p := s01Base(dc, W, H)

        s01IronPlate(dc, W, "GUILD HOLD", "THE BASTION CHARTER", p)

        // crest shield slab
        s01Slab(dc, W/2-130, 140, 260, 220, p)
        accent := cardstyle.Hex(0x801c28)
        if req.HexColor != "" {
                hx := utils.ParseHexColor(req.HexColor)
                accent = cardstyle.N(hx.R, hx.G, hx.B, 255)
        }
        cardstyle.Shield(dc, W/2, 248, 150, 170, accent, cardstyle.Lighten(p.Panel, 60), 4)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 52, firstRuneUpper(styledName(req)), W/2, 244, p.SealTx, 0.5, 0.5, 110, 24)
        name := styledName(req)
        s01Slab(dc, W/2-280, 372, 560, 96, p)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(cardstyle.Sanitize(name)), W/2, 404, p.Ink, cardstyle.Lighten(p.Panel, 60), 0.5, 0.5, W-160, 14)
        if req.Motto != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 15, "'"+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 60)+"'", W/2, 436, cardstyle.WithA(p.Ink, 210), 0.5, 0.5, W-140, 10)
        }

        // charter slab (rows + buildings row inside)
        rows := req.Rows
        if len(rows) > 4 {
                rows = rows[:4]
        }
        s01Slab(dc, 40, 470, W-80, 64+float64(len(rows))*38+44, p)
        s01SectionTitle(dc, 64, 500, "THE CHARTER", p)
        ry := 532.0
        for _, r := range rows {
                cardstyle.Engrave(dc, cardstyle.FtInterSemi, 13, strings.ToUpper(cardstyle.Sanitize(r.Label)), 64, ry, p.Ink, cardstyle.Lighten(p.Panel, 55), 0, 0.5, 240, 9)
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, cardstyle.Sanitize(r.Value), W-64, ry, cardstyle.WithA(p.Ink, 230), 1, 0.5, 300, 10)
                ry += 38
        }
        bldY := ry + 4.0
        for i, b := range req.Buildings {
                if i >= 3 {
                        break
                }
                bx := 90 + float64(i)*220
                cardstyle.Rivet(dc, bx, bldY, 5, p.Track)
                cardstyle.Text(dc, cardstyle.FtInterSemi, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 10), bx+16, bldY, p.Ink, 0, 0.5, 130, 9)
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, fmt.Sprintf("L%d", b.Level), bx+150, bldY, cardstyle.N(140, 84, 18, 255), 1, 0.5, 50, 9)
        }
        // level + xp below the slab
        lvlY := 470 + 64 + float64(len(rows))*38 + 44 + 12.0
        cardstyle.Chip(dc, 60, lvlY, 110, 34, fmt.Sprintf("LVL %d", req.Level), p.Track, cardstyle.Lighten(p.Track, 40), p.SealTx, cardstyle.FtCinzel, 14)
        pct := float64(req.XPPercent) / 100
        s01Meter(dc, 190, lvlY+10, W-320, 14, pct, p)
        cardstyle.SealMini(dc, W-60, H-44, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 12)
        if req.Caption != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 13, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 52), W/2, H-44, cardstyle.WithA(p.Ink, 180), 0.5, 0.5, 400, 9)
        }
        return dc
}
