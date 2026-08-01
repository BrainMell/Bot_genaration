package combat

import (
        "fmt"
        "image"
        "image/color"
        "math"
        "os"
        "path/filepath"
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

        // 1. Background - FIXED
        assetsPath := "assets"

        // Try to load background
        var bgPath string
        if req.Background != "" {
                // Check if it's a full path or just filename
                if filepath.IsAbs(req.Background) || fileExists(req.Background) {
                        bgPath = req.Background
                } else {
                        bgPath = filepath.Join(assetsPath, "rpgasset", "environment", req.Background)
                }
        }

        // If no background specified or doesn't exist, get random one
        if bgPath == "" || !fileExists(bgPath) {
                bgPath = getRandomEnvironment(assetsPath)
        }

        // Load and composite background
        if bgPath != "" && fileExists(bgPath) {
                bgImg, err := utils.LoadImage(bgPath)
                if err == nil {
                        bgImg = imaging.Fill(bgImg, CANVAS_W, CANVAS_H, imaging.Center, imaging.NearestNeighbor)
                        dc.DrawImage(bgImg, 0, 0)
                } else {
                        // Fallback color
                        dc.SetHexColor("#1a1a1a")
                        dc.Clear()
                }
        } else {
                // Fallback color if no background found
                dc.SetHexColor("#1a1a1a")
                dc.Clear()
        }

        // Dark Overlay (40% black)
        dc.SetColor(color.RGBA{0, 0, 0, 102})
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
                img       image.Image
                x, y      float64
                hpPercent float64
        }
        var mobQueue []RenderItem

        for i, enemy := range req.Enemies {
                if enemy.CurrentHP <= 0 && !enemy.JustDied {
                        continue
                }

                spritePath := GetEnemySpritePath(enemy.Name, avgLevel, i, enemy.IsBoss, assetsPath)
                eSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }

                // Resize
                eW := enemySpriteSize
                if enemy.IsBoss {
                        eW = enemySpriteSize * 1.5
                }
                eSprite = imaging.Resize(eSprite, int(eW), 0, imaging.NearestNeighbor)

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

                hpPerc := 0.0
                if enemy.MaxHP > 0 {
                        hpPerc = float64(enemy.CurrentHP) / float64(enemy.MaxHP)
                }
                mobQueue = append(mobQueue, RenderItem{eSprite, ex, ey, hpPerc})
        }
        // Sort by Y (Painter's Algorithm)
        sort.Slice(mobQueue, func(i, j int) bool {
                return mobQueue[i].y < mobQueue[j].y
        })

        // Draw Mobs
        for _, mob := range mobQueue {
                // Shadow
                utils.DrawShadow(dc, mob.x+float64(mob.img.Bounds().Dx())/2, mob.y+float64(mob.img.Bounds().Dy())-10, float64(mob.img.Bounds().Dx())*0.4, 0.6)
                // Sprite
                dc.DrawImage(mob.img, int(mob.x), int(mob.y))

                // ENEMY HP BAR - Stretched hp5.png
                if mob.hpPercent > 0 {
                        uiPath := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }
                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                        if err == nil {
                                barW := 100.0
                                barH := 12.0
                                // Stretch hp5.png to current HP width
                                currentBarW := int(barW * mob.hpPercent)
                                if currentBarW < 1 {
                                        currentBarW = 1
                                }
                                hpBarImg = imaging.Resize(hpBarImg, currentBarW, int(barH), imaging.NearestNeighbor)

                                // Position above head
                                bx := mob.x + (float64(mob.img.Bounds().Dx())-barW)/2
                                by := mob.y - 15
                                dc.DrawImage(hpBarImg, int(bx), int(by))
                        }
                }
        }

        // 💡 Summoner System (Phase 7): Draw summoned allies.
        // Summons are rendered on the player's side of the battlefield,
        // arranged side-by-side like enemies (same formation pattern).
        // Drawn after mobs but before UI so UI overlays on top.
        for i, summon := range req.Summons {
                if summon.CurrentHP <= 0 && !summon.JustDied {
                        continue
                }

                spritePath := GetSummonSpritePath(summon.Species, assetsPath)
                sSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }

                // Resize — summons are 80% of enemy sprite size
                sW := enemySpriteSize * 0.8
                sSprite = imaging.Resize(sSprite, int(sW), 0, imaging.NearestNeighbor)

                // 💡 Flip summon to face enemies (right), same as players
                sSprite = imaging.FlipH(sSprite)

                // Tint red if dead
                if summon.CurrentHP <= 0 {
                        sSprite = utils.TintImage(sSprite, color.RGBA{255, 0, 0, 100})
                }

                // 💡 Position — summons stand BEHIND players, slightly offset.
                // Player formation is at startX-500. Summons at startX-380 (120px right = closer to enemies).
                // Same 4-per-row pattern as enemies + players.
                sx := startX - 380
                sy := startY + 60
                sub := i % 4
                if sub == 1 || sub == 2 {
                        sx -= spX
                } else if sub == 3 {
                        sx -= spX * 2
                }
                if sub == 1 || sub == 3 {
                        sy += spY
                }
                sx += float64(i/4) * -250.0

                // Shadow
                utils.DrawShadow(dc, sx+float64(sSprite.Bounds().Dx())/2, sy+float64(sSprite.Bounds().Dy())-10, float64(sSprite.Bounds().Dx())*0.4, 0.6)

                // Sprite
                dc.DrawImage(sSprite, int(sx), int(sy))

                // HP bar (same style as enemy HP bars)
                if summon.MaxHP > 0 && summon.CurrentHP > 0 {
                        uiPath2 := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }
                        hpBarImg, err := utils.LoadImage(uiPath2("hp5.png"))
                        if err == nil {
                                barW := 80.0
                                barH := 10.0
                                hpPerc := float64(summon.CurrentHP) / float64(summon.MaxHP)
                                currentBarW := int(barW * hpPerc)
                                if currentBarW < 1 {
                                        currentBarW = 1
                                }
                                hpBarImg = imaging.Resize(hpBarImg, currentBarW, int(barH), imaging.NearestNeighbor)
                                bx := sx + (float64(sSprite.Bounds().Dx())-barW)/2
                                by := sy - 12
                                dc.DrawImage(hpBarImg, int(bx), int(by))
                        }
                }
        }

        // 3. UI Base Layer
        uiPath := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }

        drawImage := func(path string, x, y, w, h int) {
                img, err := utils.LoadImage(path)
                if err == nil {
                        if w > 0 && h > 0 {
                                img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
                        }
                        dc.DrawImage(img, normX(x), normY(y))
                }
        }

        // UI elements
        drawImage(uiPath("player_state.png"), -716, 113, 453, 244)
        drawImage(uiPath("heart.png"), -678, 209, 38, 47)
        drawImage(uiPath("mana.png"), -673, 256, 29, 44)
        drawImage(uiPath("Options_menu.png"), -97, 99, 443, 258)
        drawImage(uiPath("banner.png"), -582, -410, 800, 160)

        // 4. UI Bars (Main Player)
        if len(req.Players) > 0 {
                p := req.Players[0]

                // Draw HP and Energy Bars
                hpCoords := []int{-640, -550, -459}
                enCoords := []int{-644, -555, -465}
                hpSeg := float64(p.MaxHP) / 3.0
                enSeg := float64(p.MaxEnergy) / 3.0

                for i := 0; i < 3; i++ {
                        hCur := math.Max(0, math.Min(hpSeg, float64(p.CurrentHP)-(float64(i)*hpSeg)))
                        drawBar(dc, uiPath, normX(hpCoords[i]), normY(209), hCur, hpSeg, "hp", 121, 47)

                        eCur := math.Max(0, math.Min(enSeg, float64(p.Energy)-(float64(i)*enSeg)))
                        drawBar(dc, uiPath, normX(enCoords[i]), normY(256), eCur, enSeg, "mana", 119, 42)
                }

                // 5. Player Sprite (Main - CROPPED TOP 30%)
                spritePath := GetCharacterSpritePath(p.Class, p.SpriteIndex, assetsPath)
                pSprite, err := utils.LoadImage(spritePath)
                if err == nil {
                        if p.CurrentHP <= 0 {
                                pSprite = utils.TintImage(pSprite, color.RGBA{255, 0, 0, 100})
                        }

                        // Resize to 314px width
                        s1W := 314
                        pSprite = imaging.Resize(pSprite, s1W, 0, imaging.NearestNeighbor)

                        // Crop TOP 30% - CRITICAL FIX
                        bounds := pSprite.Bounds()
                        cropH := int(float64(bounds.Dy()) * 0.3)
                        croppedSprite := imaging.Crop(pSprite, image.Rect(0, 0, bounds.Dx(), cropH))

                        // Position at normX(-660), normY(220) - cropH
                        dc.DrawImage(croppedSprite, normX(-660), normY(220)-cropH+40)

                        // 6. Small full-body sprites on battlefield
                        if req.CombatType == "PVP" {
                                drawPvPFighter := func(player Player, x, y int, flip bool) {
                                        path := GetCharacterSpritePath(player.Class, player.SpriteIndex, assetsPath)
                                        sprite, err := utils.LoadImage(path)
                                        if err != nil {
                                                return
                                        }
                                        if player.CurrentHP <= 0 {
                                                sprite = utils.TintImage(sprite, color.RGBA{255, 0, 0, 100})
                                        }
                                        sprite = imaging.Resize(sprite, 160, 0, imaging.NearestNeighbor)
                                        if flip {
                                                sprite = imaging.FlipH(sprite)
                                        }

                                        utils.DrawShadow(dc, float64(x)+80, float64(y)+float64(sprite.Bounds().Dy()), 165, 0.6)
                                        dc.DrawImage(sprite, x, y)

                                        hpPercent := 0.0
                                        if player.MaxHP > 0 {
                                                hpPercent = math.Max(0, math.Min(1, float64(player.CurrentHP)/float64(player.MaxHP)))
                                        }
                                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                                        if err == nil && hpPercent > 0 {
                                                barW := int(120 * hpPercent)
                                                if barW < 1 {
                                                        barW = 1
                                                }
                                                hpBarImg = imaging.Resize(hpBarImg, barW, 12, imaging.NearestNeighbor)
                                                dc.DrawImage(hpBarImg, x+20, y-18)
                                        }
                                }

                                drawPvPFighter(p, int(startX-560), int(startY-30), false)
                                if len(req.Players) > 1 {
                                        drawPvPFighter(req.Players[1], int(startX-170), int(startY-30), true)
                                }
                        } else {
<<<<<<< HEAD
                                // 💡 AUDIT FIX 2026-08-01 (Round 5): draw ALL players in
                                // formation on the battlefield, not just player[0]. The
                                // animator (animated combat) already does this — the static
                                // renderer was inconsistent, only showing the first player.
                                // Now both static + animated renders show the full party
                                // standing side-by-side like enemies do.
=======
                                // PVE: draw ALL players in formation, flipped to face enemies (right)
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                s2Size := 122
                                smallSprite := imaging.Resize(pSprite, s2Size, 0, imaging.NearestNeighbor)
                                flippedSprite := imaging.FlipH(smallSprite)

<<<<<<< HEAD
                                // Player[0] at the original position
=======
                                // Player[0] position — mirrors enemy formation, on left side
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                s2X := int(startX - 500)
                                s2Y := int(startY + 30)

                                // Shadow + sprite for player[0]
                                utils.DrawShadow(dc, float64(s2X)+float64(s2Size)/2, float64(s2Y)+float64(flippedSprite.Bounds().Dy()), 150, 0.6)
                                dc.DrawImage(flippedSprite, s2X, s2Y)

<<<<<<< HEAD
                                // Draw sprite (flipped to face right toward enemies)
                                flippedSprite := imaging.FlipH(smallSprite)
                                dc.DrawImage(flippedSprite, s2X, s2Y)

                                // Draw additional players (1+) in formation
                                for pi := 1; pi < len(req.Players); pi++ {
                                        ap := req.Players[pi]
                                        if ap.CurrentHP <= 0 {
                                                continue // skip dead players
                                        }
                                        apPath := GetCharacterSpritePath(ap.Class, ap.SpriteIndex, assetsPath)
                                        apSprite, err := utils.LoadImage(apPath)
                                        if err != nil {
                                                continue
                                        }
                                        apResized := imaging.Resize(apSprite, s2Size, 0, imaging.NearestNeighbor)
                                        apFlipped := imaging.FlipH(apResized)

                                        // Formation: same pattern as enemies, mirrored.
                                        // 4 per row, offset left/right. Player 1 is at front.
=======
                                // HP bar for player[0]
                                if p.MaxHP > 0 && p.CurrentHP > 0 {
                                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                                        if err == nil {
                                                hpPerc := float64(p.CurrentHP) / float64(p.MaxHP)
                                                barW := int(80.0 * hpPerc)
                                                if barW < 1 { barW = 1 }
                                                hpBarImg = imaging.Resize(hpBarImg, barW, 10, imaging.NearestNeighbor)
                                                dc.DrawImage(hpBarImg, s2X+(s2Size/2)-40, s2Y-12)
                                        }
                                }

                                // Additional players (1+) in formation — same pattern as enemies
                                for pi := 1; pi < len(req.Players); pi++ {
                                        ap := req.Players[pi]
                                        if ap.CurrentHP <= 0 { continue }
                                        apPath := GetCharacterSpritePath(ap.Class, ap.SpriteIndex, assetsPath)
                                        apSprite, err := utils.LoadImage(apPath)
                                        if err != nil { continue }
                                        apResized := imaging.Resize(apSprite, s2Size, 0, imaging.NearestNeighbor)
                                        apFlipped := imaging.FlipH(apResized)

                                        // Formation: mirror enemy pattern, 4 per row
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                        apX := int(startX - 500)
                                        apY := int(startY + 30)
                                        sub := pi % 4
                                        spXF := 120.0
                                        spYF := 100.0
<<<<<<< HEAD
                                        if sub == 1 || sub == 2 {
                                                apX -= int(spXF)
                                        } else if sub == 3 {
                                                apX -= int(spXF * 2)
                                        }
                                        if sub == 1 || sub == 3 {
                                                apY += int(spYF)
                                        }
=======
                                        if sub == 1 || sub == 2 { apX -= int(spXF) } else if sub == 3 { apX -= int(spXF * 2) }
                                        if sub == 1 || sub == 3 { apY += int(spYF) }
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                        apX += int(float64(pi/4) * -250.0)

                                        utils.DrawShadow(dc, float64(apX)+float64(s2Size)/2, float64(apY)+float64(apFlipped.Bounds().Dy()), 150, 0.6)
                                        dc.DrawImage(apFlipped, apX, apY)

<<<<<<< HEAD
                                        // HP bar for additional players
=======
                                        // HP bar
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                        if ap.MaxHP > 0 && ap.CurrentHP > 0 {
                                                hpBarImg2, err := utils.LoadImage(uiPath("hp5.png"))
                                                if err == nil {
                                                        hpPerc2 := float64(ap.CurrentHP) / float64(ap.MaxHP)
                                                        barW2 := int(80.0 * hpPerc2)
<<<<<<< HEAD
                                                        if barW2 < 1 {
                                                                barW2 = 1
                                                        }
=======
                                                        if barW2 < 1 { barW2 = 1 }
>>>>>>> 36c8a64 (fix: re-apply lost Go fixes + player/summon formation + facing)
                                                        hpBarImg2 = imaging.Resize(hpBarImg2, barW2, 10, imaging.NearestNeighbor)
                                                        dc.DrawImage(hpBarImg2, apX+(s2Size/2)-40, apY-12)
                                                }
                                        }
                                }
                        }
                }
        }

        // 7. Banner Text (Overlaid ON the banner) - FIXED
        if req.Rank != "" || len(req.Players) > 0 {
                text := req.Rank
                if text == "" && len(req.Players) > 0 {
                        text = req.Players[0].AdventurerRank
                }
                if text == "" {
                        text = "F"
                }
                text = text + " RANK"

                if req.CombatType == "PVP" {
                        text = "PVP MATCH"
                }

                fontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
                face, err := utils.LoadFont(fontPath, 40) // 40pt (smaller as requested)
                if err == nil {
                        dc.SetFontFace(face)
                        dc.SetColor(color.RGBA{0, 0, 0, 255}) // Black

                        // Center text in banner at normX(-582), normY(-410)
                        bx, by := float64(normX(-582)), float64(normY(-410))
                        bw, bh := 800.0, 160.0

                        // Draw centered in the new 800x160 banner, shifted up 30px
                        dc.DrawStringAnchored(text, bx+bw/2, by+bh/2-30, 0.5, 0.5)
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

func GenerateEndScreen(c *gin.Context) {
        var req struct {
                Text     string `json:"text"`
                Victory  bool   `json:"victory"`
                Gold     int    `json:"gold"`
                XP       int    `json:"xp"`
                Items    string `json:"items"` // comma-separated item names
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        // 💡 NEW 2026-07-29: Render a proper victory/defeat scene instead of
        // plain white background with text. Uses a gradient background tinted
        // by victory/defeat, large fantasy-font title, and a rewards panel.
        dc := gg.NewContext(CANVAS_W, CANVAS_H)

        // Background gradient
        var rTop, gTop, bTop, rBot, gBot, bBot float64
        if req.Victory {
                // Gold gradient for victory
                rTop, gTop, bTop = 0.18, 0.13, 0.02
                rBot, gBot, bBot = 0.10, 0.07, 0.01
        } else {
                // Dark red gradient for defeat
                rTop, gTop, bTop = 0.18, 0.02, 0.02
                rBot, gBot, bBot = 0.08, 0.01, 0.01
        }
        for y := 0; y < CANVAS_H; y++ {
                t := float64(y) / float64(CANVAS_H)
                dc.SetRGB(rTop+(rBot-rTop)*t, gTop+(gBot-gTop)*t, bTop+(bBot-bTop)*t)
                dc.DrawRectangle(0, float64(y), CANVAS_W, 1)
                dc.Fill()
        }

        // Vignette
        for i := 0; i < 50; i++ {
                dc.SetRGBA(0, 0, 0, 0.015)
                dc.DrawRectangle(0, 0, float64(i), CANVAS_H)
                dc.Fill()
                dc.DrawRectangle(float64(CANVAS_W-i), 0, float64(i), CANVAS_H)
                dc.Fill()
        }

        fontPath := utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
        boldFontPath := filepath.Join("assets", "rpgasset", "ui", "Inter-Bold.ttf")

        // Title
        titleText := req.Text
        if titleText == "" {
                if req.Victory {
                        titleText = "VICTORY"
                } else {
                        titleText = "DEFEATED"
                }
        }
        if face, err := utils.LoadFont(fontPath, 110); err == nil {
                dc.SetFontFace(face)
                if req.Victory {
                        dc.SetRGB(1, 0.85, 0.3) // gold
                } else {
                        dc.SetRGB(0.9, 0.3, 0.3) // red
                }
                // Shadow
                dc.SetRGBA(0, 0, 0, 0.7)
                dc.DrawStringAnchored(titleText, float64(CANVAS_W/2)+4, 220+4, 0.5, 0.5)
                // Main text
                if req.Victory {
                        dc.SetRGB(1, 0.85, 0.3)
                } else {
                        dc.SetRGB(0.9, 0.3, 0.3)
                }
                dc.DrawStringAnchored(titleText, float64(CANVAS_W/2), 220, 0.5, 0.5)
        }

        // Subtitle / flavor
        subtitleText := ""
        if req.Victory {
                subtitleText = "The party emerges triumphant."
        } else {
                subtitleText = "Darkness claims the fallen."
        }
        if face, err := utils.LoadFont(fontPath, 28); err == nil {
                dc.SetFontFace(face)
                dc.SetRGBA(0.85, 0.85, 0.85, 0.9)
                dc.DrawStringAnchored(subtitleText, float64(CANVAS_W/2), 300, 0.5, 0.5)
        }

        // Rewards panel (victory only)
        if req.Victory && (req.Gold > 0 || req.XP > 0 || req.Items != "") {
                panelY := 360.0
                panelH := 200.0
                dc.SetRGBA(0, 0, 0, 0.5)
                dc.DrawRoundedRectangle(150, panelY, float64(CANVAS_W-300), panelH, 12)
                dc.Fill()
                dc.SetRGBA(1, 0.85, 0.3, 0.4)
                dc.DrawRoundedRectangle(150, panelY, float64(CANVAS_W-300), panelH, 12)
                dc.SetLineWidth(2)
                dc.Stroke()

                if face, err := utils.LoadFont(boldFontPath, 32); err == nil {
                        dc.SetFontFace(face)
                        dc.SetRGB(1, 0.85, 0.3)
                        dc.DrawStringAnchored("REWARDS", float64(CANVAS_W/2), panelY+35, 0.5, 0.5)
                }
                if face, err := utils.LoadFont(boldFontPath, 24); err == nil {
                        dc.SetFontFace(face)
                        lineY := panelY + 80
                        if req.Gold > 0 {
                                dc.SetRGB(1, 0.85, 0.3)
                                dc.DrawStringAnchored(fmt.Sprintf("+%d Zeni", req.Gold), float64(CANVAS_W/2), lineY, 0.5, 0.5)
                                lineY += 35
                        }
                        if req.XP > 0 {
                                dc.SetRGB(0.4, 1, 0.4)
                                dc.DrawStringAnchored(fmt.Sprintf("+%d XP", req.XP), float64(CANVAS_W/2), lineY, 0.5, 0.5)
                                lineY += 35
                        }
                        if req.Items != "" {
                                dc.SetRGB(0.9, 0.9, 1.0)
                                itemsLabel := req.Items
                                if len(itemsLabel) > 60 {
                                        itemsLabel = itemsLabel[:57] + "..."
                                }
                                dc.DrawStringAnchored("Loot: "+itemsLabel, float64(CANVAS_W/2), lineY, 0.5, 0.5)
                        }
                }
        }

        // Top + bottom border accents
        dc.SetRGBA(1, 1, 1, 0.3)
        dc.DrawRectangle(0, 0, CANVAS_W, 4)
        dc.Fill()
        dc.DrawRectangle(0, CANVAS_H-4, CANVAS_W, 4)
        dc.Fill()

        buf, err := utils.EncodeImageToBuffer(dc.Image())
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode image"})
                return
        }

        c.Data(200, "image/png", buf)
}

func normX(x int) int { return x + OFF_X }
func normY(y int) int { return y + OFF_Y }

func drawBar(dc *gg.Context, uiPath func(string) string, x, y int, current, max float64, typePrefix string, w, h int) {
        if max <= 0 {
                max = 1
        }
        percent := current / max
        spriteNum := int(math.Min(5, math.Max(1, math.Round(percent*4)+1)))

        filename := fmt.Sprintf("%s%d.png", typePrefix, spriteNum)
        img, err := utils.LoadImage(uiPath(filename))
        if err == nil {
                img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
                dc.DrawImage(img, x, y)
        }
}

func fileExists(path string) bool {
        _, err := os.Stat(path)
        return err == nil
}

func getRandomEnvironment(assetsPath string) string {
        envPath := filepath.Join(assetsPath, "rpgasset", "environment")

        entries, err := os.ReadDir(envPath)
        if err != nil {
                return ""
        }

        var files []string
        for _, entry := range entries {
                if !entry.IsDir() {
                        name := entry.Name()
                        ext := filepath.Ext(name)
                        if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
                                files = append(files, filepath.Join(envPath, name))
                        }
                }
        }

        if len(files) == 0 {
                return ""
        }

        // Actually randomize using current time (not crypto-secure, fine for visual variety)
        // Using a simple LCG seeded by time so we don't add math/rand import overhead.
        // Falls back to first file if randomization fails.
        idx := int(timeNow().UnixNano()) % len(files)
        if idx < 0 {
                idx = -idx
        }
        return files[idx]
}
