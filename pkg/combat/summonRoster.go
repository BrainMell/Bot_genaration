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
                frames   []image.Image
                delays   []int
                sprite   string
                cropRect image.Rectangle // 💡 FIXED crop region (union of all frames' visible bounds)
        }
        var gifs []gifData
        maxFrames := 1

        for _, s := range req.Summons {
                spritePath := getSummonIdleGifPath(s.Species, assetsPath)
                if spritePath == "" || !fileExists(spritePath) {
                        continue
                }

                // 💡 FIX 2026-08-04: Handle BOTH .gif and .png sprite files.
                // Ship sprites (Torrent summons) are static PNGs, not animated GIFs.
                // gif.DecodeAll fails on PNG — so check the extension.
                var fullFrames []image.Image
                var frameDelays []int

                if strings.HasSuffix(spritePath, ".gif") {
                        // Animated GIF
                        f, err := os.Open(spritePath)
                        if err != nil {
                                continue
                        }
                        g, err := gif.DecodeAll(f)
                        f.Close()
                        if err != nil || len(g.Image) == 0 {
                                continue
                        }

                        gifW := g.Config.Width
                        gifH := g.Config.Height
                        if gifW <= 0 || gifH <= 0 {
                                b := g.Image[0].Bounds()
                                gifW = b.Dx()
                                gifH = b.Dy()
                        }

                        // Composite each sub-frame onto a full-size canvas
                        // 💡 FIX: Go's gif decoder returns sub-frames with their own
                        // Rect bounds that may be offset from 0,0. We need to draw
                        // them at their correct position within the full canvas.
                        fullFrames = make([]image.Image, len(g.Image))
                        for fi, subFrame := range g.Image {
                                fullFrame := image.NewRGBA(image.Rect(0, 0, gifW, gifH))
                                // Draw the sub-frame at its own offset within the canvas
                                draw.Draw(fullFrame, subFrame.Bounds(), subFrame, subFrame.Bounds().Min, draw.Over)
                                fullFrames[fi] = fullFrame
                        }
                        frameDelays = g.Delay
                } else {
                        // Static PNG — load as single frame
                        img, err := utils.LoadImage(spritePath)
                        if err != nil {
                                continue
                        }
                        fullFrames = []image.Image{img}
                        frameDelays = []int{10}
                }

                // 💡 FIX 2026-08-04: Compute FIXED crop rect from the UNION of all
                // frames' visible bounds. This preserves the bobbing animation —
                // if we cropped each frame independently, the sprite would be
                // re-positioned to the top-left of the crop, destroying the motion.
                cropRect := computeUnionVisibleBounds(fullFrames)

                gd := gifData{
                        frames:   fullFrames,
                        delays:   frameDelays,
                        sprite:   s.Species,
                        cropRect: cropRect,
                }
                gifs = append(gifs, gd)
                if len(fullFrames) > maxFrames {
                        maxFrames = len(fullFrames)
                }
        }

        // ── 3. Calculate summon positions ──
        // 💡 CODEX LAYOUT: 5 per page in a 3-front / 2-back arrangement.
        // Front row: 3 sprites (positions 0, 1, 2) — fills first, LARGER
        // Back row: 2 sprites (positions 3, 4) — overflow, SMALLER (perspective)
        //
        // 💡 FIX 2026-08-05 (v3): Back row at original Y position (not pushed up).
        // Natural overlap is OK — back row is drawn FIRST, so front row occludes
        // it naturally (back row feet behind front row heads = realistic depth).
        // Back row is ~75% the size of front row (subtle perspective, not extreme).
        const frontSpriteH = 260
        const frontMaxSpriteW = 220
        const backSpriteH = 190
        const backMaxSpriteW = 170
        const slotW = 240
        const backRowY = 320   // back row ground line (original position, not pushed up)
        const frontRowY = 475  // front row ground line (lower = closer)
        const shadowRadiusFixed = 60.0
        _ = len(gifs)
        // Calculate total width for 3 columns (front row width)
        totalWidth := 3 * slotW
        startX := (W - totalWidth) / 2

        // ── 4. Render each frame ──
        var outFrames []*image.Paletted
        var outDelays []int

        for frameIdx := 0; frameIdx < maxFrames; frameIdx++ {
                dc := gg.NewContext(W, H)

                // Draw background
                dc.DrawImage(bgImg, 0, 0)
                dc.SetColor(color.RGBA{0, 0, 0, 140})
                dc.DrawRectangle(0, 0, W, H)
                dc.Fill()

                // Header
                dc.SetColor(color.RGBA{0, 0, 0, 160})
                dc.DrawRoundedRectangle(15, 10, float64(W-30), 50, 8)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 215, 0, 100})
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(15, 10, float64(W-30), 50, 8)
                dc.Stroke()

                dc.SetColor(color.RGBA{255, 215, 0, 255})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 22); err != nil {}
                dc.DrawStringAnchored("🐉 SUMMON ROSTER", 30, 35, 0, 0.5)

                slotStr := fmt.Sprintf("%d/%d slots", req.SlotsUsed, req.SlotsMax)
                dc.SetColor(color.RGBA{255, 255, 255, 200})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 15); err != nil {}
                dc.DrawStringAnchored(slotStr, float64(W-30), 35, 1, 0.5)

                // ── Draw each summon ──
                // 💡 FIX 2026-08-05: Draw BACK ROW FIRST (z-order), then FRONT ROW.
                // This way front row sprites occlude back row sprites naturally,
                // creating depth instead of the back row overlaying the front.
                // Back row uses smaller sprite size (perspective: further = smaller).
                drawOrder := []int{}
                for i := range gifs {
                        if i >= 3 { drawOrder = append(drawOrder, i) } // back row first
                }
                for i := range gifs {
                        if i < 3 { drawOrder = append(drawOrder, i) } // front row second
                }

                for _, i := range drawOrder {
                        gd := gifs[i]
                        gifFrameIdx := frameIdx % len(gd.frames)
                        spriteImg := gd.frames[gifFrameIdx]

                        // Determine row-specific sprite size
                        isBackRow := i >= 3
                        rowSpriteH := frontSpriteH
                        rowMaxSpriteW := frontMaxSpriteW
                        if isBackRow {
                                rowSpriteH = backSpriteH
                                rowMaxSpriteW = backMaxSpriteW
                        }

                        // Apply the FIXED crop rect (computed once from union of all frames)
                        croppedImg := applyCrop(spriteImg, gd.cropRect)

                        // Manual contain-fit (scale UP or DOWN)
                        cb := croppedImg.Bounds()
                        cW := cb.Dx()
                        cH := cb.Dy()
                        sW := float64(rowMaxSpriteW) / float64(cW)
                        sH := float64(rowSpriteH) / float64(cH)
                        fitScale := sW
                        if sH < fitScale { fitScale = sH }
                        dstW := int(float64(cW) * fitScale)
                        dstH := int(float64(cH) * fitScale)
                        if dstW < 1 { dstW = 1 }
                        if dstH < 1 { dstH = 1 }
                        resized := imaging.Resize(croppedImg, dstW, dstH, imaging.NearestNeighbor)

                        bottomPadding := 0

                        // Position: front row fills first (3 slots), back row between
                        var slotCenterX, groundY int
                        if !isBackRow {
                                slotCenterX = startX + i*slotW + slotW/2
                                groundY = frontRowY
                        } else {
                                backCol := i - 3
                                slotCenterX = startX + (backCol+1)*slotW
                                groundY = backRowY
                        }

                        spriteDrawX := slotCenterX - dstW/2
                        spriteDrawY := groundY - dstH + bottomPadding

                        // ── Shadow (FIXED size — does not scale with sprite) ──
                        shadowCenterX := float64(slotCenterX)
                        shadowCenterY := float64(groundY) - 2
                        shadowRadiusX := shadowRadiusFixed
                        shadowRadiusY := shadowRadiusX * 0.28

                        shadowCtx := gg.NewContext(int(shadowRadiusX*2)+6, int(shadowRadiusY*2)+6)
                        shadowCtx.SetColor(color.RGBA{0, 0, 0, 120})
                        shadowCtx.DrawEllipse(float64(shadowCtx.Width())/2, float64(shadowCtx.Height())/2, shadowRadiusX, shadowRadiusY)
                        shadowCtx.Fill()
                        blurredShadow := imaging.Blur(shadowCtx.Image(), 3.0)
                        dc.DrawImageAnchored(blurredShadow, int(shadowCenterX), int(shadowCenterY), 0.5, 0.5)

                        // Draw the sprite
                        dc.DrawImage(resized, spriteDrawX, spriteDrawY)

                        // ── Name + rarity badge + rarity-colored name ──
                        summon := req.Summons[i]
                        name := summon.Nickname
                        if name == "" { name = summon.Species }
                        if len(name) > 14 { name = name[:12] + "…" }

                        rarityColor := getRarityColor(summon.Rarity)

                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 14); err != nil {}

                        nameW, _ := dc.MeasureString(name)
                        badgeR := 5.0
                        badgeGap := 4.0
                        groupW := badgeR*2 + badgeGap + nameW
                        groupStartX := float64(slotCenterX) - groupW/2
                        nameY := float64(groundY + 14)

                        // Draw rarity badge
                        badgeX := groupStartX + badgeR
                        dc.SetColor(rarityColor)
                        dc.DrawCircle(badgeX, nameY, badgeR)
                        dc.Fill()
                        dc.SetColor(color.RGBA{255, 255, 255, 200})
                        dc.DrawCircle(badgeX, nameY, badgeR*0.4)
                        dc.Fill()

                        // Draw name in rarity color
                        dc.SetColor(rarityColor)
                        dc.DrawStringAnchored(name, groupStartX+badgeR*2+badgeGap+nameW/2, nameY, 0.5, 0.5)

                        // Rarity text (small, white) below name
                        dc.SetColor(color.RGBA{255, 255, 255, 180})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 11); err != nil {}
                        infoStr := fmt.Sprintf("%s", summon.Rarity)
                        dc.DrawStringAnchored(infoStr, float64(slotCenterX), float64(groundY+30), 0.5, 0.5)

                        if summon.IsDeployed {
                                dc.SetColor(color.RGBA{255, 215, 0, 255})
                                deployedY := groundY - rowSpriteH + 5
                                dc.DrawStringAnchored("⭐", float64(slotCenterX), float64(deployedY), 0.5, 0.5)
                        }
                }

                // ── Info Hub (codex-specific) ──
                hubY := 520
                hubH := 190
                dc.SetColor(color.RGBA{0, 0, 0, 180})
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 215, 0, 80})
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Stroke()

                // Codex hub title
                dc.SetColor(color.RGBA{255, 215, 0, 255})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 16); err != nil {}
                totalSpecies := len(req.Summons)
                hubTitle := fmt.Sprintf("📖 SUMMON CODEX — %d species shown", totalSpecies)
                if req.UserNickname == "CODEX" {
                    // Codex mode — show page info
                    hubTitle = fmt.Sprintf("📖 SUMMON CODEX — Browsing %d species", totalSpecies)
                }
                dc.DrawStringAnchored(hubTitle, 30, float64(hubY+15), 0, 0.5)

                // Show species list in the hub
                dc.SetColor(color.RGBA{255, 255, 255, 200})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 13); err != nil {}
                for si, s := range req.Summons {
                        row := si / 3
                        col := si % 3
                        sx := 30 + col*220
                        sy := hubY + 45 + row*25
                        speciesName := s.Nickname
                        if speciesName == "" { speciesName = s.Species }
                        line := fmt.Sprintf("%d. %s (%s)", si+1, speciesName, s.Rarity)
                        dc.DrawStringAnchored(line, float64(sx), float64(sy), 0, 0.5)
                }

                // Commands hint
                dc.SetColor(color.RGBA{255, 255, 255, 120})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 11); err != nil {}
                dc.DrawStringAnchored(".summon codex <page> — navigate pages", float64(W/2), float64(hubY+hubH-15), 0.5, 0.5)

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

        // Map our species IDs to sparklinlabs monster names.
        // 💡 2026-08-04: 1:1 mapping — each summon has its OWN dedicated sprite.
        // No sharing. The species ID IS the sprite filename.
        speciesMap := map[string]string{
                // Animated idle.gif sprites (20)
                "bat":         "bat",
                "boar":        "boar",
                "chest":       "chest",
                "dino":        "dino",
                "dragon":      "dragon",
                "ghost":       "ghost",
                "giant":       "giant",
                "mimic":       "mimic",
                "mushroom":    "mushroom",
                "octopus":     "octopus",
                "reptile":     "reptile",
                "slime":       "slime",
                "snake":       "snake",
                "yeti":        "yeti",
                // Batch 6 new summons (6)
                "plaguefang":  "plaguefang",
                "lumenmoth":   "lumenmoth",
                "emberwick":   "emberwick",
                "skitterswarm": "skitterswarm",
                "tidalmaw":    "tidalmaw",
                "fireguard":   "fireguard",
                // Static PNG sprites (3) — ship sprites for Grand Inventor
                "ship_cruiser":   "ship_cruiser",
                "ship_fighter":   "ship_fighter",
                "ship_squid":     "ship_squid",
        }

        monsterName, ok := speciesMap[species]
        if !ok {
                // Try direct match — species ID might be the sprite name directly
                monsterName = species
        }

        // Try idle.gif first (animated sprites)
        gifPath := filepath.Join(assetsPath, "rpgasset", "summons", "sparklinlabs", monsterName+"_idle.gif")
        if fileExists(gifPath) {
                return gifPath
        }
        // Try static PNG (ship sprites)
        pngPath := filepath.Join(assetsPath, "rpgasset", "summons", "sparklinlabs", monsterName+".png")
        if fileExists(pngPath) {
                return pngPath
        }
        return ""
}

// computeUnionVisibleBounds returns the UNION of all frames' visible (non-transparent)
// bounding boxes. This is used to compute a FIXED crop region that works for every
// frame in an animated GIF — so the bobbing animation is preserved.
//
// 💡 FIX 2026-08-04: If we cropped each frame independently, the sprite would be
// re-positioned to the top-left of the crop on every frame, destroying the motion.
// By computing the union once and applying the same crop to all frames, the sprite
// moves within the crop region (preserving the bob) while transparent padding is removed.
func computeUnionVisibleBounds(frames []image.Image) image.Rectangle {
        if len(frames) == 0 {
                return image.Rect(0, 0, 0, 0)
        }
        minX := int(^uint(0) >> 1) // max int
        minY := int(^uint(0) >> 1)
        maxX := -1
        maxY := -1

        for _, img := range frames {
                bounds := img.Bounds()
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
        }

        if maxX < 0 {
                // No visible pixels in any frame — return full bounds of first frame
                return frames[0].Bounds()
        }
        return image.Rect(minX, minY, maxX+1, maxY+1)
}

// applyCrop crops an image to the given rectangle. If the rect is empty or
// covers the full image, the original image is returned unchanged.
func applyCrop(img image.Image, crop image.Rectangle) image.Image {
        bounds := img.Bounds()
        if crop.Empty() || crop == bounds {
                return img
        }
        // Clamp crop to image bounds
        if crop.Min.X < bounds.Min.X { crop.Min.X = bounds.Min.X }
        if crop.Min.Y < bounds.Min.Y { crop.Min.Y = bounds.Min.Y }
        if crop.Max.X > bounds.Max.X { crop.Max.X = bounds.Max.X }
        if crop.Max.Y > bounds.Max.Y { crop.Max.Y = bounds.Max.Y }

        // Use SubImage if available (efficient, no copy)
        type subImager interface {
                SubImage(r image.Rectangle) image.Image
        }
        if si, ok := img.(subImager); ok {
                return si.SubImage(crop)
        }

        // Fallback: draw the cropped region onto a new RGBA canvas
        cropW := crop.Dx()
        cropH := crop.Dy()
        dst := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
        draw.Draw(dst, dst.Bounds(), img, crop.Min, draw.Src)
        return dst
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

// ════════════════════════════════════════════════════════════════
// 💡 SINGLE SUMMON DETAIL GIF — .summon <#>
// Renders one summon's idle.gif large + detailed info hub
// ════════════════════════════════════════════════════════════════

func GenerateSummonDetailGIF(c *gin.Context) {
        var req SummonRosterRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        if len(req.Summons) == 0 {
                c.JSON(400, gin.H{"error": "no summon provided"})
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

        // ── 2. Load the summon's idle.gif ──
        summon := req.Summons[0]
        gifPath := getSummonIdleGifPath(summon.Species, assetsPath)

        type gifData struct {
                frames   []image.Image
                delays   []int
                cropRect image.Rectangle
        }
        var gd gifData
        maxFrames := 1

        if gifPath != "" && fileExists(gifPath) {
                if strings.HasSuffix(gifPath, ".gif") {
                        f, err := os.Open(gifPath)
                        if err == nil {
                                g, err := gif.DecodeAll(f)
                                f.Close()
                                if err == nil && len(g.Image) > 0 {
                                        gifW := g.Config.Width
                                        gifH := g.Config.Height
                                        if gifW <= 0 || gifH <= 0 {
                                                b := g.Image[0].Bounds()
                                                gifW = b.Dx()
                                                gifH = b.Dy()
                                        }
                                        fullFrames := make([]image.Image, len(g.Image))
                                        for fi, subFrame := range g.Image {
                                                fullFrame := image.NewRGBA(image.Rect(0, 0, gifW, gifH))
                                                bounds := subFrame.Bounds()
                                                draw.Draw(fullFrame, bounds, subFrame, bounds.Min, draw.Over)
                                                fullFrames[fi] = fullFrame
                                        }
                                        // 💡 Compute fixed crop rect (union of all frames)
                                        cropRect := computeUnionVisibleBounds(fullFrames)
                                        gd = gifData{frames: fullFrames, delays: g.Delay, cropRect: cropRect}
                                        maxFrames = len(g.Image)
                                }
                        }
                } else {
                        // Static PNG
                        img, err := utils.LoadImage(gifPath)
                        if err == nil {
                                cropRect := computeUnionVisibleBounds([]image.Image{img})
                                gd = gifData{frames: []image.Image{img}, delays: []int{10}, cropRect: cropRect}
                                maxFrames = 1
                        }
                }
        }

        // ── 3. Render each frame ──
        var outFrames []*image.Paletted
        var outDelays []int

        for frameIdx := 0; frameIdx < maxFrames; frameIdx++ {
                dc := gg.NewContext(W, H)

                // Background
                dc.DrawImage(bgImg, 0, 0)
                dc.SetColor(color.RGBA{0, 0, 0, 130})
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

                name := summon.Nickname
                if name == "" { name = summon.Species }
                rarityColor := getRarityColor(summon.Rarity)

                // 💡 FIX 2026-08-04: Header name colored by rarity + rarity badge.
                // Draw a small colored circle (badge) before the name, then the
                // name in the rarity color. "🐉" emoji stays gold for consistency.
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 22); err != nil {}
                emojiStr := "🐉 "
                emojiW, _ := dc.MeasureString(emojiStr)
                nameW, _ := dc.MeasureString(name)
                badgeR := 7.0
                badgeGap := 6.0
                _ = emojiW + badgeR*2 + badgeGap + nameW // headerGroupW (unused, kept for clarity)
                headerStartX := 30.0

                // Draw dragon emoji in gold
                dc.SetColor(color.RGBA{255, 215, 0, 255})
                dc.DrawStringAnchored(emojiStr, headerStartX+emojiW/2, 35, 0.5, 0.5)

                // Draw rarity badge
                badgeX := headerStartX + emojiW + badgeR
                dc.SetColor(rarityColor)
                dc.DrawCircle(badgeX, 35, badgeR)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 255, 255, 220})
                dc.DrawCircle(badgeX, 35, badgeR*0.4)
                dc.Fill()

                // Draw name in rarity color
                dc.SetColor(rarityColor)
                dc.DrawStringAnchored(name, badgeX+badgeR+badgeGap+nameW/2, 35, 0.5, 0.5)

                rarityText := fmt.Sprintf("%s | Lv.%d", summon.Rarity, summon.Level)
                dc.SetColor(color.RGBA{255, 255, 255, 200})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 15); err != nil {}
                dc.DrawStringAnchored(rarityText, float64(W-30), 35, 1, 0.5)

                // ── Draw the summon sprite (large, centered) ──
                // 💡 FIX 2026-08-04: Use imaging.Fit for consistent sizing with codex.
                // Shadow radius is FIXED (not scaled with sprite).
                const spriteH = 360
                const maxSpriteW = 400
                const groundY = 410
                const shadowRadiusFixed = 80.0

                if len(gd.frames) > 0 {
                        gifFrameIdx := frameIdx % len(gd.frames)
                        spriteImg := gd.frames[gifFrameIdx]

                        // 💡 FIX 2026-08-04: Apply FIXED crop rect (same as codex).
                        // Preserves bobbing animation — crop region is constant across frames.
                        croppedImg := applyCrop(spriteImg, gd.cropRect)

                        // 💡 FIX 2026-08-04: Manual contain-fit (scale UP or DOWN).
                        // imaging.Fit does NOT upscale — if the source is smaller than
                        // the target box, it returns the original. Many sprites (chest,
                        // ships) are smaller than the target, so they'd stay tiny.
                        // We calculate the scale factor and use imaging.Resize instead.
                        cb := croppedImg.Bounds()
                        cW := cb.Dx()
                        cH := cb.Dy()
                        sW := float64(maxSpriteW) / float64(cW)
                        sH := float64(spriteH) / float64(cH)
                        fitScale := sW
                        if sH < fitScale { fitScale = sH }
                        dstW := int(float64(cW) * fitScale)
                        dstH := int(float64(cH) * fitScale)
                        if dstW < 1 { dstW = 1 }
                        if dstH < 1 { dstH = 1 }
                        resized := imaging.Resize(croppedImg, dstW, dstH, imaging.NearestNeighbor)

                        // 💡 FIX 2026-08-04: No per-frame visibleBottom adjustment.
                        // Crop already handles padding. Bobbing is preserved because
                        // the sprite position is FIXED (the sprite bobs within the box).
                        bottomPadding := 0

                        spriteDrawX := W/2 - dstW/2
                        spriteDrawY := groundY - dstH + bottomPadding

                        // Shadow (FIXED size)
                        shadowRadiusX := shadowRadiusFixed
                        shadowRadiusY := shadowRadiusX * 0.28
                        shadowCtx := gg.NewContext(int(shadowRadiusX*2)+6, int(shadowRadiusY*2)+6)
                        shadowCtx.SetColor(color.RGBA{0, 0, 0, 120})
                        shadowCtx.DrawEllipse(float64(shadowCtx.Width())/2, float64(shadowCtx.Height())/2, shadowRadiusX, shadowRadiusY)
                        shadowCtx.Fill()
                        blurredShadow := imaging.Blur(shadowCtx.Image(), 3.0)
                        dc.DrawImageAnchored(blurredShadow, W/2, int(float64(groundY)-2), 0.5, 0.5)

                        // Draw sprite
                        dc.DrawImage(resized, spriteDrawX, spriteDrawY)
                }

                // ── Detail Info Hub (bottom, much bigger) ──
                hubY := 430
                hubH := 280
                dc.SetColor(color.RGBA{0, 0, 0, 180})
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Fill()
                dc.SetColor(rarityColor)
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(15, float64(hubY), float64(W-30), float64(hubH), 10)
                dc.Stroke()

                // Summon name + deployed indicator + rarity badge
                // 💡 FIX 2026-08-04: Name colored by rarity, with badge beside it.
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 20); err != nil {}
                starStr := ""
                if summon.IsDeployed { starStr = "⭐ " }
                starW, _ := dc.MeasureString(starStr)
                nameWHub, _ := dc.MeasureString(name)
                badgeRHub := 7.0
                badgeGapHub := 6.0
                hubStartX := 30.0

                // Draw star (gold) if deployed
                if summon.IsDeployed {
                        dc.SetColor(color.RGBA{255, 215, 0, 255})
                        dc.DrawStringAnchored(starStr, hubStartX+starW/2, float64(hubY+25), 0.5, 0.5)
                }

                // Draw rarity badge
                badgeXHub := hubStartX + starW + badgeRHub
                dc.SetColor(rarityColor)
                dc.DrawCircle(badgeXHub, float64(hubY+25), badgeRHub)
                dc.Fill()
                dc.SetColor(color.RGBA{255, 255, 255, 220})
                dc.DrawCircle(badgeXHub, float64(hubY+25), badgeRHub*0.4)
                dc.Fill()

                // Draw name in rarity color
                dc.SetColor(rarityColor)
                dc.DrawStringAnchored(name, badgeXHub+badgeRHub+badgeGapHub+nameWHub/2, float64(hubY+25), 0.5, 0.5)

                // Subtitle: element, archetype, rarity
                dc.SetColor(color.RGBA{255, 255, 255, 200})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 14); err != nil {}
                subtitle := fmt.Sprintf("%s | %s | %s | Lv.%d", summon.Element, summon.Archetype, summon.Rarity, summon.Level)
                dc.DrawStringAnchored(subtitle, 30, float64(hubY+50), 0, 0.5)

                // Stats grid (2 rows, 3 columns)
                statY1 := hubY + 80
                statY2 := hubY + 120
                colW := (W - 60) / 3
                stats := []struct {
                        label string
                        value int
                        color color.RGBA
                }{
                        {"HP", summon.HP, color.RGBA{255, 107, 107, 255}},
                        {"ATK", summon.ATK, color.RGBA{255, 217, 61, 255}},
                        {"DEF", summon.DEF, color.RGBA{79, 195, 247, 255}},
                        {"MAG", summon.MAG, color.RGBA{156, 39, 176, 255}},
                        {"SPD", summon.SPD, color.RGBA{76, 175, 80, 255}},
                        {"LOY", summon.Loyalty, color.RGBA{255, 193, 7, 255}},
                }
                for si, stat := range stats {
                        col := si % 3
                        row := si / 3
                        sx := 30 + col*colW
                        sy := statY1
                        if row == 1 { sy = statY2 }

                        dc.SetColor(stat.color)
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "dogicapixelbold.otf"), 14); err != nil {}
                        dc.DrawStringAnchored(stat.label, float64(sx), float64(sy), 0, 0.5)

                        dc.SetColor(color.RGBA{255, 255, 255, 240})
                        if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 18); err != nil {}
                        valStr := fmt.Sprintf("%d", stat.value)
                        if stat.label == "LOY" { valStr = fmt.Sprintf("%d%%", stat.value) }
                        dc.DrawStringAnchored(valStr, float64(sx+45), float64(sy), 0, 0.5)
                }

                // Loyalty bar
                loyBarY := hubY + 155
                loyBarW := W - 60
                dc.SetColor(color.RGBA{0, 0, 0, 120})
                dc.DrawRectangle(30, float64(loyBarY), float64(loyBarW), 8)
                dc.Fill()
                loyPct := float64(summon.Loyalty) / 100.0
                loyColor := color.RGBA{76, 175, 80, 255}
                if summon.Loyalty < 50 { loyColor = color.RGBA{255, 152, 0, 255} }
                if summon.Loyalty < 25 { loyColor = color.RGBA{244, 67, 54, 255} }
                dc.SetColor(loyColor)
                dc.DrawRectangle(30, float64(loyBarY), float64(loyBarW)*loyPct, 8)
                dc.Fill()

                // Commands hint
                dc.SetColor(color.RGBA{255, 255, 255, 120})
                if err := dc.LoadFontFace(filepath.Join(assetsPath, "rpgasset", "fonts", "PixeloidSans.ttf"), 12); err != nil {}
                p := ".s"
                hintStr := fmt.Sprintf("%s summon deploy  |  %s summon skill  |  %s summon trial  |  %s summon forge", p, p, p, p)
                dc.DrawStringAnchored(hintStr, float64(W/2), float64(hubY+hubH-15), 0.5, 0.5)

                // Convert to paletted
                frameImg := dc.Image()
                paletted := imageToPaletted(frameImg, W, H)
                outFrames = append(outFrames, paletted)
                outDelays = append(outDelays, 10)
        }

        // ── Encode as GIF ──
        out := &gif.GIF{Image: outFrames, Delay: outDelays}
        c.Header("Content-Type", "image/gif")
        if err := gif.EncodeAll(c.Writer, out); err != nil {
                c.JSON(500, gin.H{"error": "GIF encoding failed: " + err.Error()})
        }
}
