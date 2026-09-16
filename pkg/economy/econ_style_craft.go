package economy

// style_craft.go - per-style compositions for the item-creation family
// (CRAFT / BREW / COOK / FORGE / FISH, 1000x600 landscape) and the rank-up
// DECREE card, in the 9 rebuilt design systems. Style 0/7 keeps the baked
// decree-family art. Each style speaks its own visual language here.

import (
        "math"
        "fmt"
        "image"
        "strings"

        "image-service/pkg/cardstyle"

        "github.com/fogleman/gg"
)

// styledEconomyKinds - transaction types with rebuilt compositions.
func styledEconomyKinds(t string) bool {
        switch t {
        case "CRAFT", "BREW", "COOK", "FORGE", "FISH", "DECREE":
                return true
        }
        return false
}

// renderStyledEconomy returns the styled canvas or nil to fall back.
func renderStyledEconomy(req *TransactionCardRequest) image.Image {
        if req == nil || req.Style <= 0 || req.Style == 7 || req.Style > 10 {
                return nil
        }
        if !styledEconomyKinds(req.Type) {
                return nil
        }
        var img image.Image
        switch req.Style {
        case 1:
                img = econStyle01(req)
        case 2:
                img = econStyle02(req)
        case 3:
                img = econStyle03(req)
        case 4:
                img = econStyle04(req)
        case 5:
                img = econStyle05(req)
        case 6:
                img = econStyle06(req)
        case 8:
                img = econStyle08(req)
        case 9:
                img = econStyle09(req)
        case 10:
                img = econStyle10(req)
        }
        return img
}

func econName(req *TransactionCardRequest) string {
        n := cardstyle.Sanitize(req.Nickname)
        if n == "" {
                n = "Adventurer"
        }
        return n
}

func econItem(req *TransactionCardRequest) string {
        i := cardstyle.Sanitize(req.ItemName)
        if i == "" {
                i = "Unknown Item"
        }
        return i
}

func econTypeTitle(t string) string {
        switch t {
        case "CRAFT":
                return "CRAFTED"
        case "BREW":
                return "BREWED"
        case "COOK":
                return "COOKED"
        case "FORGE":
                return "FORGED"
        case "FISH":
                return "CATCH OF THE DAY"
        case "DECREE":
                return "RANK UP"
        }
        return strings.ToUpper(t)
}

func econCaption(req *TransactionCardRequest, t string) string {
        if c := cardstyle.Sanitize(req.Details); c != "" && t != "DECREE" {
                return c
        }
        if t == "DECREE" {
                return "keep rising - the guild watches"
        }
        return fmt.Sprintf(".j %s %s", strings.ToLower(t), econItem(req))
}

// ── Style 1 STONEKEEP - anvil plaque ─────────────────────────────────

func econStyle01(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Stonekeep()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        // masonry hint
        dc.SetColor(cardstyle.N(255, 255, 255, 5))
        dc.SetLineWidth(1)
        for y := 0.0; y < H; y += 78 {
                dc.DrawLine(0, y, W, y)
                dc.Stroke()
        }
        for i, y := 0, 0.0; y < H; y, i = y+78, i+1 {
                off := 90.0
                if i%2 == 0 {
                        off = 0.0
                }
                for x := off; x < W; x += 180 {
                        dc.DrawLine(x, y, x, y+78)
                        dc.Stroke()
                }
        }
        cardstyle.Vignette(dc, W, H, 70)
        // great stone plaque
        px, py, pw, ph := 120.0, 96.0, W-240.0, H-192.0
        cardstyle.BevelRect(dc, px, py, pw, ph, cardstyle.Lighten(p.Panel, 8), cardstyle.Lighten(p.Panel, 42), cardstyle.Darken(p.Panel, 40), 3)
        for _, c := range [][2]float64{{px + 22, py + 22}, {px + pw - 22, py + 22}, {px + 22, py + ph - 22}, {px + pw - 22, py + ph - 22}} {
                cardstyle.Rivet(dc, c[0], c[1], 6, p.Panel)
        }
        // engraved content
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(econTypeTitle(req.Type)), px+pw/2, py+58, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 500, 14)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), px+pw/2, py+90, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 500, 9)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 42, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), px+pw/2, py+ph/2+20, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, pw-90, 20)
        if req.Amount > 1 {
                cardstyle.Engrave(dc, cardstyle.FtCinzel, 18, fmt.Sprintf("X%d", int(req.Amount)), px+pw/2, py+ph/2+64, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 12)
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 64)), px+pw/2, py+ph-46, cardstyle.WithA(p.Muted, 210), 0.5, 0.5, pw-110, 9)
        cardstyle.SealMini(dc, px+40, py+ph-44, 26, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 13)
        return dc.Image()
}
func econStyle02(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.GoldenArcanum()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.StarField(dc, 0, 0, W, H, 24, 0xAC2337, cardstyle.N(232, 200, 120, 255))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 14, 1.2, cardstyle.WithA(p.Accent, 170), true)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 21, 0.6, cardstyle.WithA(p.Accent, 110), false)
        cardstyle.Vignette(dc, W, H, p.Vign)

        // transmutation circle on the left
        cx, cy, R := 268.0, H/2+14, 176.0
        cardstyle.MagicCircle(dc, cx, cy, R, p.Accent)
        cardstyle.Ring(dc, cx, cy, R*0.62, cardstyle.WithA(p.Accent, 70), 0.8)
        cardstyle.Ring(dc, cx, cy, R*0.34, cardstyle.WithA(p.Accent, 90), 0.7)
        for i := 0; i < 8; i++ {
                a := float64(i) * math.Pi / 4
                cardstyle.Diamond(dc, cx+R*0.80*math.Cos(a), cy+R*0.80*math.Sin(a), 3, cardstyle.WithA(p.Accent, 120))
        }
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 52, fmt.Sprintf("%d", int(req.Amount)), cx, cy-10, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 120, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, strings.ToUpper(econTypeTitle(req.Type)), cx, cy+34, p.Muted, 0.5, 3)

        // right register
        cx0 := cx + R + 52
        cw := W - cx0 - 64
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), cx0, 118, p.Muted, 0, 2.4)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 34, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), cx0, 190, p.Accent, 0, 0.5, cw, 18)
        dc.SetColor(cardstyle.WithA(p.Accent, 150))
        dc.SetLineWidth(1)
        dc.DrawLine(cx0, 224, cx0+cw*0.7, 224)
        dc.Stroke()
        cardstyle.Diamond(dc, cx0+cw*0.7+12, 224, 3.4, p.Accent)
        if req.Details != "" {
                cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(cardstyle.Sanitize(req.Details), 60), cx0, 262, cardstyle.WithA(p.Ink, 220), 0, 0.5, cw, 11)
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 60)), cx0, 306, cardstyle.WithA(p.Muted, 190), 0, 0.5, cw, 9)
        // command chip
        cardstyle.Chip(dc, cx0, 370, 330, 40, ".J CRAFT <ITEM>", cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 14)
        // seal
        cardstyle.SealMini(dc, W-84, H-84, 34, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 16)
        return dc.Image()
}
func econStyle03(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RetroCourt()
        cardstyle.PageBase(dc, W, H, p.Bg, p.Bg2, 26)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2, p.Accent, false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 15, 0.7, cardstyle.WithA(p.Accent, 150), false)
        cardstyle.Text(dc, cardstyle.FtIMFell, 44, strings.ToUpper(econTypeTitle(req.Type)), W/2, 96, p.Ink, 0.5, 0.5, 860, 24)
        dc.SetColor(p.Ink)
        dc.SetLineWidth(1.6)
        dc.DrawLine(70, 140, W-70, 140)
        dc.Stroke()
        dc.SetLineWidth(0.6)
        dc.DrawLine(70, 146, W-70, 146)
        dc.Stroke()
        cardstyle.Fleuron(dc, W/2, 143, 4.4, p.Accent)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 18, "performed by "+econName(req), W/2, 178, p.Accent, 0.5, 0.5, 700, 11)
        cardstyle.Text(dc, cardstyle.FtIMFell, 40, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 26)), W/2, 268, p.Ink, 0.5, 0.5, 840, 18)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtIMFell, 22, fmt.Sprintf("A QUANTITY OF %d", int(req.Amount)), W/2, 316, p.Accent, 0.5, 0.5, 500, 12)
        }
        cardstyle.DotLeader(dc, 320, 680, 356, 1.2, cardstyle.WithA(p.Ink, 140))
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2, 392, p.Muted, 0.5, 0.5, 800, 10)
        cardstyle.SealMini(dc, W/2, 470, 26, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtIMFell, 14)
        return dc.Image()
}

// ── Style 4 WOODMERE - workbench plaque ──────────────────────────────

func econStyle04(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Woodmere()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Planks(dc, W, H, 0x5700D4, cardstyle.N(0, 0, 0, 60), cardstyle.N(255, 230, 180, 22))
        // hanging sign
        cardstyle.BevelRect(dc, W/2-260, 30, 520, 76, cardstyle.N(112, 76, 42, 255), cardstyle.N(168, 122, 72, 255), cardstyle.N(44, 28, 14, 255), 2.6)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, econTypeTitle(req.Type), W/2, 66, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, 460, 14)
        // vellum result panel
        dc.SetColor(cardstyle.N(36, 22, 10, 255))
        dc.DrawRoundedRectangle(120, 150, 760, 330, 10)
        dc.Fill()
        dc.SetColor(p.Panel)
        dc.DrawRoundedRectangle(132, 162, 736, 306, 8)
        dc.Fill()
        cardstyle.Stitches(dc, 142, 176, 452, 22, cardstyle.N(120, 84, 46, 255))
        cardstyle.Text(dc, cardstyle.FtCinzel, 14, "THE WORK OF", 190, 200, cardstyle.N(116, 82, 50, 255), 0, 0.5, 220, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 20, strings.ToUpper(cardstyle.Sanitize(econName(req))), 190, 232, cardstyle.N(52, 34, 20, 255), 0, 0.5, 300, 11)
        cardstyle.DotLeader(dc, 200, 800, 276, 1.2, cardstyle.N(150, 104, 56, 160))
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 40, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 24), W/2, 330, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 660, 18)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtCinzel, 20, fmt.Sprintf("X%d", int(req.Amount)), W/2, 374, cardstyle.N(150, 78, 24, 255), 0.5, 0.5, 120, 12)
        }
        cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2, 520, cardstyle.N(226, 192, 142, 230), 0.5, 0.5, 800, 10)
        cardstyle.SealMini(dc, 70, 552, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
        return dc.Image()
}

// ── Style 5 EMBLEM NOIR - operation report ───────────────────────────

func econStyle05(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.EmblemNoir()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        dc.SetColor(cardstyle.N(255, 255, 255, 5))
        dc.SetLineWidth(1)
        for y := 0.0; y < H; y += 5 {
                dc.DrawLine(0, y, W, y)
                dc.Stroke()
        }
        // registration marks
        for _, c := range [][2]float64{{26, 26}, {W - 26, 26}, {26, H - 26}, {W - 26, H - 26}} {
                dc.SetColor(cardstyle.WithA(p.Muted, 110))
                dc.SetLineWidth(1)
                dc.DrawLine(c[0]-7, c[1], c[0]+7, c[1])
                dc.Stroke()
                dc.DrawLine(c[0], c[1]-7, c[0], c[1]+7)
                dc.Stroke()
        }
        // ziggurat crown
        steps := []struct{ w, h float64 }{{W * 0.16, 7}, {W * 0.27, 7}, {W * 0.38, 7}}
        yy := 24.0
        dc.SetColor(cardstyle.WithA(p.Accent, 200))
        for i, s := range steps {
                dc.DrawRectangle(W/2-s.w/2, yy, s.w, s.h)
                dc.Fill()
                if i == 0 {
                        cardstyle.Diamond(dc, W/2, yy-7, 4.4, p.Accent)
                }
                yy += s.h + 4
        }
        // left chartered band
        bx, bw, by, bh := 56.0, 210.0, 108.0, H-108-56
        dc.SetColor(cardstyle.N(16, 16, 18, 255))
        dc.DrawRectangle(bx, by, bw, bh)
        dc.Fill()
        dc.SetColor(cardstyle.WithA(p.Accent, 210))
        dc.SetLineWidth(1.4)
        dc.DrawRectangle(bx, by, bw, bh)
        dc.Stroke()
        dc.SetLineWidth(0.7)
        dc.DrawRectangle(bx+5, by+5, bw-10, bh-10)
        dc.Stroke()
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 54, cardstyle.Sanitize(req.SealText), bx+bw/2, by+96, cardstyle.WithA(p.Accent, 60), cardstyle.WithA(p.Ink, 240), 0.5, 0.5, 120, 24)
        cardstyle.Spaced(dc, cardstyle.FtInter, 9, "CERTIFIED", bx+bw/2, by+140, p.Muted, 0.5, 5)
        // sunburst fan in the band
        sunY := by + bh - 66
        dc.SetLineWidth(0.8)
        for i := 0; i <= 12; i++ {
                a := math.Pi + math.Pi*float64(i)/12
                dc.SetColor(cardstyle.WithA(p.Accent, uint8(90+60*math.Cos(float64(i)))))
                dc.DrawLine(bx+bw/2, sunY, bx+bw/2+84*math.Cos(a), sunY+84*math.Sin(a)*0.9)
                dc.Stroke()
        }
        cardstyle.Ring(dc, bx+bw/2, sunY, 6, cardstyle.WithA(p.Accent, 200), 1.1)
        // right register
        cx0 := bx + bw + 40
        cw := W - cx0 - 56
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(econTypeTitle(req.Type)), cx0+cw/2, 132, p.Ink, 0.5, 4)
        dc.SetColor(cardstyle.WithA(p.Accent, 190))
        dc.SetLineWidth(1.2)
        dc.DrawLine(cx0, 146, cx0+cw/2-16, 146)
        dc.Stroke()
        dc.DrawLine(cx0+cw/2+16, 146, cx0+cw, 146)
        dc.Stroke()
        cardstyle.Diamond(dc, cx0+cw/2, 146, 4, p.Accent)
        cardstyle.Text(dc, cardstyle.FtInterSemi, 30, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 26)), cx0, 210, p.Ink, 0, 0.5, cw, 16)
        cardstyle.Text(dc, cardstyle.FtInter, 13, "OPERATOR: "+strings.ToUpper(cardstyle.Sanitize(econName(req))), cx0, 252, p.Muted, 0, 0.5, cw, 9)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtInter, 13, fmt.Sprintf("QUANTITY: %d", int(req.Amount)), cx0, 276, p.Muted, 0, 0.5, cw, 9)
        }
        cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 74)), cx0, 312, cardstyle.WithA(p.Muted, 180), 0, 0.5, cw, 8)
        // deco chevrons + ledger ticks
        dc.SetColor(cardstyle.WithA(p.Accent, 160))
        chW := cw * 0.5
        chX := cx0 + cw/2
        for i := 0; i < 4; i++ {
                off := chW/8 + float64(i)*chW/4
                dc.SetLineWidth(1)
                dc.DrawLine(chX-off-12, 368, chX-off, 364)
                dc.Stroke()
                dc.DrawLine(chX-off, 364, chX-off-12, 360)
                dc.Stroke()
                dc.DrawLine(chX+off+12, 368, chX+off, 364)
                dc.Stroke()
                dc.DrawLine(chX+off, 364, chX+off+12, 360)
                dc.Stroke()
        }
        cardstyle.Diamond(dc, chX, 364, 4, p.Accent)
        // wax seal anchored bottom right
        cardstyle.WaxSeal(dc, W-92, H-64, 40, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 18)
        return dc.Image()
}
func econStyle06(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.SoulForge()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.StarField(dc, 16, 16, W-16, H-16, 34, 0x50F6CF, cardstyle.N(255, 255, 255, 255))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 11, 1.3, p.PanelEd, false)
        cardstyle.Diamond(dc, W/2, 11, 5.4, p.PanelEd)
        cardstyle.Diamond(dc, W/2, H-11, 5.4, p.PanelEd)
        // title + subtitle + rule
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, strings.ToUpper(econTypeTitle(req.Type)), W/2, 74, p.Accent, 0.5, 0.5, 560, 16)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 102, p.Muted, 0.5, 2.6)
        cardstyle.DotLeader(dc, W/2-110, W/2+110, 120, 1, cardstyle.WithA(p.Accent, 140))
        // magic circle + item initial left
        cardstyle.MagicCircle(dc, 190, 330, 108, p.Accent)
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 58, firstRuneOf(econItem(req)), 190, 324, p.Accent2, p.Accent2, 0.5, 0.5, 110, 22)
        // item + caption panel
        cardstyle.PanelRounded(dc, 360, 218, 590, 224, 14, p.Panel, cardstyle.WithA(p.PanelEd, 150), 1.2)
        cardstyle.Text(dc, cardstyle.FtMedieval, 27, cardstyle.TruncateRunes(econItem(req), 26), 388, 268, p.Ink, 0, 0.5, 540, 14)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("QUANTITY %d", int(req.Amount)), 388, 304, p.Accent2, 0, 0.5, 300, 11)
        }
        cardstyle.DotLeader(dc, 388, 922, 348, 1.2, cardstyle.WithA(p.Muted, 110))
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(econCaption(req, req.Type), 66), 388, 380, p.Muted, 0, 0.5, 540, 10)
        // gold pill CTA
        cardstyle.PillCTA(dc, W/2, 512, 380, 40, cardstyle.TruncateRunes(strings.ToLower(econCaption(req, req.Type)), 40), "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 13)
        cardstyle.FooterNote(dc, W/2, H-26, cardstyle.Sanitize(req.SealText), p.Caption, cardstyle.FtMedieval, 12, 300)
        return dc.Image()
}

// ── Style 8 NEON ARCADE - crafted toast ──────────────────────────────

func econStyle08(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.NeonArcade()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Scanlines(dc, W, H, cardstyle.N(255, 64, 160, 10), 4)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2, cardstyle.HexA(0xff40a0, 200), false)
        // marquee
        dc.SetColor(cardstyle.WithA(p.Panel, 235))
        dc.DrawRoundedRectangle(W/2-330, 34, 660, 70, 6)
        dc.Fill()
        cardstyle.BracketCorners(dc, W/2-330, 34, 660, 70, 16, cardstyle.HexA(0x40e0ff, 220), 2)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 18, strings.ToUpper(econTypeTitle(req.Type)), W/2, 70, cardstyle.HexA(0xff40a0, 120), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 600, 9)
        // item panel
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(90, 160, W-180, 230, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, 90, 160, W-180, 230, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 24, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), W/2, 240, cardstyle.HexA(0x40e0ff, 130), cardstyle.Hex(0x40e0ff), 0.5, 0.5, 740, 10)
        cardstyle.Text(dc, cardstyle.FtInter, 12, "OPERATOR: "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 296, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 640, 9)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtPS2P, 11, fmt.Sprintf("X%d", int(req.Amount)), W/2, 330, cardstyle.Hex(0xff9ecb), 0.5, 0.5, 140, 8)
        }
        cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(econCaption(req, req.Type), 74), W/2, 440, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 780, 9)
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, "INSERT COIN TO CONTINUE", W/2, 560, cardstyle.HexA(0xff9ecb, 180), 0.5, 0.5, 320, 7)
        return dc.Image()
}

// ── Style 9 RUNE MONOLITH - carved tablet ────────────────────────────

func econStyle09(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RuneMonolith()
        cardstyle.VertGrad(dc, W, H, cardstyle.Darken(p.Bg, 26), cardstyle.Darken(p.Bg2, 34))
        cardstyle.Speckle(dc, 0, 0, W, H, 110, 0x9E11A8, cardstyle.N(0, 0, 0, 255))
        // floor shadow
        dc.SetColor(cardstyle.N(0, 0, 0, 130))
        dc.DrawEllipse(W/2, H-28, 400, 14)
        dc.Fill()
        // wide landscape stele
        sx, sy, sw, sh := 120.0, 54.0, W-240.0, H-100.0
        apex := 64.0
        stele := func(x, y, w, h, ap float64) {
                cx := x + w/2
                dc.ClearPath()
                dc.MoveTo(x, y+h)
                dc.LineTo(x, y+ap)
                dc.QuadraticTo(x, y+ap*0.30, cx, y)
                dc.QuadraticTo(x+w, y+ap*0.30, x+w, y+ap)
                dc.LineTo(x+w, y+h)
                dc.ClosePath()
        }
        stele(sx, sy, sw, sh, apex)
        dc.SetColor(cardstyle.Darken(p.Panel, 22))
        dc.Fill()
        dc.SetColor(cardstyle.N(0, 0, 0, 90))
        dc.SetLineWidth(3)
        dc.Stroke()
        dc.Push()
        stele(sx, sy, sw, sh, apex)
        dc.Clip()
        for i := 0; i < 20; i++ {
                f0 := float64(i) / 20
                dc.SetColor(cardstyle.Lighten(p.Panel, uint8(18*(1-f0))))
                dc.DrawRectangle(sx, sy+sh*f0, sw, sh/20+1)
                dc.Fill()
        }
        cardstyle.Speckle(dc, sx, sy, sx+sw, sy+sh, 44, 0x51B2E7, cardstyle.N(0, 0, 0, 220))
        cardstyle.GlyphCol(dc, sx+24, sy+apex+16, sy+sh-18, 44, cardstyle.N(255, 255, 255, 15))
        cardstyle.GlyphCol(dc, sx+sw-36, sy+apex+16, sy+sh-18, 44, cardstyle.N(255, 255, 255, 15))
        dc.Pop()
        stele(sx+14, sy+10, sw-28, sh-20, apex*0.8)
        dc.SetColor(cardstyle.N(0, 0, 0, 120))
        dc.SetLineWidth(2)
        dc.Stroke()

        // registers
        ix, iy := sx+56, sy+apex+30
        iw := sw - 112
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(econTypeTitle(req.Type)), ix+iw/2, iy, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, iw-120, 14)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), ix+iw/2, iy+28, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, iw-120, 9)
        // glyph band
        gy := iy + 52
        dc.SetColor(cardstyle.N(0, 0, 0, 130))
        dc.SetLineWidth(2.4)
        dc.DrawLine(ix+20, gy, ix+iw-20, gy)
        dc.Stroke()
        dc.SetColor(cardstyle.Lighten(p.Panel, 30))
        dc.SetLineWidth(1)
        dc.DrawLine(ix+20, gy+3, ix+iw-20, gy+3)
        dc.Stroke()
        dc.SetColor(p.Accent)
        for i := 0; i < 5; i++ {
                tx := ix+iw/2 - 96 + float64(i)*48
                dc.SetLineWidth(2)
                dc.DrawLine(tx, gy-4, tx+7, gy+4)
                dc.Stroke()
        }
        // item register with ember glow
        icy := iy + 130.0
        cardstyle.EmberGlow(dc, ix+iw/2, icy, 82, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 40, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 24), ix+iw/2, icy, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, iw-90, 18)
        if req.Amount > 1 {
                cardstyle.Engrave(dc, cardstyle.FtCinzel, 18, fmt.Sprintf("X%d", int(req.Amount)), ix+iw/2, icy+44, p.Accent2, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 120, 12)
        }
        // caption register
        gy2 := iy + 206.0
        dc.SetColor(cardstyle.N(0, 0, 0, 130))
        dc.SetLineWidth(2)
        dc.DrawLine(ix+20, gy2, ix+iw-20, gy2)
        dc.Stroke()
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 70)), ix+iw/2, gy2+26, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, iw-80, 9)
        return dc.Image()
}
func econStyle10(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.CrimsonCourt()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Damask(dc, W, H, 76, cardstyle.HexA(0xd4a856, 20))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2.4, p.Accent, false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 16, 1, cardstyle.WithA(p.Accent, 130), false)
        cardstyle.Cartouche(dc, W/2, 72, 520, 56, p.Panel, p.Accent, p.Ink, strings.ToUpper(econTypeTitle(req.Type)), cardstyle.FtCinzelDec, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "COMMISSIONED TO "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 122, p.Muted, 0.5, 2.4)
        // shield with item
        cardstyle.Shield(dc, W/2, 300, 300, 240, p.Panel, p.Accent, 3)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, cardstyle.TruncateRunes(strings.ToUpper(econItem(req)), 22), W/2, 276, p.Ink, 0.5, 0.5, 260, 14)
        if req.Amount > 1 {
                cardstyle.Text(dc, cardstyle.FtCinzel, 16, fmt.Sprintf("X%d", int(req.Amount)), W/2, 318, p.Accent2, 0.5, 0.5, 120, 11)
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(econCaption(req, req.Type)), 70), W/2, 470, cardstyle.WithA(p.Ink, 220), 0.5, 0.5, 820, 10)
        cardstyle.WaxSeal(dc, 80, 520, 30, p.Seal, cardstyle.Darken(p.Seal, 44), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 16)
        return dc.Image()
}

// DECREE (rank-up) per style - one composition per system.
func econDecree(req *TransactionCardRequest) image.Image {
        switch req.Style {
        case 6:
                return econDecreeSoulForge(req)
        }
        return nil // other systems: decree keeps the royal bake (it IS a decree)
}

func econDecreeSoulForge(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.SoulForge()
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.StarField(dc, 16, 16, W-16, H-16, 34, 0x50F6D0, cardstyle.N(255, 255, 255, 255))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 11, 1.3, p.PanelEd, false)
        cardstyle.Diamond(dc, W/2, 11, 5.4, p.PanelEd)
        cardstyle.Diamond(dc, W/2, H-11, 5.4, p.PanelEd)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 32, "SOUL ASCENSION", W/2, 76, p.Accent, 0.5, 0.5, 620, 16)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "A RANK DECREE FOR "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 106, p.Muted, 0.5, 2.6)
        cardstyle.DotLeader(dc, W/2-110, W/2+110, 124, 1, cardstyle.WithA(p.Accent, 140))
        // old -> new on the magic circle
        cardstyle.MagicCircle(dc, W/2, 300, 104, p.Accent)
        ledger := cardstyle.Sanitize(req.Details)
        parts := strings.Split(ledger, "->")
        if len(parts) == 2 {
                cardstyle.Text(dc, cardstyle.FtCinzel, 24, strings.ToUpper(cardstyle.Sanitize(strings.TrimSpace(parts[0]))), W/2-150, 296, p.Muted, 0.5, 0.5, 160, 12)
                cardstyle.Text(dc, cardstyle.FtCinzel, 15, "TO", W/2, 262, p.Muted, 0.5, 0.5, 60, 9)
                cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 40, strings.ToUpper(cardstyle.Sanitize(strings.TrimSpace(parts[1]))), W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 200, 16)
        } else {
                cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 36, strings.ToUpper(cardstyle.TruncateRunes(econItem(req), 18)), W/2, 296, p.Accent2, p.Accent2, 0.5, 0.5, 420, 14)
        }
        cardstyle.Text(dc, cardstyle.FtMedieval, 15, "the guild watches - keep rising", W/2, 452, p.Muted, 0.5, 0.5, 600, 10)
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, "keep rising - the guild watches", W/2, H-30, p.Caption, 0.5, 0.5, 600, 9)
        return dc.Image()
}

// firstRuneOf - first uppercased rune of s.
func firstRuneOf(s string) string {
        r := []rune(strings.ToUpper(cardstyle.Sanitize(s)))
        if len(r) == 0 {
                return "?"
        }
        return string(r[:1])
}

// QAStyledEconomy exposes the styled economy renderer for the QA harness.
func QAStyledEconomy(req *TransactionCardRequest) image.Image {
        return renderStyledEconomy(req)
}
