package combat

// ============================================
// 🐉 BOSS SPLASH RENDERER — parchment redesign 2026-09-12
// ============================================
// Owner complaint: the boss introduction card was still the OLD dark-gradient
// full-screen design — completely off from the new lamoot wood+gold+parchment
// family (BREW / HUNT / FISH / ROYAL DECREE). Rebuilt in that family.
//
// Static art baked in assets/rpgasset/ui/craft/bg_BOSS.png (build_boss_bg.py):
//   - leather name plate (66..340, 56..112)         → TIER label drawn by us
//   - wood banner title "BOSS ENCOUNTER"            → baked
//   - scene window (130,185)-(870,400) dark inset   → we paint tier glow + boss
//   - boss name line y442 + flavor line y474        → drawn by us (parchment)
//   - wax tier seal at (70,546) r26                 → drawn by us
//   - caption strip y552 ("steel yourself…")        → baked
//
// Payload compat (both shapes accepted):
//   guildAdventure:   {name, spriteFilename, flavorText, tier}
//   engine test-lock: {sprite: "enemies/x.png", name, flavorText, rank, floor}
// Tier drives the arena glow accent + wax seal letter:
//   S/SS/SSS/TRIAL/DRAGON/RAID/ABYSS/GOD (+ "???" maintenance default).
// NOTE: no gg Clip() — same clip-state leak as hunt.go; the glow is painted in
// an offscreen window-sized context so every scene pixel stays bounded.

import (
        "image"
        "image/color"
        "image/draw"
        "path/filepath"
        "strings"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

// tier accent (arena glow) — carried over from the old design's per-tier identity
var bossTierAccent = map[string][3]float64{
        "S":      {1.0, 0.30, 0.30},
        "SS":     {0.70, 0.40, 1.0},
        "SSS":    {1.0, 0.85, 0.30},
        "TRIAL":  {0.40, 0.60, 1.0},
        "DRAGON": {1.0, 0.50, 0.20},
        "RAID":   {0.30, 0.90, 1.0},
        "ABYSS":  {0.30, 0.30, 0.80},
        "GOD":    {1.0, 0.90, 0.40},
        "???":    {0.55, 0.55, 0.65},
}

// tier label on the leather name plate
var bossTierLabel = map[string]string{
        "S":      "S-RANK BOSS",
        "SS":     "SS-RANK BOSS",
        "SSS":    "SSS-RANK BOSS",
        "TRIAL":  "TRIAL BOSS",
        "DRAGON": "DRAGON BOSS",
        "RAID":   "WEEKLY RAID BOSS",
        "ABYSS":  "ABYSS BOSS",
        "GOD":    "DIVINE BOSS",
        "???":    "???",
}

// wax seal letter (short — must fit the r26 seal like hunt rank letters)
var bossTierSeal = map[string]string{
        "S": "S", "SS": "SS", "SSS": "SSS",
        "TRIAL": "T", "DRAGON": "D", "RAID": "R",
        "ABYSS": "A", "GOD": "G", "???": "?",
}

// boss window geometry — MUST match bg_BOSS.png bake (build_boss_bg.py)
const (
        bossSceneX = 130.0
        bossSceneY = 185.0
        bossSceneW = 740.0
        bossSceneH = 215.0
)

// trimTransparent crops fully-transparent borders (alpha <= threshold).
// Boss sprites like boss_3_N.png are 256x512 canvases with the visible boss
// only in the bottom ~227x180 — without trimming, Fit() sees the whole canvas
// and renders the boss tiny (~90px). QA round 1 found exactly that.
func trimTransparent(img image.Image, threshold uint8) image.Image {
        b := img.Bounds()
        var rgba *image.RGBA
        if r, ok := img.(*image.RGBA); ok && r.Rect.Eq(b) {
                rgba = r
        } else {
                rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
                draw.Draw(rgba, rgba.Rect, img, b.Min, draw.Src)
        }
        minX, minY := b.Max.X, b.Max.Y
        maxX, maxY := b.Min.X, b.Min.Y
        for y := b.Min.Y; y < b.Max.Y; y++ {
                for x := b.Min.X; x < b.Max.X; x++ {
                        if rgba.RGBAAt(x, y).A > threshold {
                                if x < minX {
                                        minX = x
                                }
                                if y < minY {
                                        minY = y
                                }
                                if x > maxX {
                                        maxX = x
                                }
                                if y > maxY {
                                        maxY = y
                                }
                        }
                }
        }
        if maxX < minX || maxY < minY {
                return img // fully transparent — draw as-is
        }
        return imaging.Crop(img, image.Rect(minX, minY, maxX+1, maxY+1))
}

// GenerateBossSplash renders the parchment boss encounter splash (1000x600).
func GenerateBossSplash(c *gin.Context) {
        var req struct {
                Name           string `json:"name"`
                SpriteFilename string `json:"spriteFilename"`
                Sprite         string `json:"sprite"` // engine test-lock shape
                FlavorText     string `json:"flavorText"`
                Tier           string `json:"tier"`
                Rank           string `json:"rank"` // engine test-lock shape
                Floor          int    `json:"floor"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        assetsPath := "assets"
        dc := gg.NewContext(1000, 600)

        // ── baked parchment base ──
        if bgImg, err := utils.LoadImage(huntAsset("bg_BOSS.png")); err == nil {
                dc.DrawImage(bgImg, 0, 0)
        } else {
                dc.SetRGB(0.09, 0.06, 0.04)
                dc.DrawRectangle(0, 0, 1000, 600)
                dc.Fill()
        }

        // normalize tier (engine test-lock sends rank instead)
        tier := strings.ToUpper(strings.TrimSpace(req.Tier))
        if tier == "" {
                tier = strings.ToUpper(strings.TrimSpace(req.Rank))
        }
        accent, ok := bossTierAccent[tier]
        if !ok {
                accent = [3]float64{0.55, 0.55, 0.65}
        }
        tierLabel, ok := bossTierLabel[tier]
        if !ok {
                tierLabel = "BOSS"
        }
        sealLetter, ok := bossTierSeal[tier]
        if !ok {
                sealLetter = "B"
                if len(tier) > 0 {
                        sealLetter = strings.ToUpper(tier[:1])
                }
        }

        // ── tier label on the leather plate ──
        huntFitText(dc, huntAsset("Cinzel.ttf"), 30, tierLabel, 240, 10)
        dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
        dc.DrawStringAnchored(tierLabel, 90, 84, 0, 0.5)

        // ── scene window: offscreen context (pixel-bounded, no Clip) ──
        // dark arena base + tier-colored radial glow + edge vignette, then a
        // single DrawImage at the window rect.
        scene := gg.NewContext(int(bossSceneW), int(bossSceneH))
        scene.SetRGB(14.0/255.0, 16.0/255.0, 22.0/255.0)
        scene.DrawRectangle(0, 0, bossSceneW, bossSceneH)
        scene.Fill()
        gcx, gcy := bossSceneW/2, bossSceneH*0.52
        for i := 0; i < 30; i++ {
                f := float64(i) / 30.0
                scene.SetRGBA(accent[0], accent[1], accent[2], 0.055*(1.0-f))
                scene.DrawCircle(gcx, gcy, 36+f*300)
                scene.Fill()
        }
        // floor line so the boss isn't floating in void
        scene.SetRGBA(1, 1, 1, 0.05)
        scene.DrawRectangle(0, bossSceneH-26, bossSceneW, 26)
        scene.Fill()
        sceneBottom := bossSceneY + bossSceneH - 3
        dc.DrawImage(scene.Image(), int(bossSceneX), int(bossSceneY))

        // ── boss sprite — centered, bottom-anchored, Lanczos (HD art) ──
        spriteRef := req.SpriteFilename
        if spriteRef == "" {
                spriteRef = req.Sprite
        }
        spriteRef = filepath.Base(strings.TrimSpace(spriteRef))
        if spriteRef != "" && spriteRef != "." {
                spritePath := filepath.Join(assetsPath, "rpgasset", "enemies", spriteRef)
                if spriteImg, err := utils.LoadImage(spritePath); err == nil {
                        // trim transparent padding, then menace-upscale small bosses
                        // (old design rendered at 450w regardless — bosses must loom)
                        spriteImg = trimTransparent(spriteImg, 8)
                        bt := spriteImg.Bounds()
                        if bt.Dx() < 300 || bt.Dy() < 170 {
                                scale := 300.0 / float64(bt.Dx())
                                if s2 := 170.0 / float64(bt.Dy()); s2 < scale {
                                        scale = s2
                                }
                                if scale > 1 {
                                        spriteImg = imaging.Resize(spriteImg,
                                                int(float64(bt.Dx())*scale), int(float64(bt.Dy())*scale),
                                                imaging.Lanczos)
                                }
                        }
                        spriteImg = imaging.Fit(spriteImg, 340, 190, imaging.Lanczos)
                        b := spriteImg.Bounds()
                        sw, sh := float64(b.Dx()), float64(b.Dy())
                        sx := bossSceneX + (bossSceneW-sw)/2
                        utils.DrawShadow(dc, sx+sw/2, sceneBottom-2, sw*0.42, 0.5)
                        dc.DrawImage(spriteImg, int(sx), int(sceneBottom-sh))
                }
        }

        // ── boss name on the parchment (y442) ──
        name := strings.Join(strings.Fields(huntSanitize(req.Name)), " ")
        if name == "" {
                name = "UNKNOWN BOSS"
        }
        huntFitText(dc, huntAsset("CinzelDecBold.ttf"), 40, name, 680, 16)
        dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
        dc.DrawStringAnchored(name, 500, 442, 0.5, 0.5)

        // ── flavor line (y474, single line, shrink-to-fit) ──
        flavor := strings.Join(strings.Fields(huntSanitize(req.FlavorText)), " ")
        if flavor != "" {
                huntFitText(dc, huntAsset("MedievalSharp.ttf"), 20, flavor, 680, 12)
                dc.SetRGB(96.0/255.0, 66.0/255.0, 38.0/255.0)
                dc.DrawStringAnchored(flavor, 500, 474, 0.5, 0.5)
        }

        // ── wax tier seal (70,546) r26 — same treatment as hunt rank seal ──
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
        huntFitText(dc, huntAsset("Cinzel.ttf"), 24, sealLetter, 60, 10)
        dc.SetRGB(250.0/255.0, 210.0/255.0, 120.0/255.0)
        dc.DrawStringAnchored(sealLetter, cx, cy-1, 0.5, 0.5)

        // 💡 2026-09-15 PERF: honor ?fmt=jpeg — the bot requests fmt=jpeg; media
        // upload time scales with bytes and this canvas is fully opaque.
        utils.RespondImage(c, dc.Image())
}
