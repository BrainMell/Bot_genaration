package combat

// ============================================
// 🏹 HUNTING CARD RENDERER
// ============================================
// Renders a stylized "hunting result" image card showing the player,
// the hunted animal, the loot item, and the biome.
//
// Payload:
//   {
//     "playerName":   "Mellow",
//     "playerClass":  "RANGER",
//     "biome":        "forest",           // forest | plains | mountains | swamp
//     "animal":       "Deer",             // Rabbit | Deer | Bear | Boar | Wolf
//     "animalSprite": "deer.png",         // optional: filename in rpgasset/enemies/
//     "item":         "Fresh Venison",
//     "itemRarity":   "UNCOMMON",
//     "xp":           25,
//     "zeni":         50,
//     "rank":         "C"
//   }
//
// Returns PNG.

import (
        "image/color"
        "math"
        "path/filepath"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

var huntBiomeBackground = map[string]string{
        "forest":    "forest1.png",
        "plains":    "sand.png",
        "mountains": "ice.png",
        "swamp":     "background2.png",
        "":          "forest1.png",
}

var huntAnimalSprite = map[string]string{
        "RABBIT": "Bat_0000_dark.png", // closest small creature
        "DEER":   "earth (1).png",     // earth creature = deer-sized
        "BEAR":   "mutated (1).png",   // mutated = bear-sized
        "BOAR":   "earth (2).png",
        "WOLF":   "Werewolf_0004_brown.png",
        "":       "Bat_0000_dark.png",
}

var huntRarityColor = map[string]string{
        "COMMON":    "#9E9E9E",
        "UNCOMMON":  "#4CAF50",
        "RARE":      "#2196F3",
        "EPIC":      "#9C27B0",
        "LEGENDARY": "#FF9800",
        "MYTHIC":    "#E91E63",
}

// GenerateHuntCard renders a hunting result image card.
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
        W, H := 800, 500
        Wf, Hf := float64(W), float64(H)
        dc := gg.NewContext(W, H)

        // ── Background ──
        bgFile := huntBiomeBackground[req.Biome]
        bgPath := filepath.Join(assetsPath, "rpgasset", "environment", bgFile)
        if bgImg, err := utils.LoadImage(bgPath); err == nil {
                bgImg = imaging.Fill(bgImg, W, H, imaging.Center, imaging.NearestNeighbor)
                dc.DrawImage(bgImg, 0, 0)
        } else {
                // Fallback gradient
                for y := 0; y < H; y++ {
                        t := float64(y) / float64(H)
                        dc.SetRGB(0.1+0.05*t, 0.15+0.05*t, 0.1+0.03*t)
                        dc.DrawRectangle(0, float64(y), Wf, 1)
                        dc.Fill()
                }
        }

        // Dark overlay
        dc.SetColor(color.RGBA{0, 0, 0, 110})
        dc.DrawRectangle(0, 0, Wf, Hf)
        dc.Fill()

        // ── Top banner ──
        dc.SetRGBA(0, 0, 0, 0.7)
        dc.DrawRectangle(0, 0, Wf, 60)
        dc.Fill()

        fontPath := filepath.Join(assetsPath, "rpgasset", "ui", "Inter-Bold.ttf")
        if face, err := utils.LoadFont(fontPath, 32); err == nil {
                dc.SetFontFace(face)
                dc.SetRGB(1, 0.85, 0.3) // gold
                dc.DrawStringAnchored("HUNT SUCCESSFUL", Wf/2, 30, 0.5, 0.5)
        }

        // ── Player sprite (left) ──
        playerSpriteFile := "Fighter1.png" // default
        if req.PlayerClass != "" {
                if sprites, ok := CharacterSprites[req.PlayerClass]; ok && len(sprites) > 0 {
                        playerSpriteFile = sprites[0]
                }
        }
        playerSpritePath := filepath.Join(assetsPath, "rpgasset", "characters", playerSpriteFile)
        if pImg, err := utils.LoadImage(playerSpritePath); err == nil {
                pImg = imaging.Resize(pImg, 220, 0, imaging.NearestNeighbor)
                // Flip horizontally so player faces right (toward animal)
                pImg = imaging.FlipH(pImg)
                // Shadow
                utils.DrawShadow(dc, 130, 380, 90, 0.6)
                dc.DrawImage(pImg, 20, 200)
        }

        // ── Animal sprite (right) ──
        animalSpriteFile := req.AnimalSprite
        if animalSpriteFile == "" {
                if req.Animal != "" {
                        animalSpriteFile = huntAnimalSprite[req.Animal]
                }
                if animalSpriteFile == "" {
                        animalSpriteFile = "Bat_0000_dark.png"
                }
        }
        animalPath := filepath.Join(assetsPath, "rpgasset", "enemies", animalSpriteFile)
        if aImg, err := utils.LoadImage(animalPath); err == nil {
                aImg = imaging.Resize(aImg, 200, 0, imaging.NearestNeighbor)
                // Tint red (defeated)
                aImg = utils.TintImage(aImg, color.RGBA{255, 0, 0, 100})
                // Shadow
                utils.DrawShadow(dc, 600, 380, 80, 0.6)
                dc.DrawImage(aImg, 540, 200)
        }

        // ── Loot panel (bottom) ──
        rarityColor := huntRarityColor[req.ItemRarity]
        if rarityColor == "" {
                rarityColor = huntRarityColor["COMMON"]
        }
        rarityRGBA := utils.ParseHexColor(rarityColor)

        // Panel background
        dc.SetRGBA(0, 0, 0, 0.75)
        drawRoundedRect(dc, 20, 410, Wf-40, 80, 8)
        dc.Fill()

        // Rarity-colored left border
        dc.SetColor(rarityRGBA)
        dc.DrawRectangle(20, 410, 6, 80)
        dc.Fill()

        // Item name (large)
        if face, err := utils.LoadFont(fontPath, 22); err == nil {
                dc.SetFontFace(face)
                dc.SetRGB(1, 1, 1)
                dc.DrawStringAnchored(req.Item, 40, 440, 0, 0.5)
        }

        // Item rarity + rewards (smaller)
        if face, err := utils.LoadFont(fontPath, 16); err == nil {
                dc.SetFontFace(face)
                dc.SetColor(rarityRGBA)
                dc.DrawStringAnchored(req.ItemRarity, 40, 470, 0, 0.5)

                // XP + Zeni on the right
                dc.SetRGB(0.4, 1, 0.4)
                dc.DrawStringAnchored("+"+itoa(req.XP)+" XP", Wf-180, 440, 0, 0.5)
                dc.SetRGB(1, 0.85, 0.3)
                dc.DrawStringAnchored("+"+itoa(req.Zeni)+" Zeni", Wf-180, 470, 0, 0.5)
        }

        // ── Player name (under player sprite) ──
        if face, err := utils.LoadFont(fontPath, 18); err == nil {
                dc.SetFontFace(face)
                dc.SetRGBA(1, 1, 1, 0.9)
                dc.DrawStringAnchored(req.PlayerName, 130, 410, 0.5, 0.5)
        }

        // ── Animal name (under animal sprite, with red "DEFEATED" tag) ──
        if face, err := utils.LoadFont(fontPath, 18); err == nil {
                dc.SetFontFace(face)
                dc.SetRGBA(1, 0.4, 0.4, 0.95)
                label := req.Animal + " (defeated)"
                if req.Animal == "" {
                        label = "defeated"
                }
                dc.DrawStringAnchored(label, 620, 410, 0.5, 0.5)
        }

        // ── Decorative top corners (rank badge) ──
        if req.Rank != "" {
                if face, err := utils.LoadFont(fontPath, 16); err == nil {
                        dc.SetFontFace(face)
                        // Rank badge top-right
                        rankRGBA := utils.ParseHexColor(getRankColor(req.Rank))
                        dc.SetColor(rankRGBA)
                        drawRoundedRect(dc, Wf-90, 10, 70, 40, 6)
                        dc.Fill()
                        dc.SetRGB(1, 1, 1)
                        dc.DrawStringAnchored(req.Rank+"-RANK", Wf-55, 30, 0.5, 0.5)
                }
        }

        // ── Vignette ──
        for i := 0; i < 30; i++ {
                dc.SetRGBA(0, 0, 0, 0.015)
                dc.DrawRectangle(0, 0, float64(i), Hf)
                dc.Fill()
                dc.DrawRectangle(float64(W-i), 0, float64(i), Hf)
                dc.Fill()
        }

        // Encode
        buf, err := utils.EncodeImageToBuffer(dc.Image())
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode hunt card"})
                return
        }

        c.Data(200, "image/png", buf)
}

// drawRoundedRect is a helper that draws a rounded rectangle path (does not fill — caller must Fill).
func drawRoundedRect(dc *gg.Context, x, y, w, h, r float64) {
        dc.DrawRoundedRectangle(x, y, w, h, r)
}

// itoa converts int to string without importing strconv (keeps imports minimal).
func itoa(n int) string {
        if n == 0 {
                return "0"
        }
        neg := n < 0
        if neg {
                n = -n
        }
        var buf [20]byte
        i := len(buf)
        for n > 0 {
                i--
                buf[i] = byte('0' + n%10)
                n /= 10
        }
        if neg {
                i--
                buf[i] = '-'
        }
        return string(buf[i:])
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

// Unused but kept for future expansion (silences linter about math import)
var _ = math.Pi
