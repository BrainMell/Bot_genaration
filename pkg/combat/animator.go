package combat

// ============================================
// ⚔️ ANIMATED COMBAT RENDERER
// ============================================
// Produces a short MP4 video showing a combat action with:
//   • VFX overlays (sword slash, fireball, lightning, dark bolt, holy)
//   • Target sprite reaction (shake + red flash)
//   • HP bar interpolation (drains smoothly across frames)
//   • Attacker windup (forward translate + return)
//   • Defeated sprite fade-out (when target dies)
//
// Pipeline:
//   Node sends CombatRequest with Action field populated
//     → Go renders N frames to a temp dir as PNGs
//     → ffmpeg encodes frames → MP4 (libx264, yuv420p, +faststart)
//     → MP4 buffer returned to Node
//     → Node sends video via WhatsApp
//
// Frame plan (12 frames @ 10 FPS = 1.2s):
//   0   idle
//   1   windup start   (attacker +5px forward)
//   2   windup peak    (attacker +10px forward)
//   3   VFX frame 1
//   4   VFX frame 2 + target shake begin + red flash 30%
//   5   VFX frame 3 + shake + flash 50%
//   6   VFX frame 4 + shake + flash 50% + HP drain 25%
//   7   VFX frame 5 + shake + flash 40% + HP drain 50%
//   8   VFX frame 6 + shake + flash 30% + HP drain 75%
//   9   VFX frame 7 + flash 20% + HP drain 100% (final HP)
//   10  settle start (no VFX, no shake, flash 10%)
//   11  settle end   (everything back to normal)

import (
        "bytes"
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
        ANIM_FRAME_COUNT = 12
        ANIM_FPS         = 10
)

// frameState holds per-frame variations applied on top of the base scene.
type frameState struct {
        // Attacker translation (windup). Positive = forward.
        // For player-side attackers, forward = +x (right). For enemy-side, forward = -x (left).
        attackerDX float64
        attackerDY float64

        // Target shake (impact). Positive/negative alternates per frame.
        targetShakeX float64
        targetShakeY float64

        // Target red flash (impact). 0.0 = no flash, 1.0 = full red.
        targetRedFlash float64

        // VFX frame to overlay on target (1-indexed). 0 = no VFX.
        vfxFrame int
        vfxPath  string // resolved VFX folder

        // HP overrides. When non-nil, override the entity's CurrentHP for this frame.
        // Used to interpolate the target's HP bar from pre-action → post-action value.
        targetHPOverride *int

        // Defeated fade (when target dies this action). 0.0 = full opacity, 1.0 = invisible.
        targetFade float64
}

// VFX folder mapping per element
var elementVfxFolder = map[string]string{
        "physical":  "attack",       // attack1-7.png (sword slash)
        "fire":      "Fire-bomb",    // Fire-bomb1-15.png
        "lightning": "Lightning",    // Lightning1-11.png
        "dark":      "Dark-Bolt",    // Dark-Bolt1-12.png
        "holy":      "Holy VFX 02",  // Holy VFX 02 1-16.png
        "ice":       "Holy VFX 02",  // re-use holy (looks cold/blue-white)
        "none":      "",             // no VFX (buffs, debuffs, healing)
}

// VFX frame count per folder (for modular arithmetic)
var vfxFrameCount = map[string]int{
        "attack":       7,
        "Fire-bomb":    15,
        "Lightning":    11,
        "Dark-Bolt":    12,
        "Holy VFX 02":  16,
}

// GenerateAnimatedCombat renders a multi-frame MP4 of a combat action.
// Falls back to a static PNG if Action is nil or ffmpeg fails.
func GenerateAnimatedCombat(c *gin.Context) {
        var req CombatRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        // If no action, fall back to static PNG
        if req.Action == nil {
                GenerateCombatImage(c)
                return
        }

        frames, err := renderAnimationFrames(&req)
        if err != nil {
                // Fallback to static on any error
                fmt.Printf("[Animator] renderAnimationFrames failed: %v — falling back to static\n", err)
                GenerateCombatImage(c)
                return
        }

        // Encode frames → MP4 via ffmpeg
        mp4Buf, err := encodeFramesToMP4(frames)
        if err != nil {
                fmt.Printf("[Animator] encodeFramesToMP4 failed: %v — falling back to static\n", err)
                GenerateCombatImage(c)
                return
        }

        c.Data(200, "video/mp4", mp4Buf)
}

// renderAnimationFrames produces ANIM_FRAME_COUNT image.Image frames.
func renderAnimationFrames(req *CombatRequest) ([]image.Image, error) {
        assetsPath := "assets"
        frames := make([]image.Image, ANIM_FRAME_COUNT)

        // Resolve VFX folder for the action's element
        vfxFolder := ""
        if req.Action.Vfx != "" {
                vfxFolder = req.Action.Vfx
        } else if f, ok := elementVfxFolder[req.Action.Element]; ok {
                vfxFolder = f
        }

        // Pre-compute target's pre-action HP (post-action HP + damage, or - heal)
        preActionHP := 0
        postActionHP := 0
        targetSide := req.Action.TargetSide
        targetIdx := req.Action.TargetIndex
        if targetSide == "enemy" && targetIdx < len(req.Enemies) {
                postActionHP = req.Enemies[targetIdx].CurrentHP
                if req.Action.Damage > 0 {
                        preActionHP = postActionHP + req.Action.Damage
                } else if req.Action.Heal > 0 {
                        preActionHP = postActionHP - req.Action.Heal
                        if preActionHP < 0 {
                                preActionHP = 0
                        }
                } else {
                        preActionHP = postActionHP
                }
                // Cap at MaxHP
                if preActionHP > req.Enemies[targetIdx].MaxHP {
                        preActionHP = req.Enemies[targetIdx].MaxHP
                }
        } else if targetSide == "player" && targetIdx < len(req.Players) {
                postActionHP = req.Players[targetIdx].CurrentHP
                if req.Action.Damage > 0 {
                        preActionHP = postActionHP + req.Action.Damage
                } else if req.Action.Heal > 0 {
                        preActionHP = postActionHP - req.Action.Heal
                        if preActionHP < 0 {
                                preActionHP = 0
                        }
                } else {
                        preActionHP = postActionHP
                }
                if preActionHP > req.Players[targetIdx].MaxHP {
                        preActionHP = req.Players[targetIdx].MaxHP
                }
        } else if targetSide == "summon" && targetIdx < len(req.Summons) {
                postActionHP = req.Summons[targetIdx].CurrentHP
                if req.Action.Damage > 0 {
                        preActionHP = postActionHP + req.Action.Damage
                } else if req.Action.Heal > 0 {
                        preActionHP = postActionHP - req.Action.Heal
                        if preActionHP < 0 {
                                preActionHP = 0
                        }
                } else {
                        preActionHP = postActionHP
                }
                if preActionHP > req.Summons[targetIdx].MaxHP {
                        preActionHP = req.Summons[targetIdx].MaxHP
                }
        }

        // Pre-compute whether target dies this action (for fade-out)
        targetDies := postActionHP <= 0 && preActionHP > 0

        // Attacker forward direction (player side attacks rightward → +x, enemy side attacks leftward → -x)
        attackerForwardX := -1.0 // default: enemy attacker moves left (toward player)
        if req.Action.AttackerSide == "player" || req.Action.AttackerSide == "summon" {
                attackerForwardX = 1.0 // player/summon attacker moves right (toward enemy)
        }

        // Per-frame state plan (see comment block at top of file)
        frameStates := make([]frameState, ANIM_FRAME_COUNT)
        for i := range frameStates {
                fs := frameState{}

                // Windup (frames 1-2)
                if i == 1 {
                        fs.attackerDX = 5 * attackerForwardX
                } else if i == 2 {
                        fs.attackerDX = 10 * attackerForwardX
                        fs.attackerDY = -3
                } else if i >= 3 && i <= 6 {
                        // Return from windup
                        fs.attackerDX = (10 - float64(i-2)*3) * attackerForwardX
                }

                // VFX (frames 3-9, 7 VFX frames cycled)
                if vfxFolder != "" && i >= 3 && i <= 9 {
                        fs.vfxPath = vfxFolder
                        fs.vfxFrame = i - 2 // frames 3-9 → VFX frames 1-7
                        maxFrames := vfxFrameCount[vfxFolder]
                        if maxFrames > 0 {
                                fs.vfxFrame = ((fs.vfxFrame - 1) % maxFrames) + 1
                        }
                }

                // Shake + red flash (frames 4-9)
                if i >= 4 && i <= 9 && req.Action.Damage > 0 && !req.Action.Missed {
                        // Shake amplitude: 8px, alternates direction
                        amplitude := 8.0
                        if i >= 7 {
                                amplitude = 4.0
                        }
                        fs.targetShakeX = amplitude * math.Sin(float64(i)*math.Pi)
                        fs.targetShakeY = amplitude * 0.3 * math.Cos(float64(i)*math.Pi)

                        // Red flash: peaks at frame 5-6, fades by frame 9
                        switch {
                        case i == 4:
                                fs.targetRedFlash = 0.30
                        case i == 5 || i == 6:
                                fs.targetRedFlash = 0.50
                        case i == 7:
                                fs.targetRedFlash = 0.40
                        case i == 8:
                                fs.targetRedFlash = 0.30
                        case i == 9:
                                fs.targetRedFlash = 0.15
                        }
                }

                // HP drain (frames 6-9): linear from preActionHP → postActionHP
                if i >= 6 && i <= 9 && req.Action.Damage > 0 && !req.Action.Missed {
                        progress := float64(i-6) / 3.0 // 0.0 at frame 6, 1.0 at frame 9
                        currentHP := int(float64(preActionHP) + float64(postActionHP-preActionHP)*progress)
                        fs.targetHPOverride = &currentHP
                } else if i >= 9 {
                        // Frame 9+: final HP
                        fs.targetHPOverride = &postActionHP
                }

                // Defeated fade-out (frames 10-11)
                if targetDies && i >= 10 {
                        if i == 10 {
                                fs.targetFade = 0.5
                        } else if i == 11 {
                                fs.targetFade = 1.0
                        }
                }

                frameStates[i] = fs
        }

        // Render each frame
        for i, fs := range frameStates {
                frame, err := renderCombatFrame(req, &fs, assetsPath)
                if err != nil {
                        return nil, fmt.Errorf("frame %d: %w", i, err)
                }
                frames[i] = frame
        }

        return frames, nil
}

// renderCombatFrame renders a single combat scene with the given per-frame state.
// This is the layout-fixed version: player side uses the SAME formation as enemy side.
func renderCombatFrame(req *CombatRequest, fs *frameState, assetsPath string) (image.Image, error) {
        dc := gg.NewContext(CANVAS_W, CANVAS_H)

        // ── Background ──
        var bgPath string
        if req.Background != "" {
                if filepath.IsAbs(req.Background) || fileExists(req.Background) {
                        bgPath = req.Background
                } else {
                        bgPath = filepath.Join(assetsPath, "rpgasset", "environment", req.Background)
                }
        }
        if bgPath == "" || !fileExists(bgPath) {
                bgPath = getRandomEnvironment(assetsPath)
        }
        if bgPath != "" && fileExists(bgPath) {
                if bgImg, err := utils.LoadImage(bgPath); err == nil {
                        bgImg = imaging.Fill(bgImg, CANVAS_W, CANVAS_H, imaging.Center, imaging.NearestNeighbor)
                        dc.DrawImage(bgImg, 0, 0)
                } else {
                        dc.SetHexColor("#1a1a1a")
                        dc.Clear()
                }
        } else {
                dc.SetHexColor("#1a1a1a")
                dc.Clear()
        }

        // Dark overlay
        dc.SetColor(color.RGBA{0, 0, 0, 102})
        dc.DrawRectangle(0, 0, CANVAS_W, CANVAS_H)
        dc.Fill()

        // ── Compute entity positions (shared by both sides for consistency) ──
        // Layout: same as enemy formation, 4 per row zigzag, mirrored for left side.
        enemySpriteSize := 190.0
        playerSpriteSize := 170.0
        summonSpriteSize := 152.0

        // Enemy anchor (right side)
        enemyAnchorX, enemyAnchorY := 780.0, 160.0
        spX, spY := 130.0, 110.0

        // Player anchor (left side) — same Y as enemies for symmetry
        playerAnchorX := enemyAnchorX - 560.0
        playerAnchorY := enemyAnchorY

        // Summon anchor — BESIDE players as own entity, not stacked behind
        summonAnchorX := playerAnchorX + 210.0 // 210px right of players = between players and enemies
        summonAnchorY := enemyAnchorY          // same Y for visual symmetry

        avgLevel := 1
        if len(req.Players) > 0 {
                sum := 0
                for _, p := range req.Players {
                        sum += p.Level
                }
                avgLevel = sum / len(req.Players)
        }

        // Compute per-entity render items (sprite, position, hp%, optional override)
        type renderItem struct {
                img        image.Image
                x, y       float64
                hpPerc     float64
                isTarget   bool
                isAttacker bool
                side       string // "enemy" | "player" | "summon"
                fadeOut    float64
        }
        var items []renderItem

        // ── Enemies ──
        for i, enemy := range req.Enemies {
                if enemy.CurrentHP <= 0 && !enemy.JustDied && !isTargetThisAction(req, "enemy", i, fs) {
                        continue
                }
                spritePath := GetEnemySpritePath(enemy.Name, avgLevel, i, enemy.IsBoss, assetsPath)
                eSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }
                eW := enemySpriteSize
                if enemy.IsBoss {
                        eW = enemySpriteSize * 1.5
                }
                eSprite = imaging.Resize(eSprite, int(eW), 0, imaging.NearestNeighbor)

                // Position
                ex, ey := enemyAnchorX, enemyAnchorY
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

                // HP override (if this is the target)
                hp := enemy.CurrentHP
                maxHP := enemy.MaxHP
                isTarget := req.Action != nil && req.Action.TargetSide == "enemy" && req.Action.TargetIndex == i
                if isTarget && fs != nil && fs.targetHPOverride != nil {
                        hp = *fs.targetHPOverride
                }

                // Red tint on death or hit flash
                if hp <= 0 {
                        eSprite = utils.TintImage(eSprite, color.RGBA{255, 0, 0, 100})
                }

                hpPerc := 0.0
                if maxHP > 0 {
                        hpPerc = float64(hp) / float64(maxHP)
                }

                items = append(items, renderItem{
                        img: eSprite, x: ex, y: ey, hpPerc: hpPerc,
                        isTarget: isTarget, side: "enemy",
                })
        }

        // ── Summons (player-side allies) ──
        for i, summon := range req.Summons {
                if summon.CurrentHP <= 0 && !summon.JustDied && !isTargetThisAction(req, "summon", i, fs) {
                        continue
                }
                spritePath := GetSummonSpritePath(summon.Species, assetsPath)
                sSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }
                sSprite = imaging.Resize(sSprite, int(summonSpriteSize), 0, imaging.NearestNeighbor)

                // Position (same formation as enemies, mirrored to left)
                sx, sy := summonAnchorX, summonAnchorY
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

                hp := summon.CurrentHP
                maxHP := summon.MaxHP
                isTarget := req.Action != nil && req.Action.TargetSide == "summon" && req.Action.TargetIndex == i
                isAttacker := req.Action != nil && req.Action.AttackerSide == "summon" && req.Action.AttackerIndex == i
                if isTarget && fs != nil && fs.targetHPOverride != nil {
                        hp = *fs.targetHPOverride
                }
                if hp <= 0 {
                        sSprite = utils.TintImage(sSprite, color.RGBA{255, 0, 0, 100})
                }

                hpPerc := 0.0
                if maxHP > 0 {
                        hpPerc = float64(hp) / float64(maxHP)
                }

                // Apply attacker windup offset
                if isAttacker && fs != nil {
                        sx += fs.attackerDX
                        sy += fs.attackerDY
                }

                fadeOut := 0.0
                if isTarget && fs != nil {
                        fadeOut = fs.targetFade
                }

                items = append(items, renderItem{
                        img: sSprite, x: sx, y: sy, hpPerc: hpPerc,
                        isTarget: isTarget, isAttacker: isAttacker, side: "summon",
                        fadeOut: fadeOut,
                })
        }

        // ── Players (as battlefield sprites — same formation as enemies, on left side) ──
        // In PVE, draw each player as a full-body sprite on the battlefield using the enemy formation pattern.
        // In PVP, draw both players using the formation pattern.
        for i, p := range req.Players {
                if p.CurrentHP <= 0 && !isTargetThisAction(req, "player", i, fs) {
                        continue // skip dead players (no justDied flag for players in current schema)
                }
                spritePath := GetCharacterSpritePath(p.Class, p.SpriteIndex, assetsPath)
                pSprite, err := utils.LoadImage(spritePath)
                if err != nil {
                        continue
                }
                pSprite = imaging.Resize(pSprite, int(playerSpriteSize), 0, imaging.NearestNeighbor)

                // Position (same formation as enemies, mirrored to left side)
                // Player 1 (i=0) is at the front; subsequent players cluster behind.
                px, py := playerAnchorX, playerAnchorY
                sub := i % 4
                if sub == 1 || sub == 2 {
                        px -= spX
                } else if sub == 3 {
                        px -= spX * 2
                }
                if sub == 1 || sub == 3 {
                        py += spY
                }
                px += float64(i/4) * -250.0

                // Flip horizontally so player faces right (toward enemies)
                pSprite = imaging.FlipH(pSprite)

                hp := p.CurrentHP
                maxHP := p.MaxHP
                isTarget := req.Action != nil && req.Action.TargetSide == "player" && req.Action.TargetIndex == i
                isAttacker := req.Action != nil && req.Action.AttackerSide == "player" && req.Action.AttackerIndex == i
                if isTarget && fs != nil && fs.targetHPOverride != nil {
                        hp = *fs.targetHPOverride
                }
                if hp <= 0 {
                        pSprite = utils.TintImage(pSprite, color.RGBA{255, 0, 0, 100})
                }

                hpPerc := 0.0
                if maxHP > 0 {
                        hpPerc = float64(hp) / float64(maxHP)
                }

                // Apply attacker windup offset
                if isAttacker && fs != nil {
                        px += fs.attackerDX
                        py += fs.attackerDY
                }

                fadeOut := 0.0
                if isTarget && fs != nil {
                        fadeOut = fs.targetFade
                }

                items = append(items, renderItem{
                        img: pSprite, x: px, y: py, hpPerc: hpPerc,
                        isTarget: isTarget, isAttacker: isAttacker, side: "player",
                        fadeOut: fadeOut,
                })
        }

        // ── Sort by Y (painter's algorithm) ──
        sort.SliceStable(items, func(i, j int) bool {
                return items[i].y < items[j].y
        })

        // ── Draw entities ──
        uiPath := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }
        for _, it := range items {
                sprite := it.img

                // Apply shake offset if target
                drawX := it.x
                drawY := it.y
                if it.isTarget && fs != nil {
                        drawX += fs.targetShakeX
                        drawY += fs.targetShakeY
                }

                // Apply fade-out (defeated)
                if it.fadeOut > 0 {
                        sprite = applyAlpha(sprite, 1.0-it.fadeOut)
                }

                // Shadow
                utils.DrawShadow(dc, drawX+float64(sprite.Bounds().Dx())/2,
                        drawY+float64(sprite.Bounds().Dy())-10,
                        float64(sprite.Bounds().Dx())*0.4, 0.6)

                // Sprite
                dc.DrawImage(sprite, int(drawX), int(drawY))

                // Red flash overlay on target (drawn on top of sprite, bounded by sprite area)
                if it.isTarget && fs != nil && fs.targetRedFlash > 0 {
                        flashImg := makeRedFlash(sprite, fs.targetRedFlash)
                        dc.DrawImage(flashImg, int(drawX), int(drawY))
                }

                // HP bar
                if it.hpPerc > 0 || it.isTarget {
                        hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
                        if err == nil {
                                barW := 100.0
                                if it.side == "summon" {
                                        barW = 80.0
                                }
                                barH := 12.0
                                if it.side == "summon" {
                                        barH = 10.0
                                }
                                hpPerc := it.hpPerc
                                if hpPerc < 0 {
                                        hpPerc = 0
                                }
                                if hpPerc > 1 {
                                        hpPerc = 1
                                }
                                currentBarW := int(barW * hpPerc)
                                if currentBarW < 1 {
                                        currentBarW = 1
                                }
                                hpBarImg = imaging.Resize(hpBarImg, currentBarW, int(barH), imaging.NearestNeighbor)
                                bx := drawX + (float64(sprite.Bounds().Dx())-barW)/2
                                by := drawY - 15
                                dc.DrawImage(hpBarImg, int(bx), int(by))
                        }
                }
        }

        // ── VFX overlay on target ──
        if fs != nil && fs.vfxFrame > 0 && fs.vfxPath != "" && req.Action != nil {
                vfxImg := loadVfxFrame(fs.vfxPath, fs.vfxFrame, assetsPath)
                if vfxImg != nil {
                        // Compute target position
                        targetPos := getEntityPosition(req, req.Action.TargetSide, req.Action.TargetIndex,
                                playerAnchorX, playerAnchorY, enemyAnchorX, enemyAnchorY, summonAnchorX, summonAnchorY, spX, spY)
                        if targetPos != nil {
                                // Scale VFX to ~200px wide, center on target's center
                                vfxW := 240
                                vfxImg = imaging.Resize(vfxImg, vfxW, 0, imaging.NearestNeighbor)
                                vx := targetPos.x + 80 - float64(vfxW)/2 // center on sprite (assume ~80px center offset)
                                vy := targetPos.y - 40
                                dc.DrawImage(vfxImg, int(vx), int(vy))
                        }
                }
        }

        // ── UI overlay (player_state panel, banner, options menu) ──
        // Same as static renderer — keeps the HUD consistent.
        drawImage := func(path string, x, y, w, h int) {
                img, err := utils.LoadImage(path)
                if err == nil {
                        if w > 0 && h > 0 {
                                img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
                        }
                        dc.DrawImage(img, normX(x), normY(y))
                }
        }
        drawImage(uiPath("player_state.png"), -716, 113, 453, 244)
        drawImage(uiPath("heart.png"), -678, 209, 38, 47)
        drawImage(uiPath("mana.png"), -673, 256, 29, 44)
        drawImage(uiPath("Options_menu.png"), -97, 99, 443, 258)
        drawImage(uiPath("banner.png"), -582, -410, 800, 160)

        // Main player HP/energy bars (always uses Players[0])
        if len(req.Players) > 0 {
                p := req.Players[0]
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
        }

        // ── Banner text ──
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
                if face, err := utils.LoadFont(fontPath, 40); err == nil {
                        dc.SetFontFace(face)
                        dc.SetColor(color.RGBA{0, 0, 0, 255})
                        bx, by := float64(normX(-582)), float64(normY(-410))
                        bw, bh := 800.0, 160.0
                        dc.DrawStringAnchored(text, bx+bw/2, by+bh/2-30, 0.5, 0.5)
                }
        }

        // ── Skill name caption (bottom center) ──
        if req.Action != nil && req.Action.SkillName != "" {
                fontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
                if face, err := utils.LoadFont(fontPath, 28); err == nil {
                        dc.SetFontFace(face)
                        // Semi-transparent black background bar
                        caption := req.Action.SkillName
                        if req.Action.IsCrit {
                                caption = "CRITICAL! " + caption
                        }
                        if req.Action.Missed {
                                caption = "MISS!"
                        }
                        textW, textH := dc.MeasureString(caption)
                        barX := float64(CANVAS_W)/2 - textW/2 - 15
                        barY := float64(CANVAS_H) - 60
                        barW := textW + 30
                        barH := textH + 10
                        dc.SetRGBA(0, 0, 0, 0.7)
                        dc.DrawRectangle(barX, barY, barW, barH)
                        dc.Fill()
                        // Text
                        if req.Action.IsCrit {
                                dc.SetRGB(1, 0.85, 0.3) // gold for crits
                        } else if req.Action.Missed {
                                dc.SetRGB(0.8, 0.8, 0.8) // grey for miss
                        } else if req.Action.Heal > 0 {
                                dc.SetRGB(0.4, 1, 0.4) // green for heal
                        } else {
                                dc.SetRGB(1, 1, 1) // white for normal
                        }
                        dc.DrawStringAnchored(caption, float64(CANVAS_W)/2, barY+barH/2, 0.5, 0.5)
                }
        }

        return dc.Image(), nil
}

// isTargetThisAction returns true if this entity is the action's target
// (used to keep dead-target rendering while the animation plays).
func isTargetThisAction(req *CombatRequest, side string, idx int, fs *frameState) bool {
        if req.Action == nil {
                return false
        }
        return req.Action.TargetSide == side && req.Action.TargetIndex == idx
}

// getEntityPosition returns the (x, y) position of an entity by side+index.
type entityPos struct{ x, y float64 }

func getEntityPosition(req *CombatRequest, side string, idx int,
        playerAnchorX, playerAnchorY, enemyAnchorX, enemyAnchorY, summonAnchorX, summonAnchorY float64,
        spX, spY float64) *entityPos {

        var anchorX, anchorY float64
        switch side {
        case "player":
                anchorX, anchorY = playerAnchorX, playerAnchorY
        case "enemy":
                anchorX, anchorY = enemyAnchorX, enemyAnchorY
        case "summon":
                anchorX, anchorY = summonAnchorX, summonAnchorY
        default:
                return nil
        }

        x, y := anchorX, anchorY
        sub := idx % 4
        if sub == 1 || sub == 2 {
                x -= spX
        } else if sub == 3 {
                x -= spX * 2
        }
        if sub == 1 || sub == 3 {
                y += spY
        }
        x += float64(idx/4) * -250.0
        return &entityPos{x: x, y: y}
}

// makeRedFlash returns a copy of the sprite with red overlay applied at the given intensity (0-1).
func makeRedFlash(img image.Image, intensity float64) image.Image {
        bounds := img.Bounds()
        dst := image.NewRGBA(bounds)
        for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
                for x := bounds.Min.X; x < bounds.Max.X; x++ {
                        r, g, b, a := img.At(x, y).RGBA()
                        r8 := uint8(r >> 8)
                        g8 := uint8(g >> 8)
                        b8 := uint8(b >> 8)
                        a8 := uint8(a >> 8)
                        if a8 == 0 {
                                continue
                        }
                        // Blend toward red (255, 50, 50)
                        nr := uint8(float64(r8)*(1-intensity) + 255*intensity)
                        ng := uint8(float64(g8)*(1-intensity) + 50*intensity)
                        nb := uint8(float64(b8)*(1-intensity) + 50*intensity)
                        dst.Set(x, y, color.RGBA{nr, ng, nb, a8})
                }
        }
        return dst
}

// applyAlpha returns a copy of the image with alpha multiplied by the given factor (0-1).
func applyAlpha(img image.Image, factor float64) image.Image {
        if factor >= 1.0 {
                return img
        }
        if factor <= 0 {
                // Return a fully transparent image of same bounds
                bounds := img.Bounds()
                return image.NewRGBA(bounds)
        }
        bounds := img.Bounds()
        dst := image.NewRGBA(bounds)
        for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
                for x := bounds.Min.X; x < bounds.Max.X; x++ {
                        r, g, b, a := img.At(x, y).RGBA()
                        a8 := uint8(float64(a>>8) * factor)
                        dst.Set(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), a8})
                }
        }
        return dst
}

// loadVfxFrame loads a specific numbered frame from a VFX folder.
// Folder names: "attack", "Fire-bomb", "Lightning", "Dark-Bolt", "Holy VFX 02".
// Frame numbering starts at 1.
func loadVfxFrame(folder string, frame int, assetsPath string) image.Image {
        // Determine the filename pattern per folder
        var filename string
        switch folder {
        case "attack":
                filename = fmt.Sprintf("attack%d.png", frame)
        case "Fire-bomb":
                filename = fmt.Sprintf("Fire-bomb%d.png", frame)
        case "Lightning":
                filename = fmt.Sprintf("Lightning%d.png", frame)
        case "Dark-Bolt":
                filename = fmt.Sprintf("Dark-Bolt%d.png", frame)
        case "Holy VFX 02":
                filename = fmt.Sprintf("Holy VFX 02 %d.png", frame)
        default:
                return nil
        }

        // Look in magic/ subfolder (Fire-bomb, Lightning, Dark-Bolt) or top-level (attack) or holy/ (Holy VFX 02)
        candidates := []string{
                filepath.Join(assetsPath, "rpgasset", "vfx", "magic", filename),
                filepath.Join(assetsPath, "rpgasset", "vfx", "attack", filename),
                filepath.Join(assetsPath, "rpgasset", "vfx", "holy", filename),
                filepath.Join(assetsPath, "rpgasset", "vfx", "dark", filename),
                filepath.Join(assetsPath, "rpgasset", "vfx", filename),
        }
        for _, p := range candidates {
                if fileExists(p) {
                        if img, err := utils.LoadImage(p); err == nil {
                                return img
                        }
                }
        }
        return nil
}

// encodeFramesToMP4 encodes a slice of image.Image frames into an MP4 buffer using ffmpeg.
func encodeFramesToMP4(frames []image.Image) ([]byte, error) {
        if len(frames) == 0 {
                return nil, fmt.Errorf("no frames to encode")
        }

        // Create temp dir for frames
        tmpDir, err := os.MkdirTemp("", "combat-anim-")
        if err != nil {
                return nil, fmt.Errorf("temp dir: %w", err)
        }
        defer os.RemoveAll(tmpDir)

        // Write each frame as PNG
        for i, frame := range frames {
                framePath := filepath.Join(tmpDir, fmt.Sprintf("frame_%03d.png", i))
                buf := new(bytes.Buffer)
                if err := encodePNG(buf, frame); err != nil {
                        return nil, fmt.Errorf("encode frame %d: %w", i, err)
                }
                if err := os.WriteFile(framePath, buf.Bytes(), 0644); err != nil {
                        return nil, fmt.Errorf("write frame %d: %w", i, err)
                }
        }

        // Output MP4 path
        outputPath := filepath.Join(tmpDir, "output.mp4")

        // ffmpeg command:
        //   ffmpeg -y -framerate 10 -i frame_%03d.png -vf "pad=ceil(iw/2)*2:ceil(ih/2)*2"
        //          -c:v libx264 -pix_fmt yuv420p -movflags +faststart output.mp4
        // The pad filter ensures even dimensions (libx264 requires width AND height
        // to be divisible by 2). Our canvas is 1024×687 — 687 is odd, so we pad to 688.
        args := []string{
                "-y",
                "-framerate", fmt.Sprintf("%d", ANIM_FPS),
                "-i", filepath.Join(tmpDir, "frame_%03d.png"),
                "-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2:color=black",
                "-c:v", "libx264",
                "-pix_fmt", "yuv420p",
                "-movflags", "+faststart",
                "-preset", "fast",
                "-crf", "23",
                outputPath,
        }

        cmd := execCommand("ffmpeg", args...)
        cmdOutput := new(bytes.Buffer)
        cmdErr := new(bytes.Buffer)
        cmd.Stdout = cmdOutput
        cmd.Stderr = cmdErr
        if err := cmd.Run(); err != nil {
                return nil, fmt.Errorf("ffmpeg: %w (stderr: %s)", err, cmdErr.String())
        }

        // Read MP4
        mp4Bytes, err := os.ReadFile(outputPath)
        if err != nil {
                return nil, fmt.Errorf("read mp4: %w", err)
        }
        if len(mp4Bytes) == 0 {
                return nil, fmt.Errorf("mp4 is empty")
        }

        return mp4Bytes, nil
}

// encodePNG writes a PNG image to the given writer.
func encodePNG(w *bytes.Buffer, img image.Image) error {
        return pngEncode(w, img)
}

// execCommand is a small wrapper around exec.Command so tests can stub it.
// (We call exec.Command directly here to avoid import-cycle surprises.)
var execCommand = newExecCommand

// pngEncode wraps the standard png.Encode function.
var pngEncode = newPngEncode
