package combat

// style08.go - NEON ARCADE "Player One HUD" (Style 8).
// Synthwave arcade cabinet: CRT marquee header with scanlines, HUD panels
// with bracket corners, segmented LED bars, blocky numerals (Press Start 2P),
// horizon grid, magenta primary / cyan secondary glow.
// Recognizable by structure: marquees + bracket panels + LED segments.

import (
        "fmt"
        "image"
        "strings"

        "image-service/pkg/cardstyle"

        "github.com/fogleman/gg"
)

func s08Base(dc *gg.Context, W, H float64) *cardstyle.Palette {
        p := cardstyle.NeonArcade()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        // horizon grid in the lower half
        dc.SetColor(cardstyle.HexA(0x40e0ff, 40))
        dc.SetLineWidth(1)
        for i := 0; i < 9; i++ {
                y := H*0.55 + float64(i)*float64(i)*3.4
                if y > H {
                        break
                }
                dc.DrawLine(0, y, W, y)
                dc.Stroke()
        }
        for i := -6; i <= 6; i++ {
                x0 := W/2 + float64(i)*36
                x1 := W/2 + float64(i)*150
                dc.DrawLine(x0, H*0.55, x1, H)
                dc.Stroke()
        }
        cardstyle.Scanlines(dc, W, H, cardstyle.N(255, 64, 160, 10), 4)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2.2, cardstyle.HexA(0xff40a0, 200), false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 16, 1, cardstyle.HexA(0x40e0ff, 150), false)
        return &p
}

// s08Marquee - CRT title bar.
func s08Marquee(dc *gg.Context, W, y, w, h float64, title, sub string, p *cardstyle.Palette) float64 {
        dc.SetColor(cardstyle.WithA(p.Panel, 235))
        dc.DrawRoundedRectangle(W/2-w/2, y, w, h, 6)
        dc.Fill()
        cardstyle.BracketCorners(dc, W/2-w/2, y, w, h, 16, cardstyle.HexA(0x40e0ff, 220), 2)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 20, strings.ToUpper(cardstyle.Sanitize(title)), W/2, y+h/2-8, cardstyle.HexA(0xff40a0, 120), cardstyle.Hex(0xff9ecb), 0.5, 0.5, w-40, 9)
        if sub != "" {
                cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(sub)), W/2, y+h-12, cardstyle.HexA(0x80a8d0, 230), 0.5, 0.5, w-40, 8)
        }
        return y + h + 18
}

// s08HUDPanel - bracket panel.
func s08HUDPanel(dc *gg.Context, x, y, w, h float64, p *cardstyle.Palette, title string) {
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(x, y, w, h, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, x, y, w, h, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        if title != "" {
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, strings.ToUpper(cardstyle.Sanitize(title)), x+14, y+16, cardstyle.HexA(0xff9ecb, 230), 0, 0.5, w-28, 7)
        }
}

// s08LEDRatio - segmented bar with numeric readout.
func s08LEDRatio(dc *gg.Context, x, y, w, frac float64, p *cardstyle.Palette, label, value string) {
        cardstyle.SegBar(dc, x, y, w, 10, frac, 14, cardstyle.HexA(0xff40a0, 235), cardstyle.HexA(0x40e0ff, 40))
        cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(label)), x, y-10, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, w*0.6, 8)
        cardstyle.Text(dc, cardstyle.FtPS2P, 10, strings.ToUpper(cardstyle.Sanitize(value)), x+w, y-9, cardstyle.HexA(0x40e0ff, 235), 1, 0.5, w*0.4, 7)
}

func s08Footer(dc *gg.Context, W, H float64, req *portraitRequest, p *cardstyle.Palette) {
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, "INSERT COIN TO CONTINUE", W/2, H-26, cardstyle.HexA(0xff9ecb, 190), 0.5, 0.5, 320, 7)
        if req.Caption != "" {
                cardstyle.Text(dc, cardstyle.FtInter, 11, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Caption), 58), 66, H-48, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, W-220, 8)
        }
        cardstyle.SealMini(dc, W-52, H-44, 18, cardstyle.HexA(0xff40a0, 225), cardstyle.Hex(0xfff0fa), cardstyle.Sanitize(req.SealText), cardstyle.FtPS2P, 9)
}

func renderStyle08(kind string, req *portraitRequest) image.Image {
        switch kind {
        case "RANK":
                return s08Rank(req).Image()
        case "ALLOCATE":
                return s08Allocate(req).Image()
        case "SKILLUP":
                return s08Skillup(req).Image()
        case "ABILITIES":
                return s08Abilities(req).Image()
        case "SKILLTREE":
                return s08Skilltree(req).Image()
        case "EQUIP":
                return s08Equip(req).Image()
        case "SHOP":
                return s08Shop(req).Image()
        case "GUILDINFO":
                return s08GuildInfo(req).Image()
        }
        return nil
}

// RANK - PLAYER STATS screen.
func s08Rank(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 34, W-80, 66, styledName(req), "PLAYER STATS.DB", p)

        s08HUDPanel(dc, 46, y, W-92, 150, p, "LEVEL")
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 40, fmt.Sprintf("%d", req.Level), W/2-60, y+84, cardstyle.HexA(0x40e0ff, 120), cardstyle.Hex(0x40e0ff), 0.5, 0.5, 160, 20)
        cardstyle.Text(dc, cardstyle.FtInter, 11, "LEVEL", W/2-60, y+122, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 100, 8)
        rankLine := strings.ToUpper(cardstyle.Sanitize(req.RankLine))
        if rankLine == "" && req.RankLetter != "" {
                rankLine = req.RankLetter + "-RANK"
        }
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, rankLine, W/2+70, y+70, cardstyle.HexA(0xff9ecb, 235), 0.5, 0.5, 180, 8)
        cardstyle.Text(dc, cardstyle.FtInter, 10, "RANK CLASS", W/2+70, y+94, cardstyle.HexA(0x80a8d0, 220), 0.5, 0.5, 170, 8)
        y += 166

        // XP meter
        s08HUDPanel(dc, 46, y, W-92, 92, p, "EXPERIENCE")
        pct := float64(req.XPPercent) / 100
        s08LEDRatio(dc, 62, y+56, W-124, pct, p, "XP", cardstyle.Sanitize(req.XPNow)+" / "+cardstyle.Sanitize(req.XPLeft))
        y += 108

        // stats as LED rows
        rows := req.Standing
        if len(rows) > 5 {
                rows = rows[:5]
        }
        ph := 34 + float64(len(rows))*46
        s08HUDPanel(dc, 46, y, W-92, ph, p, "STATUS")
        ry := y + 52
        for _, r := range rows {
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, strings.ToUpper(cardstyle.Sanitize(r.Label)), 62, ry, cardstyle.Hex(0xe0f2ff), 0, 0.5, 200, 7)
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, strings.ToUpper(cardstyle.Sanitize(r.Value)), W-62, ry, cardstyle.HexA(0x40e0ff, 235), 1, 0.5, 180, 7)
                ry += 46
        }
        y += ph + 16

        // progress quests
        prog := req.Progress
        if len(prog) > 3 {
                prog = prog[:3]
        }
        if len(prog) > 0 {
                ph2 := 34 + float64(len(prog))*52
                s08HUDPanel(dc, 46, y, W-92, ph2, p, cardstyle.Sanitize(req.ProgressTitle))
                ry = y + 52
                for _, pr := range prog {
                        frac := 0.0
                        if pr.Max > 0 {
                                frac = float64(pr.Cur) / float64(pr.Max)
                        }
                        if pr.Done {
                                frac = 1
                        }
                        s08LEDRatio(dc, 62, ry, W-124, frac, p, pr.Label, fmtProgressText(pr.Cur, pr.Max, pr.ValueText))
                        ry += 52
                }
        }
        s08Footer(dc, W, H, req, p)
        return dc
}

// ALLOCATE - UPGRADE MENU.
func s08Allocate(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 34, W-80, 66, "UPGRADE MENU", strings.ToUpper(cardstyle.Sanitize(req.Pill)), p)

        s08HUDPanel(dc, 46, y, W-92, 130, p, "UNSPENT POINTS")
        avail := parseLeadingInt(req.PointsBig, 0)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 44, fmt.Sprintf("%d", avail), W/2, y+76, cardstyle.HexA(0xff40a0, 130), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 140, 18)
        y += 146

        rows := req.Rows
        if len(rows) > 7 {
                rows = rows[:7]
        }
        ph := 34 + float64(len(rows))*44
        s08HUDPanel(dc, 46, y, W-92, ph, p, "THE SEVEN PATHS")
        ry := y + 52
        for _, r := range rows {
                code := strings.ToUpper(cardstyle.Sanitize(r.Label))
                cardstyle.Chip(dc, 62, ry-11, 44, 20, code, cardstyle.HexA(0x40e0ff, 30), cardstyle.HexA(0x40e0ff, 160), cardstyle.Hex(0x40e0ff), cardstyle.FtPS2P, 8)
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, cardstyle.StatFullName(code), 118, ry, cardstyle.Hex(0xe0f2ff), 0, 0.5, 170, 6)
                cardstyle.DotLeader(dc, 300, W-150, ry, 1, cardstyle.HexA(0x80a8d0, 110))
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, gainFromSub(r.Sub), W-62, ry, cardstyle.HexA(0x40e0ff, 235), 1, 0.5, 90, 7)
                ry += 44
        }
        y += ph + 16

        s08HUDPanel(dc, 46, y, W-92, 120, p, "SESSION")
        spent := parseLeadingInt(req.SpentNow, 0)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, fmt.Sprintf("SPENT %d PTS", spent), 62, y+48, cardstyle.Hex(0xe0f2ff), 0, 0.5, 260, 8)
        cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.Sanitize(req.SpentLeft)), 62, y+76, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, 280, 8)
        cardstyle.Text(dc, cardstyle.FtInter, 10, "HALF VALUE PER POINT AFTER HEAVY SINGLE-STAT INVESTMENT", 62, y+100, cardstyle.HexA(0x80a8d0, 180), 0, 0.5, W-140, 7)
        y += 136

        cardstyle.PillCTA(dc, W/2, y+20, 380, 40, ".J ALLOCATE <STAT> [AMOUNT]", "PRESS START", cardstyle.HexA(0xff40a0, 60), cardstyle.HexA(0xff40a0, 220), cardstyle.Hex(0xff9ecb), cardstyle.HexA(0x80a8d0, 220), cardstyle.FtPS2P, 10)
        s08Footer(dc, W, H, req, p)
        return dc
}

// SKILLUP - MASTERY UNLOCKED.
func s08Skillup(req *portraitRequest) *gg.Context {
        const W, H = 600.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 34, W-80, 66, "MASTERY UNLOCKED", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

        // big skill readout panel
        s08HUDPanel(dc, 46, y, W-92, 190, p, "SKILL")
        initials := initialLetters(cardstyle.Sanitize(req.SkillName))
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 44, initials, W/2, y+86, cardstyle.HexA(0x40e0ff, 130), cardstyle.Hex(0x40e0ff), 0.5, 0.5, 200, 16)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, strings.ToUpper(cardstyle.Sanitize(req.SkillName)), W/2, y+142, cardstyle.Hex(0xff9ecb), 0.5, 0.5, W-130, 8)
        y += 206

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
        s08HUDPanel(dc, 46, y, W-92, 120, p, "MASTERY LEVEL")
        // level pips as LED blocks
        blockW := (W - 140 - float64(maxPips-1)*8) / float64(maxPips)
        for i := 0; i < maxPips; i++ {
                c := cardstyle.HexA(0x40e0ff, 40)
                if i < cur {
                        c = cardstyle.HexA(0xff40a0, 235)
                }
                dc.SetColor(c)
                dc.DrawRoundedRectangle(70+float64(i)*(blockW+8), y+62, blockW, 18, 3)
                dc.Fill()
        }
        cardstyle.Text(dc, cardstyle.FtPS2P, 10, fmt.Sprintf("%d / %d", cur, maxPips), W/2, y+96, cardstyle.HexA(0x40e0ff, 235), 0.5, 0.5, 140, 7)
        y += 136

        tierLabel := fmt.Sprintf("TIER %d", req.Tier)
        if req.Ascended {
                tierLabel = "ASCENDED"
        }
        s08HUDPanel(dc, 46, y, W-92, 64, p, "TIER")
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, strings.ToUpper(tierLabel), W/2, y+40, cardstyle.Hex(0xff9ecb), 0.5, 0.5, 200, 8)
        s08Footer(dc, W, H, req, p)
        return dc
}

// ABILITIES - SKILL LIST.EXE.
func s08Abilities(req *portraitRequest) *gg.Context {
        W := 1000.0
        H := float64(abilitiesH(req))
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        title := strings.TrimSpace(req.DocTitle)
        if title == "" {
                title = "SKILL LIST.EXE"
        }
        y := s08Marquee(dc, W, 30, W-120, 62, title, cardstyle.Sanitize(req.DocQuote), p)
        if req.PageLabel != "" {
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, strings.ToUpper(cardstyle.Sanitize(req.PageLabel)), W-60, 44, cardstyle.HexA(0x40e0ff, 235), 1, 0.5, 170, 7)
        }

        num := req.StartNumber
        if num < 1 {
                num = 1
        }
        for gi, g := range req.Groups {
                if gi >= 6 {
                        break
                }
                gh := 36 + float64(len(g.Items))*52 + 6
                s08HUDPanel(dc, 44, y, W-88, gh, p, g.Name)
                iy := y + 52
                for _, it := range g.Items {
                        cardstyle.Chip(dc, 60, iy-11, 40, 20, fmt.Sprintf("%02d", num), cardstyle.HexA(0x40e0ff, 30), cardstyle.HexA(0x40e0ff, 160), cardstyle.Hex(0x40e0ff), cardstyle.FtPS2P, 8)
                        cardstyle.Text(dc, cardstyle.FtPS2P, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(it.Title)), 24), 114, iy-6, cardstyle.Hex(0xe0f2ff), 0, 0.5, 380, 7)
                        if it.Sub != "" {
                                cardstyle.Text(dc, cardstyle.FtInter, 10, cardstyle.TruncateRunes(cardstyle.Sanitize(it.Sub), 62), 114, iy+16, cardstyle.HexA(0x80a8d0, 230), 0, 0.5, 400, 8)
                        }
                        if it.Runes != "" {
                                cardstyle.Spaced(dc, cardstyle.FtPS2P, 9, strings.ToUpper(cardstyle.Sanitize(it.Runes)), W-64, iy-6, cardstyle.HexA(0x40e0ff, 235), 1, 2)
                        }
                        if it.Value != "" {
                                cardstyle.Text(dc, cardstyle.FtInter, 10, cardstyle.Sanitize(it.Value), W-64, iy+16, cardstyle.HexA(0x80a8d0, 230), 1, 0.5, 200, 8)
                        }
                        num++
                        iy += 52
                }
                y += gh + 14
        }
        s08Footer(dc, W, H, req, p)
        return dc
}

// SKILLTREE - CIRCUIT GRID (right-angle traces, solder pads, bottom-up).
func s08Skilltree(req *portraitRequest) *gg.Context {
        const W, H = 1200.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 30, 520, 58, "SKILL CIRCUIT", strings.ToUpper(cardstyle.Sanitize(req.ClassName)), p)
        pts := req.SkillPoints
        ptsText := "NO POINTS"
        if pts == 1 {
                ptsText = "1 POINT"
        } else if pts > 1 {
                ptsText = fmt.Sprintf("%d POINTS", pts)
        }
        cardstyle.Text(dc, cardstyle.FtPS2P, 10, ptsText, W-60, 52, cardstyle.HexA(0xff9ecb, 235), 1, 0.5, 180, 7)

        // circuit board field
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
        // vertical power rails (tiers, bottom-up)
        baseY := 884.0
        tierH := 168.0
        for t := 1; t <= maxTier; t++ {
                yy := baseY - float64(t-1)*tierH
                dc.SetColor(cardstyle.HexA(0x40e0ff, 70))
                dc.SetLineWidth(1.4)
                dc.DrawLine(60, yy, W-60, yy)
                dc.Stroke()
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, fmt.Sprintf("T%d", t), 36, yy, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 40, 6)
        }
        // ground bus
        dc.SetColor(cardstyle.HexA(0xff40a0, 130))
        dc.SetLineWidth(2)
        dc.DrawLine(60, baseY+40, W-60, baseY+40)
        dc.Stroke()

        colW := (W - 160) / float64(n)
        for bi, br := range req.Branches {
                cx := 80 + float64(bi)*colW + colW/2
                // branch label chip
                cardstyle.Chip(dc, cx-colW/2+14, y+2, colW-28, 30, strings.ToUpper(cardstyle.Sanitize(br.Name)), cardstyle.WithA(p.Panel, 230), cardstyle.HexA(0x40e0ff, 150), cardstyle.Hex(0x40e0ff), cardstyle.FtPS2P, 8)
                // vertical trace from bus to each node
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
                prevY := baseY + 40.0
                for _, t := range order {
                        list := nodesByTier[t]
                        ty := baseY - float64(t-1)*tierH
                        dc.SetColor(cardstyle.HexA(0x40e0ff, 110))
                        dc.SetLineWidth(2)
                        dc.DrawLine(cx, prevY, cx, ty)
                        dc.Stroke()
                        prevY = ty
                        for j, s := range list {
                                nx := cx + (float64(j)-float64(len(list)-1)/2)*110
                                // right-angle stub from trunk to node
                                dc.SetColor(cardstyle.HexA(0x40e0ff, 110))
                                dc.SetLineWidth(1.6)
                                dc.DrawLine(cx, ty, nx, ty)
                                dc.Stroke()
                                // solder pad node
                                lit := s.State == "learned" || s.State == "maxed"
                                pad := cardstyle.HexA(0x40e0ff, 120)
                                if lit {
                                        pad = cardstyle.HexA(0xff40a0, 235)
                                }
                                dc.SetColor(cardstyle.WithA(p.Panel, 240))
                                dc.DrawCircle(nx, ty, 24)
                                dc.Fill()
                                dc.SetColor(pad)
                                dc.SetLineWidth(2.4)
                                dc.DrawCircle(nx, ty, 24)
                                dc.Stroke()
                                dc.SetColor(pad)
                                dc.DrawCircle(nx, ty, 6)
                                dc.Fill()
                                cardstyle.Text(dc, cardstyle.FtPS2P, 8, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(s.Name)), 10), nx, ty-42, cardstyle.Hex(0xe0f2ff), 0.5, 0.5, 120, 6)
                                valCol := cardstyle.HexA(0x80a8d0, 235)
                                if s.State == "maxed" {
                                        valCol = cardstyle.Hex(0xff9ecb)
                                }
                                cardstyle.Text(dc, cardstyle.FtPS2P, 8, fmt.Sprintf("%d/%d", s.Cur, s.Max), nx, ty+38, valCol, 0.5, 0.5, 80, 6)
                        }
                }
        }
        s08Footer(dc, W, H, req, p)
        return dc
}

// EQUIP - LOADOUT screen.
func s08Equip(req *portraitRequest) *gg.Context {
        const W, H = 1500.0, 1000.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 30, 560, 60, "LOADOUT", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)

        // slot cartridge painter: one socket cell
        cartridge := func(x, yy, cw, ch float64, s portraitSlot, tag string) {
                empty := s.Empty
                fill := cardstyle.WithA(p.Panel, 215)
                if empty {
                        fill = cardstyle.HexA(0x0a0c1c, 160)
                }
                dc.SetColor(fill)
                dc.DrawRoundedRectangle(x, yy, cw, ch, 4)
                dc.Fill()
                cardstyle.BracketCorners(dc, x, yy, cw, ch, 12, cardstyle.HexA(0x40e0ff, cardstyleIfByte(empty, 80, 180)), 1.4)
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, tag, x+14, yy+24, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, 120, 7)
                for t := 0; t < s.Tier && t < 5; t++ {
                        dc.SetColor(cardstyle.HexA(0xff40a0, 235))
                        dc.DrawRectangle(x+cw-26-float64(t)*16, yy+16, 8, 8)
                        dc.Fill()
                }
                name := "EMPTY"
                ncol := cardstyle.HexA(0x80a8d0, 160)
                if !empty {
                        name = cardstyle.Sanitize(s.Name)
                        ncol = cardstyle.Hex(0xe0f2ff)
                }
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, cardstyle.TruncateRunes(strings.ToUpper(name), 18), x+14, yy+52, ncol, 0, 0.5, cw-28, 7)
                if s.TierLabel != "" {
                        cardstyle.Text(dc, cardstyle.FtInter, 10, strings.ToUpper(cardstyle.Sanitize(s.TierLabel)), x+cw-14, yy+24, cardstyle.HexA(0xff9ecb, 220), 1, 0.5, 110, 8)
                }
                if !empty && s.DurMax > 0 {
                        frac := s.Dur / s.DurMax
                        if frac > 1 {
                                frac = 1
                        }
                        cardstyle.SegBar(dc, x+14, yy+ch-34, cw-28, 9, frac, 10, cardstyle.HexA(0x40e0ff, 220), cardstyle.HexA(0x40e0ff, 36))
                        cardstyle.Text(dc, cardstyle.FtInter, 10, cardstyle.FmtF(s.Dur)+" / "+cardstyle.FmtF(s.DurMax), x+14, yy+ch-14, cardstyle.HexA(0x80a8d0, 220), 0, 0.5, cw-28, 8)
                }
        }

        // hotbar strip painter: full-width rail of sockets
        strip := func(label string, slots []portraitSlot, yy, ch float64) {
                dc.SetColor(cardstyle.HexA(0x40e0ff, 60))
                dc.SetLineWidth(1.2)
                dc.DrawLine(50, yy-10, W-50, yy-10)
                dc.Stroke()
                cardstyle.Text(dc, cardstyle.FtPS2P, 11, label, 50, yy-22, cardstyle.HexA(0x40e0ff, 220), 0, 0.5, 400, 8)
                n := len(slots)
                if n == 0 {
                        return
                }
                cw := (W - 100 - float64(n-1)*20) / float64(n)
                for j := 0; j < n && j < len(slots); j++ {
                        x := 50 + float64(j)*(cw+20)
                        cartridge(x, yy, cw, ch, slots[j], strings.ToUpper(cardstyle.Sanitize(slots[j].Slot)))
                }
        }

        // top hotbar: first 5 slots / bottom hotbar: last 4
        top := req.Slots
        if len(top) > 5 {
                top = top[:5]
        }
        bot := []portraitSlot{}
        if len(req.Slots) > 5 {
                bot = req.Slots[5:]
        }
        if len(bot) > 4 {
                bot = bot[:4]
        }
        strip("EQUIP RAIL A", top, y+36, 150)
        strip("EQUIP RAIL B", bot, H-222, 144)

        // middle band: operative window + integrity readout
        my0 := y + 226.0
        my1 := H - 296.0
        s08HUDPanel(dc, 50, my0, 430, my1-my0, p, "OPERATIVE")
        drawHeroFitted(dc, req, 265, my0+150, 300, (my1-my0)*0.52, *p)
        cardstyle.Text(dc, cardstyle.FtPS2P, 12, strings.ToUpper(cardstyle.Sanitize(styledName(req))), 265, my0+330, cardstyle.Hex(0xe0f2ff), 0.5, 0.5, 340, 8)
        if req.SealText != "" {
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, strings.ToUpper(cardstyle.Sanitize(req.SealText))+"-RANK", 265, my0+362, cardstyle.HexA(0x40e0ff, 235), 0.5, 0.5, 200, 7)
        }

        cx0 := 520.0
        s08HUDPanel(dc, cx0, my0, 460, my1-my0, p, "INTEGRITY")
        avg := 0.0
        filled := 0
        for _, s := range req.Slots {
                if !s.Empty && s.DurMax > 0 {
                        avg += s.Dur / s.DurMax
                        filled++
                }
        }
        if filled > 0 {
                avg /= float64(filled)
        }
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 44, fmt.Sprintf("%d%%", int(avg*100)), cx0+230, my0+130, cardstyle.HexA(0x40e0ff, 140), cardstyle.Hex(0x40e0ff), 0.5, 0.5, 300, 20)
        cardstyle.Text(dc, cardstyle.FtInter, 11, "SLOTS "+fmt.Sprintf("%d/9", filled)+" FILLED", cx0+230, my0+190, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 300, 8)
        cardstyle.SegBar(dc, cx0+60, my0+220, 340, 12, avg, 14, cardstyle.HexA(0x40e0ff, 230), cardstyle.HexA(0x40e0ff, 40))
        cardstyle.Text(dc, cardstyle.FtInter, 11, "REPAIR AT .J BLACKSMITH", cx0+230, my0+260, cardstyle.HexA(0x80a8d0, 210), 0.5, 0.5, 320, 8)

        nx0 := 1020.0
        s08HUDPanel(dc, nx0, my0, W-50-nx0, my1-my0, p, "NOTES")
        cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(req.Caption)), 40), nx0+30, my0+120, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, W-50-nx0-60, 9)
        dc.Push()
        dc.Translate(nx0+20, my0+150)
        cardstyle.Scanlines(dc, W-nx0-90, my1-my0-180, cardstyle.HexA(0x40e0ff, 26), 6)
        dc.Pop()
        return dc
}

// SHOP - ARMORY SHOP.
func s08Shop(req *portraitRequest) *gg.Context {
        entries := req.Entries
        if len(entries) > 8 {
                entries = entries[:8]
        }
        nE := len(entries)
        if nE < 1 {
                nE = 1
        }
        const W = 800.0
        H := 132.0 + float64(nE)*112.0 + 100.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 32, W-100, 62, "THE ARMORY SHOP", strings.ToUpper(cardstyle.Sanitize(styledName(req))), p)
        for _, e := range entries {
                eh := 96.0
                dc.SetColor(cardstyle.WithA(p.Panel, 215))
                dc.DrawRoundedRectangle(44, y, W-88, eh, 4)
                dc.Fill()
                cardstyle.BracketCorners(dc, 44, y, W-88, eh, 12, cardstyle.HexA(0x40e0ff, 150), 1.4)
                cardstyle.Text(dc, cardstyle.FtPS2P, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(e.Title)), 24), 64, y+26, cardstyle.Hex(0xe0f2ff), 0, 0.5, 420, 7)
                if e.Sub != "" {
                        cardstyle.Text(dc, cardstyle.FtInter, 10, cardstyle.TruncateRunes(cardstyle.Sanitize(e.Sub), 66), 64, y+52, cardstyle.HexA(0x80a8d0, 225), 0, 0.5, 440, 8)
                }
                if e.Runes != "" {
                        cardstyle.Spaced(dc, cardstyle.FtPS2P, 8, strings.ToUpper(cardstyle.Sanitize(e.Runes)), 64, y+76, cardstyle.HexA(0x40e0ff, 220), 0, 1.6)
                }
                cardstyle.Text(dc, cardstyle.FtPS2P, 11, strings.ToUpper(cardstyle.Sanitize(e.Value)), W-64, y+26, cardstyle.Hex(0xff9ecb), 1, 0.5, 160, 8)
                y += eh + 14
        }
        s08Footer(dc, W, H, req, p)
        return dc
}

// GUILDINFO - GUILD SERVER.
func s08GuildInfo(req *portraitRequest) *gg.Context {
        const W, H = 800.0, 800.0
        dc := gg.NewContext(int(W), int(H))
        p := s08Base(dc, W, H)

        y := s08Marquee(dc, W, 32, W-110, 60, "GUILD SERVER", "CHAPTER RECORD", p)

        // crest panel
        s08HUDPanel(dc, W/2-150, y, 300, 220, p, "SIGIL")
        drawHeroFitted(dc, req, W/2, y+120, 200, 130, *p)
        cardstyle.Text(dc, cardstyle.FtPS2P, 14, firstRuneUpper(styledName(req)), W/2, y+180, cardstyle.HexA(0x40e0ff, 235), 0.5, 0.5, 90, 9)
        name := styledName(req)
        cardstyle.Text(dc, cardstyle.FtPS2P, 13, strings.ToUpper(cardstyle.Sanitize(name)), W/2, y+250, cardstyle.Hex(0xff9ecb), 0.5, 0.5, W-140, 9)
        if req.Motto != "" {
                cardstyle.Text(dc, cardstyle.FtInter, 11, "\""+cardstyle.TruncateRunes(cardstyle.Sanitize(req.Motto), 60)+"\"", W/2, y+276, cardstyle.HexA(0x80a8d0, 230), 0.5, 0.5, W-120, 8)
        }
        y += 306

        rows := req.Rows
        if len(rows) > 4 {
                rows = rows[:4]
        }
        ph := 30 + float64(len(rows))*36
        s08HUDPanel(dc, 44, y, W-88, ph, p, "THE CHARTER")
        ry := y + 46
        for _, r := range rows {
                cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.Sanitize(r.Label)), 64, ry, cardstyle.HexA(0x80a8d0, 235), 0, 0.5, 240, 8)
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, strings.ToUpper(cardstyle.Sanitize(r.Value)), W-64, ry, cardstyle.HexA(0x40e0ff, 235), 1, 0.5, 220, 7)
                ry += 36
        }
        y += ph + 16
        // level bar + buildings
        pct := float64(req.XPPercent) / 100
        s08HUDPanel(dc, 44, y, W-88, 96, p, fmt.Sprintf("LEVEL %d", req.Level))
        cardstyle.SegBar(dc, 64, y+52, W-128, 12, pct, 16, cardstyle.HexA(0xff40a0, 235), cardstyle.HexA(0x40e0ff, 36))
        for i, b := range req.Buildings {
                if i >= 3 {
                        break
                }
                bx := W/2 + float64(i-1)*160
                cardstyle.Text(dc, cardstyle.FtInter, 10, cardstyle.TruncateRunes(strings.ToUpper(cardstyle.Sanitize(b.Name)), 9), bx, y+84, cardstyle.HexA(0x80a8d0, 220), 0.5, 0.5, 120, 8)
                cardstyle.Text(dc, cardstyle.FtPS2P, 9, fmt.Sprintf("L%d", b.Level), bx, y+102, cardstyle.HexA(0x40e0ff, 235), 0.5, 0.5, 60, 7)
        }
        s08Footer(dc, W, H, req, p)
        return dc
}
