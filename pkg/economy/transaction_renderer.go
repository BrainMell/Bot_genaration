package economy

import (
	"fmt"
	"image/color"
	"log"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	TRANS_W = 600
	TRANS_H = 250
)

type TransactionCardRequest struct {
	Nickname   string  `json:"nickname"`
	Type       string  `json:"type"` // "DEPOSIT", "WITHDRAW", "TRANSFER"
	Amount     float64 `json:"amount"`
	NewWallet  float64 `json:"newWallet"`
	NewBank    float64 `json:"newBank"`
	ZeniSymbol string  `json:"zeniSymbol"`
	PfpUrl     string  `json:"pfpUrl"`
}

func GenerateTransactionCard(c *gin.Context) {
	var req TransactionCardRequest
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

	dc := gg.NewContext(TRANS_W, TRANS_H)

	// Draw Background
	dc.SetColor(color.RGBA{15, 16, 23, 255})
	dc.DrawRectangle(0, 0, TRANS_W, TRANS_H)
	dc.Fill()

	var accentColor color.RGBA
	var iconText string
	switch req.Type {
	case "DEPOSIT":
		accentColor = color.RGBA{80, 160, 255, 255} // Blue
		iconText = "▼"
	case "WITHDRAW":
		accentColor = color.RGBA{255, 140, 0, 255} // Orange
		iconText = "▲"
	case "TRANSFER":
		accentColor = color.RGBA{255, 60, 60, 255} // Red
		iconText = "▶"
	default:
		accentColor = color.RGBA{60, 210, 130, 255} // Green
		iconText = "◆"
	}

	// Left accent bar
	dc.SetColor(accentColor)
	dc.DrawRectangle(0, 0, 4, TRANS_H)
	dc.Fill()

	fontBold := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
	fontSemi := utils.GetAssetPath("rpgasset", "ui", "Inter-SemiBold.ttf")
	fontMed := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")

	// === Header ===
	textX := 40.0
	if req.PfpUrl != "" {
		pfpImg, pfpErr := utils.DownloadImage(req.PfpUrl)
		if pfpErr == nil {
			pfpSize := 50
			pfpImg = imaging.Fill(pfpImg, pfpSize, pfpSize, imaging.Center, imaging.Lanczos)
			
			dc.DrawCircle(65, 55, float64(pfpSize)/2)
			dc.Clip()
			dc.DrawImageAnchored(pfpImg, 65, 55, 0.5, 0.5)
			dc.ResetClip()

			dc.SetColor(accentColor)
			dc.DrawCircle(65, 55, float64(pfpSize)/2)
			dc.SetLineWidth(2)
			dc.Stroke()
			
			textX = 105.0
		}
	}

	if err := dc.LoadFontFace(fontBold, 22); err == nil {
		dc.SetColor(color.RGBA{250, 250, 255, 255})
		dc.DrawString(req.Nickname, textX, 48)
	} else {
		log.Printf("failed to load fontBold (%s): %v", fontBold, err)
	}

	if err := dc.LoadFontFace(fontSemi, 12); err == nil {
		dc.SetColor(accentColor)
		dc.DrawString(fmt.Sprintf("%s SUCCESSFUL", req.Type), textX, 68)
	}

	// Separator
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawLine(40, 95, float64(TRANS_W)-40, 95)
	dc.Stroke()

	// === Middle Section ===
	
	// Transaction Amount
	if err := dc.LoadFontFace(fontMed, 14); err == nil {
		dc.SetColor(color.RGBA{200, 200, 220, 255})
		dc.DrawString("TRANSACTION AMOUNT", 40, 130)
	}

	if err := dc.LoadFontFace(fontBold, 36); err == nil {
		dc.SetColor(accentColor)
		dc.DrawString(fmt.Sprintf("%s %s%s", iconText, req.ZeniSymbol, formatNumber(req.Amount)), 40, 175)
	}

	// Balances (Right side)
	if err := dc.LoadFontFace(fontMed, 12); err == nil {
		dc.SetColor(color.RGBA{200, 200, 220, 255})
		dc.DrawString("NEW WALLET", 350, 130)
		dc.DrawString("NEW BANK", 470, 130)
	}

	if err := dc.LoadFontFace(fontSemi, 20); err == nil {
		dc.SetColor(color.RGBA{60, 210, 130, 255}) // Wallet green
		dc.DrawString(fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.NewWallet)), 350, 160)

		dc.SetColor(color.RGBA{80, 160, 255, 255}) // Bank blue
		dc.DrawString(fmt.Sprintf("%s%s", req.ZeniSymbol, formatNumber(req.NewBank)), 470, 160)
	}

	// Watermark
	if err := dc.LoadFontFace(fontSemi, 10); err == nil {
		dc.SetColor(color.RGBA{255, 255, 255, 60})
		dc.DrawStringAnchored("JOKER BOT • TRANSACTION", float64(TRANS_W)/2, float64(TRANS_H)-15, 0.5, 0.5)
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}
