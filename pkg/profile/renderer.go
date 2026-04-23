package profile

import (
	"fmt"
	"image/color"
	"math"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	PROF_W = 500
	PROF_H = 272
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

	// Flat dark background
	dc.SetColor(color.RGBA{15, 15, 26, 255})
	dc.DrawRectangle(0, 0, PROF_W, PROF_H)
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
	dc.DrawRectangle(0, 0, 3, PROF_H)
	dc.Fill()

	fontPath := utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
	largeFace, err1 := utils.LoadFont(fontPath, 18)
	medFace, err2 := utils.LoadFont(fontPath, 16)
	smallFace, err3 := utils.LoadFont(fontPath, 10)

	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Printf("Font Load Error: %v, %v, %v\nPath: %s\n", err1, err2, err3, fontPath)
		c.JSON(500, gin.H{"error": "Font loading failed. Ensure assets are correctly placed."})
		return
	}

	// === Avatar ===
	const avCX, avCY, avR = 58.0, 65.0, 38.0
	pfpLoaded := false
	if req.PfpUrl != "" {
		if img, err := utils.DownloadImage(req.PfpUrl); err == nil {
			img = imaging.Fill(img, int(avR*2), int(avR*2), imaging.Center, imaging.Lanczos)
			dc.Push()
			dc.DrawCircle(avCX, avCY, avR)
			dc.Clip()
			dc.DrawImage(img, int(avCX-avR), int(avCY-avR))
			dc.ResetClip()
			dc.Pop()
			pfpLoaded = true
		}
	}
	if !pfpLoaded {
		dc.SetColor(color.RGBA{rankColor.R / 4, rankColor.G / 4, rankColor.B / 4, 255})
		dc.DrawCircle(avCX, avCY, avR)
		dc.Fill()
		if largeFace != nil {
			dc.SetFontFace(largeFace)
		}
		dc.SetColor(color.RGBA{220, 220, 230, 200})
		dc.DrawStringAnchored(string([]rune(req.Nickname)[0:1]), avCX, avCY+2, 0.5, 0.5)
	}
	dc.SetColor(rankColor)
	dc.SetLineWidth(2)
	dc.DrawCircle(avCX, avCY, avR)
	dc.Stroke()

	// === Name block ===
	const nameX = 112.0
	if medFace != nil {
		dc.SetFontFace(medFace)
	}
	dc.SetColor(color.RGBA{240, 240, 250, 255})
	dc.DrawString(req.Nickname, nameX, 44)

	if req.Title != "" {
		if smallFace != nil {
			dc.SetFontFace(smallFace)
		}
		dc.SetColor(color.RGBA{212, 175, 55, 210})
		dc.DrawString("✦ "+req.Title, nameX, 59)
	}

	// Class badge
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{30, 30, 58, 255})
	dc.DrawRoundedRectangle(nameX, 64, 90, 18, 3)
	dc.Fill()
	dc.SetColor(color.RGBA{83, 74, 183, 180})
	dc.SetLineWidth(0.8)
	dc.DrawRoundedRectangle(nameX, 64, 90, 18, 3)
	dc.Stroke()
	dc.SetColor(color.RGBA{175, 169, 236, 255})
	dc.DrawStringAnchored(req.ClassIcon+" "+req.Class, nameX+45, 75, 0.5, 0.5)

	// Rank badge
	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(nameX+96, 64, 64, 18, 3)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 240})
	dc.DrawStringAnchored(req.Rank+" RANK", nameX+128, 75, 0.5, 0.5)

	// Info lines (aka / guild / level+GP)
	infoY := 96.0
	if req.WhatsappName != "" && req.WhatsappName != req.Nickname {
		dc.SetColor(color.RGBA{120, 120, 150, 180})
		dc.DrawString("aka "+req.WhatsappName, nameX, infoY)
		infoY += 13
	}
	if req.GuildName != "" {
		dc.SetColor(color.RGBA{212, 175, 55, 180})
		dc.DrawString("🏰 "+req.GuildName, nameX, infoY)
		infoY += 13
	}
	dc.SetColor(color.RGBA{212, 175, 55, 200})
	dc.DrawString(fmt.Sprintf("LVL %d  ·  ⭐ %s GP", req.Level, formatNum(req.GP)), nameX, infoY)

	// === Divider 1 ===
	profDivider(dc, 124)

	// === XP bar ===
	const xpBarX, xpBarY, xpBarW, xpBarH = 44.0, 133.0, 440.0, 9.0
	dc.SetColor(color.RGBA{110, 110, 150, 200})
	dc.DrawString("XP", 16, xpBarY+8)

	dc.SetColor(color.RGBA{255, 255, 255, 12})
	dc.DrawRoundedRectangle(xpBarX, xpBarY, xpBarW, xpBarH, xpBarH/2)
	dc.Fill()

	xpPct := 0.0
	if req.XPNeeded > 0 {
		xpPct = math.Min(1.0, float64(req.XP)/float64(req.XPNeeded))
	}
	if xpPct > 0 {
		fw := math.Max(xpBarH, xpPct*xpBarW)
		xpGrad := gg.NewLinearGradient(xpBarX, 0, xpBarX+fw, 0)
		xpGrad.AddColorStop(0, color.RGBA{100, 100, 255, 255})
		xpGrad.AddColorStop(1, color.RGBA{180, 100, 255, 255})
		dc.SetFillStyle(xpGrad)
		dc.DrawRoundedRectangle(xpBarX, xpBarY, fw, xpBarH, xpBarH/2)
		dc.Fill()
	}
	dc.SetColor(color.RGBA{100, 90, 140, 100})
	dc.SetLineWidth(0.5)
	dc.DrawRoundedRectangle(xpBarX, xpBarY, xpBarW, xpBarH, xpBarH/2)
	dc.Stroke()

	dc.SetColor(color.RGBA{110, 110, 150, 200})
	dc.DrawString(fmt.Sprintf("%d / %d XP  (%.0f%%)", req.XP, req.XPNeeded, xpPct*100), 16, xpBarY+xpBarH+14)

	// === Divider 2 ===
	profDivider(dc, 162)

	// === Stats 3×2 grid ===
	type stat struct {
		label, value string
		col          color.RGBA
	}
	rows := [2][3]stat{
		{
			{"QUESTS", fmt.Sprintf("%d", req.QuestsWon), color.RGBA{80, 220, 130, 255}},
			{"GAMES WON", fmt.Sprintf("%d", req.GamesWon), color.RGBA{100, 160, 255, 255}},
			{"MESSAGES", formatNum(req.MessageCount), color.RGBA{200, 160, 255, 255}},
		},
		{
			{"WALLET", fmt.Sprintf("%s%s", req.ZeniSymbol, formatNum(int(req.Wallet))), color.RGBA{80, 220, 130, 255}},
			{"BANK", fmt.Sprintf("%s%s", req.ZeniSymbol, formatNum(int(req.Bank))), color.RGBA{130, 160, 255, 255}},
			{"GP", fmt.Sprintf("⭐%s", formatNum(req.GP)), color.RGBA{255, 215, 0, 255}},
		},
	}
	colCX := [3]float64{83, 250, 417}
	rowValY := [2]float64{182, 226}
	rowLblY := [2]float64{196, 240}

	for r, row := range rows {
		for c2, s := range row {
			cx := colCX[c2]
			if largeFace != nil {
				dc.SetFontFace(largeFace)
			}
			dc.SetColor(s.col)
			dc.DrawStringAnchored(s.value, cx, rowValY[r], 0.5, 0.5)
			if smallFace != nil {
				dc.SetFontFace(smallFace)
			}
			dc.SetColor(color.RGBA{100, 100, 130, 200})
			dc.DrawStringAnchored(s.label, cx, rowLblY[r], 0.5, 0.5)
		}
	}

	// Vertical dividers
	dc.SetColor(color.RGBA{42, 42, 74, 255})
	dc.SetLineWidth(1)
	for _, vx := range [2]float64{166, 333} {
		dc.DrawLine(vx, 162, vx, 252)
		dc.Stroke()
	}
	// Horizontal mid divider
	dc.DrawLine(16, 208, 484, 208)
	dc.Stroke()

	// === Footer ===
	profDivider(dc, 252)
	if smallFace != nil {
		dc.SetFontFace(smallFace)
	}
	dc.SetColor(color.RGBA{50, 50, 80, 255})
	dc.DrawStringAnchored("JOKER BOT", PROF_W/2, 265, 0.5, 0.5)

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func profDivider(dc *gg.Context, y float64) {
	dc.SetColor(color.RGBA{42, 42, 74, 255})
	dc.SetLineWidth(1)
	dc.DrawLine(16, y, 484, y)
	dc.Stroke()
}

func formatNum(n int) string {
	f := float64(n)
	switch {
	case f >= 1_000_000:
		return fmt.Sprintf("%.1fM", f/1_000_000)
	case f >= 1_000:
		return fmt.Sprintf("%.1fK", f/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
