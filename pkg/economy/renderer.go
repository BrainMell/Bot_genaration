package economy

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	CARD_W = 500
	CARD_H = 258
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

	// Flat dark background
	dc.SetColor(color.RGBA{15, 15, 26, 255})
	dc.DrawRectangle(0, 0, CARD_W, CARD_H)
	dc.Fill()

	rankColors := map[string]color.RGBA{
		"F": {100, 100, 120, 255}, "E": {100, 149, 237, 255},
		"D": {50, 205, 50, 255}, "C": {255, 215, 0, 255},
		"B": {255, 140, 0, 255}, "A": {255, 60, 60, 255},
		"S": {220, 20, 220, 255}, "SS": {255, 20, 147, 255},
		"SSS": {255, 69, 0, 255},
	}
	rankColor, ok := rankColors[req.Rank]
	if !ok {
		rankColor = color.RGBA{150, 150, 180, 255}
	}

	// Left accent bar
	dc.SetColor(rankColor)
	dc.DrawRectangle(0, 0, 3, CARD_H)
	dc.Fill()

	fontPath := filepath.Join("assets", "rpgasset", "ui", "fantesy.ttf")
	totalFace, _ := utils.LoadFont(fontPath, 26)
	largeFace, _ := utils.LoadFont(fontPath, 18)
	medFace, _ := utils.LoadFont(fontPath, 15)
	smallFace, _ := utils.LoadFont(fontPath, 10)

	// === Header ===
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{240, 240, 250, 255})
	dc.DrawString(req.Nickname, 22, 38)

	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(338, 20, 66, 18, 3)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 240})
	dc.DrawStringAnchored(req.Rank+" RANK", 371, 31, 0.5, 0.5)

	dc.SetColor(color.RGBA{120, 120, 155, 200})
	dc.DrawString(fmt.Sprintf("· LVL %d", req.Level), 410, 32)

	// === Divider 1 ===
	econDivider(dc, 50)

	// === Total wealth ===
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{100, 100, 130, 200})
	dc.DrawStringAnchored("TOTAL WEALTH", CARD_W/2, 68, 0.5, 0.5)

	if totalFace != nil {
		dc.SetFontFace(totalFace)
	}
	dc.SetColor(color.RGBA{250, 199, 117, 255})
	dc.DrawStringAnchored(
		fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Total)),
		CARD_W/2, 98, 0.5, 0.5,
	)

	// === Divider 2 ===
	econDivider(dc, 114)

	// === Wallet ===
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	dc.SetColor(color.RGBA{80, 220, 140, 255})
	dc.DrawStringAnchored(
		fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Wallet)),
		125, 146, 0.5, 0.5,
	)
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{90, 90, 120, 200})
	dc.DrawStringAnchored("WALLET", 125, 162, 0.5, 0.5)

	// Column divider
	dc.SetColor(color.RGBA{42, 42, 74, 255})
	dc.SetLineWidth(1)
	dc.DrawLine(250, 122, 250, 173)
	dc.Stroke()

	// === Bank ===
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	dc.SetColor(color.RGBA{130, 160, 255, 255})
	dc.DrawStringAnchored(
		fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.Bank)),
		375, 146, 0.5, 0.5,
	)
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{90, 90, 120, 200})
	dc.DrawStringAnchored("BANK", 375, 162, 0.5, 0.5)

	// === Divider 3 ===
	econDivider(dc, 178)

	// === Distribution bar ===
	dc.SetColor(color.RGBA{100, 100, 130, 200})
	dc.DrawString("DISTRIBUTION", 16, 196)

	const barX, barY, barW, barH = 16.0, 201.0, 468.0, 10.0

	dc.SetColor(color.RGBA{255, 255, 255, 12})
	dc.DrawRoundedRectangle(barX, barY, barW, barH, barH/2)
	dc.Fill()

	walletPct, bankPct := 0.0, 0.0
	if req.Total > 0 {
		walletPct = math.Min(1, req.Wallet/req.Total)
		bankPct = math.Min(1-walletPct, req.Bank/req.Total)
	}
	if walletPct > 0 {
		fw := math.Max(barH, walletPct*barW)
		dc.SetColor(color.RGBA{30, 158, 117, 255})
		dc.DrawRoundedRectangle(barX, barY, fw, barH, barH/2)
		dc.Fill()
	}
	if bankPct > 0 {
		startX := barX + walletPct*barW
		fw := math.Max(0, bankPct*barW)
		dc.SetColor(color.RGBA{55, 138, 221, 255})
		dc.DrawRoundedRectangle(startX, barY, fw, barH, barH/2)
		dc.Fill()
	}

	// Percentage labels
	dc.SetColor(color.RGBA{80, 220, 140, 200})
	dc.DrawString(fmt.Sprintf("Wallet %.0f%%", walletPct*100), 16, 226)
	dc.SetColor(color.RGBA{130, 160, 255, 200})
	dc.DrawStringAnchored(fmt.Sprintf("Bank %.0f%%", bankPct*100), CARD_W/2, 226, 0.5, 0)
	if req.Frozen > 0 {
		dc.SetColor(color.RGBA{160, 190, 255, 200})
		dc.DrawStringAnchored(
			fmt.Sprintf("Frozen: %s%s", req.ZeniSymbol, formatNumber(req.Frozen)),
			CARD_W-16, 226, 1, 0,
		)
	}

	// === Divider 4 ===
	econDivider(dc, 236)

	// === Watermark ===
	dc.SetColor(color.RGBA{50, 50, 80, 255})
	dc.DrawStringAnchored("JOKER BOT", CARD_W/2, 250, 0.5, 0.5)

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func econDivider(dc *gg.Context, y float64) {
	dc.SetColor(color.RGBA{42, 42, 74, 255})
	dc.SetLineWidth(1)
	dc.DrawLine(16, y, 484, y)
	dc.Stroke()
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
