package economy

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	CARD_W = 600
	CARD_H = 300
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
	PfpUrl     string  `json:"pfpUrl"`
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

	// Draw Background
	// Draw solid background
	dc.SetColor(color.RGBA{15, 16, 23, 255})
	dc.DrawRectangle(0, 0, CARD_W, CARD_H)
	dc.Fill()

	rankColors := map[string]color.RGBA{
		"F": {100, 100, 120, 255}, "E": {100, 149, 237, 255},
		"D": {50, 205, 50, 255}, "C": {255, 215, 0, 255},
		"B": {255, 140, 0, 255}, "A": {255, 60, 60, 255},
		"S": {250, 199, 117, 255}, "SS": {255, 20, 147, 255},
		"SSS": {255, 69, 0, 255},
	}
	rankColor, ok := rankColors[req.Rank]
	if !ok {
		rankColor = color.RGBA{250, 199, 117, 255}
	}

	// Left accent bar
	dc.SetColor(rankColor)
	dc.DrawRectangle(0, 0, 4, CARD_H)
	dc.Fill()

	fontBold := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
	fontSemi := utils.GetAssetPath("rpgasset", "ui", "Inter-SemiBold.ttf")
	fontMed := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")

	// === Header ===
	textX := 40.0
	// If PfpUrl is provided, fetch and draw it as a circle
	if req.PfpUrl != "" {
		pfpImg, pfpErr := utils.DownloadImage(req.PfpUrl)
		if pfpErr == nil {
			pfpSize := 60
			pfpImg = imaging.Fill(pfpImg, pfpSize, pfpSize, imaging.Center, imaging.Lanczos)
			
			// Draw circular PFP
			dc.DrawCircle(70, 65, float64(pfpSize)/2)
			dc.Clip()
			dc.DrawImageAnchored(pfpImg, 70, 65, 0.5, 0.5)
			dc.ResetClip()

			// Add a subtle border to PFP
			dc.SetColor(rankColor)
			dc.DrawCircle(70, 65, float64(pfpSize)/2)
			dc.SetLineWidth(2)
			dc.Stroke()
			
			textX = 115.0
		}
	}

	if err := dc.LoadFontFace(fontBold, 26); err == nil {
		dc.SetColor(color.RGBA{250, 250, 255, 255})
		dc.DrawString(req.Nickname, textX, 55)
	} else {
		log.Printf("failed to load fontBold (%s): %v", fontBold, err)
	}

	if err := dc.LoadFontFace(fontMed, 14); err == nil {
		dc.SetColor(color.RGBA{200, 200, 220, 255})
		dc.DrawString(fmt.Sprintf("LVL %d", req.Level), textX, 80)
	} else {
		log.Printf("failed to load fontMed (%s): %v", fontMed, err)
	}

	// Rank Badge (Top Right)
	badgeW := 80.0
	badgeH := 26.0
	badgeX := float64(CARD_W) - 40 - badgeW
	badgeY := 45.0

	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(badgeX, badgeY, badgeW, badgeH, badgeH/2)
	dc.Fill()

	if err := dc.LoadFontFace(fontBold, 13); err == nil {
		dc.SetColor(color.RGBA{20, 20, 30, 255})
		dc.DrawStringAnchored(req.Rank+" RANK", badgeX+(badgeW/2), badgeY+(badgeH/2)-1, 0.5, 0.5)
	}

	// Separator
	dc.SetColor(color.RGBA{255, 255, 255, 30})
	dc.DrawLine(40, 115, float64(CARD_W)-40, 115)
	dc.Stroke()

	// === Middle Section (Wealth Dashboard) ===
	if err := dc.LoadFontFace(fontSemi, 11); err == nil {
		dc.SetColor(color.RGBA{200, 200, 220, 255})
		dc.DrawString("TOTAL WEALTH", 40, 145)
		dc.DrawString("WALLET", 280, 145)
		dc.DrawString("BANK", 420, 145)
	}

	// Total Wealth
	if err := dc.LoadFontFace(fontBold, 32); err == nil {
		dc.SetColor(rankColor)
		dc.DrawString(fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Total)), 40, 185)
	}

	// Wallet & Bank
	if err := dc.LoadFontFace(fontSemi, 24); err == nil {
		dc.SetColor(color.RGBA{60, 210, 130, 255})
		dc.DrawString(fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Wallet)), 280, 182)

		dc.SetColor(color.RGBA{80, 160, 255, 255})
		dc.DrawString(fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Bank)), 420, 182)
	}

	// === Bottom Section (Distribution Bar) ===
	barX := 40.0
	barY := 225.0
	barW := float64(CARD_W) - 80.0
	barH := 12.0

	dc.SetColor(color.RGBA{255, 255, 255, 20})
	dc.DrawRoundedRectangle(barX, barY, barW, barH, barH/2)
	dc.Fill()

	walletPct, bankPct := 0.0, 0.0
	if req.Total > 0 {
		walletPct = math.Min(1, req.Wallet/req.Total)
		bankPct = math.Min(1-walletPct, req.Bank/req.Total)
	}

	if walletPct > 0 {
		w := math.Max(barH, barW*walletPct)
		dc.SetColor(color.RGBA{60, 210, 130, 255})
		dc.DrawRoundedRectangle(barX, barY, w, barH, barH/2)
		dc.Fill()
	}

	if bankPct > 0 {
		startX := barX + (barW * walletPct)
		w := math.Max(barH, barW*bankPct)
		dc.SetColor(color.RGBA{80, 160, 255, 255})
		dc.DrawRoundedRectangle(startX, barY, w, barH, barH/2)
		dc.Fill()
	}

	if err := dc.LoadFontFace(fontMed, 12); err == nil {
		dc.SetColor(color.RGBA{60, 210, 130, 255})
		dc.DrawString(fmt.Sprintf("Wallet %.0f%%", walletPct*100), 40, 255)

		dc.SetColor(color.RGBA{80, 160, 255, 255})
		dc.DrawStringAnchored(fmt.Sprintf("Bank %.0f%%", bankPct*100), float64(CARD_W)-40, 255, 1.0, 0)
	}

	// Watermark
	if err := dc.LoadFontFace(fontSemi, 10); err == nil {
		dc.SetColor(color.RGBA{255, 255, 255, 60})
		dc.DrawStringAnchored("JOKER BOT", float64(CARD_W)/2, float64(CARD_H)-20, 0.5, 0.5)
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func formatNumber(n float64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", n/1_000)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}
