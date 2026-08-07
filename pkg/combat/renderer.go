package combat

import (
        "fmt"
        "image"
        "image/color"
        "image/draw"
        "math"
        "os"
        "path/filepath"
        "sort"
        "strings"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

// 💡 FIX 2026-08-07 (#2): cropToVisibleBounds crops transparent padding from
// a sprite image so the visible content's bottom edge aligns with the ground.
// Without this, sprites with bottom transparent padding "float" above the
// ground line because their bounding-box bottom isn't their visible bottom.
func cropToVisibleBounds(img image.Image) image.Image {
        bounds := img.Bounds()
        minX := bounds.Max.X
        minY := bounds.Max.Y
        maxX := bounds.Min.X
        maxY := bounds.Min.Y

        for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
                for x := bounds.Min.X; x < bounds.Max.X; x++ {
                        _, _, _, a := img.At(x, y).RGBA()
                        if a > 10 {
                                if x < minX { minX = x }
                                if x > maxX { maxX = x }
                                if y < minY { minY = y }
                                if y > maxY { maxY = y }
                        }
                }
        }
        if maxX < 0 || maxY < 0 {
                return img // fully transparent — return as-is
        }
        // 💡 FIX: Always use NewRGBA (0,0 origin), NOT SubImage.
        // SubImage returns offset bounds which cause DrawImage to draw at
        // wrong position (position drift = ghosting/floating artifact).
        cropW := maxX - minX + 1
        cropH := maxY - minY + 1
        dst := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
        draw.Draw(dst, dst.Bounds(), img, image.Pt(minX, minY), draw.Src)
        return dst
}

const (
        CANVAS_W = 1024
        CANVAS_H = 687
        OFF_X    = 694
        OFF_Y    = 356
        // 💡 FIX 2026-08-07: Horizon line + ground zone (Spec 1A).
        // Sky zone: Y=0 to HORIZON_Y. No sprite feet may be placed here.
        // Ground zone: HORIZON_Y to GROUND_BOTTOM_Y. All sprite feet go here.
        HORIZON_Y       = 309 // 45% of CANVAS_H — sky/ground boundary
        GROUND_BOTTOM_Y = 515 // 75% of CANVAS_H — top of UI panel
        MAIN_FEET_Y     = 505 // main player/enemy feet (above UI panel top at 469... wait)
        BACK_FEET_Y     = 410 // summon/background entity feet (higher = further away)
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

        // If no background specified or doesn't exist, use a deterministic default.
        // 💡 FIX: was getRandomEnvironment (random every render). Now uses
        // rank-based selection so the same dungeon always gets the same background.
        if bgPath == "" || !fileExists(bgPath) {
                bgPath = getRankBackground(req.Rank, assetsPath)
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
        // 💡 FIX 2026-08-07: Reduced from 102 (40%) to 80 (31%) to prevent
        // dark sprite areas (wolf legs, bat wings) from blending into the
        // background and appearing "floating" to viewers.
        dc.SetColor(color.RGBA{0, 0, 0, 80})
        dc.DrawRectangle(0, 0, CANVAS_W, CANVAS_H)
        dc.Fill()

        // 💡 TURN INDICATOR HELPER (2026-08-03):
        // Draws a golden glowing ellipse under the active attacker, BEFORE
        // the sprite is drawn (so the sprite stands on top of it).
        // Squashed vertically (0.4 ratio) for perspective.
        // Two layers: outer golden glow + inner bright ring.
        // Only draws if req.Action is set (TURN phase renders).
        drawTurnIndicator := func(cx, cy, spriteW float64) {
                r := spriteW * 0.45
                if r < 30 {
                        r = 30
                }
                // Outer golden glow (high alpha so it's clearly visible over dark backgrounds)
                dc.SetColor(color.RGBA{255, 215, 0, 220})
                dc.DrawEllipse(cx, cy, r, r*0.4)
                dc.Fill()
                // Inner bright ring (fully opaque, warm white-gold)
                dc.SetColor(color.RGBA{255, 255, 200, 255})
                dc.DrawEllipse(cx, cy, r*0.8, r*0.32)
                dc.Fill()
        }
        // isAttacker checks if the entity at (side, index) is the active attacker.
        isAttacker := func(side string, index int) bool {
                if req.Action == nil {
                        return false
                }
                return req.Action.AttackerSide == side && req.Action.AttackerIndex == index
        }

        // 2. Mobs / Enemies
        // 💡 FIX 2026-08-07: Feet-anchor + ground zone (Spec 1A).
        // Enemies are on the RIGHT side, feet in ground zone (HORIZON_Y to GROUND_BOTTOM_Y).
        // Main enemy (index 0): feet at MAIN_FEET_Y (lower = closer to camera).
        // Background enemies (index 1+): feet at BACK_FEET_Y (higher = further away, depth cue).
        enemySpriteSize := 190.0
        enemyStartX := 800.0 // center X for main enemy
        spX := 120.0         // horizontal formation spacing

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
                origIndex int // 💡 carries the original enemy index through the Y-sort
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
                // 💡 FIX 2026-08-07 (#2): Crop transparent padding so feet sit on ground
                eSprite = cropToVisibleBounds(eSprite)

                // 💡 FIX 2026-08-07: Side-based facing — enemies on RIGHT side face LEFT (Spec 1C)
                if flipForSide(filepath.Base(spritePath), false) {
                        eSprite = imaging.FlipH(eSprite)
                }

                // Tint Red if dead
                if enemy.CurrentHP <= 0 {
                        eSprite = utils.TintImage(eSprite, color.RGBA{80, 0, 80, 180})
                }

                // 💡 FIX 2026-08-07: Feet-anchor positioning in ground zone.
                // feetX = horizontal center, feetY = ground line for this entity.
                // Main enemy (sub 0): feet at MAIN_FEET_Y. Others: BACK_FEET_Y (depth).
                //
                // 💡 FIX 2026-08-08: Formation overlap bug — sub=1 and sub=2 were
                // both at feetX-spX (same X), causing 2 of 3 enemies to overlap.
                // Now uses a 2x2 grid: sub=0 front-center, sub=1 back-left,
                // sub=2 back-right, sub=3 far-back-center.
                feetX := enemyStartX
                feetY := float64(MAIN_FEET_Y)
                sub := i % 4
                switch sub {
                case 1:
                        feetX -= spX // back-left
                case 2:
                        feetX += spX // back-right (was -= spX, overlapping sub=1)
                case 3:
                        feetX -= spX * 2 // far-back-left
                }
                if sub > 0 {
                        feetY = float64(BACK_FEET_Y) // background enemies higher (depth)
                }
                feetX += float64(i/4) * -250.0

                // Convert feet position to draw position (top-left anchor for DrawImage)
                spriteW := eSprite.Bounds().Dx()
                spriteH := eSprite.Bounds().Dy()
                ex := feetX - float64(spriteW)/2
                ey := feetY - float64(spriteH)

                hpPerc := 0.0
                if enemy.MaxHP > 0 {
                        hpPerc = float64(enemy.CurrentHP) / float64(enemy.MaxHP)
                }
                mobQueue = append(mobQueue, RenderItem{eSprite, ex, ey, hpPerc, i})
        }
        // Sort by Y (Painter's Algorithm)
        sort.Slice(mobQueue, func(i, j int) bool {
                return mobQueue[i].y < mobQueue[j].y
        })

        // Draw Mobs
        for _, mob := range mobQueue {
                // Shadow (drawn FIRST, under everything) — at feet position
                utils.DrawShadow(dc, mob.x+float64(mob.img.Bounds().Dx())/2, mob.y+float64(mob.img.Bounds().Dy())-2, float64(mob.img.Bounds().Dx())*0.6, 0.85)
                // 💡 TURN INDICATOR: draw golden ellipse ON TOP of shadow, UNDER sprite.
                // Must be after shadow so the golden circle is visible (shadow would cover it otherwise).
                if isAttacker("enemy", mob.origIndex) {
                        cx := mob.x + float64(mob.img.Bounds().Dx())/2
                        cy := mob.y + float64(mob.img.Bounds().Dy()) - 5
                        drawTurnIndicator(cx, cy, float64(mob.img.Bounds().Dx()))
                }
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
        // 💡 FIX 2026-08-07: Summon placement per Spec 1B + 1D.
        // - Positioned relative to player (X offset, feet Y slightly lower = in front)
        // - Scaled to ~85% of player sprite size (not fixed enemy-relative size)
        // - Drawn BEFORE players (z-order: behind owner)
        // - Facing: inherits owner's side (left side → face RIGHT)
        // 💡 FIX 2026-08-07 (#4): playerCenterX=500 — sprite X range 439→561
        // is FULLY outside the player_state.png panel (X:-22→431).
        // Zero overlap, not masked by z-order. Gap to enemy at X=800 is 300px.
        playerCenterX := 500.0 // player formation center X (past UI panel, left of center)
        playerSpriteW := 122   // player PvE sprite width
        summonSpriteW := int(float64(playerSpriteW) * 0.85) // 85% of player size

        for i, summon := range req.Summons {
                if summon.CurrentHP <= 0 && !summon.JustDied {
                        continue
                }

                spritePath := GetSummonSpritePath(summon.Species, assetsPath)
                sSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }

                // Resize — 85% of player sprite size (Spec 1B)
                sSprite = imaging.Resize(sSprite, summonSpriteW, 0, imaging.NearestNeighbor)
                // 💡 FIX 2026-08-07 (#2): Crop transparent padding so feet sit on ground
                sSprite = cropToVisibleBounds(sSprite)

                // 💡 FIX 2026-08-07 (#5): Summon facing via flipForSide.
                // Summon sprites now have facing directions in SummonSpriteFacing map.
                // flipForSide checks the map and flips only if needed.
                if strings.Contains(summon.Species, "ship_") {
                        sSprite = imaging.Rotate90(sSprite)
                } else {
                        summonSpriteFile := filepath.Base(GetSummonSpritePath(summon.Species, assetsPath))
                        if flipForSide(summonSpriteFile, true) {
                                sSprite = imaging.FlipH(sSprite)
                        }
                }

                // Tint red if dead
                if summon.CurrentHP <= 0 {
                        sSprite = utils.TintImage(sSprite, color.RGBA{80, 0, 80, 180})
                }

                // 💡 FIX 2026-08-08 (#3): Summon staggered behind player, not stacked on top.
                // BEFORE: summonFeetX = playerCenterX - 80 (only 80px offset). With sprite
                // widths ~103 (summon) + 122 (player), the half-widths sum to ~112px.
                // 80px < 112px → sprites overlapped horizontally.
                //
                // AFTER: 150px offset (summonFeetX = 500 - 150 = 350). Sprite ranges:
                //   Player: 439-561 (width 122, center 500)
                //   Summon: 299-401 (width 103, center 350)
                //   Gap between summon right edge (401) and player left edge (439) = 38px. ✅
                // Combined with BACK_FEET_Y=410 (vs player at 505), the summon is clearly
                // staggered behind and to the left — depth + horizontal separation.
                summonFeetX := playerCenterX - 150
                summonFeetY := float64(BACK_FEET_Y) // higher = further back (depth)
                // Formation offset for multiple summons
                sub := i % 4
                if sub == 1 || sub == 2 {
                        summonFeetX -= float64(spX) * 0.8
                } else if sub == 3 {
                        summonFeetX -= float64(spX) * 1.6
                }
                if sub > 0 {
                        summonFeetY = float64(BACK_FEET_Y) - 15 // background summons even higher
                }

                // Convert feet to draw position
                sSpriteW := sSprite.Bounds().Dx()
                sSpriteH := sSprite.Bounds().Dy()
                sx := summonFeetX - float64(sSpriteW)/2
                sy := summonFeetY - float64(sSpriteH)

                // Shadow at feet
                utils.DrawShadow(dc, summonFeetX, summonFeetY-2, float64(sSpriteW)*0.6, 0.85)

                // Turn indicator
                if isAttacker("summon", i) {
                        drawTurnIndicator(summonFeetX, summonFeetY-5, float64(sSpriteW))
                }

                // Sprite
                dc.DrawImage(sSprite, int(sx), int(sy))

                // HP bar
                if summon.MaxHP > 0 && summon.CurrentHP > 0 {
                        uiPath2 := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }
                        hpBarImg, err := utils.LoadImage(uiPath2("hp5.png"))
                        if err == nil {
                                barW := 70.0
                                barH := 10.0
                                hpPerc := float64(summon.CurrentHP) / float64(summon.MaxHP)
                                currentBarW := int(barW * hpPerc)
                                if currentBarW < 1 {
                                        currentBarW = 1
                                }
                                hpBarImg = imaging.Resize(hpBarImg, currentBarW, int(barH), imaging.NearestNeighbor)
                                bx := summonFeetX - barW/2
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
        // 💡 FIX 2026-08-08: PvP now draws a MIRRORED player_state panel on the right
        // side for player 2 (was missing entirely). Options_menu.png is PvE-only.
        // In PvE: left panel (player_state) + right panel (Options_menu action buttons).
        // In PvP: left panel (player 1 state) + right panel (player 2 state, mirrored).
        drawImage(uiPath("player_state.png"), -716, 113, 453, 244)
        drawImage(uiPath("heart.png"), -678, 209, 38, 47)
        drawImage(uiPath("mana.png"), -673, 256, 29, 44)
        if req.CombatType == "PVP" && len(req.Players) > 1 {
                // 💡 FIX 2026-08-08 #1: Right-side player 2 state panel (was missing).
                // Mirrors the left panel: same player_state.png, flipped horizontally.
                // Position: normX(-123) = -123 + 694 = 571. Width 453 → X=571 to 1024.
                // (22px clipped off right edge, same as left panel clips 22px off left.)
                panelImg, err := utils.LoadImage(uiPath("player_state.png"))
                if err == nil {
                        panelImg = imaging.Resize(panelImg, 453, 244, imaging.NearestNeighbor)
                        panelImg = imaging.FlipH(panelImg) // mirror for right side
                        dc.DrawImage(panelImg, normX(-123), normY(113))
                }
                // Heart icon (flipped position — on right side of panel)
                heartImg, err := utils.LoadImage(uiPath("heart.png"))
                if err == nil {
                        heartImg = imaging.Resize(heartImg, 38, 47, imaging.NearestNeighbor)
                        dc.DrawImage(heartImg, normX(-89), normY(209))
                }
                // Mana icon
                manaImg, err := utils.LoadImage(uiPath("mana.png"))
                if err == nil {
                        manaImg = imaging.Resize(manaImg, 29, 44, imaging.NearestNeighbor)
                        dc.DrawImage(manaImg, normX(-84), normY(256))
                }
        } else if req.CombatType != "PVP" {
                drawImage(uiPath("Options_menu.png"), -97, 99, 443, 258)
        }
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
                // 💡 FIX 2026-08-05: Use summon sprite if player is a summon (mode='summon')
                var spritePath string
                if p.Mode == "summon" && p.Species != "" {
                        spritePath = GetSummonSpritePath(p.Species, assetsPath)
                } else {
                        spritePath = GetCharacterSpritePath(p.Class, p.SpriteIndex, assetsPath)
                }
                pSprite, err := utils.LoadImage(spritePath)
                if err == nil {
                        if p.CurrentHP <= 0 {
                                pSprite = utils.TintImage(pSprite, color.RGBA{80, 0, 80, 180})
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

                        // 💡 FIX 2026-08-08 #1: Player 2 portrait + HP/EN bars (PvP only).
                        // Mirrors player 1's portrait on the right-side panel.
                        if req.CombatType == "PVP" && len(req.Players) > 1 {
                                p2 := req.Players[1]

                                // Player 2 HP/EN bars (mirrored X coordinates)
                                // Player 1 bar X coords: -640, -550, -459 (3 segments)
                                // Player 2 mirrored: negate and shift: 640-694=... actually
                                // mirror around canvas center (512). Player 1 at normX(-640)=54,
                                // mirror to 1024-54=970, which is normX(970-694)=normX(276).
                                // But the right panel is at X=571-1024. Bars should be within that.
                                // Right panel inner area: X~600 to ~1010.
                                // 3 segments at X=620, 730, 820 (roughly mirroring left's 54, 144, 235)
                                p2HpCoords := []int{326, 236, 145}  // normX(326)=1020, normX(236)=930, normX(145)=839
                                p2EnCoords := []int{322, 232, 141}
                                p2HpSeg := float64(p2.MaxHP) / 3.0
                                p2EnSeg := float64(p2.MaxEnergy) / 3.0

                                for i := 0; i < 3; i++ {
                                        hCur := math.Max(0, math.Min(p2HpSeg, float64(p2.CurrentHP)-(float64(i)*p2HpSeg)))
                                        drawBar(dc, uiPath, normX(p2HpCoords[i]), normY(209), hCur, p2HpSeg, "hp", 121, 47)

                                        eCur := math.Max(0, math.Min(p2EnSeg, float64(p2.Energy)-(float64(i)*p2EnSeg)))
                                        drawBar(dc, uiPath, normX(p2EnCoords[i]), normY(256), eCur, p2EnSeg, "mana", 119, 42)
                                }

                                // Player 2 portrait (cropped top 30%, flipped)
                                var p2SpritePath string
                                if p2.Mode == "summon" && p2.Species != "" {
                                        p2SpritePath = GetSummonSpritePath(p2.Species, assetsPath)
                                } else {
                                        p2SpritePath = GetCharacterSpritePath(p2.Class, p2.SpriteIndex, assetsPath)
                                }
                                p2Sprite, err := utils.LoadImage(p2SpritePath)
                                if err == nil {
                                        if p2.CurrentHP <= 0 {
                                                p2Sprite = utils.TintImage(p2Sprite, color.RGBA{80, 0, 80, 180})
                                        }
                                        p2Sprite = imaging.Resize(p2Sprite, 314, 0, imaging.NearestNeighbor)
                                        p2Bounds := p2Sprite.Bounds()
                                        p2CropH := int(float64(p2Bounds.Dy()) * 0.3)
                                        p2Cropped := imaging.Crop(p2Sprite, image.Rect(0, 0, p2Bounds.Dx(), p2CropH))
                                        p2Cropped = imaging.FlipH(p2Cropped) // mirror for right side
                                        // Position: mirror of player 1's normX(-660)=34.
                                        // Mirror: 1024-34-314 = 676. normX(676-694) = normX(-18).
                                        // Actually player 1 at normX(-660)=34, width 314 → X=34-348.
                                        // Player 2 mirror: X=1024-348 to 1024-34 = 676-990.
                                        // normX(-18) = 676. So use normX(-18).
                                        dc.DrawImage(p2Cropped, normX(-18), normY(220)-p2CropH+40)
                                }
                        }

                        // 6. Small full-body sprites on battlefield
                        if req.CombatType == "PVP" {
                                // 💡 FIX 2026-08-08: Unified PvP positioning with PvE.
                                // BEFORE: PvP used X=300/690 with sprite width=160 — player 1 at X=300
                                // overlapped the left UI panel (X=-22 to 431). PvP summon sprites were
                                // 160px (different from PvE's 103px). These separate constants caused
                                // the "fixed in PvE but not PvP" pattern.
                                //
                                // AFTER: PvP now uses the SAME constants as PvE:
                                //   - Player 1: X=500 (same as PvE playerCenterX, outside left panel)
                                //   - Player 2: X=720 (outside right panel, which is skipped in PvP)
                                //   - Sprite width: 122px humans, 103px summons (85% per Spec 1B)
                                //   - Feet Y: MAIN_FEET_Y=505 (grounded)
                                //
                                // Options_menu.png (right UI panel) is now PvP-only-skipped below
                                // so player 2 can use the right half of the canvas without overlap.
                                drawPvPFighter := func(player Player, feetX, feetY int, flip bool, isAtk bool) {
                                        var path string
                                        if player.Mode == "summon" && player.Species != "" {
                                                path = GetSummonSpritePath(player.Species, assetsPath)
                                        } else {
                                                path = GetCharacterSpritePath(player.Class, player.SpriteIndex, assetsPath)
                                        }
                                        sprite, err := utils.LoadImage(path)
                                        if err != nil {
                                                return
                                        }
                                        if player.CurrentHP <= 0 {
                                                sprite = utils.TintImage(sprite, color.RGBA{80, 0, 80, 180})
                                        }
                                        // 💡 FIX 2026-08-08: Match PvE sprite sizes.
                                        // Humans: 122px (same as PvE playerSpriteW).
                                        // Summons: 103px (85% of 122, same as PvE summonSpriteW per Spec 1B).
                                        var spriteW int
                                        if player.Mode == "summon" && player.Species != "" {
                                                spriteW = int(float64(playerSpriteW) * 0.85) // 103 — matches PvE summon
                                        } else {
                                                spriteW = playerSpriteW // 122 — matches PvE player
                                        }
                                        sprite = imaging.Resize(sprite, spriteW, 0, imaging.NearestNeighbor)
                                        sprite = cropToVisibleBounds(sprite)
                                        if flip {
                                                sprite = imaging.FlipH(sprite)
                                        }

                                        // Convert feet to draw position
                                        sW := sprite.Bounds().Dx()
                                        sH := sprite.Bounds().Dy()
                                        drawX := feetX - sW/2
                                        drawY := feetY - sH

                                        // Shadow at feet
                                        utils.DrawShadow(dc, float64(feetX), float64(feetY)-2, float64(sW)*0.6, 0.85)

                                        if isAtk {
                                                drawTurnIndicator(float64(feetX), float64(feetY)-5, float64(sW))
                                        }

                                        dc.DrawImage(sprite, drawX, drawY)

                                        hpPercent := 0.0
                                        if player.MaxHP > 0 {
                                                hpPercent = math.Max(0, math.Min(1, float64(player.CurrentHP)/float64(player.MaxHP)))
                                        }
                                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                                        if err == nil && hpPercent > 0 {
                                                barW := int(100 * hpPercent)
                                                if barW < 1 {
                                                        barW = 1
                                                }
                                                hpBarImg = imaging.Resize(hpBarImg, barW, 10, imaging.NearestNeighbor)
                                                dc.DrawImage(hpBarImg, drawX+10, drawY-15)
                                        }
                                }

                                // 💡 FIX 2026-08-08: PvP positions unified with PvE.
                                // Player 1 (left): X=500 — same as PvE playerCenterX.
                                //   Sprite range: 439-561. Left panel ends at 431. Gap=8px. ✅
                                // Player 2 (right): X=720 — Options_menu skipped in PvP.
                                //   Sprite range: 659-781. No right panel to overlap. ✅
                                var p1Flip bool
                                if p.Mode == "summon" && p.Species != "" {
                                        p1SpriteFile := filepath.Base(GetSummonSpritePath(p.Species, assetsPath))
                                        p1Flip = flipForSide(p1SpriteFile, true)
                                } else {
                                        p1SpriteFile := GetCharacterSpriteFile(p.Class, p.SpriteIndex, assetsPath)
                                        p1Flip = flipForSide(filepath.Base(p1SpriteFile), true)
                                }
                                drawPvPFighter(p, 500, MAIN_FEET_Y, p1Flip, isAttacker("player", 0))
                                if len(req.Players) > 1 {
                                        var p2Flip bool
                                        if req.Players[1].Mode == "summon" && req.Players[1].Species != "" {
                                                p2SpriteFile := filepath.Base(GetSummonSpritePath(req.Players[1].Species, assetsPath))
                                                p2Flip = flipForSide(p2SpriteFile, false)
                                        } else {
                                                p2SpriteFile := GetCharacterSpriteFile(req.Players[1].Class, req.Players[1].SpriteIndex, assetsPath)
                                                p2Flip = flipForSide(filepath.Base(p2SpriteFile), false)
                                        }
                                        drawPvPFighter(req.Players[1], 720, MAIN_FEET_Y, p2Flip, isAttacker("player", 1))
                                }
                        } else {
                                // PVE: draw ALL players in formation on the battlefield.
                                // 💡 FIX 2026-08-07: Feet-anchor + ground zone (Spec 1A/1C).
                                // Players on LEFT side, feet in ground zone, face RIGHT.
                                s2Size := 122
                                var smallSprite image.Image = imaging.Resize(pSprite, s2Size, 0, imaging.NearestNeighbor)
                                // 💡 FIX 2026-08-07 (#2): Crop transparent padding so feet sit on ground
                                smallSprite = cropToVisibleBounds(smallSprite)
                                pSpriteFile := GetCharacterSpriteFile(p.Class, p.SpriteIndex, assetsPath)
                                var flippedSprite image.Image
                                if flipForSide(pSpriteFile, true) {
                                        flippedSprite = imaging.FlipH(smallSprite)
                                } else {
                                        flippedSprite = smallSprite
                                }

                                // Player[0] — feet at (playerCenterX, MAIN_FEET_Y)
                                p0FeetX := int(playerCenterX)
                                p0FeetY := MAIN_FEET_Y
                                p0SpriteH := flippedSprite.Bounds().Dy()
                                s2X := p0FeetX - s2Size/2
                                s2Y := p0FeetY - p0SpriteH

                                // Shadow at feet
                                utils.DrawShadow(dc, float64(p0FeetX), float64(p0FeetY)-2, 170, 0.85)

                                if isAttacker("player", 0) {
                                        drawTurnIndicator(float64(p0FeetX), float64(p0FeetY)-5, float64(s2Size))
                                }

                                dc.DrawImage(flippedSprite, s2X, s2Y)

                                // HP bar for player[0]
                                if p.MaxHP > 0 && p.CurrentHP > 0 {
                                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                                        if err == nil {
                                                hpPerc := float64(p.CurrentHP) / float64(p.MaxHP)
                                                barW := int(80.0 * hpPerc)
                                                if barW < 1 { barW = 1 }
                                                hpBarImg = imaging.Resize(hpBarImg, barW, 10, imaging.NearestNeighbor)
                                                dc.DrawImage(hpBarImg, p0FeetX-40, s2Y-12)
                                        }
                                }

                                // Additional players (1+) in formation
                                for pi := 1; pi < len(req.Players); pi++ {
                                        ap := req.Players[pi]
                                        if ap.CurrentHP <= 0 { continue }
                                        apPath := GetCharacterSpritePath(ap.Class, ap.SpriteIndex, assetsPath)
                                        apSprite, err := utils.LoadImage(apPath)
                                        if err != nil { continue }
                                        var apResized image.Image = imaging.Resize(apSprite, s2Size, 0, imaging.NearestNeighbor)
                                        // 💡 FIX 2026-08-07 (#2): Crop transparent padding so feet sit on ground
                                        apResized = cropToVisibleBounds(apResized)
                                        apSpriteFile := GetCharacterSpriteFile(ap.Class, ap.SpriteIndex, assetsPath)
                                        var apFlipped image.Image
                                        if flipForSide(apSpriteFile, true) {
                                                apFlipped = imaging.FlipH(apResized)
                                        } else {
                                                apFlipped = apResized
                                        }

                                        // Formation: feet-anchor, background players higher (depth)
                                        apFeetX := int(playerCenterX)
                                        apFeetY := BACK_FEET_Y
                                        sub := pi % 4
                                        if sub == 1 || sub == 2 { apFeetX -= 120 } else if sub == 3 { apFeetX -= 240 }
                                        apFeetX += int(float64(pi/4) * -250.0)
                                        apSpriteH := apFlipped.Bounds().Dy()
                                        apX := apFeetX - s2Size/2
                                        apY := apFeetY - apSpriteH

                                        utils.DrawShadow(dc, float64(apFeetX), float64(apFeetY)-2, 170, 0.85)

                                        if isAttacker("player", pi) {
                                                drawTurnIndicator(float64(apFeetX), float64(apFeetY)-5, float64(s2Size))
                                        }

                                        dc.DrawImage(apFlipped, apX, apY)

                                        if ap.MaxHP > 0 && ap.CurrentHP > 0 {
                                                hpBarImg2, err := utils.LoadImage(uiPath("hp5.png"))
                                                if err == nil {
                                                        hpPerc2 := float64(ap.CurrentHP) / float64(ap.MaxHP)
                                                        barW2 := int(80.0 * hpPerc2)
                                                        if barW2 < 1 { barW2 = 1 }
                                                        hpBarImg2 = imaging.Resize(hpBarImg2, barW2, 10, imaging.NearestNeighbor)
                                                        dc.DrawImage(hpBarImg2, apFeetX-40, apY-12)
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
                // 💡 FIX 2026-08-07: Moved subtitle from Y=300 to Y=375 to prevent
                // overlap with the 110px title (which spans ~165-275 at Y=220 center).
                dc.DrawStringAnchored(subtitleText, float64(CANVAS_W/2), 375, 0.5, 0.5)
        }

        // Rewards panel (victory only)
        if req.Victory && (req.Gold > 0 || req.XP > 0 || req.Items != "") {
                panelY := 420.0
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

// getRankBackground returns a deterministic background based on the dungeon rank.
// Same rank → same background every time. No randomness.
// Uses sparklinlabs backgrounds for variety + original backgrounds for higher ranks.
func getRankBackground(rank string, assetsPath string) string {
        envPath := filepath.Join(assetsPath, "rpgasset", "environment")
        var filename string
        switch strings.ToUpper(rank) {
        case "F":
                filename = "spark_1.png"      // Grassland
        case "E":
                filename = "spark_2.png"      // Forest
        case "D":
                filename = "spark_3.png"      // Cave
        case "C":
                filename = "spark_4.png"      // Desert
        case "B":
                filename = "spark_5.png"      // Snow
        case "A":
                filename = "spark_6.png"      // Volcano
        case "S", "SS":
                filename = "spark_7.png"      // Dark castle
        case "SSS":
                filename = "spark_8.png"      // Abyss
        case "GOD":
                filename = "spark_10.png"     // Divine realm
        case "PVP":
                filename = "spark_15.png"     // Arena
        case "ABYSS":
                filename = "spark_1-night.png" // Night version for Abyss
        case "TRIAL":
                filename = "spark_3-night.png" // Night cave for trials
        default:
                filename = "spark_1.png"
        }
        path := filepath.Join(envPath, filename)
        if fileExists(path) {
                return path
        }
        // Fallback: try original backgrounds
        fallbacks := []string{"forest.png", "env1.png", "background1.png"}
        for _, fb := range fallbacks {
                fbPath := filepath.Join(envPath, fb)
                if fileExists(fbPath) {
                        return fbPath
                }
        }
        // Last resort: first available file
        entries, err := os.ReadDir(envPath)
        if err != nil {
                return ""
        }
        for _, entry := range entries {
                if !entry.IsDir() {
                        ext := filepath.Ext(entry.Name())
                        if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
                                return filepath.Join(envPath, entry.Name())
                        }
                }
        }
        return ""
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
