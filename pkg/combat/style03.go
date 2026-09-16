package combat

// style03.go - RETRO COURT "The Guild Playbill" (Style 3).
// Victorian broadsheet: typographic, NO panels. Oversized masthead, hairline
// double rules, fleuron separators, small-caps subheads, dotted-leader bill
// rows, halftone paper. IM Fell English typography, burgundy accents.
// Recognizable by structure: pure typography + rules + dotted bills.

import (
        "fmt"
        "image"
        "strings"

        "image-service/pkg/cardstyle"

        "github.com/fogleman/gg"
)

func s03Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
        p := cardstyle.RetroCourt()
        cardstyle.PageBase(dc, W, H, p.Bg, p.Bg2, 26)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2, p.Accent, false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 15, 0.7, cardstyle.WithA(p.Accent, 150), false)
        cardstyle.Vignette(dc, W, H, p.Vign)
        return &p
}

// s03Masthead - oversized playbill title with double rules.
func s03Masthead(dc *gg.Context, W float64, y float64, title, sub string, size int) float64 {
        p := cardstyle.RetroCourt()
        cardstyle.Text(dc, cardstyle.FtIMFell, size, strings.ToUpper(cardstyle.Sanitize(title)), W/2, y, p.Ink, 0.5, 0.5, W-70, 20)
        y += float64(size) * 0.72
        if sub != "" {
                cardstyle.Spaced(dc, cardstyle.FtIMFell, 12, strings.ToUpper(cardstyle.Sanitize(sub)), W/2, y, p.Accent, 0.5, 2.6)
                y += 18
        }
        dc.SetColor(p.Ink)
        dc.SetLineWidth(1.6)
        dc.DrawLine(52, y+4, W-52, y+4)
        dc.Stroke()
        dc.SetLineWidth(0.6)
        dc.DrawLine(52, y+9, W-52, y+9)
        dc.Stroke()
        cardstyle.Fleuron(dc, W/2, y+6, 4, p.Accent)
        return y + 28
}

// s03BillRow - dotted-leader playbill row.
func s03BillRow(dc *gg.Context, x, y, w float64, label, value, note string, p *cardstyle.Palette, big bool) {
        size := 17
        if big {
                size = 20
        }
        cardstyle.Text(dc, cardstyle.FtIMFell, size, cardstyle.Sanitize(label), x, y, p.Ink, 0, 0.5, w*0.55, 10)
        lw := cardstyle.SpacedWidth(dc, cardstyle.FtIMFell, size, cardstyle.TruncateRunes(cardstyle.Sanitize(label), 40), 0)
        cardstyle.DotLeader(dc, x+lw+14, x+w-cardstyle.SpacedWidth(dc, cardstyle.FtIMFell, size, cardstyle.Sanitize(value), 0)-10, y, 1.1, cardstyle.WithA(p.Ink, 150))
        cardstyle.Text(dc, cardstyle.FtIMFell, size, cardstyle.Sanitize(value), x+w, y, p.Accent, 1, 0.5, w*0.4, 10)
        if note != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 12, cardstyle.Sanitize(note), x, y+18, p.Muted, 0, 0.5, w*0.6, 8)
        }
}

func s03SubHead(dc *gg.Context, x, y float64, text string, p *cardstyle.Palette) {
        cardstyle.Text(dc, cardstyle.FtIMFell, 15, strings.ToUpper(cardstyle.Sanitize(text)), x, y, p.Accent, 0, 0.5, 400, 10)
        dc.SetColor(cardstyle.WithA(p.Ink, 130))
        dc.SetLineWidth(0.7)
        dc.DrawLine(x, y+12, x+cardstyle.SpacedWidth(dc, cardstyle.FtIMFell, 15, strings.ToUpper(cardstyle.Sanitize(text)), 0)+16, y+12)
        dc.Stroke()
}

func s03Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
        cardstyle.Fleuron(dc, 52, H-40, 4, p.Accent)
        if req.Caption != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 14, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 62), 78, H-40, p.Muted, 0, 0.5, W-160, 9)
        }
        cardstyle.SealMini(dc, W-52, H-40, 19, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtIMFell, 11)
}

func renderStyle03(kind string, req *portraitRequest) image.Image {
        switch kind {
        case "RANK":
                return s03Rank(req).Image()
        case "ALLOCATE":
                return s03Allocate(req).Image()
        case "SKILLUP":
                return s03Skillup(req).Image()
        case "ABILITIES":
                return s03Abilities(req).Image()
        case "SKILLTREE":
                return s03Skilltree(req).Image()
        case "EQUIP":
                return s03Equip(req).Image()
        case "SHOP":
                return s03Shop(req).Image()
        case "GUILDINFO":
                return s03GuildInfo(req).Image()
        }
        return nil
}

// RANK - masthead + programme bill + standing column.
// s03Classifieds - newspaper "classified advertisements" box filling short pages.
func s03Classifieds(dc *gg.Context, x, y, w, yMax float64, p *cardstyle.Palette) {
        if yMax-y < 120 {
                return
        }
        h := yMax - y
        if h > 210 {
                h = 210
        }
        // bordered ad box with double rules
        dc.SetColor(cardstyle.WithA(p.Track, 60))
        dc.DrawRectangle(x, y, w, h)
        dc.Fill()
        dc.SetColor(cardstyle.WithA(p.Accent, 170))
        dc.SetLineWidth(1.4)
        dc.DrawRectangle(x, y, w, h)
        dc.Stroke()
        dc.SetLineWidth(0.6)
        dc.DrawRectangle(x+5, y+5, w-10, h-10)
        dc.Stroke()
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "CLASSIFIED ADVERTISEMENTS", x+w/2, y+24, p.Accent, 0.5, 2.4)
        dc.SetLineWidth(0.8)
        dc.DrawLine(x+16, y+38, x+w-16, y+38)
        dc.Stroke()
        ads := []string{
                "WANTED: BRAVE SOULS FOR DEEP DELVING - INQUIRE WITHIN",
                "LOST: ONE WIZARD'S HAT, SLIGHTLY SINGED - GENEROUS REWARD",
                "FOR SALE: GENTLY CURSED AMULETS, ALL SALES FINAL",
        }
        by := y + 62.0
        step := (h - 78) / 3
        if step > 34 {
                step = 34
        }
        for i, ad := range ads {
                cardstyle.Text(dc, cardstyle.FtIMFell, 12, ad, x+16, by+float64(i)*step, p.Muted, 0, 0.5, w-32, 9)
        }
}

func s03Rank(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 62, styledName(req), "THE GUILD PLAYBILL PRESENTS", 34)
        cardstyle.Spaced(dc, cardstyle.FtIMFell, 12, fmt.Sprintf("LEVEL %d", req.Level), W/2, y, p.Accent, 0.5, 2.6)
        y += 24
        rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
        if rankLine == "" && req.RankLetter != "" {
                rankLine = req.RankLetter + "-RANK ADVENTURER"
        }
        if rankLine != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, rankLine, W/2, y, p.Ink, 0.5, 0.5, W-90, 10)
                y += 26
        }

        s03SubHead(dc, 56, y+8, "PROGRAMME", p)
        y += 34
        // experience as bill row
        cardstyle.Text(dc, cardstyle.FtIMFell, 15, "EXPERIENCE", 56, y, p.Ink, 0, 0.5, 180, 9)
        lw := cardstyle.SpacedWidth(dc, cardstyle.FtIMFell, 15, "EXPERIENCE", 0)
        cardstyle.DotLeader(dc, 56+lw+12, W-190, y, 1.1, cardstyle.WithA(p.Ink, 140))
        cardstyle.Text(dc, cardstyle.FtIMFell, 15, cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft), W-56, y, p.Accent, 1, 0.5, 200, 9)
        y += 22
        pct := float64(req.XPPercent) / 100
        dc.SetColor(p.Track)
        dc.DrawRoundedRectangle(56, y, W-112, 7, 3)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.DrawRoundedRectangle(56, y, (W-112)*pct, 7, 3)
        dc.Fill()
        y += 30

        rows := req.Standing
        if len(rows) > 5 {
                rows = rows[:5]
        }
        for _, r := range rows {
                s03BillRow(dc, 56, y, W-112, r.Label, r.Value, "", p, false)
                y += 30
        }

        y += 12
        s03SubHead(dc, 56, y, cardstyle.Sanitize(req.ProgressTitle), p)
        y += 30
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
                s03BillRow(dc, 56, y, W-112, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText), "", p, false)
                dc.SetColor(p.Track)
                dc.DrawRoundedRectangle(56, y+13, W-112, 5, 2)
                dc.Fill()
                dc.SetColor(p.Accent)
                dc.DrawRoundedRectangle(56, y+13, (W-112)*frac, 5, 2)
                dc.Fill()
                y += 44
        }
        s03Footer(dc, W, H, req, p)
        return dc
}

// ALLOCATE - "THE SEVEN WONDERS" bill.
func s03Allocate(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 62, "ALLOCATION", "BY ORDER OF THE GUILD TREASURER", 36)
        avail := parseLeadingInt(req.PointsBig, 0)
        cardstyle.Text(dc, cardstyle.FtIMFell, 17, fmt.Sprintf("%d UNSPENT POINTS", avail), W/2, y, p.Ink, 0.5, 0.5, W-100, 11)
        y += 24
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 13, strings.ToLower(cardstyle.Sanitize(req.Pill)), W/2, y, p.Muted, 0.5, 0.5, W-100, 9)
        y += 26

        s03SubHead(dc, 56, y, "THE SEVEN WONDERS", p)
        y += 32
        rows := req.Rows
        if len(rows) > 7 {
                rows = rows[:7]
        }
        for _, r := range rows {
                code := strings.ToUpper(cardstyle.Sanitize(r.Label))
                s03BillRow(dc, 56, y, W-112, cardstyle.StatFullName(code), gainFromSub(r.Sub), "", p, true)
                y += 36
        }
        y += 8
        dc.SetColor(p.Ink)
        dc.SetLineWidth(1.2)
        dc.DrawLine(56, y, W-56, y)
        dc.Stroke()
        y += 26
        spent := parseLeadingInt(req.SpentNow, 0)
        s03BillRow(dc, 56, y, W-112, "SPENT", fmt.Sprintf("%d pts", spent), cardstyle.Sanitize(req.SpentLeft), p, true)
        y += 52
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 13, ".j allocate <stat> [amount]", W/2, y, p.Accent, 0.5, 0.5, W-90, 9)
        y += 22
        cardstyle.FooterNote(dc, W/2, y, "Half value per point after heavy single-stat investment", p.Muted, cardstyle.FtIMFell, 12, W-90)
        s03Footer(dc, W, H, req, p)
        return dc
}

// SKILLUP - star billing announcement.
func s03Skillup(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 62, "SKILL UP", "A STAR BILLING ANNOUNCEMENT", 34)
        cardstyle.Spaced(dc, cardstyle.FtIMFell, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), W/2, y, p.Muted, 0.5, 2.4)
        y += 26

        cardstyle.Fleuron(dc, W/2, y+6, 5, p.Accent)
        y += 30
        cardstyle.Text(dc, cardstyle.FtIMFell, 30, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, y, p.Ink, 0.5, 0.5, W-90, 16)
        y += 42
        tierLabel := fmt.Sprintf("TIER %d", req.Tier)
        if req.Ascended {
                tierLabel = "ASCENDED"
        }
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 16, tierLabel, W/2, y, p.Accent, 0.5, 0.5, W-90, 10)
        y += 36

        // level pips as small print dots
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
        totalW := float64(maxPips-1) * 28
        sx := W/2 - totalW/2
        for i := 0; i < maxPips; i++ {
                if i < cur {
                        dc.SetColor(p.Accent)
                        dc.DrawCircle(sx+float64(i)*28, y, 5.5)
                        dc.Fill()
                } else {
                        dc.SetColor(cardstyle.WithA(p.Ink, 110))
                        dc.SetLineWidth(1.2)
                        dc.DrawCircle(sx+float64(i)*28, y, 5)
                        dc.Stroke()
                }
        }
        y += 44

        s03SubHead(dc, 56, y, "THE BILL", p)
        y += 30
        frac := 0.0
        if maxPips > 0 {
                frac = float64(cur) / float64(maxPips)
        }
        s03BillRow(dc, 56, y, W-112, "MASTERY", fmt.Sprintf("%d / %d", cur, maxPips), "", p, true)
        dc.SetColor(p.Track)
        dc.DrawRoundedRectangle(56, y+13, W-112, 5, 2)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.DrawRoundedRectangle(56, y+13, (W-112)*frac, 5, 2)
        dc.Fill()
        s03Classifieds(dc, 56, H-104-210, W-112, H-104, p)
        s03Footer(dc, W, H, req, p)
        return dc
}

// ABILITIES - evening programme list.
func s03Abilities(req *portraitRequest) *gg.Context {
        W := 1000.0
        H := float64(abilitiesH(req))
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        title := strings.TrimSpace(req.DocTitle)
        if title == "" {
                title = "THE PROGRAMME"
        }
        y := s03Masthead(dc, W, 56, title, "AN EVENING OF ABILITIES", 30)
        if strings.TrimSpace(req.DocQuote) != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 14, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.DocQuote), 84)+"\"", W/2, y, p.Muted, 0.5, 0.5, W-160, 9)
                y += 22
        }
        if req.PageLabel != "" {
                cardstyle.Text(dc, cardstyle.FtIMFell, 12, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-56, 66, p.Accent, 1, 0.5, 160, 8)
        }

        num := req.StartNumber
        if num < 1 {
                num = 1
        }
        for gi, g := range req.Groups {
                if gi >= 6 {
                        break
                }
                s03SubHead(dc, 60, y, g.Name, p)
                y += 28
                for _, it := range g.Items {
                        cardstyle.Text(dc, cardstyle.FtIMFell, 13, fmt.Sprintf("%02d.", num), 60, y, p.Accent, 0, 0.5, 40, 8)
                        s03BillRow(dc, 112, y, W-172, it.Title, it.Runes, it.Sub, p, false)
                        if it.Value != "" {
                                cardstyle.Text(dc, cardstyle.FtIMFellIt, 11, cardstyle.Sanitize(it.Value), W-60, y+18, p.Muted, 1, 0.5, 180, 8)
                        }
                        num++
                        y += 48
                }
                y += 10
                cardstyle.Fleuron(dc, W/2, y-6, 3.4, cardstyle.WithA(p.Accent, 150))
                y += 12
        }
        s03Classifieds(dc, 56, H-104-210, W-112, H-104, p)
        s03Footer(dc, W, H, req, p)
        return dc
}

// SKILLTREE - parlour game winding path with numbered stations.
func s03Skilltree(req *portraitRequest) *gg.Context {
        const W, H = 1200.0, 900.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        s03Masthead(dc, W, 52, "THE PARLOUR OF MASTERY", strings.ToUpper(cardstyle.Sanitize(req.ClassName)), 30)
        pts := req.SkillPoints
        ptsText := "NO POINTS"
        if pts == 1 {
                ptsText = "1 POINT"
        } else if pts > 1 {
                ptsText = fmt.Sprintf("%d POINTS", pts)
        }
        cardstyle.Text(dc, cardstyle.FtIMFell, 14, ptsText, W-70, 66, p.Accent, 1, 0.5, 160, 9)

        // winding route: serpentine stations across the board
        n := len(req.Branches)
        if n == 0 {
                n = 1
        }
        // collect stations: branch-major, tier order
        type station struct {
                node  portraitNode
                br    int
                x, y  float64
                label string
        }
        stations := []station{}
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
        cols := n
        if cols < 3 {
                cols = 3
        }
        cellW := (W - 160) / float64(cols)
        rowH := 118.0
        for bi, br := range req.Branches {
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
                        for _, s := range byTier[t] {
                                stations = append(stations, station{node: s, br: bi})
                        }
                }
        }
        // assign serpentine rows
        for i := range stations {
                row := i / cols
                col := i % cols
                if row%2 == 1 {
                        col = cols - 1 - col
                }
                stations[i].x = 80 + float64(col)*cellW + cellW/2
                stations[i].y = 252 + float64(row)*rowH
                stations[i].label = fmt.Sprintf("%02d", i+1)
        }
        // connect route
        dc.SetColor(cardstyle.WithA(p.Ink, 120))
        dc.SetLineWidth(1.4)
        dc.SetDash(6, 6)
        for i := 0; i < len(stations)-1; i++ {
                dc.DrawLine(stations[i].x, stations[i].y, stations[i+1].x, stations[i+1].y)
                dc.Stroke()
        }
        dc.SetDash()
        // stations
        for i, st := range stations {
                lit := st.node.State == "learned" || st.node.State == "maxed"
                r := 26.0
                if lit {
                        dc.SetColor(p.Accent)
                        dc.DrawCircle(st.x, st.y, r)
                        dc.Fill()
                        dc.SetColor(cardstyle.Hex(0xf4e2c4))
                } else {
                        dc.SetColor(cardstyle.Hex(0xf0e4c4))
                        dc.DrawCircle(st.x, st.y, r)
                        dc.Fill()
                        dc.SetColor(p.Accent)
                }
                dc.SetLineWidth(2)
                dc.DrawCircle(st.x, st.y, r)
                dc.Stroke()
                cardstyle.Text(dc, cardstyle.FtIMFell, 15, st.label, st.x, st.y-2, p.Ink, 0.5, 0.5, 40, 9)
                cardstyle.Text(dc, cardstyle.FtIMFell, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(st.node.Name)), 14), st.x, st.y+r+14, p.Ink, 0.5, 0.5, cellW-14, 8)
                if st.node.State == "maxed" {
                        cardstyle.Fleuron(dc, st.x+r-4, st.y-r+4, 3, p.Accent)
                }
                _ = i
        }
        // branch legend row
        lx := 80.0
        for bi, br := range req.Branches {
                dc.SetColor(p.Accent)
                dc.DrawCircle(lx+6, H-52, 5)
                dc.Fill()
                cardstyle.Text(dc, cardstyle.FtIMFell, 12, strings.ToUpper(cardstyle.Sanitize(br.Name)), lx+20, H-52, p.Muted, 0, 0.5, 140, 8)
                lx += 60 + cardstyle.SpacedWidth(dc, cardstyle.FtIMFell, 12, strings.ToUpper(cardstyle.Sanitize(br.Name)), 0)
                _ = bi
        }
        s03Footer(dc, W, H, req, p)
        return dc
}

// EQUIP - costume inventory bill.
func s03Equip(req *portraitRequest) *gg.Context {
        const W, H = 1200.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 54, "THE WARDROBE", strings.ToUpper(cardstyle.Sanitize(styledName(req))), 30)
        s03SubHead(dc, 70, y, "COSTUME INVENTORY", p)
        y += 34
        // hero vignette left column + bill right
        cardstyle.Shield(dc, 190, y+180, 220, 330, cardstyle.Hex(0xf0e4c4), p.Accent, 2.4)
        drawHeroFitted(dc, req, 190, y+160, 190, 220, *p)
        cardstyle.Text(dc, cardstyle.FtIMFell, 16, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 190, y+382, p.Ink, 0.5, 0.5, 240, 10)
        if req.SealText != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 13, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", 190, y+410, p.Accent, 0.5, 0.5, 200, 9)
        }

        bx, bw := 400.0, W-470.0
        for i, s := range req.Slots {
                var label, value, note string
                slotCode := strings.ToUpper(cardstyle.Sanitize(s.Slot))
                if s.Empty {
                        label = slotCode
                        value = "EMPTY"
                        note = "-"
                } else {
                        label = slotCode + " - " + cardstyle.Sanitize(s.Name)
                        value = strings.ToUpper(cardstyle.Sanitize(s.TierLabel))
                        if s.DurMax > 0 {
                                note = fmt.Sprintf("DURABILITY %s / %s", cardstyle.FmtF(s.Dur), cardstyle.FmtF(s.DurMax))
                        } else {
                                note = "SOLID"
                        }
                }
                s03BillRow(dc, bx, y+float64(i)*52, bw, label, value, note, p, false)
                if !s.Empty && s.DurMax > 0 {
                        frac := s.Dur / s.DurMax
                        if frac > 1 {
                                frac = 1
                        }
                        dc.SetColor(p.Track)
                        dc.DrawRoundedRectangle(bx, y+float64(i)*52+30, bw*0.55, 4, 2)
                        dc.Fill()
                        dc.SetColor(p.Accent)
                        dc.DrawRoundedRectangle(bx, y+float64(i)*52+30, bw*0.55*frac, 4, 2)
                        dc.Fill()
                }
        }
        s03Footer(dc, W, H, req, p)
        return dc
}

// SHOP - playbill advertisements.
func s03Shop(req *portraitRequest) *gg.Context {
        entries := req.Entries
        if len(entries) > 8 {
                entries = entries[:8]
        }
        nE := len(entries)
        if nE < 1 {
                nE = 1
        }
        const W = 800.0
        H := 196.0 + float64(nE)*62.0 + 344.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 56, "THE EMPORIUM", "ADVERTISEMENTS", 32)
        for _, e := range entries {
                s03BillRow(dc, 60, y, W-120, e.Title, e.Value, e.Sub, p, true)
                if e.Runes != "" {
                        cardstyle.Text(dc, cardstyle.FtIMFell, 12, cardstyle.Sanitize(e.Runes), 60, y+36, p.Accent, 0, 0.5, 300, 8)
                }
                y += 62
                dc.SetColor(cardstyle.WithA(p.Ink, 90))
                dc.SetLineWidth(0.6)
                dc.DrawLine(60, y-14, W-60, y-14)
                dc.Stroke()
        }
        s03Classifieds(dc, 56, H-104-210, W-112, H-104, p)
        s03Footer(dc, W, H, req, p)
        return dc
}

// GUILDINFO - society letterhead.
func s03GuildInfo(req *portraitRequest) *gg.Context {
        const W, H = 800.0, 800.0
        dc := gg.NewContext(int(W), int(H))
        p := s03Base(dc, W, H)

        y := s03Masthead(dc, W, 56, styledName(req), "SOCIETY OF ADVENTURERS", 30)
        if req.Motto != "" {
                cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 66)+"\"", W/2, y, p.Muted, 0.5, 0.5, W-110, 10)
                y += 24
        }
        // crest medallion
        dc.SetColor(cardstyle.Hex(0xf0e4c4))
        dc.DrawCircle(W/2, y+64, 52)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.SetLineWidth(2)
        dc.DrawCircle(W/2, y+64, 52)
        dc.Stroke()
        dc.SetLineWidth(0.8)
        dc.DrawCircle(W/2, y+64, 44)
        dc.Stroke()
        cardstyle.Text(dc, cardstyle.FtIMFell, 40, firstRuneUpper(styledName(req)), W/2, y+60, p.Accent, 0.5, 0.5, 80, 18)
        y += 142

        s03SubHead(dc, 70, y, "THE CHARTER", p)
        y += 30
        rows := req.Rows
        if len(rows) > 4 {
                rows = rows[:4]
        }
        for _, r := range rows {
                s03BillRow(dc, 70, y, W-140, r.Label, r.Value, "", p, false)
                y += 32
        }
        y += 8
        s03BillRow(dc, 70, y, W-140, "LEVEL", fmt.Sprintf("%d", req.Level), "", p, true)
        y += 30
        pct := float64(req.XPPercent) / 100
        dc.SetColor(p.Track)
        dc.DrawRoundedRectangle(70, y, W-140, 6, 3)
        dc.Fill()
        dc.SetColor(p.Accent)
        dc.DrawRoundedRectangle(70, y, (W-140)*pct, 6, 3)
        dc.Fill()
        y += 30
        for i, b := range req.Buildings {
                if i >= 3 {
                        break
                }
                s03BillRow(dc, 70+float64(i)*((W-140)/3), y, (W-140)/3-16, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 9), fmt.Sprintf("L%d", b.Level), "", p, false)
        }
        s03Footer(dc, W, H, req, p)
        return dc
}
