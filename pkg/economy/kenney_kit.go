package economy

// kenney_kit.go - shared drawing kit for the Kenney-style money cards.
// Faithful port of the owner-approved redesigns (download/money_cards/*.png):
// bevel depth buttons, autofit text, face-centered labels, EMV chip, coin.

import (
        "fmt"
        "image/color"
        "strings"

        "image-service/pkg/utils"

        "github.com/fogleman/gg"
)

const (
        cardW = 1000.0
        cardH = 600.0
)

// ---- palette (sampled from the Kenney UI Pack "Double" PNGs) ----
var (
        knGrey   = kenSet{face: rgb(218, 220, 231), depth: rgb(102, 104, 128), edge: rgb(233, 235, 244)}
        knGreen  = kenSet{face: rgb(22, 187, 119), depth: rgb(4, 109, 65), edge: rgb(47, 215, 146)}
        knBlue   = kenSet{face: rgb(28, 159, 215), depth: rgb(20, 101, 135), edge: rgb(54, 189, 247)}
        knYellow = kenSet{face: rgb(255, 204, 0), depth: rgb(180, 128, 0), edge: rgb(255, 234, 156)}
        knRed    = kenSet{face: rgb(238, 39, 71), depth: rgb(135, 16, 35), edge: rgb(255, 98, 123)}
        knPurple = kenSet{face: rgb(150, 80, 220), depth: rgb(96, 40, 150), edge: rgb(190, 120, 255)}
        knOrange = kenSet{face: rgb(255, 120, 50), depth: rgb(170, 60, 20), edge: rgb(255, 160, 100)}
        knLight  = rgb(218, 220, 231) // Kenney border-style face
)

type kenSet struct {
        face, depth, edge color.RGBA
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }

// ---- fonts ----
func fontFuture() string { return utils.GetAssetPath("rpgasset", "ui", "KenneyFuture.ttf") }

func fontFutureNarrow() string {
        return utils.GetAssetPath("rpgasset", "ui", "KenneyFutureNarrow.ttf")
}

// loadFit loads the font at `size`, shrinking until s fits maxW (min minSize).
// Returns the chosen size. dc keeps the final face loaded.
func loadFit(dc *gg.Context, path string, size int, s string, maxW float64, minSize int) int {
        if size < minSize {
                size = minSize
        }
        for {
                dc.LoadFontFace(path, float64(size))
                w, _ := dc.MeasureString(s)
                if w <= maxW || size <= minSize {
                        break
                }
                size--
        }
        return size
}

func textLM(dc *gg.Context, s string, x, y float64, col color.RGBA) {
        dc.SetColor(col)
        dc.DrawStringAnchored(s, x, y, 0, 0.5)
}

func textRM(dc *gg.Context, s string, x, y float64, col color.RGBA) {
        dc.SetColor(col)
        dc.DrawStringAnchored(s, x, y, 1, 0.5)
}

func textCenter(dc *gg.Context, s string, cx, cy float64, col color.RGBA) {
        dc.SetColor(col)
        dc.DrawStringAnchored(s, cx, cy, 0.5, 0.5)
}

// faceCY - Kenney depth buttons: bright face is the top 87.5%, so the visual
// center of the face sits at y0 + 0.4375*h.
func faceCY(y0, h float64) float64 { return y0 + h*0.4375 }

// ---- base background: gradient + glow + dot grid + rounded panel ----
func drawBase(dc *gg.Context) {
        W, H := cardW, cardH
        // vertical gradient (24,27,48) -> (13,14,28)
        for y := 0; y < int(H); y++ {
                t := float64(y) / (H - 1)
                r := uint8(24 + (13-24)*t)
                g := uint8(27 + (14-27)*t)
                b := uint8(48 + (28-48)*t)
                dc.SetColor(rgb(r, g, b))
                dc.DrawLine(0, float64(y)+0.5, W, float64(y)+0.5)
                dc.Stroke()
        }
        // soft blue glow, top-right (center at right edge, y=60)
        for i := 380; i > 0; i -= 4 {
                t := float64(i) / 380.0
                a := uint8(50 * (1 - t) * (1 - t))
                dc.SetColor(color.NRGBA{R: 90, G: 110, B: 255, A: a})
                dc.DrawCircle(W, 60, float64(i))
                dc.Fill()
        }
        // dot grid
        dc.SetColor(color.NRGBA{R: 255, G: 255, B: 255, A: 10})
        for gx := 60.0; gx < W; gx += 60 {
                for gy := 60.0; gy < H; gy += 60 {
                        dc.DrawCircle(gx, gy, 1.2)
                        dc.Fill()
                }
        }
        // panel
        dc.SetColor(color.NRGBA{R: 30, G: 34, B: 62, A: 250})
        dc.DrawRoundedRectangle(36, 36, W-72, H-72, 30)
        dc.Fill()
        dc.SetColor(rgb(72, 82, 140))
        dc.SetLineWidth(2)
        dc.DrawRoundedRectangle(36, 36, W-72, H-72, 30)
        dc.Stroke()
}

// ---- Kenney depth buttons ----
func cornerR(h float64) float64 {
        r := h * 0.22
        if r < 8 {
                r = 8
        }
        if r > 26 {
                r = 26
        }
        return r
}

// kenButtonFlat - solid face + darker bottom depth strip.
func kenButtonFlat(dc *gg.Context, x, y, w, h float64, set kenSet) {
        r := cornerR(h)
        bevel := h * 0.125
        dc.SetColor(set.depth)
        dc.DrawRoundedRectangle(x, y, w, h, r)
        dc.Fill()
        dc.SetColor(set.face)
        dc.DrawRoundedRectangle(x, y, w, h-bevel, r)
        dc.Fill()
}

// kenButtonBorder - colored depth + light face + colored ring (badge/title style).
func kenButtonBorder(dc *gg.Context, x, y, w, h float64, set kenSet) {
        r := cornerR(h)
        bevel := h * 0.125
        dc.SetColor(set.depth)
        dc.DrawRoundedRectangle(x, y, w, h, r)
        dc.Fill()
        dc.SetColor(knLight)
        dc.DrawRoundedRectangle(x, y, w, h-bevel, r)
        dc.Fill()
        dc.SetColor(set.edge)
        dc.SetLineWidth(3)
        dc.DrawRoundedRectangle(x+1.5, y+1.5, w-3, h-bevel-3, r*0.92)
        dc.Stroke()
}

// ---- chip + coin ----
func drawChip(dc *gg.Context) {
        dc.SetColor(rgb(246, 196, 84))
        dc.DrawRoundedRectangle(66, 160, 80, 60, 12)
        dc.Fill()
        dc.SetColor(rgb(160, 120, 40))
        dc.SetLineWidth(3)
        dc.DrawRoundedRectangle(66, 160, 80, 60, 12)
        dc.Stroke()
        dc.SetLineWidth(2)
        dc.DrawLine(66, 180, 146, 180)
        dc.Stroke()
        dc.DrawLine(66, 200, 146, 200)
        dc.Stroke()
        dc.DrawLine(96, 160, 96, 220)
        dc.Stroke()
        dc.DrawLine(116, 160, 116, 220)
        dc.Stroke()
}

func drawCoin(dc *gg.Context, cx, cy, r float64, symbol string) {
        dc.SetColor(rgb(246, 196, 84))
        dc.DrawCircle(cx, cy, r)
        dc.Fill()
        dc.SetColor(rgb(160, 120, 40))
        dc.SetLineWidth(3)
        dc.DrawCircle(cx, cy, r)
        dc.Stroke()
        dc.SetColor(rgb(255, 236, 180))
        dc.SetLineWidth(2)
        dc.DrawCircle(cx, cy, r-6)
        dc.Stroke()
        sym := sanitize(symbol)
        if sym == "" {
                sym = "Z"
        }
        loadFit(dc, fontFutureNarrow(), int(r*1.1), sym, r*1.2, 8)
        textCenter(dc, sym, cx, cy-1, rgb(90, 62, 14))
}

// ---- header (name plate + accent badge + chip) ----
func drawHead(dc *gg.Context, name string, badge string, accent kenSet) {
        // name plate - Grey flat, dark text
        npw, nph, npx, npy := 320.0, 64.0, 66.0, 62.0
        kenButtonFlat(dc, npx, npy, npw, nph, knGrey)
        loadFit(dc, fontFuture(), 32, name, npw-60, 12)
        textLM(dc, name, npx+30, faceCY(npy, nph), rgb(36, 40, 64))
        // rank / info badge - accent border style, width measured from text
        badge = sanitize(badge)
        loadFit(dc, fontFutureNarrow(), 20, badge, 10000, 20)
        tw, _ := dc.MeasureString(badge)
        bw := tw + 48
        if bw < 170 {
                bw = 170
        }
        bh, bx, by := 56.0, cardW-66-bw, 66.0
        kenButtonBorder(dc, bx, by, bw, bh, accent)
        textCenter(dc, badge, bx+bw/2, faceCY(by, bh), rgb(40, 36, 18))
        // EMV chip
        drawChip(dc)
}

// ---- kv pill (label above value, both left-aligned, inside the panel) ----
func kvPill(dc *gg.Context, x0, y0, x1, y1 float64, label, value string,
        labelSize, valueSize int, labelCol, valueCol color.RGBA, pad float64) {
        h := y1 - y0
        cy := faceCY(y0, h)
        loadFit(dc, fontFutureNarrow(), labelSize, label, (x1-x0)-2*pad, 10)
        textLM(dc, label, x0+pad, cy-h*0.16, labelCol)
        loadFit(dc, fontFuture(), valueSize, value, (x1-x0)-2*pad, 12)
        textLM(dc, value, x0+pad, cy+h*0.17, valueCol)
}

// ---- title plate (border style, centered) ----
func drawTitlePlate(dc *gg.Context, msg string, set kenSet, txtCol color.RGBA) {
        pw, ph := 420.0, 66.0
        px, py := (cardW-pw)/2, 180.0
        kenButtonBorder(dc, px, py, pw, ph, set)
        loadFit(dc, fontFuture(), 24, msg, pw-60, 12)
        textCenter(dc, msg, cardW/2, faceCY(py, ph), txtCol)
}

// ---- caption line ----
func drawCaption(dc *gg.Context, s string) {
        s = sanitize(s)
        loadFit(dc, fontFutureNarrow(), 18, s, cardW-200, 10)
        textCenter(dc, s, cardW/2, 548, rgb(110, 120, 160))
}

// ---- right-pointing arrow (wallet -> bank) ----
func drawArrow(dc *gg.Context, cx, cy float64) {
        dc.MoveTo(cx-26, cy-36)
        dc.LineTo(cx+34, cy)
        dc.LineTo(cx-26, cy+36)
        dc.ClosePath()
        dc.SetColor(rgb(255, 204, 0))
        dc.Fill()
        dc.SetColor(rgb(160, 120, 40))
        dc.SetLineWidth(5)
        dc.SetLineJoin(gg.LineJoinRound)
        dc.Stroke()
}

// ---- numbers ----
func groupDigits(s string) string {
        neg := strings.HasPrefix(s, "-")
        if neg {
                s = s[1:]
        }
        n := len(s)
        if n > 3 {
                var b strings.Builder
                pre := n % 3
                if pre > 0 {
                        b.WriteString(s[:pre])
                        if n > pre {
                                b.WriteString(",")
                        }
                }
                for i := pre; i < n; i += 3 {
                        b.WriteString(s[i : i+3])
                        if i+3 < n {
                                b.WriteString(",")
                        }
                }
                s = b.String()
        }
        if neg {
                s = "-" + s
        }
        return s
}

// formatFull - full comma-grouped integer, e.g. 10145678 -> "10,145,678".
func formatFull(n float64) string {
        return groupDigits(fmt.Sprintf("%.0f", n))
}

// formatSigned - "+1,234" / "-1,234".
func formatSigned(n float64) string {
        s := formatFull(n)
        if n >= 0 {
                return "+" + s
        }
        return s
}

// sanitize - keep printable ASCII + middle dot so the Kenney fonts never
// render tofu boxes for nicknames/items coming from WhatsApp.
func sanitize(s string) string {
        var b strings.Builder
        for _, r := range s {
                if r == '·' || (r >= 0x20 && r <= 0x7E) {
                        b.WriteRune(r)
                }
        }
        out := strings.TrimSpace(b.String())
        return out
}

// ---- transaction type accents ----
type txAccent struct {
        set       kenSet
        amountCol color.RGBA
        titleCol  color.RGBA
}

func txStyle(t string) txAccent {
        switch t {
        case "DEPOSIT":
                return txAccent{knBlue, rgb(120, 190, 245), rgb(10, 58, 86)}
        case "WITHDRAW":
                return txAccent{knRed, rgb(248, 148, 140), rgb(124, 22, 30)}
        case "CRAFT", "TRANSFER":
                return txAccent{knGreen, rgb(120, 225, 140), rgb(14, 62, 34)}
        case "BREW":
                return txAccent{knPurple, rgb(200, 150, 250), rgb(58, 20, 96)}
        case "COOK":
                return txAccent{knYellow, rgb(250, 210, 110), rgb(94, 74, 8)}
        case "FORGE":
                return txAccent{knOrange, rgb(250, 160, 110), rgb(124, 36, 14)}
        default:
                return txAccent{knGrey, rgb(200, 205, 225), rgb(46, 48, 66)}
        }
}

// drawAmountLine - big value + gold ZENI suffix, centered as a group.
func drawAmountLine(dc *gg.Context, amount string, cy float64, col color.RGBA, maxAmountW float64) {
        amount = sanitize(amount)
        // measure ZENI first (narrow face), then fit + draw the amount (Future face),
        // then draw ZENI - order matters because the face is shared state.
        loadFit(dc, fontFutureNarrow(), 26, "ZENI", 200, 12)
        zw, _ := dc.MeasureString("ZENI")
        loadFit(dc, fontFuture(), 70, amount, maxAmountW, 20)
        aw, _ := dc.MeasureString(amount)
        gap := 28.0
        x0 := cardW/2 - (aw+gap+zw)/2
        textLM(dc, amount, x0, cy, col)
        loadFit(dc, fontFutureNarrow(), 26, "ZENI", 200, 12)
        textLM(dc, "ZENI", x0+aw+gap, cy, rgb(200, 176, 110))
}
