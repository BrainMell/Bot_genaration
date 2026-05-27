package profile

import (
	"fmt"
	"image/color"
	"math"
	"net/url"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
)



const (
	PROF_W = 800
	PROF_H = 460
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

	// RPG Stats
	HP      float64 `json:"hp"`
	ATK     float64 `json:"atk"`
	DEF     float64 `json:"def"`
	MAG     float64 `json:"mag"`
	SPD     float64 `json:"spd"`
	Luck    float64 `json:"luck"`
	Crit    float64 `json:"crit"`
	Evasion float64 `json:"evasion"`

	// Gear Stats
	EquipHP   float64 `json:"equipHp"`
	EquipATK  float64 `json:"equipAtk"`
	EquipDEF  float64 `json:"equipDef"`
	EquipMAG  float64 `json:"equipMag"`
	EquipSPD  float64 `json:"equipSpd"`
	EquipLuck float64 `json:"equipLuck"`

	// Gear Item Names
	GearMainHand string `json:"gearMainHand"`
	GearOffHand  string `json:"gearOffHand"`
	GearArmor    string `json:"gearArmor"`
	GearHelmet   string `json:"gearHelmet"`
	GearBoots    string `json:"gearBoots"`
	GearRing     string `json:"gearRing"`
	GearAmulet   string `json:"gearAmulet"`
	GearCloak    string `json:"gearCloak"`
	GearGloves   string `json:"gearGloves"`
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

	// Rank Color Themes
	rankColors := map[string]color.RGBA{
		"F":   {140, 140, 160, 255}, // Gray/Stone
		"E":   {100, 180, 255, 255}, // Light Blue/Sky
		"D":   {60, 220, 130, 255},  // Green/Emerald
		"C":   {255, 220, 80, 255},  // Gold/Yellow
		"B":   {255, 140, 0, 255},   // Orange/Fiery
		"A":   {255, 70, 70, 255},    // Crimson/Red
		"S":   {180, 100, 255, 255},  // Royal Purple/Amethyst
		"SS":  {255, 60, 180, 255},   // Deep Pink/Rose
		"SSS": {255, 50, 50, 255},    // Hellfire/Dragon Red
	}
	rankSecColors := map[string]color.RGBA{
		"F":   {180, 180, 200, 255},
		"E":   {160, 210, 255, 255},
		"D":   {120, 245, 170, 255},
		"C":   {255, 240, 150, 255},
		"B":   {255, 190, 80, 255},
		"A":   {255, 130, 130, 255},
		"S":   {220, 160, 255, 255},
		"SS":  {255, 140, 210, 255},
		"SSS": {255, 120, 120, 255},
	}

	rankColor, ok := rankColors[req.Rank]
	if !ok {
		rankColor = color.RGBA{180, 100, 255, 255}
	}
	rankSecColor, ok2 := rankSecColors[req.Rank]
	if !ok2 {
		rankSecColor = color.RGBA{220, 160, 255, 255}
	}

	// Dynamic theme borders/dividers tinted with Rank Color
	dividerColor := color.RGBA{
		R: uint8(math.Max(25, float64(rankColor.R)/4.5)),
		G: uint8(math.Max(25, float64(rankColor.G)/4.5)),
		B: uint8(math.Max(38, float64(rankColor.B)/4.5)),
		A: 255,
	}
	cardBorderColor := color.RGBA{
		R: uint8(math.Max(30, float64(rankColor.R)/4.0)),
		G: uint8(math.Max(30, float64(rankColor.G)/4.0)),
		B: uint8(math.Max(45, float64(rankColor.B)/4.0)),
		A: 255,
	}

	// Left accent bar
	dc.SetColor(rankColor)
	dc.DrawRectangle(0, 0, 4, PROF_H)
	dc.Fill()

	// Load Fonts
	fontBold := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
	fontSemi := utils.GetAssetPath("rpgasset", "ui", "Inter-SemiBold.ttf")
	fontMed := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")
	log.Printf("Font paths - Bold: %s, Semi: %s, Med: %s", fontBold, fontSemi, fontMed)

	bold22, err1 := utils.LoadFont(fontBold, 22)
	bold14, err2 := utils.LoadFont(fontBold, 14)
	semi12, err3 := utils.LoadFont(fontSemi, 12)
	med10, err4 := utils.LoadFont(fontMed, 10)
	med8, err5 := utils.LoadFont(fontMed, 8)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		fmt.Printf("Font Load Error: %v, %v, %v, %v, %v\n", err1, err2, err3, err4, err5)
		c.JSON(500, gin.H{"error": "Font loading failed. Ensure Inter fonts are present in assets/rpgasset/ui/"})
		return
	}

	// === Left Panel: Avatar & Identity ===
	const avCX, avCY, avR = 135.0, 75.0, 46.0
    // Avatar handling with fallback
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
        // Try UI Avatars placeholder based on nickname
        placeholderURL := "https://ui-avatars.com/api/?name=" + url.QueryEscape(req.Nickname) + "&size=92&background=random&color=fff&bold=true&format=png"
        if img, err := utils.DownloadImage(placeholderURL); err == nil {
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
        // Solid circle with initial fallback
        dc.SetColor(color.RGBA{rankColor.R / 4, rankColor.G / 4, rankColor.B / 4, 255})
        dc.DrawCircle(avCX, avCY, avR)
        dc.Fill()
        dc.SetFontFace(bold22)
        dc.SetColor(color.RGBA{220, 220, 230, 200})
        dc.DrawStringAnchored(string([]rune(req.Nickname)[0:1]), avCX, avCY+2, 0.5, 0.5)
    }
	// PFP border
	dc.SetColor(rankColor)
	dc.SetLineWidth(3)
	dc.DrawCircle(avCX, avCY, avR)
	dc.Stroke()

	// Nickname
	dc.SetFontFace(bold22)
	dc.SetColor(color.RGBA{245, 245, 255, 255})
	dc.DrawStringAnchored(req.Nickname, avCX, 143, 0.5, 0.5)

	// Title
	textY := 172.0
	if req.Title != "" {
		dc.SetFontFace(semi12)
		dc.SetColor(color.RGBA{212, 175, 55, 255}) // Gold
		dc.DrawStringAnchored("✦ "+req.Title+" ✦", avCX, textY, 0.5, 0.5)
		textY += 22
	}

	// Class Badge
	dc.SetColor(color.RGBA{22, 22, 42, 255})
	dc.DrawRoundedRectangle(avCX-75, textY-12, 150, 20, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{rankColor.R / 2, rankColor.G / 2, rankColor.B / 2, 200})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(avCX-75, textY-12, 150, 20, 4)
	dc.Stroke()

	dc.SetFontFace(semi12)
	dc.SetColor(color.RGBA{
		R: uint8(math.Min(255, float64(rankColor.R)*0.8+50)),
		G: uint8(math.Min(255, float64(rankColor.G)*0.8+50)),
		B: uint8(math.Min(255, float64(rankColor.B)*0.8+50)),
		A: 255,
	})
	dc.DrawStringAnchored(req.ClassIcon+" "+req.Class, avCX, textY-3, 0.5, 0.5) // Offset to center perfectly
	textY += 26

	// Rank Badge
	dc.SetColor(rankColor)
	dc.DrawRoundedRectangle(avCX-55, textY-12, 110, 20, 4)
	dc.Fill()
	dc.SetFontFace(bold14)
	dc.SetColor(color.RGBA{20, 20, 30, 255})
	dc.DrawStringAnchored(req.Rank+" RANK", avCX, textY-5, 0.5, 0.5) // Adjusted upward
	textY += 26

	// Guild & Whatsapp Name info
	dc.SetFontFace(med10)
	if req.WhatsappName != "" && req.WhatsappName != req.Nickname {
		dc.SetColor(color.RGBA{120, 120, 155, 255})
		dc.DrawStringAnchored("aka "+req.WhatsappName, avCX, textY, 0.5, 0.5)
		textY += 16
	}
	if req.GuildName != "" {
		dc.SetColor(color.RGBA{212, 175, 55, 200}) // Goldish
		dc.DrawStringAnchored("🏰 "+req.GuildName, avCX, textY, 0.5, 0.5)
		textY += 16
	}

	// Level & GP stats
	dc.SetFontFace(semi12)
	dc.SetColor(color.RGBA{212, 175, 55, 255})
	dc.DrawStringAnchored(fmt.Sprintf("LVL %d  ·  ⭐ %s GP", req.Level, formatNum(req.GP)), avCX, 305, 0.5, 0.5)

	// === XP bar ===
	const xpBarX, xpBarY, xpBarW, xpBarH = 30.0, 335.0, 210.0, 8.0
	dc.SetFontFace(med8)
	dc.SetColor(color.RGBA{120, 120, 150, 255})
	dc.DrawString("XP PROGRESS", xpBarX, xpBarY-6)

	// Bar BG
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
		xpGrad.AddColorStop(0, rankColor)
		xpGrad.AddColorStop(1, rankSecColor)
		dc.SetFillStyle(xpGrad)
		dc.DrawRoundedRectangle(xpBarX, xpBarY, fw, xpBarH, xpBarH/2)
		dc.Fill()
	}
	dc.SetColor(color.RGBA{100, 90, 140, 100})
	dc.SetLineWidth(0.5)
	dc.DrawRoundedRectangle(xpBarX, xpBarY, xpBarW, xpBarH, xpBarH/2)
	dc.Stroke()

	dc.SetFontFace(med10)
	dc.SetColor(color.RGBA{140, 140, 170, 255})
	dc.DrawStringAnchored(fmt.Sprintf("%d / %d XP  (%.1f%%)", req.XP, req.XPNeeded, xpPct*100), avCX, xpBarY+20, 0.5, 0.5)

	// === Vertical Divider ===
	dc.SetColor(dividerColor)
	dc.SetLineWidth(1.5)
	dc.DrawLine(265, 20, 265, PROF_H-20)
	dc.Stroke()

	// === Right Panel: Stats & Gear ===
	// Title for Stats
	dc.SetFontFace(bold14)
	dc.SetColor(rankColor)
	dc.DrawString("📊 CHARACTER STATS", 295, 38)

	// 2 Column, 4 Row Stats Layout
	rowY := [4]float64{70, 105, 140, 175}
	colX1 := 295.0
	colX2 := 545.0

	// Draw Column 1
	drawStatRow(dc, semi12, bold14, colX1, rowY[0], "❤️", "HP", req.HP, req.EquipHP, false)
	drawStatRow(dc, semi12, bold14, colX1, rowY[1], "🛡️", "DEF", req.DEF, req.EquipDEF, false)
	drawStatRow(dc, semi12, bold14, colX1, rowY[2], "💨", "SPD", req.SPD, req.EquipSPD, false)
	drawStatRow(dc, semi12, bold14, colX1, rowY[3], "💥", "CRIT", req.Crit, 0, true)

	// Draw Column 2
	drawStatRow(dc, semi12, bold14, colX2, rowY[0], "⚔️", "ATK", req.ATK, req.EquipATK, false)
	drawStatRow(dc, semi12, bold14, colX2, rowY[1], "🔮", "MAG", req.MAG, req.EquipMAG, false)
	drawStatRow(dc, semi12, bold14, colX2, rowY[2], "🍀", "LCK", req.Luck, req.EquipLuck, false)
	drawStatRow(dc, semi12, bold14, colX2, rowY[3], "🕊️", "EVA", req.Evasion, 0, true)

	// === Horizontal Divider ===
	dc.SetColor(dividerColor)
	dc.SetLineWidth(1.5)
	dc.DrawLine(295, 205, 770, 205)
	dc.Stroke()

	// Title for Gear
	dc.SetFontFace(bold14)
	dc.SetColor(color.RGBA{170, 170, 195, 255})
	dc.DrawString("🛡️ EQUIPPED GEAR", 295, 235)

	// 3x3 Gear Card layout
	gearColX := [3]float64{295, 453, 611}
	gearRowY := [3]float64{252, 312, 372}

	// Row 1
	drawGearSlot(dc, med8, med10, gearColX[0], gearRowY[0], "⚔️", "MAIN HAND", req.GearMainHand, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[1], gearRowY[0], "🗡️", "OFF HAND", req.GearOffHand, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[2], gearRowY[0], "👕", "ARMOR", req.GearArmor, cardBorderColor)

	// Row 2
	drawGearSlot(dc, med8, med10, gearColX[0], gearRowY[1], "⛑️", "HELMET", req.GearHelmet, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[1], gearRowY[1], "🧤", "GLOVES", req.GearGloves, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[2], gearRowY[1], "👢", "BOOTS", req.GearBoots, cardBorderColor)

	// Row 3
	drawGearSlot(dc, med8, med10, gearColX[0], gearRowY[2], "💍", "RING", req.GearRing, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[1], gearRowY[2], "📿", "AMULET", req.GearAmulet, cardBorderColor)
	drawGearSlot(dc, med8, med10, gearColX[2], gearRowY[2], "🧥", "CLOAK", req.GearCloak, cardBorderColor)

	// Watermark
	dc.SetFontFace(med8)
	dc.SetColor(color.RGBA{45, 45, 75, 255})
	dc.DrawStringAnchored("Made By Mellow™", PROF_W-20, PROF_H-12, 1.0, 0.5)

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func drawStatRow(dc *gg.Context, fontLabel, fontVal font.Face, x, y float64, statIcon, statName string, baseVal, bonusVal float64, isPercent bool) {
	// Label
	dc.SetFontFace(fontLabel)
	dc.SetColor(color.RGBA{160, 160, 185, 255})
	dc.DrawString(statIcon+" "+statName, x, y)

	// Base Value
	dc.SetFontFace(fontVal)
	valStr := ""
	if isPercent {
		valStr = fmt.Sprintf("%.1f%%", baseVal)
	} else {
		valStr = fmt.Sprintf("%.0f", baseVal)
	}

	wStr, _ := dc.MeasureString(valStr)

	dc.SetColor(color.RGBA{245, 245, 250, 255})
	dc.DrawString(valStr, x+90, y)

	// Bonus Value
	if bonusVal > 0 {
		dc.SetColor(color.RGBA{60, 210, 130, 255}) // Green
		bonusStr := fmt.Sprintf(" (+%.0f)", bonusVal)
		dc.DrawString(bonusStr, x+90+wStr+4, y)
	} else if bonusVal < 0 {
		dc.SetColor(color.RGBA{240, 70, 70, 255}) // Red
		bonusStr := fmt.Sprintf(" (-%.0f)", math.Abs(bonusVal))
		dc.DrawString(bonusStr, x+90+wStr+4, y)
	}


func drawGearSlot(dc *gg.Context, fontLabel, fontVal font.Face, x, y float64, slotIcon, slotLabel, itemName string, borderColor color.RGBA) {
	w := 148.0
	h := 50.0

	// BG Card
	dc.SetColor(color.RGBA{22, 22, 38, 255})
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()

	// Green border for occupied slots, dim border for empty
	border := borderColor
	if itemName != "" && itemName != "None" {
		border = color.RGBA{60, 210, 130, 255}
	}
	dc.SetColor(border)
	dc.SetLineWidth(0.8)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Stroke()

	// Slot Title
	dc.SetFontFace(fontLabel)
	dc.SetColor(color.RGBA{120, 120, 150, 255})
	dc.DrawString(slotIcon+" "+slotLabel, x+8, y+16)

	// Item Value
	dc.SetFontFace(fontVal)
	if itemName == "" || itemName == "None" {
		dc.SetColor(color.RGBA{70, 70, 95, 255})
		dc.DrawString("Empty", x+8, y+36)
	} else {
		dc.SetColor(color.RGBA{240, 240, 255, 255})
		runes := []rune(itemName)
		if len(runes) > 17 {
			itemName = string(runes[:14]) + "..."
		}
		dc.DrawString(slotIcon+" "+itemName, x+8, y+36)
	}
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
