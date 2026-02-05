package combat

import (
	"fmt"  // ← ADDED
	"image"
	"image/color"
	"math"
	"sort"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	CANVAS_W = 1024
	CANVAS_H = 687
	OFF_X    = 694
	OFF_Y    = 356
)

func GenerateCombatImage(c *gin.Context) {
	var req CombatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Create Canvas
	dc := gg.NewContext(CANVAS_W, CANVAS_H)
	
	// 1. Background
	// Assume assets are in ./assets
	assetsPath := "assets" 
	
	bgPath := utils.GetAssetPath("rpgasset", "environment", req.Background)
	if req.Background == "" {
		// Fallback if empty
		bgPath = utils.GetAssetPath("rpgasset", "environment", "forest.jpg") // Example default
	}

	bgImg, err := utils.LoadImage(bgPath)
	if err == nil {
		bgImg = imaging.Fill(bgImg, CANVAS_W, CANVAS_H, imaging.Center, imaging.Lanczos)
		dc.DrawImage(bgImg, 0, 0)
	} else {
		// Fallback color
		dc.SetHexColor("#1a1a1a")
		dc.Clear()
	}

	// Dark Overlay
	dc.SetColor(color.RGBA{0, 0, 0, 100}) // 0x00000066 -> 102 (40%)
	dc.DrawRectangle(0, 0, CANVAS_W, CANVAS_H)
	dc.Fill()

	// 2. Mobs / Enemies
	enemySpriteSize := 190.0
	startX, startY := 780.0, 160.0
	spX, spY := 130.0, 110.0

	// Determine avg level for sprite selection
	avgLevel := 1
	if len(req.Players) > 0 {
		sum := 0
		for _, p := range req.Players {
			sum += p.Level
		}
		avgLevel = sum / len(req.Players)
	}

	type RenderItem struct {
		img image.Image
		x, y float64
	}
	var mobQueue []RenderItem

	for i, enemy := range req.Enemies {
		if enemy.CurrentHP <= 0 && !enemy.JustDied {
			continue
		}

		spritePath := GetEnemySpritePath(avgLevel, i, enemy.IsBoss, assetsPath)
		eSprite, err := utils.LoadImage(spritePath)
		if err != nil {
			continue
		}

		// Resize
		eW := enemySpriteSize
		if enemy.IsBoss {
			eW = enemySpriteSize * 1.5
		}
		eSprite = imaging.Resize(eSprite, int(eW), 0, imaging.Lanczos)

		// Tint Red if dead
		if enemy.CurrentHP <= 0 {
			eSprite = utils.TintImage(eSprite, color.RGBA{255, 0, 0, 100})
		}

		// Calculate Position
		ex, ey := startX, startY
		sub := i % 4
		if sub == 1 || sub == 2 {
			ex -= spX
		} else if sub == 3 {
			ex -= spX * 2
		}
		if sub == 1 || sub == 3 {
			ey += spY
		}
		ex += float64(i/4) * -250.0

		mobQueue = append(mobQueue, RenderItem{eSprite, ex, ey})
	}

	// Sort by Y (Painter's Algorithm)
	sort.Slice(mobQueue, func(i, j int) bool {
		return mobQueue[i].y < mobQueue[j].y
	})

	// Draw Mobs
	for _, mob := range mobQueue {
		// Shadow
		utils.DrawShadow(dc, mob.x + float64(mob.img.Bounds().Dx())/2, mob.y + float64(mob.img.Bounds().Dy()) - 10, float64(mob.img.Bounds().Dx())*0.4, 0.6)
		// Sprite
		dc.DrawImage(mob.img, int(mob.x), int(mob.y))
	}

	// 3. UI Layer
	uiPath := func(f string) string { return utils.GetAssetPath("rpgasset", "ui", f) }
	
	drawImage := func(path string, x, y, w, h int) {
		img, err := utils.LoadImage(path)
		if err == nil {
			if w > 0 && h > 0 {
				img = imaging.Resize(img, w, h, imaging.Lanczos)
			}
			dc.DrawImage(img, normX(x), normY(y))
		}
	}

	// player_state.png -716, 113
	drawImage(uiPath("player_state.png"), -716, 113, 453, 244)
	// heart.png -678, 209
	drawImage(uiPath("heart.png"), -678, 209, 38, 47)
	// mana.png -673, 256
	drawImage(uiPath("mana.png"), -673, 256, 29, 44)
	// Options_menu.png -97, 99
	drawImage(uiPath("Options_menu.png"), -97, 99, 443, 258)
	// banner.png -496, -339
	drawImage(uiPath("banner.png"), -496, -339, 573, 118)

	// 4. Bars & Player Sprite
	if len(req.Players) > 0 {
		p := req.Players[0]
		
		// Draw Bars
		hpCoords := []int{-640, -550, -459}
		enCoords := []int{-644, -555, -465}
		hpSeg := float64(p.MaxHP) / 3.0
		enSeg := float64(p.MaxEnergy) / 3.0

		for i := 0; i < 3; i++ {
			hCur := math.Max(0, math.Min(hpSeg, float64(p.CurrentHP) - (float64(i)*hpSeg)))
			drawBar(dc, uiPath, normX(hpCoords[i]), normY(209), hCur, hpSeg, "hp", 121, 47)

			eCur := math.Max(0, math.Min(enSeg, float64(p.Energy) - (float64(i)*enSeg)))
			drawBar(dc, uiPath, normX(enCoords[i]), normY(256), eCur, enSeg, "mana", 119, 42)
		}

		// Draw Player Sprite
		spritePath := GetCharacterSpritePath(p.Class, p.SpriteIndex, assetsPath)
		pSprite, err := utils.LoadImage(spritePath)
		if err == nil {
			if p.CurrentHP <= 0 {
				pSprite = utils.TintImage(pSprite, color.RGBA{255, 0, 0, 100})
			}
			
			s1W := 314
			pSprite = imaging.Resize(pSprite, s1W, 0, imaging.Lanczos)
			
			dc.DrawImage(pSprite, normX(-660), normY(191) - pSprite.Bounds().Dy() + 50)
		}
	}

	// 5. Rank Text
	if req.Rank != "" {
		text := req.Rank + " RANK"
		if req.CombatType == "PVP" {
			text = "PVP MATCH"
		}
		
		fontPath := utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
		face, err := utils.LoadFont(fontPath, 100)
		if err == nil {
			dc.SetFontFace(face)
			dc.SetColor(color.Black)
			
			bx, by := float64(normX(-496)), float64(normY(-339))
			bw, bh := 573.0, 118.0
			
			dc.DrawStringAnchored(text, bx + bw/2, by + bh/2 - 10, 0.5, 0.5)
		}
	}

	// Encode
	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to encode image"})
		return
	}

	c.Data(200, "image/png", buf)
}

func normX(x int) int { return x + OFF_X }
func normY(y int) int { return y + OFF_Y }

func drawBar(dc *gg.Context, uiPath func(string)string, x, y int, current, max float64, typePrefix string, w, h int) {
	if max <= 0 { max = 1 }
	percent := current / max
	spriteNum := int(math.Min(5, math.Max(1, math.Round(percent*4)+1)))
	
	filename := fmt.Sprintf("%s%d.png", typePrefix, spriteNum)
	img, err := utils.LoadImage(uiPath(filename))
	if err == nil {
		img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
		dc.DrawImage(img, x, y)
	}
}
