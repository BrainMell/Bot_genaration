package economy

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	CARD_W = 800
	CARD_H = 450
)

type EconomyCardRequest struct {
	Nickname   string  `json:"nickname"`
	Wallet     float64 `json:"wallet"`
	Bank       float64 `json:"bank"`
	Total      float64 `json:"total"`
	Frozen     float64 `json:"frozen"`
	ZeniSymbol string  `json:"zeniSymbol"`
	Rank       string  `json:"rank"`
	Level      int     `json:"level"`
}

func GenerateEconomyCard(c *gin.Context) {
	var req EconomyCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.ZeniSymbol == "" {
		req.ZeniSymbol = "Z"
	}
	if req.Nickname == "" {
		req.Nickname = "Adventurer"
	}

	dc := gg.NewContext(CARD_W, CARD_H)

	// === Background gradient (dark navy to purple) ===
	grad := gg.NewLinearGradient(0, 0, float64(CARD_W), float64(CARD_H))
	grad.AddColorStop(0, color.RGBA{13, 13, 26, 255})
	grad.AddColorStop(0.5, color.RGBA{20, 20, 50, 255})
	grad.AddColorStop(1, color.RGBA{26, 10, 40, 255})
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, CARD_W, CARD_H)
	dc.Fill()

	// === Subtle grid pattern overlay ===
	dc.SetColor(color.RGBA{255, 255, 255, 8})
	dc.SetLineWidth(0.5)
	for x := 0.0; x < CARD_W; x += 40 {
		dc.DrawLine(x, 0, x, CARD_H)
		dc.Stroke()
	}
	for y := 0.0; y < CARD_H; y += 40 {
		dc.DrawLine(0, y, CARD_W, y)
		dc.Stroke()
	}

	// === Gold border ===
	dc.SetColor(color.RGBA{212, 175, 55, 255})
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(8, 8, CARD_W-16, CARD_H-16, 12)
	dc.Stroke()

	// === Inner accent lines ===
	dc.SetColor(color.RGBA{212, 175, 55, 60})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(14, 14, CARD_W-28, CARD_H-28, 10)
	dc.Stroke()

	// === Decorative corner markers ===
	dc.SetColor(color.RGBA{212, 175, 55, 120})
	// Top-left
	dc.DrawLine(20, 20, 60, 20)
	dc.Stroke()
	dc.DrawLine(20, 20, 20, 60)
	dc.Stroke()
	// Top-right
	dc.DrawLine(float64(CARD_W-20), 20, float64(CARD_W-60), 20)
	dc.Stroke()
	dc.DrawLine(float64(CARD_W-20), 20, float64(CARD_W-20), 60)
	dc.Stroke()
	// Bottom-left
	dc.DrawLine(20, float64(CARD_H-20), 60, float64(CARD_H-20))
	dc.Stroke()
	dc.DrawLine(20, float64(CARD_H-20), 20, float64(CARD_H-60))
	dc.Stroke()
	// Bottom-right
	dc.DrawLine(float64(CARD_W-20), float64(CARD_H-20), float64(CARD_W-60), float64(CARD_H-20))
	dc.Stroke()
	dc.DrawLine(float64(CARD_W-20), float64(CARD_H-20), float64(CARD_W-20), float64(CARD_H-60))
	dc.Stroke()

	// === Font loading ===
	fontPath := filepath.Join("assets", "rpgasset", "ui", "fantesy.ttf")
	titleFace, _ := utils.LoadFont(fontPath, 32)
	largeFace, _ := utils.LoadFont(fontPath, 48)
	medFace, _ := utils.LoadFont(fontPath, 22)
	smallFace, _ := utils.LoadFont(fontPath, 16)

	// === Header: Nickname ===
	if titleFace != nil {
		dc.SetFontFace(titleFace)
	}
	// Gold text with drop shadow
	dc.SetColor(color.RGBA{0, 0, 0, 120})
	dc.DrawString(req.Nickname, 42, 67)
	dc.SetColor(color.RGBA{212, 175, 55, 255})
	dc.DrawString(req.Nickname, 40, 65)

	// === Rank badge ===
	rankColors := map[string]color.RGBA{
		"F":   {100, 100, 100, 255},
		"E":   {100, 149, 237, 255},
		"D":   {50, 205, 50, 255},
		"C":   {255, 215, 0, 255},
		"B":   {255, 140, 0, 255},
		"A":   {255, 50, 50, 255},
		"S":   {220, 20, 220, 255},
		"SS":  {255, 20, 147, 255},
		"SSS": {255, 69, 0, 255},
	}
	rankColor, ok := rankColors[req.Rank]
	if !ok {
		rankColor = color.RGBA{150, 150, 150, 255}
	}

	// Rank pill background
	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(float64(CARD_W-160), 36, 130, 40, 8)
	dc.Fill()

	// Rank text
	rankText := req.Rank + " RANK"
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawStringAnchored(rankText, float64(CARD_W-95), 60, 0.5, 0.5)

	// Level display
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{180, 180, 220, 255})
	lvlText := fmt.Sprintf("LVL %d", req.Level)
	dc.DrawString(lvlText, float64(CARD_W-160), 100)

	// === Divider gradient line ===
	divGrad := gg.NewLinearGradient(40, 0, float64(CARD_W-40), 0)
	divGrad.AddColorStop(0, color.RGBA{212, 175, 55, 0})
	divGrad.AddColorStop(0.3, color.RGBA{212, 175, 55, 200})
	divGrad.AddColorStop(0.7, color.RGBA{212, 175, 55, 200})
	divGrad.AddColorStop(1, color.RGBA{212, 175, 55, 0})
	dc.SetStrokeStyle(divGrad)
	dc.SetLineWidth(1)
	dc.DrawLine(40, 110, float64(CARD_W-40), 110)
	dc.Stroke()

	// === TOTAL wealth (center, prominent) ===
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	totalStr := fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Total))

	// Layered glow effect
	dc.SetColor(color.RGBA{212, 175, 55, 30})
	for i := 3; i >= 1; i-- {
		offset := float64(i * 2)
		dc.DrawStringAnchored(totalStr, float64(CARD_W/2)+offset, 170+offset, 0.5, 0.5)
		dc.DrawStringAnchored(totalStr, float64(CARD_W/2)-offset, 170-offset, 0.5, 0.5)
	}
	dc.SetColor(color.RGBA{255, 215, 0, 255})
	dc.DrawStringAnchored(totalStr, float64(CARD_W/2), 170, 0.5, 0.5)

	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{150, 150, 180, 255})
	dc.DrawStringAnchored("TOTAL WEALTH", float64(CARD_W/2), 196, 0.5, 0.5)

	// === Wallet panel (left) ===
	panelY := 220.0
	panelH := 90.0

	walletGrad := gg.NewLinearGradient(40, panelY, 360, panelY+panelH)
	walletGrad.AddColorStop(0, color.RGBA{30, 60, 30, 200})
	walletGrad.AddColorStop(1, color.RGBA{20, 40, 20, 200})
	dc.SetFillStyle(walletGrad)
	dc.DrawRoundedRectangle(40, panelY, 320, panelH, 10)
	dc.Fill()
	dc.SetColor(color.RGBA{50, 200, 100, 150})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(40, panelY, 320, panelH, 10)
	dc.Stroke()

	// Wallet icon circle
	dc.SetColor(color.RGBA{50, 200, 100, 255})
	dc.DrawCircle(70, panelY+45, 18)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.DrawStringAnchored("W", 70, panelY+52, 0.5, 0.5)

	// Wallet label & amount
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{150, 220, 150, 255})
	dc.DrawString("WALLET", 100, panelY+35)
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	dc.SetColor(color.RGBA{100, 255, 130, 255})
	walletStr := fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Wallet))
	dc.DrawString(walletStr, 100, panelY+78)

	// === Bank panel (right) ===
	bankGrad := gg.NewLinearGradient(440, panelY, 760, panelY+panelH)
	bankGrad.AddColorStop(0, color.RGBA{20, 30, 80, 200})
	bankGrad.AddColorStop(1, color.RGBA{15, 20, 60, 200})
	dc.SetFillStyle(bankGrad)
	dc.DrawRoundedRectangle(440, panelY, 320, panelH, 10)
	dc.Fill()
	dc.SetColor(color.RGBA{80, 120, 255, 150})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(440, panelY, 320, panelH, 10)
	dc.Stroke()

	// Bank icon circle
	dc.SetColor(color.RGBA{80, 120, 255, 255})
	dc.DrawCircle(470, panelY+45, 18)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.DrawStringAnchored("B", 470, panelY+52, 0.5, 0.5)

	// Bank label & amount
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{150, 180, 255, 255})
	dc.DrawString("BANK", 500, panelY+35)
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	dc.SetColor(color.RGBA{130, 160, 255, 255})
	bankStr := fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Bank))
	dc.DrawString(bankStr, 500, panelY+78)

	// === Wealth distribution bar ===
	barY := 330.0
	barW := float64(CARD_W) - 80
	barH := 16.0

	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{150, 150, 180, 255})
	dc.DrawString("DISTRIBUTION", 40, barY-6)

	// Track bar background
	dc.SetColor(color.RGBA{255, 255, 255, 20})
	dc.DrawRoundedRectangle(40, barY, barW, barH, barH/2)
	dc.Fill()

	// Wallet portion (green)
	walletPct := 0.0
	if req.Total > 0 {
		walletPct = req.Wallet / req.Total
	}
	if walletPct > 0 {
		walletBarW := math.Max(barH, walletPct*barW)
		walletBarGrad := gg.NewLinearGradient(40, 0, 40+walletBarW, 0)
		walletBarGrad.AddColorStop(0, color.RGBA{50, 200, 100, 255})
		walletBarGrad.AddColorStop(1, color.RGBA{100, 230, 150, 255})
		dc.SetFillStyle(walletBarGrad)
		dc.DrawRoundedRectangle(40, barY, walletBarW, barH, barH/2)
		dc.Fill()
	}

	// Bank portion (blue), starts where wallet ends
	bankPct := 0.0
	if req.Total > 0 {
		bankPct = req.Bank / req.Total
	}
	if bankPct > 0 && walletPct < 1 {
		bankStartX := 40 + walletPct*barW
		bankBarW := math.Min(bankPct*barW, barW-walletPct*barW)
		bankBarGrad := gg.NewLinearGradient(bankStartX, 0, bankStartX+bankBarW, 0)
		bankBarGrad.AddColorStop(0, color.RGBA{80, 120, 255, 255})
		bankBarGrad.AddColorStop(1, color.RGBA{120, 160, 255, 255})
		dc.SetFillStyle(bankBarGrad)
		dc.DrawRoundedRectangle(bankStartX, barY, bankBarW, barH, barH/2)
		dc.Fill()
	}

	// Bar border
	dc.SetColor(color.RGBA{212, 175, 55, 80})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(40, barY, barW, barH, barH/2)
	dc.Stroke()

	// === Footer stats row ===
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	footerY := 390.0

	// Wallet percentage (left)
	dc.SetColor(color.RGBA{100, 200, 100, 200})
	dc.DrawString(fmt.Sprintf("Wallet %.0f%%", walletPct*100), 40, footerY)

	// Bank percentage (center)
	dc.SetColor(color.RGBA{100, 130, 255, 200})
	dc.DrawStringAnchored(fmt.Sprintf("Bank %.0f%%", bankPct*100), float64(CARD_W/2), footerY, 0.5, 0)

	// Frozen amount (right, only if non-zero)
	if req.Frozen > 0 {
		dc.SetColor(color.RGBA{150, 200, 255, 200})
		dc.DrawStringAnchored(
			fmt.Sprintf("Frozen: %s%s", req.ZeniSymbol, formatNumber(req.Frozen)),
			float64(CARD_W-40), footerY, 1, 0,
		)
	}

	// === Bottom decorative line ===
	dc.SetStrokeStyle(divGrad)
	dc.SetLineWidth(1)
	dc.DrawLine(40, float64(CARD_H-30), float64(CARD_W-40), float64(CARD_H-30))
	dc.Stroke()

	// === Watermark ===
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{212, 175, 55, 60})
	dc.DrawStringAnchored("JOKER BOT", float64(CARD_W/2), float64(CARD_H-15), 0.5, 0.5)

	// === Encode & respond ===
	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

// formatNumber formats a float64 as a compact human-readable string.
func formatNumber(n float64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}

// loadProfilePic attempts to download and crop a profile picture from a URL.
// Returns nil silently on any failure — callers must handle a nil return.
func loadProfilePic(url string) interface{} {
	if url == "" {
		return nil
	}
	img, err := utils.DownloadImage(url)
	if err != nil {
		return nil
	}
	return imaging.Fill(img, 64, 64, imaging.Center, imaging.Lanczos)
}
