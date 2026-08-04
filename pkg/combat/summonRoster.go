package combat

import (
        "fmt"
        "image"
        "image/color"
        "image/draw"
        "image/gif"
        "os"
        "path/filepath"
        "strings"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

// SummonRosterRequest is the payload from the Node bot for rendering
// an animated summon roster GIF.
type SummonRosterRequest struct {
        UserNickname string         `json:"userNickname"`
        SlotsUsed    int            `json:"slotsUsed"`
        SlotsMax     int            `json:"slotsMax"`
        Summons      []RosterSummon `json:"summons"`
        ActiveIndex  int            `json:"activeIndex"` // -1 if none
}

// RosterSummon represents one summon in the roster.
type RosterSummon struct {
        Species    string `json:"species"`
        Nickname   string `json:"nickname"`
        Level      int    `json:"level"`
        Rarity     string `json:"rarity"`
        Element    string `json:"element"`
        Archetype  string `json:"archetype"`
        Loyalty    int    `json:"loyalty"`
        HP         int    `json:"hp"`
        ATK        int    `json:"atk"`
        DEF        int    `json:"def"`
        MAG        int    `json:"mag"`
        SPD        int    `json:"spd"`
        IsDeployed bool   `json:"isDeployed"`
}

// GenerateSummonRosterGIF renders an animated GIF showing the player's
// summons standing side-by-side on a sparklinlabs background, with an
// info hub at the bottom.
func GenerateSummonRosterGIF(c *gin.Context) {
        var req SummonRosterRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        assetsPath := "assets"
        const W = 720
        const H = 720

        // ── 1. Load background ──
        bgPath := filepath.Join(assetsPath, "rpgasset", "environment", "spark_7.png")
        bgImg, err := utils.LoadImage(bgPath)
        if err != nil {
                bgImg = createSolidBackground(W, H, color.RGBA{26, 26, 46, 255})
        } else {
                bgImg = imaging.Fill(bgImg, W, H, imaging.Center, imaging.NearestNeighbor)
        }

        // ── 2. Load each summon's idle.gif ──
        // 💡 FIX 2026-08-04: GIF frames are sub-frames (only changed pixels)
        // with varying dimensions. Resizing each independently causes sprites
        // to grow/shrink/jump during animation.
        // Fix: composite each frame onto the full GIF logical canvas first.
        type gifData struct {
                frames []image.Image
                delays []int
                sprite string
        }
        var gifs []gifData
        maxFrames := 1

        for _, s := range req.Summons {
                gifPath := getSummonIdleGifPath(s.Species, assetsPath)
                if gifPath == "" || !fileExists(gifPath) {
                        continue
                }

                f, err := os.Open(gifPath)
                if err != nil {
                        continue
                }
                g, err := gif.DecodeAll(f)
                f.Close()
                if err != nil || len(g.Image) == 0 {
                        continue
                }

                // Get GIF logical canvas size
                gifW := g.Config.Width
                gifH := g.Config.Height
                if gifW <= 0 || gifH <= 0 {
                        b := g.Image[0].Bounds()
                        gifW = b.Dx()
                        gifH = b.Dy()
                }

                // Composite each sub-frame onto a full-size canvas
                fullFrames := make([]image.Image, len(g.Image))
                var prevFrame *image.RGBA
                for fi, subFrame := range g.Image {
                        fullFrame := image.NewRGBA(image.Rect(0, 0, gifW, gifH))
                        if prevFrame != nil {
                                // Copy previous frame (for proper GIF disposal)
                                copy(fullFrame.Pix, prevFrame.Pix)
                        }
                        // Draw the sub-frame at its proper offset
                        bounds := subFrame.Bounds()
                        draw.Draw(fullFrame, bounds, subFrame, bounds.Min, draw.Over)
                        fullFrames[fi] = fullFrame
                        prevFrame = fullFrame
                }

                gd := gifData{
                        frames: fullFrames,
                        delays: g.Delay,
                        sprite: s.Species,
                }
                gifs = append(gifs, gd)
                if len(g.Image) > maxFrames {
                        maxFrames = len(g.Image)
                }
        }

        // ── 3. Calculate summon positions (side by side, centered) ──
        summonSize := 150
        summonSpacing := 20
        numSummons := len(gifs)
        totalWidth := numSummons*summonSize + (numSummons-1)*summonSpacing
        startX := (W - totalWidth) / 2
        summonY := 180

        // ── 4. Render each frame ──
        var outFrames []*image.Paletted
        var outDelays []int

        for frameIdx := 0; frameIdx < maxFrames; frameIdx++ {
                // Create a new context for this frame
                dc := gg.NewContext(W, H)

                // Draw background
                dc.DrawImage(bgImg, 0, 0)

                // Dark overlay
                dc.SetColor(color.RGBA{0, 0, 0, 140})
                dc.DrawRectangle(0, 0, W, H)
                dc.Fill()

                // ── Header ──
                dc.SetColor(color.RGBA{0, 0, 0, 160})
                dc.DrawRoundedRectangle(15, 10, float64(W-30), 50, 8)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 215, 0, 100})
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(15, 10, float64(W-30), 50, 8)
                dc.Stroke()

                // Title
                dc.SetColor(color.RGBA{255, 215, 0, 255})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 22); err != nil {
                        
                }
                dc.DrawStringAnchored("🐉 SUMMON ROSTER", 30, 35, 0, 0.5)

                // Slots
                slotStr := fmt.Sprintf("%d/%d slots", req.SlotsUsed, req.SlotsMax)
                dc.SetColor(color.RGBA{255, 255, 255, 200})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 15); err != nil {
                        
                }
                dc.DrawStringAnchored(slotStr, float64(W-30), 35, 1, 0.5)

                // ── Draw each summon's idle frame ──
                for i, gd := range gifs {
                        gifFrameIdx := frameIdx % len(gd.frames)
                        spriteImg := gd.frames[gifFrameIdx]

                        // Resize the sprite to fixed square (prevents frame-to-frame size variation)
                        resized := imaging.Resize(spriteImg, summonSize, summonSize, imaging.NearestNeighbor)

                        // Draw shadow under summon
                        x := startX + i*(summonSize+summonSpacing)
                        utils.DrawShadow(dc, float64(x)+float64(summonSize)/2, float64(summonY+summonSize)-10, float64(summonSize)*0.4, 0.6)

                        // Draw the sprite
                        dc.DrawImage(resized, x, summonY)

                        // Draw name + level below sprite
                        summon := req.Summons[i]
                        name := summon.Nickname
                        if name == "" {
                                name = summon.Species
                        }
                        if len(name) > 14 {
                                name = name[:12] + "…"
                        }

                        rarityColor := getRarityColor(summon.Rarity)
                        dc.SetColor(rarityColor)
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 13); err != nil {
                                
                        }
                        dc.DrawStringAnchored(name, float64(x+summonSize/2), float64(summonY+summonSize+12), 0.5, 0.5)

                        dc.SetColor(color.RGBA{255, 255, 255, 180})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 12); err != nil {
                                
                        }
                        infoStr := fmt.Sprintf("Lv.%d %s", summon.Level, summon.Rarity)
                        dc.DrawStringAnchored(infoStr, float64(x+summonSize/2), float64(summonY+summonSize+28), 0.5, 0.5)

                        // Deployed indicator
                        if summon.IsDeployed {
                                dc.SetColor(color.RGBA{255, 215, 0, 255})
                                dc.DrawStringAnchored("⭐", float64(x+summonSize-15), float64(summonY+5), 0.5, 0.5)
                        }

                        // Loyalty bar
                        barW := summonSize - 20
                        barX := x + 10
                        barY := summonY + summonSize + 38
                        dc.SetColor(color.RGBA{0, 0, 0, 120})
                        dc.DrawRectangle(float64(barX), float64(barY), float64(barW), 5)
                        dc.Fill()
                        loyaltyPct := float64(summon.Loyalty) / 100.0
                        loyColor := color.RGBA{76, 175, 80, 255}
                        if summon.Loyalty < 50 {
                                loyColor = color.RGBA{255, 152, 0, 255}
                        }
                        if summon.Loyalty < 25 {
                                loyColor = color.RGBA{244, 67, 54, 255}
                        }
                        dc.SetColor(loyColor)
                        dc.DrawRectangle(float64(barX), float64(barY), float64(barW)*loyaltyPct, 5)
                        dc.Fill()
                }

                // ── Info Hub (bottom panel) ──
                hubY := 480
                hubH := 230
                dc.SetColor(color.RGBA{0, 0, 0, 180})
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 215, 0, 80})
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Stroke()

                // Hub title
                dc.SetColor(color.RGBA{255, 215, 0, 255})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 15); err != nil {
                        
                }
                dc.DrawStringAnchored("📋 SUMMON INFO HUB", 30, float64(hubY+15), 0, 0.5)

                // Show active summon details or summary
                if req.ActiveIndex >= 0 && req.ActiveIndex < len(req.Summons) {
                        s := req.Summons[req.ActiveIndex]
                        name := s.Nickname
                        if name == "" {
                                name = s.Species
                        }

                        dc.SetColor(color.RGBA{255, 215, 0, 255})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 18); err != nil {
                                
                        }
                        dc.DrawStringAnchored("⭐ "+name, 30, float64(hubY+42), 0, 0.5)

                        dc.SetColor(color.RGBA{255, 255, 255, 200})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 13); err != nil {
                                
                        }
                        detailStr := fmt.Sprintf("Lv.%d %s | %s | %s", s.Level, s.Rarity, s.Element, s.Archetype)
                        dc.DrawStringAnchored(detailStr, 30, float64(hubY+65), 0, 0.5)

                        // Stats row
                        statY := hubY + 90
                        stats := []struct {
                                label string
                                value int
                                color color.RGBA
                        }{
                                {"HP", s.HP, color.RGBA{255, 107, 107, 255}},
                                {"ATK", s.ATK, color.RGBA{255, 217, 61, 255}},
                                {"DEF", s.DEF, color.RGBA{79, 195, 247, 255}},
                                {"MAG", s.MAG, color.RGBA{156, 39, 176, 255}},
                                {"SPD", s.SPD, color.RGBA{76, 175, 80, 255}},
                        }
                        for si, stat := range stats {
                                sx := 30 + si*130
                                dc.SetColor(stat.color)
                                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 12); err != nil {
                                        
                                }
                                dc.DrawStringAnchored(stat.label, float64(sx), float64(statY), 0, 0.5)
                                dc.SetColor(color.RGBA{255, 255, 255, 230})
                                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 14); err != nil {
                                        
                                }
                                dc.DrawStringAnchored(fmt.Sprintf("%d", stat.value), float64(sx+35), float64(statY), 0, 0.5)
                        }

                        // Loyalty
                        loyStr := fmt.Sprintf("Loyalty: %d%%", s.Loyalty)
                        dc.SetColor(color.RGBA{255, 255, 255, 150})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 12); err != nil {
                                
                        }
                        dc.DrawStringAnchored(loyStr, 30, float64(hubY+120), 0, 0.5)
                } else {
                        dc.SetColor(color.RGBA{255, 255, 255, 150})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 13); err != nil {
                                
                        }
                        dc.DrawStringAnchored("No summon deployed. Use .summon <#> deploy to deploy one.", 30, float64(hubY+42), 0, 0.5)
                        summaryStr := fmt.Sprintf("📊 Collection: %d summons | Slots: %d/%d", len(req.Summons), req.SlotsUsed, req.SlotsMax)
                        dc.DrawStringAnchored(summaryStr, 30, float64(hubY+65), 0, 0.5)
                }

                // Hub footer — commands hint
                dc.SetColor(color.RGBA{255, 255, 255, 100})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 11); err != nil {
                        
                }
                dc.DrawStringAnchored(".summon <#> — view  |  .summon <#> deploy  |  .summon help", float64(W/2), float64(hubY+hubH-15), 0.5, 0.5)

                // Convert to paletted for GIF
                frameImg := dc.Image()
                paletted := imageToPaletted(frameImg, W, H)
                outFrames = append(outFrames, paletted)
                outDelays = append(outDelays, 10) // 10 * 10ms = 100ms per frame
        }

        // ── 5. Encode as GIF ──
        out := &gif.GIF{
                Image: outFrames,
                Delay: outDelays,
        }

        c.Header("Content-Type", "image/gif")
        if err := gif.EncodeAll(c.Writer, out); err != nil {
                c.JSON(500, gin.H{"error": "GIF encoding failed: " + err.Error()})
        }
}

// getSummonIdleGifPath maps a summon species to its idle.gif file.
func getSummonIdleGifPath(species string, assetsPath string) string {
        species = strings.ToLower(strings.TrimSpace(species))

        // Map our species IDs to sparklinlabs monster names
        speciesMap := map[string]string{
                // Starters
                "stoneguard":     "giant",
                "emberdrake":     "dragon",
                "mistwisp":       "ghost",
                "bloompixie":     "mushroom",
                // Evolved starters
                "iron_sentinel":      "giant",
                "mountain_titan":     "giant",
                "flare_wyrm":         "dragon",
                "infernal_dragon":    "dragon",
                "frost_spectre":      "ghost",
                "abyssal_phantom":    "ghost",
                "blossom_sylph":      "mushroom",
                "world_tree_spirit":  "mushroom",
                // Direct sparklinlabs names
                "bat":       "bat",
                "boar":      "boar",
                "dino":      "dino",
                "dragon":    "dragon",
                "ghost":     "ghost",
                "giant":     "giant",
                "mimic":     "mimic",
                "mushroom":  "mushroom",
                "reptile":   "reptile",
                "slime":     "slime",
                "snake":     "snake",
        }

        monsterName, ok := speciesMap[species]
        if !ok {
                // Try direct match
                monsterName = species
        }

        gifPath := filepath.Join(assetsPath, "rpgasset", "summons", "sparklinlabs", monsterName+"_idle.gif")
        if fileExists(gifPath) {
                return gifPath
        }
        return ""
}

// getRarityColor returns the RGBA color for a rarity string.
func getRarityColor(rarity string) color.RGBA {
        switch strings.ToUpper(rarity) {
        case "MYTHIC":
                return color.RGBA{233, 30, 99, 255}
        case "LEGENDARY":
                return color.RGBA{255, 152, 0, 255}
        case "EPIC":
                return color.RGBA{156, 39, 176, 255}
        case "RARE":
                return color.RGBA{33, 150, 243, 255}
        case "UNCOMMON":
                return color.RGBA{76, 175, 80, 255}
        default:
                return color.RGBA{158, 158, 158, 255}
        }
}


// imageToPaletted converts an image.Image to a paletted image for GIF encoding.
func imageToPaletted(img image.Image, w, h int) *image.Paletted {
        // Use a 256-color palette
        palette := generatePalette(img)
        paletted := image.NewPaletted(image.Rect(0, 0, w, h), palette)
        for y := 0; y < h; y++ {
                for x := 0; x < w; x++ {
                        paletted.Set(x, y, img.At(x, y))
                }
        }
        return paletted
}

// generatePalette creates a simple 256-color palette from the image.
func generatePalette(img image.Image) color.Palette {
        // Use a fixed palette with common RPG colors
        return color.Palette{
                color.RGBA{0, 0, 0, 255},       // black
                color.RGBA{255, 255, 255, 255}, // white
                color.RGBA{255, 215, 0, 255},   // gold
                color.RGBA{255, 0, 0, 255},     // red
                color.RGBA{0, 255, 0, 255},     // green
                color.RGBA{0, 0, 255, 255},     // blue
                color.RGBA{255, 255, 0, 255},   // yellow
                color.RGBA{255, 0, 255, 255},   // magenta
                color.RGBA{0, 255, 255, 255},   // cyan
                color.RGBA{128, 128, 128, 255}, // gray
                color.RGBA{192, 192, 192, 255}, // light gray
                color.RGBA{64, 64, 64, 255},    // dark gray
                color.RGBA{255, 128, 0, 255},   // orange
                color.RGBA{128, 0, 128, 255},   // purple
                color.RGBA{0, 128, 0, 255},     // dark green
                color.RGBA{128, 0, 0, 255},     // dark red
                color.RGBA{0, 0, 128, 255},     // dark blue
                color.RGBA{255, 192, 203, 255}, // pink
                color.RGBA{139, 69, 19, 255},   // brown
                color.RGBA{210, 180, 140, 255}, // tan
                color.RGBA{34, 139, 34, 255},   // forest green
                color.RGBA{72, 61, 139, 255},   // dark slate blue
                color.RGBA{47, 79, 79, 255},    // dark slate gray
                color.RGBA{106, 90, 205, 255},  // slate blue
                color.RGBA{100, 149, 237, 255}, // cornflower blue
                color.RGBA{30, 144, 255, 255},  // dodger blue
                color.RGBA{95, 158, 160, 255},  // cadet blue
                color.RGBA{60, 179, 113, 255},  // medium sea green
                color.RGBA{32, 178, 170, 255},  // light sea green
                color.RGBA{255, 69, 0, 255},    // orange red
                color.RGBA{255, 99, 71, 255},   // tomato
                color.RGBA{255, 140, 0, 255},   // dark orange
                color.RGBA{218, 165, 32, 255},  // goldenrod
                color.RGBA{184, 134, 11, 255},  // dark goldenrod
                color.RGBA{238, 130, 238, 255}, // violet
                color.RGBA{221, 160, 221, 255}, // plum
                color.RGBA{186, 85, 211, 255},  // medium orchid
                color.RGBA{148, 0, 211, 255},   // dark violet
                color.RGBA{75, 0, 130, 255},    // indigo
                color.RGBA{25, 25, 112, 255},   // midnight blue
                color.RGBA{240, 248, 255, 255}, // alice blue
                color.RGBA{176, 224, 230, 255}, // powder blue
                color.RGBA{135, 206, 235, 255}, // sky blue
                color.RGBA{135, 206, 250, 255}, // light sky blue
                color.RGBA{173, 216, 230, 255}, // light blue
                color.RGBA{224, 255, 255, 255}, // light cyan
                color.RGBA{240, 255, 240, 255}, // honeydew
                color.RGBA{255, 250, 205, 255}, // lemon chiffon
                color.RGBA{250, 250, 210, 255}, // light goldenrod yellow
                color.RGBA{211, 211, 211, 255}, // light gray
                color.RGBA{245, 222, 179, 255}, // wheat
                color.RGBA{160, 82, 45, 255},   // sienna
                color.RGBA{101, 67, 33, 255},   // dark brown
                color.RGBA{105, 105, 105, 255}, // dim gray
                color.RGBA{169, 169, 169, 255}, // dark gray
                color.RGBA{255, 228, 181, 255}, // moccasin
                color.RGBA{255, 239, 213, 255}, // papaya whip
                color.RGBA{255, 218, 185, 255}, // peach puff
                color.RGBA{255, 222, 173, 255}, // navajo white
                color.RGBA{189, 183, 107, 255}, // dark khaki
                color.RGBA{128, 128, 0, 255},   // olive
                color.RGBA{85, 107, 47, 255},   // dark olive green
                color.RGBA{143, 188, 143, 255}, // dark sea green
                color.RGBA{46, 139, 87, 255},   // sea green
                color.RGBA{102, 205, 170, 255}, // medium aquamarine
                color.RGBA{127, 255, 212, 255}, // aquamarine
                color.RGBA{176, 196, 222, 255}, // steel blue
                color.RGBA{25, 25, 25, 255},    // very dark gray
                color.RGBA{50, 50, 50, 255},
                color.RGBA{75, 75, 75, 255},
                color.RGBA{100, 100, 100, 255},
                color.RGBA{125, 125, 125, 255},
                color.RGBA{150, 150, 150, 255},
                color.RGBA{175, 175, 175, 255},
                color.RGBA{200, 200, 200, 255},
                color.RGBA{225, 225, 225, 255},
                color.RGBA{10, 10, 10, 255},
                color.RGBA{20, 20, 20, 255},
                color.RGBA{30, 30, 30, 255},
                color.RGBA{40, 40, 40, 255},
                color.RGBA{15, 15, 30, 255},
                color.RGBA{15, 30, 15, 255},
                color.RGBA{30, 15, 15, 255},
                color.RGBA{15, 15, 45, 255},
                color.RGBA{45, 15, 15, 255},
                color.RGBA{15, 45, 15, 255},
                color.RGBA{15, 30, 45, 255},
                color.RGBA{45, 15, 30, 255},
                color.RGBA{30, 45, 15, 255},
                color.RGBA{50, 50, 0, 255},
                color.RGBA{0, 50, 50, 255},
                color.RGBA{50, 0, 50, 255},
                color.RGBA{50, 25, 0, 255},
                color.RGBA{0, 50, 25, 255},
                color.RGBA{25, 0, 50, 255},
                color.RGBA{50, 50, 25, 255},
                color.RGBA{25, 50, 50, 255},
                color.RGBA{50, 25, 50, 255},
                color.RGBA{90, 90, 90, 255},
                color.RGBA{110, 110, 110, 255},
                color.RGBA{130, 130, 130, 255},
                color.RGBA{170, 170, 170, 255},
                color.RGBA{190, 190, 190, 255},
                color.RGBA{210, 210, 210, 255},
                color.RGBA{230, 230, 230, 255},
                color.RGBA{240, 240, 240, 255},
                color.RGBA{250, 250, 250, 255},
                color.RGBA{5, 5, 5, 255},
                color.RGBA{255, 250, 240, 255},
                color.RGBA{248, 248, 255, 255},
                color.RGBA{240, 255, 255, 255},
                color.RGBA{255, 240, 245, 255},
                color.RGBA{255, 240, 255, 255},
                color.RGBA{240, 240, 255, 255},
                color.RGBA{255, 255, 240, 255},
                color.RGBA{255, 255, 250, 255},
                color.RGBA{250, 240, 230, 255},
                color.RGBA{250, 235, 215, 255},
                color.RGBA{245, 245, 220, 255},
                color.RGBA{255, 228, 196, 255},
                color.RGBA{255, 248, 220, 255},
                color.RGBA{255, 250, 205, 255},
                color.RGBA{250, 250, 210, 255},
                color.RGBA{255, 255, 224, 255},
                color.RGBA{238, 232, 170, 255},
                color.RGBA{189, 183, 107, 255},
                color.RGBA{128, 128, 0, 255},
                color.RGBA{85, 107, 47, 255},
                color.RGBA{107, 142, 35, 255},
                color.RGBA{34, 139, 34, 255},
                color.RGBA{60, 179, 113, 255},
                color.RGBA{46, 139, 87, 255},
                color.RGBA{50, 205, 50, 255},
                color.RGBA{152, 251, 152, 255},
                color.RGBA{143, 188, 143, 255},
                color.RGBA{0, 100, 0, 255},
                color.RGBA{0, 128, 0, 255},
                color.RGBA{34, 139, 34, 255},
                color.RGBA{0, 250, 154, 255},
                color.RGBA{72, 209, 204, 255},
                color.RGBA{0, 206, 209, 255},
                color.RGBA{64, 224, 208, 255},
                color.RGBA{0, 255, 255, 255},
                color.RGBA{224, 255, 255, 255},
                color.RGBA{95, 158, 160, 255},
                color.RGBA{70, 130, 180, 255},
                color.RGBA{100, 149, 237, 255},
                color.RGBA{0, 191, 255, 255},
                color.RGBA{30, 144, 255, 255},
                color.RGBA{25, 25, 112, 255},
                color.RGBA{0, 0, 139, 255},
                color.RGBA{0, 0, 205, 255},
                color.RGBA{65, 105, 225, 255},
                color.RGBA{72, 61, 139, 255},
                color.RGBA{106, 90, 205, 255},
                color.RGBA{123, 104, 238, 255},
                color.RGBA{147, 112, 219, 255},
                color.RGBA{139, 0, 139, 255},
                color.RGBA{148, 0, 211, 255},
                color.RGBA{153, 50, 204, 255},
                color.RGBA{186, 85, 211, 255},
                color.RGBA{128, 0, 128, 255},
                color.RGBA{75, 0, 130, 255},
                color.RGBA{221, 160, 221, 255},
                color.RGBA{238, 130, 238, 255},
                color.RGBA{218, 112, 214, 255},
                color.RGBA{199, 21, 133, 255},
                color.RGBA{219, 112, 147, 255},
                color.RGBA{255, 20, 147, 255},
                color.RGBA{255, 105, 180, 255},
                color.RGBA{255, 182, 193, 255},
                color.RGBA{188, 143, 143, 255},
                color.RGBA{205, 92, 92, 255},
                color.RGBA{233, 150, 122, 255},
                color.RGBA{240, 128, 128, 255},
                color.RGBA{255, 99, 71, 255},
                color.RGBA{255, 69, 0, 255},
                color.RGBA{255, 140, 0, 255},
                color.RGBA{255, 160, 122, 255},
                color.RGBA{255, 165, 0, 255},
                color.RGBA{255, 215, 0, 255},
                color.RGBA{238, 221, 130, 255},
                color.RGBA{218, 165, 32, 255},
                color.RGBA{184, 134, 11, 255},
                color.RGBA{160, 82, 45, 255},
                color.RGBA{139, 69, 19, 255},
                color.RGBA{101, 67, 33, 255},
                color.RGBA{210, 105, 30, 255},
                color.RGBA{222, 184, 135, 255},
                color.RGBA{245, 222, 179, 255},
                color.RGBA{255, 228, 181, 255},
                color.RGBA{255, 235, 205, 255},
                color.RGBA{255, 239, 213, 255},
                color.RGBA{255, 245, 238, 255},
                color.RGBA{245, 245, 245, 255},
                color.RGBA{255, 228, 225, 255},
                color.RGBA{255, 240, 245, 255},
                color.RGBA{255, 218, 185, 255},
                color.RGBA{255, 222, 173, 255},
                color.RGBA{255, 250, 240, 255},
                color.RGBA{253, 245, 230, 255},
                color.RGBA{250, 240, 230, 255},
                color.RGBA{250, 235, 215, 255},
                color.RGBA{255, 239, 213, 255},
                color.RGBA{255, 235, 205, 255},
                color.RGBA{255, 228, 196, 255},
                color.RGBA{255, 248, 220, 255},
                color.RGBA{255, 255, 224, 255},
                color.RGBA{255, 250, 205, 255},
                color.RGBA{250, 250, 210, 255},
                color.RGBA{245, 245, 220, 255},
                color.RGBA{240, 230, 140, 255},
                color.RGBA{238, 232, 170, 255},
                color.RGBA{240, 230, 140, 255},
                color.RGBA{189, 183, 107, 255},
                color.RGBA{128, 128, 0, 255},
                color.RGBA{85, 107, 47, 255},
                color.RGBA{107, 142, 35, 255},
                color.RGBA{34, 139, 34, 255},
                color.RGBA{60, 179, 113, 255},
                color.RGBA{46, 139, 87, 255},
                color.RGBA{50, 205, 50, 255},
                color.RGBA{152, 251, 152, 255},
                color.RGBA{143, 188, 143, 255},
                color.RGBA{0, 100, 0, 255},
                color.RGBA{0, 128, 0, 255},
                color.RGBA{0, 250, 154, 255},
                color.RGBA{72, 209, 204, 255},
                color.RGBA{0, 206, 209, 255},
                color.RGBA{64, 224, 208, 255},
                color.RGBA{0, 255, 255, 255},
                color.RGBA{224, 255, 255, 255},
                color.RGBA{127, 255, 212, 255},
                color.RGBA{176, 224, 230, 255},
                color.RGBA{95, 158, 160, 255},
                color.RGBA{70, 130, 180, 255},
                color.RGBA{100, 149, 237, 255},
                color.RGBA{0, 191, 255, 255},
                color.RGBA{30, 144, 255, 255},
                color.RGBA{173, 216, 230, 255},
                color.RGBA{135, 206, 250, 255},
                color.RGBA{135, 206, 235, 255},
                color.RGBA{176, 196, 222, 255},
        }
}

// createSolidBackground creates a simple solid color background.
func createSolidBackground(w, h int, c color.RGBA) image.Image {
        dc := gg.NewContext(w, h)
        dc.SetColor(c)
        dc.DrawRectangle(0, 0, float64(w), float64(h))
        dc.Fill()
        return dc.Image()
}
