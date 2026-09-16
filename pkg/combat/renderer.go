package combat

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
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
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < 0 || maxY < 0 {
		return img // fully transparent - return as-is
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
	HORIZON_Y       = 309 // 45% of CANVAS_H - sky/ground boundary
	GROUND_BOTTOM_Y = 515 // 75% of CANVAS_H - top of UI panel
	MAIN_FEET_Y     = 505 // main player/enemy feet (above UI panel top at 469... wait)
	BACK_FEET_Y     = 410 // summon/background entity feet (higher = further away)
)

func GenerateCombatImage(c *gin.Context) {
	var req CombatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Debug log for req.Action. If nil, turn crystal falls back to player[0].
	if req.Action != nil {
		log.Printf("[combat-render] Action present: attackerSide=%s attackerIndex=%d targetSide=%s targetIndex=%d combatType=%s",
			req.Action.AttackerSide, req.Action.AttackerIndex,
			req.Action.TargetSide, req.Action.TargetIndex,
			req.CombatType)
	} else {
		log.Printf("[combat-render] Action is nil - turn crystal falls back to player[0]. combatType=%s players=%d enemies=%d summons=%d",
			req.CombatType, len(req.Players), len(req.Enemies), len(req.Summons))
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

	// FIX 2026-09-11 R2 (owner directive: montage_E_4 style): the duel
	// arena is now spark_5.png - the bright open beach arena. The Node
	// client sends spark_5 for PvP too; this guard covers direct callers. Previously PvP without an
	// explicit background fell through to getRankBackground(req.Rank) - and the
	// Node client hardcodes spark_1.png (grassland) when no backgroundPath is
	// given, so every production duel rendered on the grassland. Node now sends
	// spark_15 for PvP too; this guard covers direct API callers.
	if req.CombatType == "PVP" && (bgPath == "" || !fileExists(bgPath)) {
		bgPath = filepath.Join(assetsPath, "rpgasset", "environment", "spark_5.png")
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

	// 💡 2026-09-11 (owner directive): REMOVE the ground-ellipse turn
	// indicator - it read as a big colored blob under the sprite.
	// Replacement: a floating crystal drawn ABOVE the active unit's head.
	// Crystals are QUEUED here and painted in a dedicated pass AFTER the
	// nameplate pills so they hover on top of HP bars / name tags.
	// Asset: assets/rpgasset/ui/crystal.png (owner-supplied pixel art).
	var crystalQueue []struct {
		cx, headTopY, spriteH float64
	}
	crystalSrc, crystalLoadErr := utils.LoadImage(filepath.Join(assetsPath, "rpgasset", "ui", "crystal.png"))
	if crystalLoadErr != nil {
		log.Printf("[combat-render] turn crystal asset missing (%v) - turn indicator disabled", crystalLoadErr)
	}
	queueTurnCrystal := func(cx, headTopY, spriteH float64) {
		crystalQueue = append(crystalQueue, struct {
			cx, headTopY, spriteH float64
		}{cx, headTopY, spriteH})
	}
	// isAttacker checks if the entity at (side, index) is the active attacker.
	// When req.Action is nil (static PvE renders), falls back to highlighting
	// player[0] so the turn indicator always renders.
	isAttacker := func(side string, index int) bool {
		if req.Action == nil {
			return side == "player" && index == 0
		}
		return req.Action.AttackerSide == side && req.Action.AttackerIndex == index
	}

	// 2. Mobs / Enemies - slot table (EnemySlots) + crop-then-resize-by-height.
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
		feetY     float64 // FIX 2026-09-11: depth sort key (top-y favored tall sprites)
		hpPercent float64
		origIndex int    // carries the original enemy index through the Y-sort
		name      string // for nameplate pill
	}
	var mobQueue []RenderItem

	// FIX 2026-09-11: nameplate pills are now collected while sprites draw and
	// painted in ONE collision-resolved pass after every sprite exists. Before,
	// each mob drew its pill immediately, so (a) a later-drawn sprite covered
	// the pill of the mob behind it (ghost label clipped in E12) and (b) pills
	// of stacked mobs overlapped each other into unreadable label soup (E5/E8/
	// E11). pillH mirrors drawNameplatePill's height.
	const pillH = 16.0
	type pillItem struct {
		cx, bottomY float64
		name        string
	}
	var pillQueue []pillItem

	// 💡 FIX 2026-09-11 (visual audit): resolve every enemy sprite up front,
	// then spread same-family duplicates across the family pool so one
	// encounter never shows identical twin sprites side by side.
	// Dead-and-unseen enemies resolve to "" and are ignored by the dedupe.
	resolvedEnemyFiles := make([]string, len(req.Enemies))
	for i, enemy := range req.Enemies {
		if enemy.CurrentHP <= 0 && !enemy.JustDied {
			continue
		}
		rotIdx := i
		// FIX 2026-09-11: rotation previously ALWAYS used the enemy array
		// position, so a payload spriteIndex was dead data. Honor spriteIndex
		// when the Node side sends one (>0); position remains the fallback.
		if enemy.SpriteIndex > 0 {
			rotIdx = enemy.SpriteIndex
		}
		// 💡 FIX 2026-09-11 (facing audit): bucket by the ENEMY's own level.
		// The payload's per-enemy level was dropped by types.go, so the
		// fallback buckets keyed off the party's average level and every
		// mob rendered as the party-tier element (ice wolves as fire blobs).
		lvl := enemy.Level
		if lvl <= 0 {
			lvl = avgLevel // legacy payloads without per-enemy level
		}
		resolvedEnemyFiles[i] = filepath.Base(GetEnemySpritePath(enemy.Name, lvl, rotIdx, enemy.IsBoss, assetsPath))
	}
	resolvedEnemyFiles = DedupeEnemySprites(resolvedEnemyFiles)

	for i, enemy := range req.Enemies {
		if enemy.CurrentHP <= 0 && !enemy.JustDied {
			continue
		}

		spritePath := filepath.Join(assetsPath, "rpgasset", "enemies", resolvedEnemyFiles[i])
		eSprite, err := utils.LoadImage(spritePath)
		if err != nil {
			continue
		}

		// Crop FIRST, then resize by fixed HEIGHT. Width floats by aspect ratio.
		eSprite = cropToVisibleBounds(eSprite)
		targetH := pveEnemySpriteH // 170
		if enemy.IsBoss {
			targetH = pveBossSpriteH // 210
		}
		eSprite = imaging.Resize(eSprite, 0, targetH, imaging.NearestNeighbor)

		// Side-based facing - enemies on RIGHT side face LEFT
		if flipForSide(filepath.Base(spritePath), false) {
			eSprite = imaging.FlipH(eSprite)
		}

		// Tint Red if dead
		if enemy.CurrentHP <= 0 {
			eSprite = utils.TintImage(eSprite, color.RGBA{80, 0, 80, 180})
		}

		// Position from EnemySlots table - no inline math.
		slot := slotFor(EnemySlots, i)
		feetX := slot.X
		feetY := slot.Y

		// Convert feet position to draw position (top-left anchor for DrawImage)
		spriteW := eSprite.Bounds().Dx()
		spriteH := eSprite.Bounds().Dy()
		ex := feetX - float64(spriteW)/2
		ey := feetY - float64(spriteH)

		hpPerc := 0.0
		if enemy.MaxHP > 0 {
			hpPerc = float64(enemy.CurrentHP) / float64(enemy.MaxHP)
		}
		mobQueue = append(mobQueue, RenderItem{img: eSprite, x: ex, y: ey, feetY: feetY, hpPercent: hpPerc, origIndex: i, name: enemy.Name})
	}
	// Sort by FEET Y (Painter's Algorithm)
	// FIX 2026-09-11: was sorting by sprite TOP y, which drew tall sprites
	// (210px bosses) before short ones at the same feet line regardless of
	// actual depth, making front mobs paint behind boss bodies.
	sort.Slice(mobQueue, func(i, j int) bool {
		return mobQueue[i].feetY < mobQueue[j].feetY
	})

	// Draw Mobs
	for _, mob := range mobQueue {
		// Shadow (drawn FIRST, under everything) - at feet position
		utils.DrawShadow(dc, mob.x+float64(mob.img.Bounds().Dx())/2, mob.y+float64(mob.img.Bounds().Dy())-2, float64(mob.img.Bounds().Dx())*0.6, 0.85)
		// 💡 TURN INDICATOR: floating crystal above the attacker's head.
		if isAttacker("enemy", mob.origIndex) {
			cx := mob.x + float64(mob.img.Bounds().Dx())/2
			queueTurnCrystal(cx, mob.y, float64(mob.img.Bounds().Dy()))
		}
		// Sprite
		dc.DrawImage(mob.img, int(mob.x), int(mob.y))

		// ENEMY HP BAR - 💡 FIX 2026-09-11 (audit): the bare hp5 sliver
		// floated with no track and shrank to a 1px red hairline at low HP
		// ("HP bar texture too low/too small"). Now: fixed-width dark
		// track + border, hp5 fill clipped to remaining HP inside it.
		hpBarTopY := mob.y - 15
		if mob.hpPercent > 0 {
			uiPath := func(f string) string { return filepath.Join(assetsPath, "rpgasset", "ui", f) }
			hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
			if err == nil {
				bx := mob.x + (float64(mob.img.Bounds().Dx())-100.0)/2
				by := mob.y - 15
				hpBarTopY = by
				drawOverheadHPBar(dc, bx, by, 100, 12, mob.hpPercent, hpBarImg)
			}
		}

		// Nameplate pill above HP bar - queued for the collision pass
		cx := mob.x + float64(mob.img.Bounds().Dx())/2
		pillQueue = append(pillQueue, pillItem{cx: cx, bottomY: hpBarTopY - 2, name: mob.name})
	}

	// 💡 Summoner System: Draw summoned allies.
	// Slot table (SummonSlots) + crop-then-resize-by-height (75px).

	// FIX (2026-08-16): Skip in PvP -- regular PvP summons use ownerIndex
	// for side-based positioning (added below), not the PvE left-side slots.
	if req.CombatType != "PVP" {
		for i, summon := range req.Summons {
			if summon.CurrentHP <= 0 && !summon.JustDied {
				continue
			}

			spritePath := GetSummonSpritePath(summon.Species, assetsPath)
			sSprite, err := utils.LoadImage(spritePath)
			if err != nil {
				continue
			}

			// Crop FIRST, then resize by fixed HEIGHT (75px).
			sSprite = cropToVisibleBounds(sSprite)
			sSprite = imaging.Resize(sSprite, 0, pveSummonSpriteH, imaging.NearestNeighbor)

			// Summon facing via flipForSide.
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

			// Position from SummonSlots table - no inline math.
			slot := slotFor(SummonSlots, i)
			summonFeetX := slot.X
			summonFeetY := slot.Y

			// Convert feet to draw position
			sSpriteW := sSprite.Bounds().Dx()
			sSpriteH := sSprite.Bounds().Dy()
			sx := summonFeetX - float64(sSpriteW)/2
			sy := summonFeetY - float64(sSpriteH)

			// Shadow at feet
			utils.DrawShadow(dc, summonFeetX, summonFeetY-2, float64(sSpriteW)*0.6, 0.85)

			// Turn indicator - floating crystal above head
			if isAttacker("summon", i) {
				queueTurnCrystal(summonFeetX, sy, float64(sSpriteH))
			}

			// Sprite
			dc.DrawImage(sSprite, int(sx), int(sy))

			// HP bar
			hpBarTopY := sy - 12
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
					hpBarTopY = by
					dc.DrawImage(hpBarImg, int(bx), int(by))
				}
			}

			// Nameplate pill above HP bar - queued for the collision pass
			summonName := summon.Name
			if summonName == "" {
				summonName = summon.Species
			}
			pillQueue = append(pillQueue, pillItem{cx: summonFeetX, bottomY: hpBarTopY - 2, name: summonName})
		}
	} // end if combatType != "PVP"

	// FIX (2026-08-16): Regular PvP summon rendering -- split by ownerIndex.
	if req.CombatType == "PVP" && !(req.CombatType == "PVP" && len(req.Players) >= 2 && req.Players[0].Mode == "summon" && req.Players[0].Species != "" && req.Players[1].Mode == "summon" && req.Players[1].Species != "") && len(req.Summons) > 0 {
		// FIX 2026-09-11: right summon sat at x=820 while its owner stood at
		// x=800 - the slime painted straight over the master's face (E4).
		// Slots moved OUTWARD of both owners (owners at 220/800).
		pvpSummonSlots := []struct{ x, y float64 }{
			{110, 420},
			{915, 420},
		}
		for i, summon := range req.Summons {
			if summon.CurrentHP <= 0 && !summon.JustDied {
				continue
			}
			spritePath := GetSummonSpritePath(summon.Species, assetsPath)
			sSprite, err := utils.LoadImage(spritePath)
			if err != nil {
				continue
			}
			sSprite = cropToVisibleBounds(sSprite)
			sSprite = imaging.Resize(sSprite, 0, pveSummonSpriteH, imaging.NearestNeighbor)
			ownerIdx := summon.OwnerIndex
			if ownerIdx < 0 || ownerIdx > 1 {
				ownerIdx = i % 2
			}
			if strings.Contains(summon.Species, "ship_") {
				sSprite = imaging.Rotate90(sSprite)
			} else {
				summonSpriteFile := filepath.Base(spritePath)
				if flipForSide(summonSpriteFile, ownerIdx == 0) {
					sSprite = imaging.FlipH(sSprite)
				}
			}
			if summon.CurrentHP <= 0 {
				sSprite = utils.TintImage(sSprite, color.RGBA{80, 0, 80, 180})
			}
			pos := pvpSummonSlots[ownerIdx]
			summonFeetX := pos.x
			summonFeetY := pos.y
			sSpriteW := sSprite.Bounds().Dx()
			sSpriteH := sSprite.Bounds().Dy()
			sx := summonFeetX - float64(sSpriteW)/2
			sy := summonFeetY - float64(sSpriteH)
			utils.DrawShadow(dc, summonFeetX, summonFeetY-2, float64(sSpriteW)*0.6, 0.85)
			if isAttacker("summon", i) {
				queueTurnCrystal(summonFeetX, sy, float64(sSpriteH))
			}
			dc.DrawImage(sSprite, int(sx), int(sy))
			hpBarTopY := sy - 12
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
					hpBarTopY = by
					dc.DrawImage(hpBarImg, int(bx), int(by))
				}
			}
			summonName := summon.Name
			if summonName == "" {
				summonName = summon.Species
			}
			pillQueue = append(pillQueue, pillItem{cx: summonFeetX, bottomY: hpBarTopY - 2, name: summonName})
		}
	}

	// 💡 FIX 2026-08-08: Dedicated PvP-summon render path.
	// Decoupled from drawPvPFighter (shared with 1v1) so summon scale,
	// position, and z-order can be tuned independently.
	//
	// Detection: both players are summons (mode="summon" + species set).
	isPvPSummonDuel := req.CombatType == "PVP" && len(req.Players) >= 2 &&
		req.Players[0].Mode == "summon" && req.Players[0].Species != "" &&
		req.Players[1].Mode == "summon" && req.Players[1].Species != ""

	// PvP-summon sprites drawn HERE (before UI panels) so panels occlude
	// any upper-body overlap (z-order fix #3).
	if isPvPSummonDuel {
		// #2: Scale by HEIGHT (not width) so visually different-shaped sprites
		// (square plaguefang vs wide giant) end up perceptually similar in size.
		// Target height: 180px post-crop. Width varies by aspect ratio.
		const pvpSummonSpriteH = 180

		// #4: Repositioned outward, symmetric, 100px+ clear of panel edges.
		// Left panel inner edge: X=431. Left summon at X=220 → range ~115-325.
		//   Gap to panel: 431-325 = 106px ✅ (>100px)
		// Right panel inner edge: X=571. Right summon at X=800 → range ~695-905.
		//   Gap to panel: 695-571 = 124px ✅ (>100px)
		// Gap between summons: 695-325 = 370px ✅
		//
		// 💡 FIX 2026-08-08: Feet Y raised from MAIN_FEET_Y (505) to 450.
		// Panel top edge is at Y=469 (normY(113)). At Y=505, sprite feet
		// extended 36px INTO the panel zone, causing clipping of legs/shield.
		// Shadow extends 16px below feet (radius_y=18, center at feetY-2).
		// At Y=465, shadow bottom=481, overlapped panel top (469) by 12px.
		// At Y=445, shadow bottom=461, 8px clear of panel top. ✅
		const pvpSummonFeetY = 455
		pvpSummonPositions := []struct{ x, y int }{
			{220, pvpSummonFeetY},
			{800, pvpSummonFeetY},
		}

		for i, pos := range pvpSummonPositions {
			player := req.Players[i]
			spritePath := GetSummonSpritePath(player.Species, assetsPath)
			sprite, err := utils.LoadImage(spritePath)
			if err != nil {
				continue
			}
			if player.CurrentHP <= 0 {
				sprite = utils.TintImage(sprite, color.RGBA{80, 0, 80, 180})
			}

			// #2: Crop FIRST (remove transparent padding), THEN resize to target height.
			// This ensures both sprites have the same visible height after resize,
			// regardless of how much padding their raw assets have.
			sprite = cropToVisibleBounds(sprite)
			rawH := sprite.Bounds().Dy()
			if rawH > 0 {
				sprite = imaging.Resize(sprite, 0, pvpSummonSpriteH, imaging.NearestNeighbor)
			}

			// Facing: left side faces RIGHT, right side faces LEFT
			spriteFile := filepath.Base(spritePath)
			if flipForSide(spriteFile, i == 0) {
				sprite = imaging.FlipH(sprite)
			}

			sW := sprite.Bounds().Dx()
			sH := sprite.Bounds().Dy()
			drawX := pos.x - sW/2
			drawY := pos.y - sH

			// Shadow at feet - wide flat ellipse (looks like ground shadow, doesn't overlap panel)
			// rx = 60% of sprite width (wide), ry = 12px (flat). Total height = 24px.
			utils.DrawShadowEllipse(dc, float64(pos.x), float64(pos.y)-2, float64(sW)*0.6, 12, 0.85)

			if isAttacker("player", i) {
				queueTurnCrystal(float64(pos.x), float64(drawY), float64(sH))
			}

			dc.DrawImage(sprite, drawX, drawY)

			// FIX 2026-09-11: duel summons had NO overhead HP bar and no
			// nameplate (E3 showed both duelists bar-less while 1v1 humans
			// had bars). Draw the same bar drawPvPFighter uses, and queue
			// the nameplate for the collision pass.
			hpPercent := 0.0
			if player.MaxHP > 0 {
				hpPercent = math.Max(0, math.Min(1, float64(player.CurrentHP)/float64(player.MaxHP)))
			}
			if hpBarImg, err := utils.LoadImage(filepath.Join(assetsPath, "rpgasset", "ui", "hp5.png")); err == nil && hpPercent > 0 {
				barW := int(100 * hpPercent)
				if barW < 1 {
					barW = 1
				}
				hpBarImg = imaging.Resize(hpBarImg, barW, 10, imaging.NearestNeighbor)
				dc.DrawImage(hpBarImg, drawX+10, drawY-15)
			}
			duelName := player.Name
			if duelName == "" {
				duelName = player.Species
			}
			pillQueue = append(pillQueue, pillItem{cx: float64(pos.x), bottomY: float64(drawY) - 18, name: duelName})
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
		// 💡 FIX 2026-08-08 #3: Right panel mirrored with same margin as left.
		// Left panel: X=-22 (22px off-canvas left), width 453 → right edge at 431.
		// Right panel: right edge at canvas_w - 22 = 1002 → X = 1002 - 453 = 549.
		// normX(-145) = -145 + 694 = 549. Right edge = 549 + 453 = 1002. ✅
		panelImg, err := utils.LoadImage(uiPath("player_state.png"))
		if err == nil {
			panelImg = imaging.Resize(panelImg, 453, 244, imaging.NearestNeighbor)
			panelImg = imaging.FlipH(panelImg)
			dc.DrawImage(panelImg, normX(-145), normY(113))
		}
		// Heart icon - mirrored position (right side of right panel)
		heartImg, err := utils.LoadImage(uiPath("heart.png"))
		if err == nil {
			heartImg = imaging.Resize(heartImg, 38, 47, imaging.NearestNeighbor)
			dc.DrawImage(heartImg, normX(-111), normY(209))
		}
		manaImg, err := utils.LoadImage(uiPath("mana.png"))
		if err == nil {
			manaImg = imaging.Resize(manaImg, 29, 44, imaging.NearestNeighbor)
			dc.DrawImage(manaImg, normX(-106), normY(256))
		}
	} else if req.CombatType != "PVP" {
		drawImage(uiPath("Options_menu.png"), -97, 99, 443, 258)
	}
	drawImage(uiPath("banner.png"), -582, -410, 800, 160)

	// 💡 FIX 2026-08-08: Text name labels on ALL PvP panels (1v1 + summon).
	// Portraits removed for all PvP - text label replaces them.
	// Placement spec:
	//   - Horizontally centered within EACH panel's own width (panel-center, not canvas-center)
	//   - Vertically centered in the top strip (between panel top border and scroll-divider line)
	//   - Font size shrinks to fit panel inner width with 10px margin on each side
	//
	// Panel geometry (canvas coordinates):
	//   Left panel:  X=-22 to 431, center X = 204.5
	//   Right panel: X=549 to 1002, center X = 775.5
	//   Top border (ornate):  Y=469 to ~489 (panel Y=20)
	//   Scroll divider line:  Y=549 (panel Y=80)
	//   Top strip center:     Y=(489+549)/2 = 519
	if req.CombatType == "PVP" && len(req.Players) >= 2 {
		// Panel centers (using full panel bounds, not visible bounds)
		panelCenters := []float64{204.5, 775.5}
		// Top strip vertical center (between ornate top border and scroll divider)
		const labelY = 519.0
		// Panel inner width (between ornate side borders, ~20px each side)
		const panelInnerW = 453.0 - 40.0 // 413px
		// Max text width with 10px margin each side
		const maxTextW = panelInnerW - 20.0 // 393px

		boldFontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
		for i, player := range req.Players {
			// Draw HP/EN bars for player 2 in PvP-summon (player 1 bars drawn in section 4)
			if i == 1 {
				// P2 bars: positioned WITHIN right panel (same offsets as left panel bars).
				// Left panel bars at canvas X=54, 144, 235 (offsets 76, 166, 257 from panel left at -22).
				// Right panel at X=549. Same offsets: 549+76=625, 549+166=715, 549+257=806.
				// Reversed segment order (seg3, seg2, seg1) + flipped images for mirrored fill.
				// normX values: 625-694=-69, 715-694=21, 806-694=112.
				p2HpCoords := []int{-69, 21, 112}
				p2EnCoords := []int{-73, 17, 108}
				p2HpSeg := float64(player.MaxHP) / 3.0
				p2EnSeg := float64(player.MaxEnergy) / 3.0
				for j := 0; j < 3; j++ {
					segIdx := 2 - j // reverse: j=0→segIdx=2, j=1→segIdx=1, j=2→segIdx=0
					hCur := math.Max(0, math.Min(p2HpSeg, float64(player.CurrentHP)-(float64(segIdx)*p2HpSeg)))
					drawBarFlipped(dc, uiPath, normX(p2HpCoords[j]), normY(209), hCur, p2HpSeg, "hp", 121, 47)
					eCur := math.Max(0, math.Min(p2EnSeg, float64(player.Energy)-(float64(segIdx)*p2EnSeg)))
					drawBarFlipped(dc, uiPath, normX(p2EnCoords[j]), normY(256), eCur, p2EnSeg, "mana", 119, 42)
				}
			}

			name := player.Name
			if name == "" {
				name = player.Species
			}
			if name == "" {
				continue
			}

			// Try font sizes from 28 down to 14 until the name fits
			for fontSize := 28; fontSize >= 14; fontSize -= 2 {
				face, err := utils.LoadFont(boldFontPath, float64(fontSize))
				if err != nil {
					continue
				}
				dc.SetFontFace(face)
				textW, _ := dc.MeasureString(name)

				if textW <= maxTextW || fontSize == 14 {
					// Draw text centered in panel
					// Shadow for readability
					dc.SetColor(color.RGBA{0, 0, 0, 200})
					dc.DrawStringAnchored(name, panelCenters[i]+2, labelY+2, 0.5, 0.5)
					// White text
					dc.SetColor(color.RGBA{255, 255, 255, 255})
					dc.DrawStringAnchored(name, panelCenters[i], labelY, 0.5, 0.5)
					break
				}
			}
		}
	}

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

		// 💡 FIX 2026-09-11 (visual audit round 2): REMOVED the portrait block.
		// It resized the player sprite to 314px, cropped the top 30% and drew it
		// INSIDE the left state panel (x≈34, y≈500) where the oversized head
		// covered the panel title and the HP/MP bars crossed the face
		// (pixel-verified in 9 of 41 audit scenes; worst on AVATAR where the hat
		// covered the whole panel). PvP panels are text-only and clean - PvE now
		// matches. The dead PvP-P2 portrait sub-block (unreachable behind the
		// != PVP guard) is removed with it.
		_ = assetsPath

		// 6. Small full-body sprites on battlefield
		// 💡 FIX 2026-08-08: Three-way split:
		//   - PvP-summon: sprites drawn in dedicated path above (skip here)
		//   - PvP-1v1: drawPvPFighter (shared function)
		//   - PvE: player formation on battlefield
		if req.CombatType == "PVP" && !(req.CombatType == "PVP" && len(req.Players) >= 2 && req.Players[0].Mode == "summon" && req.Players[0].Species != "" && req.Players[1].Mode == "summon" && req.Players[1].Species != "") {
			// 💡 FIX 2026-08-08: Unified PvP positioning with PvE.
			// BEFORE: PvP used X=300/690 with sprite width=160 - player 1 at X=300
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
				// Crop FIRST, then resize by fixed HEIGHT.
				// Humans → 150px, summon-mode players → 130px.
				sprite = cropToVisibleBounds(sprite)
				var targetH int
				if player.Mode == "summon" && player.Species != "" {
					targetH = pvpSummonSpriteH // 130
				} else {
					targetH = pvpHumanSpriteH // 150
				}
				sprite = imaging.Resize(sprite, 0, targetH, imaging.NearestNeighbor)
				if flip {
					sprite = imaging.FlipH(sprite)
				}

				// Convert feet to draw position
				sW := sprite.Bounds().Dx()
				sH := sprite.Bounds().Dy()
				drawX := feetX - sW/2
				drawY := feetY - sH

				// Shadow at feet - wide flat ellipse (looks like ground shadow, doesn't overlap panel)
				utils.DrawShadowEllipse(dc, float64(feetX), float64(feetY)-2, float64(sW)*0.6, 12, 0.85)

				if isAtk {
					queueTurnCrystal(float64(feetX), float64(drawY), float64(sH))
				}

				dc.DrawImage(sprite, drawX, drawY)

				hpPercent := 0.0
				if player.MaxHP > 0 {
					hpPercent = math.Max(0, math.Min(1, float64(player.CurrentHP)/float64(player.MaxHP)))
				}
				hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
				if err == nil && hpPercent > 0 {
					// 💡 FIX 2026-09-11: fixed 100px track + border so the bar
					// no longer collapses to a floating hairline at low HP.
					drawOverheadHPBar(dc, float64(drawX+10), float64(drawY-15), 100, 10, hpPercent, hpBarImg)
				}
			}

			// 💡 FIX 2026-08-08: PvP-1v1 positions match PvP-summon (symmetric, away from center).
			// Player 1 (left): X=220 - same as PvP-summon left position.
			//   Was X=500 (almost at canvas center 512, looked like "in the middle").
			// Player 2 (right): X=800 - same as PvP-summon right position.
			// Feet Y: 450 - raised from 465 to clear shadow from panel.
			//   Shadow extends 16px below feet (radius_y=18, center at feetY-2).
			//   At Y=465, shadow bottom=481, overlapped panel top (469) by 12px.
			//   At Y=445, shadow bottom=461, 8px clear of panel top. ✅
			const pvp1v1FeetY = 455
			var p1Flip bool
			if p.Mode == "summon" && p.Species != "" {
				p1SpriteFile := filepath.Base(GetSummonSpritePath(p.Species, assetsPath))
				p1Flip = flipForSide(p1SpriteFile, true)
			} else {
				p1SpriteFile := GetCharacterSpriteFile(p.Class, p.SpriteIndex, assetsPath)
				p1Flip = flipForSide(filepath.Base(p1SpriteFile), true)
			}
			drawPvPFighter(p, 220, pvp1v1FeetY, p1Flip, isAttacker("player", 0))
			if len(req.Players) > 1 {
				var p2Flip bool
				if req.Players[1].Mode == "summon" && req.Players[1].Species != "" {
					p2SpriteFile := filepath.Base(GetSummonSpritePath(req.Players[1].Species, assetsPath))
					p2Flip = flipForSide(p2SpriteFile, false)
				} else {
					p2SpriteFile := GetCharacterSpriteFile(req.Players[1].Class, req.Players[1].SpriteIndex, assetsPath)
					p2Flip = flipForSide(filepath.Base(p2SpriteFile), false)
				}
				drawPvPFighter(req.Players[1], 800, pvp1v1FeetY, p2Flip, isAttacker("player", 1))
			}
		} else if !(req.CombatType == "PVP" && len(req.Players) >= 2 && req.Players[0].Mode == "summon" && req.Players[0].Species != "" && req.Players[1].Mode == "summon" && req.Players[1].Species != "") {
			// PVE: draw ALL players in formation on the battlefield.
			// Uses playerQueue + Y-sort (same pattern as enemies) so
			// front-row players draw ON TOP of back-row players.
			type PlayerRenderItem struct {
				img       image.Image
				x, y      float64
				feetX     float64
				feetY     float64
				spriteW   float64
				origIndex int
				name      string
				maxHP     int
				curHP     int
			}
			var playerQueue []PlayerRenderItem

			for pi, cp := range req.Players {
				if cp.CurrentHP <= 0 && pi > 0 {
					continue
				}
				var cpPath string
				if cp.Mode == "summon" && cp.Species != "" {
					cpPath = GetSummonSpritePath(cp.Species, assetsPath)
				} else {
					cpPath = GetCharacterSpritePath(cp.Class, cp.SpriteIndex, assetsPath)
				}
				cpSprite, err := utils.LoadImage(cpPath)
				if err != nil {
					continue
				}
				if cp.CurrentHP <= 0 {
					cpSprite = utils.TintImage(cpSprite, color.RGBA{80, 0, 80, 180})
				}
				// crop-first-then-resize-by-height
				cpSprite = cropToVisibleBounds(cpSprite)
				cpSprite = imaging.Resize(cpSprite, 0, pvePlayerSpriteH, imaging.NearestNeighbor)

				// facing: left side faces RIGHT
				var cpSpriteFile string
				if cp.Mode == "summon" && cp.Species != "" {
					cpSpriteFile = filepath.Base(GetSummonSpritePath(cp.Species, assetsPath))
				} else {
					cpSpriteFile = GetCharacterSpriteFile(cp.Class, cp.SpriteIndex, assetsPath)
				}
				if flipForSide(cpSpriteFile, true) {
					cpSprite = imaging.FlipH(cpSprite)
				}

				cpSlot := slotFor(PlayerSlots, pi)
				cpFeetX := cpSlot.X
				cpFeetY := cpSlot.Y
				cpSpriteW := cpSprite.Bounds().Dx()
				cpSpriteH := cpSprite.Bounds().Dy()
				cpX := cpFeetX - float64(cpSpriteW)/2
				cpY := cpFeetY - float64(cpSpriteH)

				cpName := cp.Name
				if cpName == "" {
					cpName = cp.Species
				}

				playerQueue = append(playerQueue, PlayerRenderItem{
					img: cpSprite, x: cpX, y: cpY,
					feetX: cpFeetX, feetY: cpFeetY,
					spriteW:   float64(cpSpriteW),
					origIndex: pi, name: cpName,
					maxHP: cp.MaxHP, curHP: cp.CurrentHP,
				})
			}

			// Sort by feet Y (back to front - lower Y = further back = drawn first).
			sort.Slice(playerQueue, func(i, j int) bool {
				return playerQueue[i].feetY < playerQueue[j].feetY
			})

			// Draw all players in Y-sorted order.
			for _, pr := range playerQueue {
				utils.DrawShadow(dc, pr.feetX, pr.feetY-2, pr.spriteW*0.6, 0.85)
				if isAttacker("player", pr.origIndex) {
					queueTurnCrystal(pr.feetX, pr.y, float64(pr.img.Bounds().Dy()))
				}
				dc.DrawImage(pr.img, int(pr.x), int(pr.y))

				hpBarTopY := pr.y - 12
				if pr.maxHP > 0 && pr.curHP > 0 {
					hpBarImg, err := utils.LoadImage(uiPath("hp5.png"))
					if err == nil {
						hpPerc := float64(pr.curHP) / float64(pr.maxHP)
						bx := pr.feetX - 40
						by := pr.y - 12
						hpBarTopY = by
						// 💡 FIX 2026-09-11: fixed 80px track + border.
						drawOverheadHPBar(dc, bx, by, 80, 10, hpPerc, hpBarImg)
					}
				}
				pillQueue = append(pillQueue, pillItem{cx: pr.feetX, bottomY: hpBarTopY - 2, name: pr.name})
			}

			// Main player (player[0]) name inside the left UI panel top strip.
			if len(req.Players) > 0 {
				p0 := req.Players[0]
				p0Name := p0.Name
				if p0Name == "" {
					p0Name = p0.Species
				}
				if p0Name != "" {
					p0BoldFont := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
					const p0PanelCenterX = 204.5
					const p0LabelY = 519.0
					const p0PanelInnerW = 453.0 - 40.0
					const p0MaxTextW = p0PanelInnerW - 20.0
					for fontSize := 28; fontSize >= 14; fontSize -= 2 {
						face, ferr := utils.LoadFont(p0BoldFont, float64(fontSize))
						if ferr != nil {
							continue
						}
						dc.SetFontFace(face)
						textW, _ := dc.MeasureString(p0Name)
						if textW <= p0MaxTextW || fontSize == 14 {
							dc.SetColor(color.RGBA{0, 0, 0, 200})
							dc.DrawStringAnchored(p0Name, p0PanelCenterX+2, p0LabelY+2, 0.5, 0.5)
							dc.SetColor(color.RGBA{255, 255, 255, 255})
							dc.DrawStringAnchored(p0Name, p0PanelCenterX, p0LabelY, 0.5, 0.5)
							break
						}
					}
				}
			}
		}
	}

	// 💡 FIX 2026-09-11: single collision-resolved nameplate pass.
	// All pills (enemies, summons, players, duelists) were queued while their
	// sprites drew. Now: sort top-of-screen first, then push any pill DOWN
	// until its rect no longer intersects an already-placed pill, then draw.
	// Result: no pill is ever covered by a sprite (they all draw last) and no
	// two pills overlap (E5/E8/E11 label soup fixed). Shifting goes DOWN from
	// the bar (into empty sky above the formation) which reads naturally.
	{
		const passGap = 2.0
		type placedRect struct{ x0, y0, x1, y1 float64 }
		var placed []placedRect
		// top-most pills first (smallest bottomY)
		sort.SliceStable(pillQueue, func(i, j int) bool { return pillQueue[i].bottomY < pillQueue[j].bottomY })
		for _, pq := range pillQueue {
			if pq.name == "" {
				continue
			}
			// measure with the same font logic drawNameplatePill uses
			w := pillWidthFor(dc, pq.name, assetsPath)
			by := pq.bottomY
			px0 := pq.cx - w/2
			px1 := pq.cx + w/2
			for tries := 0; tries < 40; tries++ {
				collision := false
				for _, r := range placed {
					if px0 < r.x1 && px1 > r.x0 && by > r.y0 && by-pillH < r.y1 {
						collision = true
						break
					}
				}
				if !collision {
					break
				}
				by += pillH + passGap // push down (toward the viewer, empty field)
				if by > 452 {         // never slide into the UI panel zone
					break
				}
			}
			placed = append(placed, placedRect{px0, by - pillH, px1, by})
			drawNameplatePill(dc, pq.name, pq.cx, by, assetsPath)
		}
	}

	// 💡 2026-09-11: Floating turn-crystal pass - painted LAST so the crystal
	// hovers over HP bars and nameplates, never underneath them. Anchored
	// above the head with clearance for the name pill (pillH) that sits on
	// the HP bar. Replaces the removed ground-ellipse indicator.
	if crystalLoadErr == nil && len(crystalQueue) > 0 {
		for _, cq := range crystalQueue {
			ch := cq.spriteH * 0.30
			if ch < 24 {
				ch = 24
			}
			if ch > 90 {
				ch = 90
			}
			cw := ch * float64(crystalSrc.Bounds().Dx()) / float64(crystalSrc.Bounds().Dy())
			if cw < 12 {
				cw = 12
			}
			img := imaging.Resize(crystalSrc, int(cw), int(ch), imaging.NearestNeighbor)
			bottom := cq.headTopY - pillH - 22
			cx := cq.cx
			cy := bottom - ch/2
			// Soft glow halo (outer faint, inner brighter)
			// 💡 color.NRGBA (straight alpha) - color.RGBA with channel > alpha
			// is an INVALID premultiplied color and overflows the src-over
			// composite into garbage saturated rings (the old "green circle"
			// bug was the same overflow on the golden ellipse).
			dc.SetColor(color.NRGBA{R: 160, G: 210, B: 255, A: 40})
			dc.DrawCircle(cx, cy, cw*1.05)
			dc.Fill()
			dc.SetColor(color.NRGBA{R: 110, G: 170, B: 255, A: 70})
			dc.DrawCircle(cx, cy, cw*0.72)
			dc.Fill()
			dc.DrawImage(img, int(cx-cw/2), int(bottom-ch))
		}
	}

	// 7. Banner Text (Overlaid ON the banner) - FIXED
	if req.Rank != "" || len(req.Players) > 0 || req.Floor > 0 {
		var text string

		// 💡 Abyss floor: show "FLOOR N" when Floor > 0
		if req.Floor > 0 {
			text = fmt.Sprintf("FLOOR %d", req.Floor)
		} else if req.CombatType == "PVP" {
			text = "PVP MATCH"
		} else {
			text = req.Rank
			if text == "" && len(req.Players) > 0 {
				text = req.Players[0].AdventurerRank
			}
			if text == "" {
				text = "F"
			}
			text = text + " RANK"
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
	buf, ctype, err := utils.EncodeImageToBufferFormat(dc.Image(), c.Query("fmt"), 90)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to encode image"})
		return
	}

	c.Data(200, ctype, buf)
}

func GenerateEndScreen(c *gin.Context) {
	// 2026-09-14 REDESIGN (owner: "update the victory and defeat image
	// cards with the style"): the old gradient + rewards-panel renderer
	// is retired. The handler now delegates to WriteEndCard (portrait.go),
	// which paints the lamoot wood+gold+parchment portrait family on
	// bg_VICTORY/bg_DEFEAT. Payload keeps its legacy fields
	// (text/victory/gold/xp/items) and gains optional player/enemy
	// sprite, rank, floor, background and caption fields.
	var req EndCardPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	WriteEndCard(c, req)
}

func normX(x int) int { return x + OFF_X }
func normY(y int) int { return y + OFF_Y }

// drawOverheadHPBar draws a fixed-width overhead HP bar: dark track + 1px
// border + hp5 texture fill clipped to the current HP percentage.
// 💡 NEW 2026-09-11 (visual audit): replaces the raw hp5 sliver that shrank
// to 1px and floated with no track at low HP.
func drawOverheadHPBar(dc *gg.Context, x, y, w, h float64, pct float64, fillImg image.Image) {
	// 💡 ROUND-4 FIX (owner: "HP bar texture too low"): the hp5.png texture is
	// a PANEL segment sprite - only ~26% of its pixels are opaque, so the
	// overhead bar read as an empty dark sliver even at FULL HP. The fill is
	// now DRAWN directly: continuous red->orange gradient at full opacity
	// plus a 1px gloss line. Always readable at any HP value.
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	dc.SetRGBA(0.04, 0.04, 0.06, 0.92)
	dc.DrawRoundedRectangle(x, y, w, h, 3)
	dc.Fill()
	if pct > 0 {
		innerX := x + 1
		innerY := y + 1
		innerW := w - 2
		innerH := h - 2
		fillW := innerW * pct
		if fillW < 1.5 {
			fillW = 1.5
		}
		steps := int(fillW)
		if steps < 1 {
			steps = 1
		}
		for i := 0; i < steps; i++ {
			t := float64(i) / float64(steps)
			r := 0.86 + 0.14*t
			g := 0.12 + 0.55*t
			b := 0.10 + 0.05*t
			dc.SetRGBA(r, g, b, 1)
			dc.DrawRectangle(innerX+float64(i), innerY, 1.25, innerH)
			dc.Fill()
		}
		dc.SetRGBA(1, 1, 1, 0.35)
		dc.DrawRectangle(innerX, innerY, fillW, 1.5)
		dc.Fill()
	}
	dc.SetRGBA(0, 0, 0, 0.9)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x+0.5, y+0.5, w-1, h-1, 3)
	dc.Stroke()
}

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

// drawBarFlipped draws the HP/EN bar with the image flipped horizontally.
// Used for the right (mirrored) panel so the bar fills right-to-left in canvas
// coordinates = left-to-right in panel-local coordinates.
func drawBarFlipped(dc *gg.Context, uiPath func(string) string, x, y int, current, max float64, typePrefix string, w, h int) {
	if max <= 0 {
		max = 1
	}
	percent := current / max
	spriteNum := int(math.Min(5, math.Max(1, math.Round(percent*4)+1)))

	filename := fmt.Sprintf("%s%d.png", typePrefix, spriteNum)
	img, err := utils.LoadImage(uiPath(filename))
	if err == nil {
		img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
		img = imaging.FlipH(img)
		dc.DrawImage(img, x, y)
	}
}

// drawNameplatePill draws a small translucent name pill above a battlefield
// entity's HP bar. Sits in a consistent slot relative to each entity's own HP
// bar, so it never competes with the sprite for space and never reaches the
// entity above it in the depth stack.
// pillWidthFor mirrors drawNameplatePill's font sizing to return the pill width
// the collision pass will need, without drawing anything.
func pillWidthFor(dc *gg.Context, name string, assetsPath string) float64 {
	if name == "" {
		return 0
	}
	boldFontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
	const pillMaxW = 90.0
	const pillMinW = 50.0
	const padX = 6.0
	for fontSize := 12; fontSize >= 9; fontSize -= 1 {
		face, err := utils.LoadFont(boldFontPath, float64(fontSize))
		if err != nil {
			continue
		}
		dc.SetFontFace(face)
		w, _ := dc.MeasureString(name)
		if w <= pillMaxW-2*padX || fontSize == 9 {
			pillW := w + 2*padX
			if pillW < pillMinW {
				pillW = pillMinW
			}
			return pillW
		}
	}
	return pillMinW
}

func drawNameplatePill(dc *gg.Context, name string, centerX, bottomY float64, assetsPath string) {
	if name == "" {
		return
	}
	boldFontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
	const pillH = 16.0
	const pillMaxW = 90.0
	const pillMinW = 50.0
	const padX = 6.0

	var chosenSize float64
	var textW float64
	for fontSize := 12; fontSize >= 9; fontSize -= 1 {
		face, err := utils.LoadFont(boldFontPath, float64(fontSize))
		if err != nil {
			continue
		}
		dc.SetFontFace(face)
		w, _ := dc.MeasureString(name)
		if w <= pillMaxW-2*padX || fontSize == 9 {
			chosenSize = float64(fontSize)
			textW = w
			break
		}
	}
	if chosenSize == 0 {
		return
	}

	pillW := textW + 2*padX
	if pillW < pillMinW {
		pillW = pillMinW
	}
	pillX := centerX - pillW/2
	pillY := bottomY - pillH

	// Pill background (translucent dark)
	dc.SetColor(color.RGBA{0, 0, 0, 165})
	dc.DrawRoundedRectangle(pillX, pillY, pillW, pillH, 3)
	dc.Fill()
	// Pill border (thin white)
	dc.SetColor(color.RGBA{255, 255, 255, 180})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(pillX, pillY, pillW, pillH, 3)
	dc.Stroke()

	// Name text (white, centered in pill)
	if face, err := utils.LoadFont(boldFontPath, chosenSize); err == nil {
		dc.SetFontFace(face)
		dc.SetColor(color.RGBA{255, 255, 255, 255})
		dc.DrawStringAnchored(name, centerX, pillY+pillH/2, 0.5, 0.5)
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
		filename = "spark_1.png" // Grassland
	case "E":
		filename = "spark_2.png" // Forest
	case "D":
		filename = "spark_3.png" // Cave
	case "C":
		filename = "spark_4.png" // Desert
	case "B":
		filename = "spark_5.png" // Snow
	case "A":
		filename = "spark_6.png" // Volcano
	case "S", "SS":
		filename = "spark_7.png" // Dark castle
	case "SSS":
		filename = "spark_8.png" // Abyss
	case "GOD":
		filename = "spark_10.png" // Divine realm
	case "PVP":
		filename = "spark_15.png" // Arena
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
