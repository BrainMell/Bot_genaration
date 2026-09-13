package combat

// ============================================
// 🏹 HUNTING CARD RENDERER — parchment redesign 2026-09-12
// ============================================
// Owner complaint: old card showed placeholder sprites ("rabbit" = pink bat,
// "deer" = earth golem, "bear" = troll) and flipped the player unconditionally
// (right-facing classes faced AWAY from the animal). Rebuilt in the approved
// lamoot wood+gold+parchment family (same as CRAFT/BREW cards).
//
// Static art baked in assets/rpgasset/ui/craft/bg_HUNT.png (build_hfd_bgs.py):
//   - leather name plate (66..340, 56..112)        → nickname drawn by us
//   - wood banner title "HUNT SUCCESSFUL"          → baked
//   - scene window (130,185)-(870,400) dark inset  → we paint forest+player+animal
//   - "SPOILS OF THE HUNT" label + divider y440    → baked
//   - ledger line y463                             → item left, rarity/XP/Zeni right
//   - wax rank seal at (70,546) r26                → drawn by us
//   - caption strip y552                           → drawn by us
//
// Animals now use real LPC art (bluecarrou LPC Animals 2022 + PixelFarm bunny,
// CC-BY-SA/GPL — see docs/asset_credits.md): rabbit_lpc / deer_lpc / bear_lpc.
// Payload unchanged: {playerName, playerClass, biome, animal, animalSprite,
//                     item, itemRarity, xp, zeni, rank}

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

var huntBiomeBackground = map[string]string{
	"forest":    "forest.png",
	"plains":    "background1.png", // grassland
	"mountains": "ice.png",
	"swamp":     "spark_4.png",
	"":          "forest.png",
}

// 2026-09-12: REAL animal art (was: RABBIT=bat, DEER=earth golem, BEAR=troll).
var huntAnimalSprite = map[string]string{
	"RABBIT": "rabbit_lpc.png",
	"DEER":   "deer_lpc.png",
	"BEAR":   "bear_lpc.png",
	"BOAR":   "boar_still.png",
	"WOLF":   "wolf_frame.png",
	"":       "rabbit_lpc.png",
}

var huntRarityColor = map[string]string{
	"COMMON":    "#9E9E9E",
	"UNCOMMON":  "#4CAF50",
	"RARE":      "#2196F3",
	"EPIC":      "#9C27B0",
	"LEGENDARY": "#FF9800",
	"MYTHIC":    "#E91E63",
}

// darker variants for the parchment ledger (mid-grays vanish on lamoot paper)
var huntLedgerRarityColor = map[string]string{
	"COMMON":    "#5E5E5E",
	"UNCOMMON":  "#1B5E20",
	"RARE":      "#0D47A1",
	"EPIC":      "#4A148C",
	"LEGENDARY": "#8D4004",
	"MYTHIC":    "#880E4F",
}

// hunt window geometry — MUST match bg_HUNT.png bake (build_hfd_bgs.py)
const (
	huntSceneX = 130.0
	huntSceneY = 185.0
	huntSceneW = 740.0
	huntSceneH = 215.0
)

func huntAsset(name string) string { return utils.GetAssetPath("rpgasset", "ui", "craft/"+name) }

// huntSanitize — printable ASCII + middle dot only (same rule as economy.sanitize);
// WhatsApp nicknames can carry emoji that Cinzel would render as tofu.
func huntSanitize(s string) string {
	var b []rune
	for _, r := range s {
		if r == '·' || (r >= 0x20 && r <= 0x7E) {
			b = append(b, r)
		}
	}
	return strings.TrimSpace(string(b))
}

// huntFitText loads font at size, shrinking until s fits maxW (min minSize).
func huntFitText(dc *gg.Context, path string, size int, s string, maxW float64, minSize int) {
	for size > minSize {
		if face, err := utils.LoadFont(path, float64(size)); err == nil {
			dc.SetFontFace(face)
			if w, _ := dc.MeasureString(s); w <= maxW {
				return
			}
		}
		size -= 2
	}
	if face, err := utils.LoadFont(path, float64(minSize)); err == nil {
		dc.SetFontFace(face)
	}
}

// GenerateHuntCard renders the parchment hunting result card (1000x600).
func GenerateHuntCard(c *gin.Context) {
	var req struct {
		PlayerName   string `json:"playerName"`
		PlayerClass  string `json:"playerClass"`
		Biome        string `json:"biome"`
		Animal       string `json:"animal"`
		AnimalSprite string `json:"animalSprite"`
		Item         string `json:"item"`
		ItemRarity   string `json:"itemRarity"`
		XP           int    `json:"xp"`
		Zeni         int    `json:"zeni"`
		Rank         string `json:"rank"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	assetsPath := "assets"
	dc := gg.NewContext(1000, 600)

	// ── baked parchment base ──
	if bgImg, err := utils.LoadImage(huntAsset("bg_HUNT.png")); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, 1000, 600)
		dc.Fill()
	}

	// ── nickname on the leather plate ──
	name := huntSanitize(req.PlayerName)
	if name == "" {
		name = "Adventurer"
	}
	huntFitText(dc, huntAsset("Cinzel.ttf"), 30, name, 240, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(name, 90, 84, 0, 0.5)

	// ── scene window: forest + player (left, facing RIGHT) + animal (right, facing LEFT) ──
	// NOTE: no gg Clip() here — clip state leaked in this gg version and erased the
	// ledger/seal/caption drawn afterwards. Every scene element is pixel-bounded to
	// the window rect by construction (Fill produces exactly WxH; sprites are
	// bottom-anchored at y=397 with height ≤ 192, x-ranges 190..846).
	sceneBottom := huntSceneY + huntSceneH - 3
	if bgFile, ok := huntBiomeBackground[req.Biome]; ok {
		if sceneImg, err := utils.LoadImage(filepath.Join(assetsPath, "rpgasset", "environment", bgFile)); err == nil {
			sceneImg = imaging.Fill(sceneImg, int(huntSceneW), int(huntSceneH), imaging.Center, imaging.NearestNeighbor)
			dc.DrawImage(sceneImg, int(huntSceneX), int(huntSceneY))
		}
	}
	// light mood overlay so sprites pop
	dc.SetColor(color.RGBA{0, 0, 0, 46})
	dc.DrawRectangle(huntSceneX, huntSceneY, huntSceneW, huntSceneH)
	dc.Fill()

	// player sprite — FIX: flip ONLY when native facing is LEFT (was unconditional FlipH)
	playerFile := "Fighter1.png"
	if req.PlayerClass != "" {
		if sprites, ok := CharacterSprites[req.PlayerClass]; ok && len(sprites) > 0 {
			playerFile = sprites[0]
		}
	}
	if pImg, err := utils.LoadImage(filepath.Join(assetsPath, "rpgasset", "characters", playerFile)); err == nil {
		pImg = imaging.Fit(pImg, 210, 192, imaging.NearestNeighbor)
		if GetSpriteFacing(playerFile) == "LEFT" {
			pImg = imaging.FlipH(pImg)
		}
		b := pImg.Bounds()
		pw, ph := float64(b.Dx()), float64(b.Dy())
		px := huntSceneX + 60
		utils.DrawShadow(dc, px+pw/2, sceneBottom-2, pw*0.45, 0.5)
		dc.DrawImage(pImg, int(px), int(sceneBottom-ph))
	}
	animalFile := req.AnimalSprite
	if animalFile == "" {
		animalFile = huntAnimalSprite[req.Animal]
		if animalFile == "" {
			animalFile = "rabbit_lpc.png"
		}
	}
	if aImg, err := utils.LoadImage(filepath.Join(assetsPath, "rpgasset", "enemies", animalFile)); err == nil {
		// LPC frames are tiny (29..63px) and imaging.Fit never upscales —
		// integer-scale with NEAREST first so the pixel art stays crisp.
		b0 := aImg.Bounds()
		aw0, ah0 := b0.Dx(), b0.Dy()
		if aw0 < 240 && ah0 < 175 {
			scale := 240 / aw0
			if s2 := 175 / ah0; s2 < scale {
				scale = s2
			}
			if scale > 1 {
				aImg = imaging.Resize(aImg, aw0*scale, ah0*scale, imaging.NearestNeighbor)
			}
		}
		aImg = imaging.Fit(aImg, 240, 175, imaging.NearestNeighbor)
		if GetSpriteFacing(animalFile) == "RIGHT" {
			aImg = imaging.FlipH(aImg)
		}
		b := aImg.Bounds()
		aw, ah := float64(b.Dx()), float64(b.Dy())
		ax := huntSceneX + huntSceneW - 24 - aw
		utils.DrawShadow(dc, ax+aw/2, sceneBottom-2, aw*0.42, 0.5)
		dc.DrawImage(aImg, int(ax), int(sceneBottom-ah))
	}

	// "<ANIMAL> CAPTURED" tag pill, top-right inside the scene window
	animalLabel := req.Animal
	if animalLabel == "" {
		animalLabel = "CREATURE"
	}
	tag := animalLabel + " — CAPTURED"
	huntFitText(dc, huntAsset("Cinzel.ttf"), 18, tag, 400, 12)
	tagW, _ := dc.MeasureString(tag)
	dc.SetColor(color.RGBA{8, 8, 8, 150})
	dc.DrawRoundedRectangle(huntSceneX+huntSceneW-tagW-26, huntSceneY+10, tagW+16, 26, 8)
	dc.Fill()
	dc.SetRGBA(1, 1, 1, 0.96)
	dc.DrawStringAnchored(tag, huntSceneX+huntSceneW-18, huntSceneY+23, 1, 0.5)

	// ── spoils ledger (y463): item left · rarity | +XP | +Zeni right ──
	item := huntSanitize(req.Item)
	if item == "" {
		item = "Unknown Item"
	}
	huntFitText(dc, huntAsset("CinzelDecBold.ttf"), 30, item, 430, 14)
	dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
	dc.DrawStringAnchored(item, 150, 463, 0, 0.5)

	rarity := req.ItemRarity
	if rarity == "" {
		rarity = "COMMON"
	}
	const ledgerRight = 860.0
	const segGap = 18.0
	zStr := fmt.Sprintf("+%d Z", req.Zeni)
	xStr := fmt.Sprintf("+%d XP", req.XP)

	huntFitText(dc, huntAsset("Cinzel.ttf"), 20, zStr, 200, 12)
	wZ, _ := dc.MeasureString(zStr)
	dc.SetRGB(150.0/255.0, 110.0/255.0, 30.0/255.0)
	dc.DrawStringAnchored(zStr, ledgerRight, 463, 1, 0.5)

	huntFitText(dc, huntAsset("Cinzel.ttf"), 20, xStr, 200, 12)
	wX, _ := dc.MeasureString(xStr)
	dc.SetRGB(96.0/255.0, 110.0/255.0, 50.0/255.0)
	dc.DrawStringAnchored(xStr, ledgerRight-wZ-segGap, 463, 1, 0.5)

	huntFitText(dc, huntAsset("Cinzel.ttf"), 20, rarity, 220, 12)
	ledgerCol := huntLedgerRarityColor[rarity]
	if ledgerCol == "" {
		ledgerCol = huntLedgerRarityColor["COMMON"]
	}
	dc.SetColor(utils.ParseHexColor(ledgerCol))
	dc.DrawStringAnchored(rarity, ledgerRight-wZ-segGap-wX-segGap, 463, 1, 0.5)

	// ── wax rank seal (70,546) r26 ──
	if req.Rank != "" {
		cx, cy, r := 70.0, 546.0, 26.0
		dc.SetColor(color.NRGBA{R: 128, G: 28, B: 40, A: 245})
		dc.DrawCircle(cx, cy, r)
		dc.Fill()
		dc.SetRGB(70.0/255.0, 12.0/255.0, 20.0/255.0)
		dc.SetLineWidth(3)
		dc.DrawCircle(cx, cy, r)
		dc.Stroke()
		dc.SetRGB(220.0/255.0, 150.0/255.0, 90.0/255.0)
		dc.SetLineWidth(2)
		dc.DrawCircle(cx, cy, r-6)
		dc.Stroke()
		huntFitText(dc, huntAsset("Cinzel.ttf"), 24, req.Rank, 60, 10)
		dc.SetRGB(250.0/255.0, 210.0/255.0, 120.0/255.0)
		dc.DrawStringAnchored(req.Rank, cx, cy-1, 0.5, 0.5)
	}

	// ── caption ──
	if face, err := utils.LoadFont(huntAsset("MedievalSharp.ttf"), 22); err == nil {
		dc.SetFontFace(face)
		dc.SetRGB(176.0/255.0, 140.0/255.0, 96.0/255.0)
		dc.DrawStringAnchored("the wilds yield — sell at HQ or craft with it", 500, 552, 0.5, 0.5)
	}

	buf, ctype, err := utils.EncodeImageToBufferFormat(dc.Image(), c.Query("fmt"), 90)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to encode hunt card"})
		return
	}
	c.Data(200, ctype, buf)
}

// getRankColor returns a hex color for the given rank letter.
func getRankColor(rank string) string {
	switch rank {
	case "F":
		return "#9E9E9E"
	case "E":
		return "#8D6E63"
	case "D":
		return "#795548"
	case "C":
		return "#558B2F"
	case "B":
		return "#2E7D32"
	case "A":
		return "#1565C0"
	case "S":
		return "#7B1FA2"
	case "SS":
		return "#C2185B"
	case "SSS":
		return "#E65100"
	case "GOD":
		return "#FFD700"
	case "DRAGON":
		return "#FF6F00"
	default:
		return "#9E9E9E"
	}
}
