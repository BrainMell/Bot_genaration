package economy

// econ_style_money.go - themed BALANCE / TRANSFER / DEPOSIT / WITHDRAW for
// the 9 rebuilt card styles (1-6, 8-10; styles 0/7 keep the Kenney base).
//
// Design contract (owner rule): each theme gets ONE original base for the
// money family and reuses it across the kinds with kind-specific
// modifications - the same reuse pattern Royal Decree uses when it shares
// drawRPGCraft across CRAFT/BREW/COOK/FORGE/FISH and drawRPGDecree for
// rank-ups. BALANCE = the theme's static treasury register; TRANSFER /
// DEPOSIT / WITHDRAW = the same theme's directed-flow composition
// (FROM -> TO with the moved amount on the flow).

import (
        "fmt"
        "image"
        "image/color"
        "math"
        "strings"

        "image-service/pkg/cardstyle"

        "github.com/fogleman/gg"
)

// ── shared money helpers ─────────────────────────────────────────────

// moneyMeta returns (title, fromLabel, toLabel, fromValue, toValue).
func moneyMeta(req *TransactionCardRequest) (string, string, string, float64, float64) {
        switch strings.ToUpper(strings.TrimSpace(req.Type)) {
        case "DEPOSIT":
                return "DEPOSIT", "FROM WALLET", "TO BANK", req.NewWallet - req.Amount, req.NewBank
        case "WITHDRAW":
                return "WITHDRAW", "FROM BANK", "TO WALLET", req.NewBank + req.Amount, req.NewWallet
        default: // TRANSFER
                return "TRANSFER", "FROM WALLET", "TO ADVENTURER", req.NewWallet + req.Amount, req.NewWallet - req.Amount
        }
}

func isMoneyKind(t string) bool {
        switch strings.ToUpper(strings.TrimSpace(t)) {
        case "TRANSFER", "DEPOSIT", "WITHDRAW":
                return true
        }
        return false
}

func zeni(symbol string, v float64) string {
        return fmt.Sprintf("%s %s", symbol, formatFull(v))
}

// renderStyledMoney dispatches a money movement card to its style painter.
func renderStyledMoney(req *TransactionCardRequest) image.Image {
        if req == nil || req.Style <= 0 || req.Style == 7 || req.Style > 10 {
                return nil
        }
        if !isMoneyKind(req.Type) {
                return nil
        }
        switch req.Style {
        case 1:
                return moneyStyle01(req)
        case 2:
                return moneyStyle02(req)
        case 3:
                return moneyStyle03(req)
        case 4:
                return moneyStyle04(req)
        case 5:
                return moneyStyle05(req)
        case 6:
                return moneyStyle06(req)
        case 8:
                return moneyStyle08(req)
        case 9:
                return moneyStyle09(req)
        case 10:
                return moneyStyle10(req)
        }
        return nil
}

// renderStyledBalance dispatches a balance card to its style painter.
func renderStyledBalance(req *EconomyCardRequest) image.Image {
        if req == nil || req.Style <= 0 || req.Style == 7 || req.Style > 10 {
                return nil
        }
        switch req.Style {
        case 1:
                return balanceStyle01(req)
        case 2:
                return balanceStyle02(req)
        case 3:
                return balanceStyle03(req)
        case 4:
                return balanceStyle04(req)
        case 5:
                return balanceStyle05(req)
        case 6:
                return balanceStyle06(req)
        case 8:
                return balanceStyle08(req)
        case 9:
                return balanceStyle09(req)
        case 10:
                return balanceStyle10(req)
        }
        return nil
}

// balanceFrozen hides zero frozen rows (frozen is rare).
func balanceFrozen(f float64) float64 {
        if f < 0.5 {
                return 0
        }
        return f
}

// walletBankPct mirrors the Kenney card distribution maths.
func walletBankPct(wallet, bank, total float64) (float64, float64) {
        if total <= 0 {
                return 0, 0
        }
        wp := math.Min(1, math.Max(0, wallet/total))
        return wp, math.Min(1-wp, math.Max(0, bank/total))
}

// flowArrow draws a chunky directional arrow from (x0,y) to (x1,y).
func flowArrow(dc *gg.Context, x0, x1, y float64, col color.NRGBA, lw float64, right bool) {
        if right {
                dc.SetColor(col)
                dc.SetLineWidth(lw)
                dc.DrawLine(x0, y, x1-16, y)
                dc.Stroke()
                dc.DrawLine(x1-16, y, x1-30, y-11)
                dc.Stroke()
                dc.DrawLine(x1-16, y, x1-30, y+11)
                dc.Stroke()
                return
        }
        dc.SetColor(col)
        dc.SetLineWidth(lw)
        dc.DrawLine(x1, y, x0+16, y)
        dc.Stroke()
        dc.DrawLine(x0+16, y, x0+30, y-11)
        dc.Stroke()
        dc.DrawLine(x0+16, y, x0+30, y+11)
        dc.Stroke()
}

// ── Style 1 STONEKEEP - the granite vault ────────────────────────────

func s01VaultBase(dc *gg.Context, W, H float64, p cardstyle.Palette) (float64, float64, float64, float64) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
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
        px, py, pw, ph := 96.0, 84.0, W-192.0, H-168.0
        cardstyle.BevelRect(dc, px, py, pw, ph, cardstyle.Lighten(p.Panel, 8), cardstyle.Lighten(p.Panel, 42), cardstyle.Darken(p.Panel, 40), 3)
        for _, c := range [][2]float64{{px + 20, py + 20}, {px + pw - 20, py + 20}, {px + 20, py + ph - 20}, {px + pw - 20, py + ph - 20}} {
                cardstyle.Rivet(dc, c[0], c[1], 6, p.Panel)
        }
        return px, py, pw, ph
}

func moneyStyle01(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Stonekeep()
        title, fromL, toL, _, _ := moneyMeta(req)
        px, py, pw, ph := s01VaultBase(dc, W, H, p)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 27, title, px+pw/2, py+54, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 500, 14)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), px+pw/2, py+86, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 500, 9)
        // the moved amount, chiseled deep and lit by torchlight
        cardstyle.TorchGlow(dc, px+pw/2, py+170, 120, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 46, zeni(req.ZeniSymbol, req.Amount), px+pw/2, py+170, p.Accent2, cardstyle.Darken(p.Panel, 70), 0.5, 0.5, pw-140, 20)
        // from / to slabs
        sy := py + 232.0
        sh := 96.0
        cardstyle.BevelRect(dc, px+44, sy, pw/2-72, sh, cardstyle.Darken(p.Panel, 6), cardstyle.Lighten(p.Panel, 26), cardstyle.Darken(p.Panel, 46), 2)
        cardstyle.BevelRect(dc, px+pw/2+28, sy, pw/2-72, sh, cardstyle.Darken(p.Panel, 6), cardstyle.Lighten(p.Panel, 26), cardstyle.Darken(p.Panel, 46), 2)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, fromL, px+44+(pw/2-72)/2, sy+30, p.Muted, 0.5, 0.5, pw/2-100, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, toL, px+pw/2+28+(pw/2-72)/2, sy+30, p.Muted, 0.5, 0.5, pw/2-100, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 21, zeni(req.ZeniSymbol, req.NewWallet), px+44+(pw/2-72)/2, sy+62, p.Ink, 0.5, 0.5, pw/2-100, 12)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 21, zeni(req.ZeniSymbol, req.NewBank), px+pw/2+28+(pw/2-72)/2, sy+62, p.Ink, 0.5, 0.5, pw/2-100, 12)
        // carved flow arrow between the slabs
        right := req.Type != "WITHDRAW"
        flowArrow(dc, px+pw/2-84, px+pw/2+84, sy+sh/2, cardstyle.WithA(p.Accent, 230), 4, right)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 60)), px+pw/2, py+ph-40, cardstyle.WithA(p.Muted, 210), 0.5, 0.5, pw-110, 9)
        cardstyle.SealMini(dc, px+44, py+ph-44, 24, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 12)
        return dc.Image()
}

func balanceStyle01(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Stonekeep()
        px, py, pw, ph := s01VaultBase(dc, W, H, p)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 27, "THE VAULT OF "+strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 18)), px+pw/2, py+54, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 60), 0.5, 0.5, pw-120, 13)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(req.Rank)+" RANK - LEVEL "+fmt.Sprintf("%d", req.Level), px+pw/2, py+86, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 500, 9)
        cardstyle.TorchGlow(dc, px+pw/2, py+180, 130, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, "TOTAL WEALTH", px+pw/2, py+136, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 300, 10)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 54, zeni(req.ZeniSymbol, req.Total), px+pw/2, py+192, p.Accent2, cardstyle.Darken(p.Panel, 70), 0.5, 0.5, pw-160, 22)
        sy := py + 244.0
        sh := 92.0
        cardstyle.BevelRect(dc, px+44, sy, pw/2-72, sh, cardstyle.Darken(p.Panel, 6), cardstyle.Lighten(p.Panel, 26), cardstyle.Darken(p.Panel, 46), 2)
        cardstyle.BevelRect(dc, px+pw/2+28, sy, pw/2-72, sh, cardstyle.Darken(p.Panel, 6), cardstyle.Lighten(p.Panel, 26), cardstyle.Darken(p.Panel, 46), 2)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, "WALLET", px+44+(pw/2-72)/2, sy+28, p.Muted, 0.5, 0.5, pw/2-100, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, "BANK", px+pw/2+28+(pw/2-72)/2, sy+28, p.Muted, 0.5, 0.5, pw/2-100, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, zeni(req.ZeniSymbol, req.Wallet), px+44+(pw/2-72)/2, sy+60, p.Ink, 0.5, 0.5, pw/2-100, 12)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, zeni(req.ZeniSymbol, req.Bank), px+pw/2+28+(pw/2-72)/2, sy+60, p.Ink, 0.5, 0.5, pw/2-100, 12)
        wp, bp := walletBankPct(req.Wallet, req.Bank, req.Total)
        mx, my, mw := px+104, sy+118, pw-168
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("WALLET %d%%", int(wp*100)), mx+mw*0.25, my-12, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 200, 9)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, fmt.Sprintf("BANK %d%%", int(bp*100)), mx+mw*0.75, my-12, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 200, 9)
        cardstyle.MeterBar(dc, mx, my, mw, 16, wp, cardstyle.Darken(p.Panel, 30), p.Accent, cardstyle.Lighten(p.Accent, 40), cardstyle.Darken(p.Panel, 50), 8)
        cardstyle.Text(dc, cardstyle.FtCinzel, 11, "SEAL OF THE EXCHEQUER", px+pw/2, py+ph-36, cardstyle.WithA(p.Muted, 190), 0.5, 0.5, 400, 8)
        cardstyle.SealMini(dc, px+44, py+ph-40, 24, cardstyle.Darken(p.Seal, 20), p.SealTx, cardstyle.Sanitize(req.Rank), cardstyle.FtCinzel, 12)
        return dc.Image()
}

// ── Style 2 GOLDEN ARCANUM - the rite of coin ────────────────────────

func s02ArcaneBase(dc *gg.Context, W, H float64, p cardstyle.Palette) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.StarField(dc, 0, 0, W, H, 24, 0xAC2337, cardstyle.N(232, 200, 120, 255))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 14, 1.2, cardstyle.WithA(p.Accent, 170), true)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 21, 0.6, cardstyle.WithA(p.Accent, 110), false)
        cardstyle.Vignette(dc, W, H, p.Vign)
}

func arcaneCircle(dc *gg.Context, cx, cy, R float64, p cardstyle.Palette) {
        cardstyle.MagicCircle(dc, cx, cy, R, p.Accent)
        cardstyle.Ring(dc, cx, cy, R*0.62, cardstyle.WithA(p.Accent, 70), 0.8)
        cardstyle.Ring(dc, cx, cy, R*0.34, cardstyle.WithA(p.Accent, 90), 0.7)
        for i := 0; i < 8; i++ {
                a := float64(i) * math.Pi / 4
                cardstyle.Diamond(dc, cx+R*0.80*math.Cos(a), cy+R*0.80*math.Sin(a), 3, cardstyle.WithA(p.Accent, 120))
        }
}

func moneyStyle02(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.GoldenArcanum()
        s02ArcaneBase(dc, W, H, p)
        title, fromL, toL, _, _ := moneyMeta(req)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 15, strings.ToUpper(title), W/2, 66, p.Accent, 0.5, 4)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "CONDUCTED BY "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 94, p.Muted, 0.5, 2.4)
        // twin sigils joined by a gold arc of travel
        cxL, cxR, cy, R := 268.0, W-268.0, 330.0, 128.0
        arcaneCircle(dc, cxL, cy, R, p)
        arcaneCircle(dc, cxR, cy, R, p)
        dc.SetColor(cardstyle.WithA(p.Accent, 200))
        dc.SetLineWidth(1.4)
        dc.DrawLine(cxL+R*0.86, cy-R*0.5, cxR-R*0.86, cy-R*0.5)
        dc.Stroke()
        right := req.Type != "WITHDRAW"
        if right {
                cardstyle.Diamond(dc, cxR-R*0.86-14, cy-R*0.5, 5, p.Accent)
        } else {
                cardstyle.Diamond(dc, cxL+R*0.86+14, cy-R*0.5, 5, p.Accent)
        }
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 50, zeni(req.ZeniSymbol, req.Amount), W/2, cy-R*0.5-40, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 420, 18)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, fromL, cxL, cy+R+26, p.Muted, 0.5, 0.5, 240, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, toL, cxR, cy+R+26, p.Muted, 0.5, 0.5, 240, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 19, zeni(req.ZeniSymbol, req.NewWallet), cxL, cy+R+56, p.Ink, 0.5, 0.5, 260, 11)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 19, zeni(req.ZeniSymbol, req.NewBank), cxR, cy+R+56, p.Ink, 0.5, 0.5, 260, 11)
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, cardstyle.TruncateRunes(econCaption(req, req.Type), 70), W/2, H-64, cardstyle.WithA(p.Muted, 190), 0.5, 0.5, 760, 9)
        cardstyle.SealMini(dc, W-84, H-84, 30, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 14)
        return dc.Image()
}

func balanceStyle02(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.GoldenArcanum()
        s02ArcaneBase(dc, W, H, p)
        // grand total inside the great circle
        cx, cy, R := 268.0, H/2+10, 176.0
        arcaneCircle(dc, cx, cy, R, p)
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 44, formatFull(req.Total), cx, cy-8, cardstyle.WithA(cardstyle.Hex(0xffe9a8), 220), p.Accent, 0.5, 0.5, 260, 16)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 10, "TOTAL WEALTH", cx, cy+36, p.Muted, 0.5, 3)
        // right register
        cx0 := cx + R + 52
        cw := W - cx0 - 64
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 13, "THE TREASURY OF "+strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 16)), cx0, 108, p.Muted, 0, 2.4)
        rows := [][2]string{
                {"WALLET", zeni(req.ZeniSymbol, req.Wallet)},
                {"BANK", zeni(req.ZeniSymbol, req.Bank)},
        }
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                rows = append(rows, [2]string{"FROZEN", zeni(req.ZeniSymbol, fr)})
        }
        yy := 168.0
        for _, r := range rows {
                cardstyle.Diamond(dc, cx0+8, yy, 3.4, p.Accent)
                cardstyle.Text(dc, cardstyle.FtCinzel, 16, r[0], cx0+26, yy, p.Ink, 0, 0.5, 220, 10)
                cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, r[1], cx0+cw, yy, p.Accent, 1, 0.5, 300, 12)
                dc.SetColor(cardstyle.WithA(p.Accent, 110))
                dc.SetLineWidth(0.8)
                dc.DrawLine(cx0+26, yy+26, cx0+cw-20, yy+26)
                dc.Stroke()
                yy += 64
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(req.Rank)+" RANK - LEVEL "+fmt.Sprintf("%d", req.Level), cx0, yy+18, cardstyle.WithA(p.Muted, 200), 0, 0.5, 300, 9)
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, "the ledger remembers every coin", cx0, yy+52, cardstyle.WithA(p.Muted, 170), 0, 0.5, cw, 9)
        cardstyle.Chip(dc, cx0, H-104, 300, 38, ".J BALANCE", cardstyle.WithA(p.Track, 220), p.Accent, p.Ink, cardstyle.FtCinzel, 13)
        cardstyle.SealMini(dc, W-84, H-84, 34, p.Seal, p.SealTx, cardstyle.Sanitize(req.Rank), cardstyle.FtCinzel, 16)
        return dc.Image()
}

// ── Style 3 RETRO COURT - the parlour ledger ─────────────────────────

func s03ParlourBase(dc *gg.Context, W, H float64, p cardstyle.Palette) {
        cardstyle.PageBase(dc, W, H, p.Bg, p.Bg2, 26)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 10, 2, p.Accent, false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 15, 0.7, cardstyle.WithA(p.Accent, 150), false)
}

func parlourRule(dc *gg.Context, p cardstyle.Palette, W, y float64) {
        dc.SetColor(p.Ink)
        dc.SetLineWidth(1.6)
        dc.DrawLine(70, y, W-70, y)
        dc.Stroke()
        dc.SetLineWidth(0.6)
        dc.DrawLine(70, y+6, W-70, y+6)
        dc.Stroke()
        cardstyle.Fleuron(dc, W/2, y+3, 4.4, p.Accent)
}

func moneyStyle03(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RetroCourt()
        s03ParlourBase(dc, W, H, p)
        title, fromL, toL, _, _ := moneyMeta(req)
        cardstyle.Text(dc, cardstyle.FtIMFell, 44, strings.ToUpper(title), W/2, 92, p.Ink, 0.5, 0.5, 860, 24)
        parlourRule(dc, p, W, 138)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 18, "entered by "+econName(req), W/2, 172, p.Accent, 0.5, 0.5, 700, 11)
        cardstyle.Text(dc, cardstyle.FtIMFell, 46, zeni(req.ZeniSymbol, req.Amount), W/2, 252, p.Ink, 0.5, 0.5, 700, 20)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, "the sum changing hands this day", W/2, 292, p.Muted, 0.5, 0.5, 500, 10)
        // ledger lines with dot leaders
        cardstyle.DotLeader(dc, 250, 560, 356, 1.2, cardstyle.WithA(p.Ink, 140))
        cardstyle.Text(dc, cardstyle.FtIMFell, 18, fromL, 240, 356, p.Ink, 0, 0.5, 300, 11)
        cardstyle.Text(dc, cardstyle.FtIMFell, 18, zeni(req.ZeniSymbol, req.NewWallet), 570, 356, p.Ink, 1, 0.5, 190, 10)
        cardstyle.DotLeader(dc, 250, 560, 402, 1.2, cardstyle.WithA(p.Ink, 140))
        cardstyle.Text(dc, cardstyle.FtIMFell, 18, toL, 240, 402, p.Ink, 0, 0.5, 300, 11)
        cardstyle.Text(dc, cardstyle.FtIMFell, 18, zeni(req.ZeniSymbol, req.NewBank), 570, 402, p.Ink, 1, 0.5, 190, 10)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 80), W/2, 470, p.Muted, 0.5, 0.5, 800, 10)
        cardstyle.SealMini(dc, W/2, 522, 24, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtIMFell, 13)
        return dc.Image()
}

func balanceStyle03(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RetroCourt()
        s03ParlourBase(dc, W, H, p)
        cardstyle.Text(dc, cardstyle.FtIMFell, 40, "STATEMENT OF ACCOUNT", W/2, 88, p.Ink, 0.5, 0.5, 860, 20)
        parlourRule(dc, p, W, 132)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 18, "for "+cardstyle.Sanitize(req.Nickname)+" - "+strings.ToUpper(req.Rank)+" rank, level "+fmt.Sprintf("%d", req.Level), W/2, 166, p.Accent, 0.5, 0.5, 800, 11)
        yy := 226.0
        row := func(label, value string) {
                cardstyle.DotLeader(dc, 240, 620, yy, 1.2, cardstyle.WithA(p.Ink, 150))
                cardstyle.Text(dc, cardstyle.FtIMFell, 20, label, 230, yy, p.Ink, 0, 0.5, 340, 12)
                cardstyle.Text(dc, cardstyle.FtIMFell, 20, value, 630, yy, p.Ink, 1, 0.5, 140, 11)
                yy += 58
        }
        row("WALLET", zeni(req.ZeniSymbol, req.Wallet))
        row("BANK", zeni(req.ZeniSymbol, req.Bank))
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                row("FROZEN", zeni(req.ZeniSymbol, fr))
        }
        parlourRule(dc, p, W, yy+6)
        cardstyle.Text(dc, cardstyle.FtIMFell, 30, "TOTAL WEALTH", 230, yy+52, p.Accent, 0, 0.5, 380, 15)
        cardstyle.Text(dc, cardstyle.FtIMFell, 30, zeni(req.ZeniSymbol, req.Total), 630, yy+52, p.Accent, 1, 0.5, 140, 14)
        cardstyle.Text(dc, cardstyle.FtIMFellIt, 14, cardstyle.TruncateRunes("a penny kept is a penny earned - .j balance to return", 70), W/2, H-62, p.Muted, 0.5, 0.5, 800, 9)
        return dc.Image()
}

// ── Style 4 WOODMERE - the coin till ─────────────────────────────────

func s04TillBase(dc *gg.Context, W, H float64, p cardstyle.Palette, title string) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Planks(dc, W, H, 0x5700D4, cardstyle.N(0, 0, 0, 60), cardstyle.N(255, 230, 180, 22))
        cardstyle.BevelRect(dc, W/2-280, 30, 560, 76, cardstyle.N(112, 76, 42, 255), cardstyle.N(168, 122, 72, 72), cardstyle.N(44, 28, 14, 255), 2.6)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 26, title, W/2, 66, cardstyle.N(240, 202, 130, 255), 0.5, 0.5, 500, 14)
}

func moneyStyle04(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Woodmere()
        title, fromL, toL, _, _ := moneyMeta(req)
        s04TillBase(dc, W, H, p, strings.ToUpper(title))
        // vellum workbench panel
        dc.SetColor(cardstyle.N(36, 22, 10, 255))
        dc.DrawRoundedRectangle(100, 140, W-200, 356, 10)
        dc.Fill()
        dc.SetColor(p.Panel)
        dc.DrawRoundedRectangle(112, 152, W-224, 332, 8)
        dc.Fill()
        cardstyle.Stitches(dc, 124, 166, 468, 22, cardstyle.N(120, 84, 46, 255))
        cardstyle.Text(dc, cardstyle.FtCinzel, 14, "THE SUM MOVED", W/2, 208, cardstyle.N(116, 82, 50, 255), 0.5, 0.5, 300, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 44, zeni(req.ZeniSymbol, req.Amount), W/2, 258, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, 560, 18)
        // two trays + carved groove with a rolling coin
        ty, tw, th := 320.0, 300.0, 108.0
        cardstyle.BevelRect(dc, 150, ty, tw, th, cardstyle.N(150, 108, 62, 255), cardstyle.N(196, 148, 92, 255), cardstyle.N(58, 38, 18, 255), 2)
        cardstyle.BevelRect(dc, W-150-tw, ty, tw, th, cardstyle.N(150, 108, 62, 255), cardstyle.N(196, 148, 92, 255), cardstyle.N(58, 38, 18, 255), 2)
        cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(fromL), 150+tw/2, ty+30, cardstyle.N(96, 64, 34, 255), 0.5, 0.5, tw-40, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 14, strings.ToUpper(toL), W-150-tw/2, ty+30, cardstyle.N(96, 64, 34, 255), 0.5, 0.5, tw-40, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, zeni(req.ZeniSymbol, req.NewWallet), 150+tw/2, ty+66, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, tw-40, 12)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 24, zeni(req.ZeniSymbol, req.NewBank), W-150-tw/2, ty+66, cardstyle.N(52, 34, 20, 255), 0.5, 0.5, tw-40, 12)
        // groove + coin
        gy := ty + th/2
        dc.SetColor(cardstyle.N(58, 38, 18, 255))
        dc.DrawRectangle(150+tw+8, gy-4, W-300-tw*2-16, 8)
        dc.Fill()
        right := req.Type != "WITHDRAW"
        cx := W/2 + 60
        if !right {
                cx = W/2 - 60
        }
        dc.SetColor(cardstyle.N(226, 178, 92, 255))
        dc.DrawCircle(cx, gy, 22)
        dc.Fill()
        dc.SetColor(cardstyle.N(150, 104, 48, 255))
        dc.SetLineWidth(3)
        dc.DrawCircle(cx, gy, 22)
        dc.Stroke()
        dc.SetColor(cardstyle.N(150, 104, 48, 255))
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 16, req.ZeniSymbol, cx, gy, cardstyle.N(150, 104, 48, 255), 0.5, 0.5, 30, 8)
        cardstyle.Text(dc, cardstyle.FtMedieval, 15, cardstyle.TruncateRunes(econCaption(req, req.Type), 76), W/2, 528, cardstyle.N(226, 192, 142, 230), 0.5, 0.5, 800, 10)
        cardstyle.SealMini(dc, 70, 552, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 11)
        return dc.Image()
}

func balanceStyle04(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.Woodmere()
        s04TillBase(dc, W, H, p, "THE COIN TILL")
        dc.SetColor(cardstyle.N(36, 22, 10, 255))
        dc.DrawRoundedRectangle(100, 140, W-200, 356, 10)
        dc.Fill()
        dc.SetColor(p.Panel)
        dc.DrawRoundedRectangle(112, 152, W-224, 332, 8)
        dc.Fill()
        cardstyle.Stitches(dc, 124, 166, 468, 22, cardstyle.N(120, 84, 46, 255))
        cardstyle.Text(dc, cardstyle.FtCinzel, 14, "THE PURSE OF", 190, 200, cardstyle.N(116, 82, 50, 255), 0, 0.5, 220, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 22, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 18)), 190, 234, cardstyle.N(52, 34, 20, 255), 0, 0.5, 320, 11)
        yy := 286.0
        row := func(label, value string) {
                cardstyle.DotLeader(dc, 200, 700, yy, 1.2, cardstyle.N(150, 104, 56, 160))
                cardstyle.Text(dc, cardstyle.FtCinzel, 19, label, 190, yy, cardstyle.N(96, 64, 34, 255), 0, 0.5, 260, 10)
                cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, value, 710, yy, cardstyle.N(52, 34, 20, 255), 1, 0.5, 170, 11)
                yy += 54
        }
        row("WALLET", zeni(req.ZeniSymbol, req.Wallet))
        row("BANK", zeni(req.ZeniSymbol, req.Bank))
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                row("FROZEN", zeni(req.ZeniSymbol, fr))
        }
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 34, zeni(req.ZeniSymbol, req.Total), 710, yy+8, cardstyle.N(150, 78, 24, 255), 1, 0.5, 190, 14)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, "TOTAL WEALTH", 710, yy+38, cardstyle.N(96, 64, 34, 255), 1, 0.5, 190, 9)
        wp, _ := walletBankPct(req.Wallet, req.Bank, req.Total)
        cardstyle.NotchMeter(dc, 190, H-96, 300, 10, int(math.Round(wp*10)), cardstyle.N(150, 104, 56, 120), cardstyle.N(150, 78, 24, 255))
        cardstyle.Text(dc, cardstyle.FtMedieval, 14, "counted by hand, sealed by the till-keeper", W/2, 528, cardstyle.N(226, 192, 142, 230), 0.5, 0.5, 700, 9)
        cardstyle.SealMini(dc, 70, 552, 20, p.Seal, p.SealTx, cardstyle.Sanitize(req.Rank), cardstyle.FtCinzel, 11)
        return dc.Image()
}

// ── Style 5 EMBLEM NOIR - the house account ──────────────────────────

func s05NoirBase(dc *gg.Context, W, H float64, p cardstyle.Palette) float64 {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        dc.SetColor(cardstyle.N(255, 255, 255, 5))
        dc.SetLineWidth(1)
        for y := 0.0; y < H; y += 5 {
                dc.DrawLine(0, y, W, y)
                dc.Stroke()
        }
        for _, c := range [][2]float64{{26, 26}, {W - 26, 26}, {26, H - 26}, {W - 26, H - 26}} {
                dc.SetColor(cardstyle.WithA(p.Muted, 110))
                dc.SetLineWidth(1)
                dc.DrawLine(c[0]-7, c[1], c[0]+7, c[1])
                dc.Stroke()
                dc.DrawLine(c[0], c[1]-7, c[0], c[1]+7)
                dc.Stroke()
        }
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
        return 56.0
}

func moneyStyle05(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.EmblemNoir()
        s05NoirBase(dc, W, H, p)
        title, fromL, toL, _, _ := moneyMeta(req)
        // left chartered band - amount on file
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
        cardstyle.Spaced(dc, cardstyle.FtInter, 9, "AMOUNT", bx+bw/2, by+40, p.Muted, 0.5, 5)
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 30, zeni(req.ZeniSymbol, req.Amount), bx+bw/2, by+92, cardstyle.WithA(p.Accent, 50), cardstyle.WithA(p.Ink, 240), 0.5, 0.5, 170, 13)
        cardstyle.Spaced(dc, cardstyle.FtInter, 9, "ON FILE", bx+bw/2, by+126, p.Muted, 0.5, 5)
        // right register
        cx0 := bx + bw + 40
        cw := W - cx0 - 56
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, strings.ToUpper(title), cx0+cw/2, 132, p.Ink, 0.5, 4)
        dc.SetColor(cardstyle.WithA(p.Accent, 190))
        dc.SetLineWidth(1.2)
        dc.DrawLine(cx0, 146, cx0+cw/2-16, 146)
        dc.Stroke()
        dc.DrawLine(cx0+cw/2+16, 146, cx0+cw, 146)
        dc.Stroke()
        cardstyle.Diamond(dc, cx0+cw/2, 146, 4, p.Accent)
        // from / to registers with deco chevron flow
        ry := 208.0
        reg := func(label, value string) {
                cardstyle.Text(dc, cardstyle.FtInter, 12, label, cx0, ry, p.Muted, 0, 0.5, 220, 9)
                cardstyle.Text(dc, cardstyle.FtInterSemi, 26, value, cx0+cw, ry+6, p.Ink, 1, 0.5, 340, 13)
                ry += 74
        }
        reg(fromL, zeni(req.ZeniSymbol, req.NewWallet))
        reg(toL, zeni(req.ZeniSymbol, req.NewBank))
        right := req.Type != "WITHDRAW"
        flowArrow(dc, cx0+cw*0.2, cx0+cw*0.8, ry-38, cardstyle.WithA(p.Accent, 230), 2.4, right)
        cardstyle.Text(dc, cardstyle.FtInter, 11, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 74)), cx0, ry+6, cardstyle.WithA(p.Muted, 180), 0, 0.5, cw, 8)
        cardstyle.WaxSeal(dc, W-92, H-64, 40, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtInter, 18)
        return dc.Image()
}

func balanceStyle05(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.EmblemNoir()
        s05NoirBase(dc, W, H, p)
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
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 40, formatFull(req.Total), bx+bw/2, by+86, cardstyle.WithA(p.Accent, 60), cardstyle.WithA(p.Ink, 240), 0.5, 0.5, 180, 16)
        cardstyle.Spaced(dc, cardstyle.FtInter, 9, "TOTAL WEALTH", bx+bw/2, by+124, p.Muted, 0.5, 4)
        sunY := by + bh - 66
        dc.SetLineWidth(0.8)
        for i := 0; i <= 12; i++ {
                a := math.Pi + math.Pi*float64(i)/12
                dc.SetColor(cardstyle.WithA(p.Accent, uint8(90+60*math.Cos(float64(i)))))
                dc.DrawLine(bx+bw/2, sunY, bx+bw/2+84*math.Cos(a), sunY+84*math.Sin(a)*0.9)
                dc.Stroke()
        }
        cardstyle.Ring(dc, bx+bw/2, sunY, 6, cardstyle.WithA(p.Accent, 200), 1.1)
        cx0 := bx + bw + 40
        cw := W - cx0 - 56
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 14, "HOUSE ACCOUNT", cx0+cw/2, 132, p.Ink, 0.5, 4)
        dc.SetColor(cardstyle.WithA(p.Accent, 190))
        dc.SetLineWidth(1.2)
        dc.DrawLine(cx0, 146, cx0+cw/2-16, 146)
        dc.Stroke()
        dc.DrawLine(cx0+cw/2+16, 146, cx0+cw, 146)
        dc.Stroke()
        cardstyle.Diamond(dc, cx0+cw/2, 146, 4, p.Accent)
        cardstyle.Text(dc, cardstyle.FtInterSemi, 26, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 20)), cx0, 200, p.Ink, 0, 0.5, cw, 14)
        cardstyle.Text(dc, cardstyle.FtInter, 12, "CLIENT: "+strings.ToUpper(req.Rank)+" RANK - LEVEL "+fmt.Sprintf("%d", req.Level), cx0, 232, p.Muted, 0, 0.5, cw, 9)
        ry := 278.0
        reg := func(label, value string) {
                cardstyle.Text(dc, cardstyle.FtInter, 12, label, cx0, ry, p.Muted, 0, 0.5, 220, 9)
                cardstyle.Text(dc, cardstyle.FtInterSemi, 26, value, cx0+cw, ry+6, p.Ink, 1, 0.5, 340, 13)
                dc.SetColor(cardstyle.WithA(p.Accent, 120))
                dc.SetLineWidth(0.8)
                dc.DrawLine(cx0, ry+26, cx0+cw, ry+26)
                dc.Stroke()
                ry += 64
        }
        reg("WALLET", zeni(req.ZeniSymbol, req.Wallet))
        reg("BANK", zeni(req.ZeniSymbol, req.Bank))
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                reg("FROZEN", zeni(req.ZeniSymbol, fr))
        }
        cardstyle.Text(dc, cardstyle.FtInter, 11, "DISCRETION ASSURED - .J BALANCE", cx0, ry+2, cardstyle.WithA(p.Muted, 180), 0, 0.5, cw, 8)
        cardstyle.WaxSeal(dc, W-92, H-64, 40, p.Seal, cardstyle.Darken(p.Seal, 40), p.SealTx, cardstyle.Sanitize(req.Rank), cardstyle.FtInter, 18)
        return dc.Image()
}

// ── Style 6 SOUL FORGE - the astral exchequer ────────────────────────

func s06ForgeBase(dc *gg.Context, W, H float64, p cardstyle.Palette) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.StarField(dc, 16, 16, W-16, H-16, 34, 0x50F6CF, cardstyle.N(255, 255, 255, 255))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 11, 1.3, p.PanelEd, false)
        cardstyle.Diamond(dc, W/2, 11, 5.4, p.PanelEd)
        cardstyle.Diamond(dc, W/2, H-11, 5.4, p.PanelEd)
}

func moneyStyle06(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.SoulForge()
        s06ForgeBase(dc, W, H, p)
        title, fromL, toL, _, _ := moneyMeta(req)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, strings.ToUpper(title), W/2, 72, p.Accent, 0.5, 0.5, 560, 16)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 100, p.Muted, 0.5, 2.6)
        // soul thread: two diamond nodes joined by a dotted gold thread
        cy := 330.0
        nxL, nxR := 236.0, W-236.0
        cardstyle.DotLeader(dc, nxL+34, nxR-34, cy, 1.6, cardstyle.WithA(p.Accent, 170))
        cardstyle.DiamondOutline(dc, nxL, cy, 30, p.Accent, 1.6)
        cardstyle.DiamondOutline(dc, nxR, cy, 30, p.Accent, 1.6)
        cardstyle.Diamond(dc, nxL, cy, 8, p.Accent)
        cardstyle.Diamond(dc, nxR, cy, 8, p.Accent)
        right := req.Type != "WITHDRAW"
        cardstyle.Diamond(dc, W/2, cy, 6, cardstyle.Hex(0xffe9a8))
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, fromL, nxL, cy+56, p.Muted, 0.5, 0.5, 220, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, toL, nxR, cy+56, p.Muted, 0.5, 0.5, 240, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 21, zeni(req.ZeniSymbol, req.NewWallet), nxL, cy+86, p.Ink, 0.5, 0.5, 260, 11)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 21, zeni(req.ZeniSymbol, req.NewBank), nxR, cy+86, p.Ink, 0.5, 0.5, 260, 11)
        cardstyle.GlowText(dc, cardstyle.FtCinzelDec, 48, zeni(req.ZeniSymbol, req.Amount), W/2, cy-64, cardstyle.WithA(cardstyle.Hex(0x7ef2e2), 200), cardstyle.Hex(0xaefcf1), 0.5, 0.5, 420, 18)
        _ = right
        cardstyle.PillCTA(dc, W/2, 500, 400, 40, cardstyle.TruncateRunes(strings.ToLower(econCaption(req, req.Type)), 42), "", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 13)
        cardstyle.FooterNote(dc, W/2, H-26, cardstyle.Sanitize(req.SealText), p.Caption, cardstyle.FtMedieval, 12, 300)
        return dc.Image()
}

func balanceStyle06(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.SoulForge()
        s06ForgeBase(dc, W, H, p)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 30, "THE ASTRAL EXCHEQUER", W/2, 72, p.Accent, 0.5, 0.5, 620, 16)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 20))+" - "+strings.ToUpper(req.Rank)+" RANK - LVL "+fmt.Sprintf("%d", req.Level), W/2, 100, p.Muted, 0.5, 2.6)
        // three hairline shelves, cyan numerals, diamond bullets
        shelves := []struct{ label, value string; big bool }{
                {"TOTAL WEALTH", zeni(req.ZeniSymbol, req.Total), true},
                {"WALLET", zeni(req.ZeniSymbol, req.Wallet), false},
                {"BANK", zeni(req.ZeniSymbol, req.Bank), false},
        }
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                shelves = append(shelves, struct{ label, value string; big bool }{"FROZEN", zeni(req.ZeniSymbol, fr), false})
        }
        yy := 178.0
        for _, s := range shelves {
                size := 20
                if s.big {
                        size = 34
                }
                cardstyle.Diamond(dc, 150, yy, 4.4, p.Accent)
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, s.label, 176, yy, p.Muted, 0, 0.5, 260, 10)
                cardstyle.DotLeader(dc, 176+280, W-396, yy, 1, cardstyle.WithA(p.Accent, 120))
                cardstyle.GlowText(dc, cardstyle.FtCinzelDec, size, s.value, W-176, yy, cardstyle.WithA(cardstyle.Hex(0x7ef2e2), 120), cardstyle.Hex(0x9ffcef), 1, 0.5, 380, 12)
                yy += 86
        }
        cardstyle.PillCTA(dc, W/2, 512, 380, 40, ".j balance", "the night keeps count", cardstyle.WithA(p.Accent, 235), cardstyle.WithA(cardstyle.Lighten(p.Accent, 40), 190), cardstyle.N(42, 35, 24, 255), p.Muted, cardstyle.FtCinzel, 14)
        return dc.Image()
}

// ── Style 8 NEON ARCADE - the credit terminal ────────────────────────

func s08TerminalBase(dc *gg.Context, W, H float64, p cardstyle.Palette, title string) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Scanlines(dc, W, H, cardstyle.N(255, 64, 160, 10), 4)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2, cardstyle.HexA(0xff40a0, 200), false)
        dc.SetColor(cardstyle.WithA(p.Panel, 235))
        dc.DrawRoundedRectangle(W/2-330, 34, 660, 70, 6)
        dc.Fill()
        cardstyle.BracketCorners(dc, W/2-330, 34, 660, 70, 16, cardstyle.HexA(0x40e0ff, 220), 2)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 18, strings.ToUpper(title), W/2, 70, cardstyle.HexA(0xff40a0, 120), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 620, 9)
}

func moneyStyle08(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.NeonArcade()
        title, fromL, toL, _, _ := moneyMeta(req)
        s08TerminalBase(dc, W, H, p, title)
        cardstyle.Text(dc, cardstyle.FtInter, 12, "OPERATOR: "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 128, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 640, 9)
        // from / to slot panels + chunky pixel arrow
        py, pw, ph := 180.0, 330.0, 130.0
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(60, py, pw, ph, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, 60, py, pw, ph, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(W-60-pw, py, pw, ph, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, W-60-pw, py, pw, ph, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, fromL, 60+pw/2, py+36, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, pw-40, 8)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, toL, W-60-pw/2, py+36, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, pw-40, 8)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 17, zeni(req.ZeniSymbol, req.NewWallet), 60+pw/2, py+84, cardstyle.HexA(0x40e0ff, 120), cardstyle.Hex(0x40e0ff), 0.5, 0.5, pw-40, 10)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 17, zeni(req.ZeniSymbol, req.NewBank), W-60-pw/2, py+84, cardstyle.HexA(0x40e0ff, 120), cardstyle.Hex(0x40e0ff), 0.5, 0.5, pw-40, 10)
        // chunky pixel arrow
        right := req.Type != "WITHDRAW"
        ay := py + ph/2
        dc.SetColor(cardstyle.Hex(0xff40a0))
        if right {
                for i := 0; i < 4; i++ {
                        dc.DrawRectangle(470+float64(i)*14, ay-4, 10, 8)
                        dc.Fill()
                }
                dc.DrawRectangle(524, ay-12, 8, 24)
                dc.Fill()
        } else {
                for i := 0; i < 4; i++ {
                        dc.DrawRectangle(530-float64(i)*14, ay-4, 10, 8)
                        dc.Fill()
                }
                dc.DrawRectangle(468, ay-12, 8, 24)
                dc.Fill()
        }
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 26, zeni(req.ZeniSymbol, req.Amount), W/2, 400, cardstyle.HexA(0xff40a0, 130), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 520, 12)
        cardstyle.Text(dc, cardstyle.FtInter, 12, cardstyle.TruncateRunes(econCaption(req, req.Type), 74), W/2, 452, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 780, 9)
        cardstyle.SegBar(dc, 110, 500, W-220, 14, 0.62, 20, cardstyle.Hex(0x40e0ff), cardstyle.HexA(0x40e0ff, 40))
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, "TRANSACTION COMPLETE - INSERT COIN TO CONTINUE", W/2, 560, cardstyle.HexA(0xff9ecb, 180), 0.5, 0.5, 620, 7)
        return dc.Image()
}

func balanceStyle08(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.NeonArcade()
        s08TerminalBase(dc, W, H, p, "CREDIT BALANCE")
        cardstyle.Text(dc, cardstyle.FtInter, 12, "OPERATOR: "+strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 18))+" - "+strings.ToUpper(req.Rank)+" - LVL "+fmt.Sprintf("%d", req.Level), W/2, 128, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 760, 9)
        // total readout
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 34, zeni(req.ZeniSymbol, req.Total), W/2, 216, cardstyle.HexA(0xff40a0, 130), cardstyle.Hex(0xff9ecb), 0.5, 0.5, 760, 14)
        cardstyle.Text(dc, cardstyle.FtPS2P, 10, "TOTAL WEALTH", W/2, 262, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, 400, 8)
        // wallet / bank panels with distribution bar
        py, ph := 300.0, 130.0
        pw := 380.0
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(90, py, pw, ph, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, 90, py, pw, ph, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        dc.SetColor(cardstyle.WithA(p.Panel, 225))
        dc.DrawRoundedRectangle(W-90-pw, py, pw, ph, 4)
        dc.Fill()
        cardstyle.BracketCorners(dc, W-90-pw, py, pw, ph, 14, cardstyle.HexA(0x40e0ff, 190), 1.6)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, "WALLET", 90+pw/2, py+34, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, pw-40, 8)
        cardstyle.Text(dc, cardstyle.FtPS2P, 11, "BANK", W-90-pw/2, py+34, cardstyle.HexA(0x80a8d0, 235), 0.5, 0.5, pw-40, 8)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 16, zeni(req.ZeniSymbol, req.Wallet), 90+pw/2, py+80, cardstyle.HexA(0x40e0ff, 120), cardstyle.Hex(0x40e0ff), 0.5, 0.5, pw-40, 10)
        cardstyle.GlowText(dc, cardstyle.FtPS2P, 16, zeni(req.ZeniSymbol, req.Bank), W-90-pw/2, py+80, cardstyle.HexA(0x40e0ff, 120), cardstyle.Hex(0x40e0ff), 0.5, 0.5, pw-40, 10)
        wp, bp := walletBankPct(req.Wallet, req.Bank, req.Total)
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, fmt.Sprintf("%d%%", int(wp*100)), 90+pw/2, py+112, cardstyle.Hex(0x40e0ff), 0.5, 0.5, 120, 7)
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, fmt.Sprintf("%d%%", int(bp*100)), W-90-pw/2, py+112, cardstyle.Hex(0x40e0ff), 0.5, 0.5, 120, 7)
        cardstyle.SegBar(dc, 110, 478, W-220, 16, wp, 24, cardstyle.Hex(0xff40a0), cardstyle.HexA(0x40e0ff, 40))
        cardstyle.Text(dc, cardstyle.FtPS2P, 8, "INSERT COIN TO CONTINUE", W/2, 560, cardstyle.HexA(0xff9ecb, 180), 0.5, 0.5, 320, 7)
        return dc.Image()
}

// ── Style 9 RUNE MONOLITH - the counting stones ──────────────────────

func s09Stele(dc *gg.Context, x, y, w, h, ap float64, p cardstyle.Palette, light bool) {
        cx := x + w/2
        dc.ClearPath()
        dc.MoveTo(x, y+h)
        dc.LineTo(x, y+ap)
        dc.QuadraticTo(x, y+ap*0.30, cx, y)
        dc.QuadraticTo(x+w, y+ap*0.30, x+w, y+ap)
        dc.LineTo(x+w, y+h)
        dc.ClosePath()
        dc.SetColor(cardstyle.Darken(p.Panel, 22))
        dc.Fill()
        dc.SetColor(cardstyle.N(0, 0, 0, 90))
        dc.SetLineWidth(3)
        dc.Stroke()
        dc.Push()
        dc.ClearPath()
        dc.MoveTo(x, y+h)
        dc.LineTo(x, y+ap)
        dc.QuadraticTo(x, y+ap*0.30, cx, y)
        dc.QuadraticTo(x+w, y+ap*0.30, x+w, y+ap)
        dc.LineTo(x+w, y+h)
        dc.ClosePath()
        dc.Clip()
        for i := 0; i < 20; i++ {
                f0 := float64(i) / 20
                dc.SetColor(cardstyle.Lighten(p.Panel, uint8(18*(1-f0))))
                dc.DrawRectangle(x, y+h*f0, w, h/20+1)
                dc.Fill()
        }
        cardstyle.Speckle(dc, x, y, x+w, y+h, 44, 0x51B2E7, cardstyle.N(0, 0, 0, 220))
        cardstyle.GlyphCol(dc, x+20, y+ap+12, y+h-14, 40, cardstyle.N(255, 255, 255, 15))
        cardstyle.GlyphCol(dc, x+w-30, y+ap+12, y+h-14, 40, cardstyle.N(255, 255, 255, 15))
        dc.Pop()
        // gg v1.3.0 Pop() does NOT restore the clipping mask (it keeps the
        // post-Clip mask) - without ResetClip every later draw would be
        // clipped to this stele.
        dc.ResetClip()
        if light {
                dc.ClearPath()
                dc.MoveTo(x+12, y+h-8)
                dc.LineTo(x+12, y+ap*0.9)
                dc.QuadraticTo(x+12, y+ap*0.28, cx, y+8)
                dc.QuadraticTo(x+w-12, y+ap*0.28, x+w-12, y+ap*0.9)
                dc.LineTo(x+w-12, y+h-8)
                dc.ClosePath()
                dc.SetColor(cardstyle.N(0, 0, 0, 120))
                dc.SetLineWidth(2)
                dc.Stroke()
        }
}

func moneyStyle09(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RuneMonolith()
        cardstyle.VertGrad(dc, W, H, cardstyle.Darken(p.Bg, 26), cardstyle.Darken(p.Bg2, 34))
        cardstyle.Speckle(dc, 0, 0, W, H, 110, 0x9E11A8, cardstyle.N(0, 0, 0, 255))
        title, fromL, toL, _, _ := moneyMeta(req)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, strings.ToUpper(title), W/2, 74, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Bg, 50), 0.5, 0.5, 560, 13)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, "BY THE HAND OF "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 102, p.Muted, cardstyle.Darken(p.Bg, 50), 0.5, 0.5, 520, 9)
        // two standing stones + carved channel with ember spark
        dc.SetColor(cardstyle.N(0, 0, 0, 130))
        dc.DrawEllipse(W/2, H-36, 420, 16)
        dc.Fill()
        sy, sh := 150.0, 340.0
        s09Stele(dc, 96, sy, 320, sh, 46, p, true)
        s09Stele(dc, W-96-320, sy, 320, sh, 46, p, true)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, fromL, 96+160, sy+56, cardstyle.WithA(p.Ink, 210), 0.5, 0.5, 280, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, toL, W-96-160, sy+56, cardstyle.WithA(p.Ink, 210), 0.5, 0.5, 280, 9)
        cardstyle.EmberGlow(dc, 96+160, sy+130, 70, p.Accent)
        cardstyle.EmberGlow(dc, W-96-160, sy+130, 70, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 30, zeni(req.ZeniSymbol, req.NewWallet), 96+160, sy+130, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 280, 13)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 30, zeni(req.ZeniSymbol, req.NewBank), W-96-160, sy+130, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, 280, 13)
        cardstyle.GlyphCol(dc, 96+160-60, sy+180, sy+300, 40, cardstyle.N(255, 255, 255, 20))
        cardstyle.GlyphCol(dc, W-96-160+40, sy+180, sy+300, 40, cardstyle.N(255, 255, 255, 20))
        // carved channel + ember spark
        dc.SetColor(cardstyle.N(0, 0, 0, 140))
        dc.DrawRectangle(436, H/2-6, W-872, 12)
        dc.Fill()
        cardstyle.EmberGlow(dc, W/2, H/2, 56, p.Accent)
        cardstyle.Diamond(dc, W/2, H/2, 9, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 34, zeni(req.ZeniSymbol, req.Amount), W/2, H/2-44, p.Accent2, cardstyle.Darken(p.Bg, 55), 0.5, 0.5, 340, 14)
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.TruncateRunes(econCaption(req, req.Type), 66)), W/2, H-72, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 780, 9)
        return dc.Image()
}

func balanceStyle09(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.RuneMonolith()
        cardstyle.VertGrad(dc, W, H, cardstyle.Darken(p.Bg, 26), cardstyle.Darken(p.Bg2, 34))
        cardstyle.Speckle(dc, 0, 0, W, H, 110, 0x9E11A8, cardstyle.N(0, 0, 0, 255))
        // single wide stele as the ledger
        dc.SetColor(cardstyle.N(0, 0, 0, 130))
        dc.DrawEllipse(W/2, H-30, 420, 14)
        dc.Fill()
        sx, sy, sw, sh := 120.0, 54.0, W-240.0, H-96.0
        s09Stele(dc, sx, sy, sw, sh, 64, p, true)
        ix, iy := sx+56, sy+92
        iw := sw - 112
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 26, "THE COUNTING STONE", ix+iw/2, iy, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, iw-140, 13)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 12, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 18))+" - "+strings.ToUpper(req.Rank)+" RANK - LVL "+fmt.Sprintf("%d", req.Level), ix+iw/2, iy+28, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, iw-120, 9)
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
        // total register with ember glow
        icy := iy + 128.0
        cardstyle.EmberGlow(dc, ix+iw/2, icy, 84, p.Accent)
        cardstyle.Engrave(dc, cardstyle.FtCinzel, 13, "TOTAL WEALTH", ix+iw/2, icy-42, p.Muted, cardstyle.Darken(p.Panel, 60), 0.5, 0.5, 300, 9)
        cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 44, zeni(req.ZeniSymbol, req.Total), ix+iw/2, icy, cardstyle.Lighten(p.Ink, 14), cardstyle.Darken(p.Panel, 70), 0.5, 0.5, iw-100, 18)
        // wallet / bank registers
        ry := icy + 66.0
        reg := func(label, value string) {
                cardstyle.Engrave(dc, cardstyle.FtCinzel, 14, label, ix+8, ry, p.Muted, cardstyle.Darken(p.Panel, 60), 0, 0.5, 220, 9)
                cardstyle.Engrave(dc, cardstyle.FtCinzelDec, 22, value, ix+iw-8, ry, cardstyle.Lighten(p.Ink, 12), cardstyle.Darken(p.Panel, 70), 1, 0.5, 300, 11)
                ry += 52
        }
        reg("WALLET", zeni(req.ZeniSymbol, req.Wallet))
        reg("BANK", zeni(req.ZeniSymbol, req.Bank))
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                reg("FROZEN", zeni(req.ZeniSymbol, fr))
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 12, strings.ToUpper("the stone keeps every count"), ix+iw/2, sy+sh-34, cardstyle.WithA(p.Ink, 190), 0.5, 0.5, 500, 9)
        return dc.Image()
}

// ── Style 10 CRIMSON COURT - the treasurer's plate ───────────────────

func s10CourtBase(dc *gg.Context, W, H float64, p cardstyle.Palette) {
        cardstyle.VertGrad(dc, W, H, p.Bg, p.Bg2)
        cardstyle.Damask(dc, W, H, 76, cardstyle.HexA(0xd4a856, 20))
        cardstyle.FrameHairline(dc, 0, 0, W, H, 9, 2.4, p.Accent, false)
        cardstyle.FrameHairline(dc, 0, 0, W, H, 16, 1, cardstyle.WithA(p.Accent, 130), false)
}

func moneyStyle10(req *TransactionCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.CrimsonCourt()
        s10CourtBase(dc, W, H, p)
        title, fromL, toL, _, _ := moneyMeta(req)
        cardstyle.Cartouche(dc, W/2, 72, 520, 56, p.Panel, p.Accent, p.Ink, strings.ToUpper(title), cardstyle.FtCinzelDec, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, "COMMISSIONED BY "+strings.ToUpper(cardstyle.Sanitize(econName(req))), W/2, 122, p.Muted, 0.5, 2.4)
        // twin shields joined by a gold filigree arrow
        sy := 296.0
        cardstyle.Shield(dc, 240, sy, 260, 210, p.Panel, p.Accent, 3)
        cardstyle.Shield(dc, W-240, sy, 260, 210, p.Panel, p.Accent, 3)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, fromL, 240, sy-62, cardstyle.WithA(p.Ink, 220), 0.5, 0.5, 240, 9)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, toL, W-240, sy-62, cardstyle.WithA(p.Ink, 220), 0.5, 0.5, 240, 9)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, zeni(req.ZeniSymbol, req.NewWallet), 240, sy+34, p.Ink, 0.5, 0.5, 230, 11)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 22, zeni(req.ZeniSymbol, req.NewBank), W-240, sy+34, p.Ink, 0.5, 0.5, 230, 11)
        right := req.Type != "WITHDRAW"
        flowArrow(dc, W/2-140, W/2+140, sy, cardstyle.WithA(p.Accent, 240), 3, right)
        cardstyle.Diamond(dc, W/2, sy, 6, p.Accent)
        cardstyle.Text(dc, cardstyle.FtCinzelDec, 40, zeni(req.ZeniSymbol, req.Amount), W/2, sy+104, p.Accent2, 0.5, 0.5, 420, 16)
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper(econCaption(req, req.Type)), 70), W/2, H-92, cardstyle.WithA(p.Ink, 220), 0.5, 0.5, 820, 10)
        cardstyle.WaxSeal(dc, 80, H-64, 30, p.Seal, cardstyle.Darken(p.Seal, 44), p.SealTx, cardstyle.Sanitize(req.SealText), cardstyle.FtCinzel, 16)
        return dc.Image()
}

func balanceStyle10(req *EconomyCardRequest) image.Image {
        const W, H = 1000.0, 600.0
        dc := gg.NewContext(int(W), int(H))
        p := cardstyle.CrimsonCourt()
        s10CourtBase(dc, W, H, p)
        cardstyle.Cartouche(dc, W/2, 72, 560, 56, p.Panel, p.Accent, p.Ink, "THE TREASURY", cardstyle.FtCinzelDec, 22)
        cardstyle.Spaced(dc, cardstyle.FtCinzel, 11, strings.ToUpper(cardstyle.TruncateRunes(cardstyle.Sanitize(req.Nickname), 18))+" - "+strings.ToUpper(req.Rank)+" RANK - LVL "+fmt.Sprintf("%d", req.Level), W/2, 122, p.Muted, 0.5, 2.4)
        // three stacked cartouches
        yy := 216.0
        plate := func(label, value string, wide bool) {
                w := 640.0
                if wide {
                        w = 760.0
                }
                cardstyle.Cartouche(dc, W/2, yy, w, 74, p.Panel, p.Accent, p.Ink, "", cardstyle.FtCinzel, 14)
                cardstyle.Text(dc, cardstyle.FtCinzel, 14, label, W/2-w/2+34, yy, cardstyle.WithA(p.Ink, 220), 0, 0.5, 240, 10)
                cardstyle.Text(dc, cardstyle.FtCinzelDec, 28, value, W/2+w/2-34, yy, p.Ink, 1, 0.5, 340, 13)
                yy += 104
        }
        plate("TOTAL WEALTH", zeni(req.ZeniSymbol, req.Total), true)
        plate("WALLET", zeni(req.ZeniSymbol, req.Wallet), false)
        plate("BANK", zeni(req.ZeniSymbol, req.Bank), false)
        if fr := balanceFrozen(req.Frozen); fr > 0 {
                plate("FROZEN", zeni(req.ZeniSymbol, fr), false)
        }
        cardstyle.Text(dc, cardstyle.FtCinzel, 13, cardstyle.TruncateRunes(strings.ToUpper("the court keeps its own counsel - .j balance"), 60), W/2, H-72, cardstyle.WithA(p.Ink, 200), 0.5, 0.5, 820, 10)
        cardstyle.WaxSeal(dc, 80, H-64, 30, p.Seal, cardstyle.Darken(p.Seal, 44), p.SealTx, cardstyle.Sanitize(req.Rank), cardstyle.FtCinzel, 16)
        return dc.Image()
}

// ── QA hooks ─────────────────────────────────────────────────────────

// QAStyledBalance renders a styled balance card from the raw request
// (QA harness only).
func QAStyledBalance(req *EconomyCardRequest) image.Image {
        return renderStyledBalance(req)
}
