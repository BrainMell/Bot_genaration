package profile

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
	PROF_W = 800
	PROF_H = 500
)

type ProfileCardRequest struct {
	Nickname     string  `json:"nickname"`
	WhatsappName string  `json:"whatsappName"`
	Level        int     `json:"level"`
	XP           int     `json:"xp"`
	XPNeeded     int     `json:"xpNeeded"`
	GP           int     `json:"gp"`
	Rank         string  `json:"rank"`
	Class        string  `json:"class"`
	ClassIcon    string  `json:"classIcon"`
	GuildName    string  `json:"guildName"`
	Wallet       float64 `json:"wallet"`
	Bank         float64 `json:"bank"`
	ZeniSymbol   string  `json:"zeniSymbol"`
	QuestsWon    int     `json:"questsWon"`
	GamesWon     int     `json:"gamesWon"`
	MessageCount int     `json:"messageCount"`
	PfpUrl       string  `json:"pfpUrl"`
	Title        string  `json:"title"`
}

func GenerateProfileCard(c *gin.Context) {
	var req ProfileCardRequest
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
	if req.Class == "" {
		req.Class = "Fighter"
	}
	if req.Rank == "" {
		req.Rank = "F"
	}

	dc := gg.NewContext(PROF_W, PROF_H)

	// === Deep dark background ===
	grad := gg.NewLinearGradient(0, 0, float64(PROF_W), float64(PROF_H))
	grad.AddColorStop(0, color.RGBA{10, 10, 25, 255})
	grad.AddColorStop(0.6, color.RGBA{15, 15, 40, 255})
	grad.AddColorStop(1, color.RGBA{20, 10, 35, 255})
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, PROF_W, PROF_H)
	dc.Fill()

	// === Left accent strip ===
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

	// Left colored bar
	dc.SetColor(rankColor)
	dc.DrawRectangle(0, 0, 6, PROF_H)
	dc.Fill()

	// Gradient fade from left bar
	leftGrad := gg.NewLinearGradient(0, 0, 120, 0)
	leftGrad.AddColorStop(0, color.RGBA{rankColor.R, rankColor.G, rankColor.B, 60})
	leftGrad.AddColorStop(1, color.RGBA{0, 0, 0, 0})
	dc.SetFillStyle(leftGrad)
	dc.DrawRectangle(6, 0, 120, PROF_H)
	dc.Fill()

	// === Gold border ===
	dc.SetColor(color.RGBA{212, 175, 55, 180})
	dc.SetLineWidth(1.5)
	dc.DrawRoundedRectangle(8, 8, PROF_W-16, PROF_H-16, 8)
	dc.Stroke()

	// === Font loading ===
	fontPath := filepath.Join("assets", "rpgasset", "ui", "fantesy.ttf")
	titleFace, _ := utils.LoadFont(fontPath, 36)
	largeFace, _ := utils.LoadFont(fontPath, 52)
	medFace, _ := utils.LoadFont(fontPath, 20)
	smallFace, _ := utils.LoadFont(fontPath, 14)

	// === Profile picture (left side) ===
	pfpX, pfpY, pfpSize := 30.0, 30.0, 120.0
	pfpLoaded := false

	if req.PfpUrl != "" {
		if pfpImg, err := utils.DownloadImage(req.PfpUrl); err == nil {
			pfpImg = imaging.Fill(pfpImg, int(pfpSize), int(pfpSize), imaging.Center, imaging.Lanczos)
			// Clip to circle by drawing in a circular mask area
			dc.Push()
			dc.DrawCircle(pfpX+pfpSize/2, pfpY+pfpSize/2, pfpSize/2)
			dc.Clip()
			dc.DrawImage(pfpImg, int(pfpX), int(pfpY))
			dc.ResetClip()
			dc.Pop()
			// Circle border
			dc.SetColor(rankColor)
			dc.SetLineWidth(3)
			dc.DrawCircle(pfpX+pfpSize/2, pfpY+pfpSize/2, pfpSize/2)
			dc.Stroke()
			pfpLoaded = true
		}
	}

	if !pfpLoaded {
		// Default circle avatar
		dc.SetColor(color.RGBA{rankColor.R / 3, rankColor.G / 3, rankColor.B / 3, 255})
		dc.DrawCircle(pfpX+pfpSize/2, pfpY+pfpSize/2, pfpSize/2)
		dc.Fill()
		dc.SetColor(rankColor)
		dc.SetLineWidth(3)
		dc.DrawCircle(pfpX+pfpSize/2, pfpY+pfpSize/2, pfpSize/2)
		dc.Stroke()
		// Initial letter
		if titleFace != nil {
			dc.SetFontFace(titleFace)
		}
		dc.SetColor(color.RGBA{255, 255, 255, 200})
		initial := string([]rune(req.Nickname)[0:1])
		dc.DrawStringAnchored(initial, pfpX+pfpSize/2, pfpY+pfpSize/2+2, 0.5, 0.5)
	}

	// === Name + Title ===
	nameX := pfpX + pfpSize + 20
	if titleFace != nil {
		dc.SetFontFace(titleFace)
	}
	// Shadow
	dc.SetColor(color.RGBA{0, 0, 0, 150})
	dc.DrawString(req.Nickname, nameX+2, 62)
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(req.Nickname, nameX, 60)

	if req.Title != "" {
		if medFace != nil {
			dc.SetFontFace(medFace)
		}
		dc.SetColor(color.RGBA{212, 175, 55, 200})
		dc.DrawString("✦ "+req.Title, nameX, 86)
	}

	// Class + Rank badges
	badgeY := 100.0
	if req.WhatsappName != "" && req.WhatsappName != req.Nickname {
		if smallFace != nil {
			dc.SetFontFace(smallFace)
		}
		dc.SetColor(color.RGBA{150, 150, 180, 180})
		dc.DrawString("aka "+req.WhatsappName, nameX, badgeY)
		badgeY += 18
	}

	// Class badge
	dc.SetColor(color.RGBA{40, 40, 80, 220})
	dc.DrawRoundedRectangle(nameX, badgeY, 140, 28, 6)
	dc.Fill()
	dc.SetColor(color.RGBA{150, 180, 255, 255})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(nameX, badgeY, 140, 28, 6)
	dc.Stroke()
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{200, 220, 255, 255})
	classText := req.ClassIcon + " " + req.Class
	dc.DrawStringAnchored(classText, nameX+70, badgeY+16, 0.5, 0.5)

	// Rank badge
	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(nameX+150, badgeY, 70, 28, 6)
	dc.Fill()
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawStringAnchored(req.Rank+" RANK", nameX+185, badgeY+16, 0.5, 0.5)

	// Guild
	if req.GuildName != "" {
		if smallFace != nil {
			dc.SetFontFace(smallFace)
		}
		dc.SetColor(color.RGBA{212, 175, 55, 180})
		dc.DrawString("🏰 "+req.GuildName, nameX, badgeY+38)
	}

	// === Divider ===
	divGrad := gg.NewLinearGradient(20, 0, float64(PROF_W-20), 0)
	divGrad.AddColorStop(0, color.RGBA{212, 175, 55, 0})
	divGrad.AddColorStop(0.2, color.RGBA{212, 175, 55, 150})
	divGrad.AddColorStop(0.8, color.RGBA{212, 175, 55, 150})
	divGrad.AddColorStop(1, color.RGBA{212, 175, 55, 0})
	dc.SetStrokeStyle(divGrad)
	dc.SetLineWidth(1)
	dc.DrawLine(20, 170, float64(PROF_W-20), 170)
	dc.Stroke()

	// === Level section ===
	levelY := 190.0
	if largeFace != nil {
		dc.SetFontFace(largeFace)
	}
	dc.SetColor(color.RGBA{255, 215, 0, 255})
	levelStr := fmt.Sprintf("%d", req.Level)
	dc.DrawStringAnchored(levelStr, 80, levelY+40, 0.5, 0.5)
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{150, 150, 180, 255})
	dc.DrawStringAnchored("LEVEL", 80, levelY+65, 0.5, 0.5)

	// === XP Bar ===
	xpBarX := 150.0
	xpBarW := float64(PROF_W) - xpBarX - 30
	xpBarY := levelY + 15
	xpBarH := 18.0

	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{180, 180, 220, 255})
	dc.DrawString("XP PROGRESS", xpBarX, xpBarY-3)

	// XP bar background
	dc.SetColor(color.RGBA{255, 255, 255, 15})
	dc.DrawRoundedRectangle(xpBarX, xpBarY, xpBarW, xpBarH, xpBarH/2)
	dc.Fill()

	xpPct := 0.0
	if req.XPNeeded > 0 {
		xpPct = math.Min(1.0, float64(req.XP)/float64(req.XPNeeded))
	}
	if xpPct > 0 {
		xpFillW := math.Max(xpBarH, xpPct*xpBarW)
		xpGrad := gg.NewLinearGradient(xpBarX, 0, xpBarX+xpFillW, 0)
		xpGrad.AddColorStop(0, color.RGBA{100, 100, 255, 255})
		xpGrad.AddColorStop(1, color.RGBA{180, 100, 255, 255})
		dc.SetFillStyle(xpGrad)
		dc.DrawRoundedRectangle(xpBarX, xpBarY, xpFillW, xpBarH, xpBarH/2)
		dc.Fill()
	}

	dc.SetColor(color.RGBA{212, 175, 55, 80})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(xpBarX, xpBarY, xpBarW, xpBarH, xpBarH/2)
	dc.Stroke()

	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{180, 180, 220, 200})
	xpText := fmt.Sprintf("%d / %d XP  (%.0f%%)", req.XP, req.XPNeeded, xpPct*100)
	dc.DrawString(xpText, xpBarX, xpBarY+xpBarH+14)

	// GP display
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{255, 215, 0, 200})
	gpStr := fmt.Sprintf("⭐ %s GP", formatNum(req.GP))
	dc.DrawString(gpStr, xpBarX+xpBarW-100, xpBarY+xpBarH+14)

	// === Stats grid ===
	statsY := 285.0
	dc.SetStrokeStyle(divGrad)
	dc.SetLineWidth(1)
	dc.DrawLine(20, statsY-10, float64(PROF_W-20), statsY-10)
	dc.Stroke()

	statCols := []struct {
		label, value string
		col          color.RGBA
	}{
		{"QUESTS", fmt.Sprintf("%d", req.QuestsWon), color.RGBA{100, 200, 100, 255}},
		{"GAMES WON", fmt.Sprintf("%d", req.GamesWon), color.RGBA{100, 160, 255, 255}},
		{"MESSAGES", fmt.Sprintf("%s", formatNum(req.MessageCount)), color.RGBA{200, 150, 255, 255}},
		{"WALLET", fmt.Sprintf("%s%s", req.ZeniSymbol, formatNum(int(req.Wallet))), color.RGBA{100, 220, 130, 255}},
		{"BANK", fmt.Sprintf("%s%s", req.ZeniSymbol, formatNum(int(req.Bank))), color.RGBA{130, 150, 255, 255}},
	}

	colW := float64(PROF_W-60) / float64(len(statCols))
	for i, stat := range statCols {
		cx := 30 + float64(i)*colW + colW/2

		// Panel bg
		panelGrad := gg.NewLinearGradient(cx-colW/2+5, statsY, cx+colW/2-5, statsY+75)
		panelGrad.AddColorStop(0, color.RGBA{20, 20, 50, 180})
		panelGrad.AddColorStop(1, color.RGBA{15, 15, 40, 180})
		dc.SetFillStyle(panelGrad)
		dc.DrawRoundedRectangle(cx-colW/2+5, statsY, colW-10, 75, 8)
		dc.Fill()
		dc.SetColor(color.RGBA{stat.col.R, stat.col.G, stat.col.B, 80})
		dc.SetLineWidth(1)
		dc.DrawRoundedRectangle(cx-colW/2+5, statsY, colW-10, 75, 8)
		dc.Stroke()

		// Value (large)
		if medFace != nil {
			dc.SetFontFace(medFace)
		}
		dc.SetColor(stat.col)
		dc.DrawStringAnchored(stat.value, cx, statsY+38, 0.5, 0.5)

		// Label (small)
		if smallFace != nil {
			dc.SetFontFace(smallFace)
		}
		dc.SetColor(color.RGBA{150, 150, 180, 200})
		dc.DrawStringAnchored(stat.label, cx, statsY+60, 0.5, 0.5)
	}

	// === Footer ===
	dc.SetStrokeStyle(divGrad)
	dc.SetLineWidth(1)
	dc.DrawLine(20, float64(PROF_H-35), float64(PROF_W-35), float64(PROF_H-35))
	dc.Stroke()

	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{212, 175, 55, 60})
	dc.DrawStringAnchored("JOKER BOT", float64(PROF_W/2), float64(PROF_H-18), 0.5, 0.5)

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func formatNum(n int) string {
	f := float64(n)
	if f >= 1_000_000 {
		return fmt.Sprintf("%.1fM", f/1_000_000)
	}
	if f >= 1_000 {
		return fmt.Sprintf("%.1fK", f/1_000)
	}
	return fmt.Sprintf("%d", n)
}
