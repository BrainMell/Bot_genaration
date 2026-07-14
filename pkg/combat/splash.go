package combat

import (
        "math"
        "path/filepath"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

// GenerateBossSplash renders a full-screen boss splash image.
// Payload: { name, spriteFilename, flavorText, tier }
// tier controls the color theme: "S" (red), "SS" (purple), "SSS" (gold),
// "TRIAL" (blue), "DRAGON" (orange), "RAID" (cyan).
// Returns PNG (the JS side can convert to GIF if needed).
func GenerateBossSplash(c *gin.Context) {
        var req struct {
                Name           string `json:"name"`
                SpriteFilename string `json:"spriteFilename"`
                FlavorText     string `json:"flavorText"`
                Tier           string `json:"tier"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        assetsPath := "assets"

        // 1. Background - dark gradient based on tier
        dc := gg.NewContext(CANVAS_W, CANVAS_H)
        var bgR, bgG, bgB float64
        switch req.Tier {
        case "S":
                bgR, bgG, bgB = 0.15, 0.02, 0.02 // dark red
        case "SS":
                bgR, bgG, bgB = 0.10, 0.02, 0.15 // dark purple
        case "SSS":
                bgR, bgG, bgB = 0.15, 0.10, 0.02 // dark gold
        case "TRIAL":
                bgR, bgG, bgB = 0.02, 0.05, 0.15 // dark blue
        case "DRAGON":
                bgR, bgG, bgB = 0.18, 0.06, 0.02 // dark orange
        case "RAID":
                bgR, bgG, bgB = 0.02, 0.15, 0.18 // dark cyan
        default:
                bgR, bgG, bgB = 0.05, 0.05, 0.08 // near-black
        }

        // Vertical gradient: darker at top/bottom, slightly lighter in middle
        for y := 0; y < CANVAS_H; y++ {
                t := float64(y) / float64(CANVAS_H)
                intensity := 1.0 - math.Abs(t-0.5)*0.6
                dc.SetRGB(bgR*intensity, bgG*intensity, bgB*intensity)
                dc.DrawRectangle(0, float64(y), CANVAS_W, 1)
                dc.Fill()
        }

        // 2. Vignette overlay - darken edges
        for i := 0; i < 60; i++ {
                dc.SetRGBA(0, 0, 0, 0.02)
                dc.DrawRectangle(0, 0, float64(i), CANVAS_H)
                dc.Fill()
                dc.DrawRectangle(float64(CANVAS_W-i), 0, float64(i), CANVAS_H)
                dc.Fill()
        }

        // 3. Boss sprite - centered, large
        if req.SpriteFilename != "" {
                spritePath := filepath.Join(assetsPath, "rpgasset", "enemies", req.SpriteFilename)
                if spriteImg, err := utils.LoadImage(spritePath); err == nil {
                        spriteImg = imaging.Resize(spriteImg, 450, 0, imaging.Lanczos)
                        sx := float64(CANVAS_W/2 - spriteImg.Bounds().Dx()/2)
                        sy := float64(CANVAS_H/2 - spriteImg.Bounds().Dy()/2 - 30)
                        // Glow effect
                        for g := 8; g > 0; g-- {
                                dc.SetRGBA(1, 1, 1, 0.04)
                                dc.DrawImageAnchored(spriteImg, int(float64(CANVAS_W/2)+float64(g)), int(float64(CANVAS_H/2-30)+float64(g)), 0.5, 0.5)
                        }
                        dc.DrawImage(spriteImg, int(sx), int(sy))
                }
        }

        // 4. Boss name - top center, large fantasy font
        fontPath := utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
        if req.Name != "" {
                if face, err := utils.LoadFont(fontPath, 64); err == nil {
                        dc.SetFontFace(face)
                        switch req.Tier {
                        case "S":
                                dc.SetRGB(1, 0.3, 0.3)
                        case "SS":
                                dc.SetRGB(0.7, 0.4, 1)
                        case "SSS":
                                dc.SetRGB(1, 0.85, 0.3)
                        case "TRIAL":
                                dc.SetRGB(0.4, 0.6, 1)
                        case "DRAGON":
                                dc.SetRGB(1, 0.5, 0.2)
                        case "RAID":
                                dc.SetRGB(0.3, 0.9, 1)
                        default:
                                dc.SetRGB(1, 1, 1)
                        }
                        dc.DrawStringAnchored(req.Name, float64(CANVAS_W/2), 80, 0.5, 0.5)
                }
        }

        // 5. Tier label - small, above name
        if req.Tier != "" {
                if face, err := utils.LoadFont(fontPath, 28); err == nil {
                        dc.SetFontFace(face)
                        dc.SetRGBA(1, 1, 1, 0.7)
                        tierLabel := "BOSS"
                        switch req.Tier {
                        case "S":
                                tierLabel = "S-RANK BOSS"
                        case "SS":
                                tierLabel = "SS-RANK BOSS"
                        case "SSS":
                                tierLabel = "SSS-RANK BOSS"
                        case "TRIAL":
                                tierLabel = "TRIAL BOSS"
                        case "DRAGON":
                                tierLabel = "DRAGON BOSS"
                        case "RAID":
                                tierLabel = "WEEKLY RAID BOSS"
                        }
                        dc.DrawStringAnchored(tierLabel, float64(CANVAS_W/2), 30, 0.5, 0.5)
                }
        }

        // 6. Flavor text - bottom center, italic, smaller
        if req.FlavorText != "" {
                if face, err := utils.LoadFont(fontPath, 24); err == nil {
                        dc.SetFontFace(face)
                        dc.SetRGBA(0.9, 0.9, 0.9, 0.85)
                        words := splitWords(req.FlavorText)
                        lines := wrapText(words, 60)
                        startY := float64(CANVAS_H - 60 - (len(lines)-1)*30)
                        for i, line := range lines {
                                dc.DrawStringAnchored(line, float64(CANVAS_W/2), startY+float64(i*30), 0.5, 0.5)
                        }
                }
        }

        // 7. Bottom + top border accent
        dc.SetRGBA(1, 1, 1, 0.3)
        dc.DrawRectangle(0, float64(CANVAS_H-4), CANVAS_W, 4)
        dc.Fill()
        dc.DrawRectangle(0, 0, CANVAS_W, 4)
        dc.Fill()

        buf, err := utils.EncodeImageToBuffer(dc.Image())
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode splash image"})
                return
        }

        c.Data(200, "image/png", buf)
}

// splitWords splits a string into words by whitespace.
func splitWords(s string) []string {
        var words []string
        current := ""
        for _, ch := range s {
                if ch == ' ' || ch == '\t' || ch == '\n' {
                        if current != "" {
                                words = append(words, current)
                                current = ""
                        }
                } else {
                        current += string(ch)
                }
        }
        if current != "" {
                words = append(words, current)
        }
        return words
}

// wrapText takes words and a max line length, returns wrapped lines.
func wrapText(words []string, maxLen int) []string {
        var lines []string
        current := ""
        for _, w := range words {
                if len(current)+len(w)+1 > maxLen {
                        if current != "" {
                                lines = append(lines, current)
                        }
                        current = w
                } else {
                        if current == "" {
                                current = w
                        } else {
                                current += " " + w
                        }
                }
        }
        if current != "" {
                lines = append(lines, current)
        }
        if len(lines) == 0 {
                lines = append(lines, "")
        }
        return lines
}
