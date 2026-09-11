package economy

// craft_renderer.go — RPG-style card for item-creation types
// (CRAFT / BREW / COOK / FORGE). Deliberately NOT the Kenney money look:
// lamoot wood + gold frame + parchment panel + game-icons type emblem,
// owner-approved mock download/craft_cards/MOCK_*.png.
// Static art is baked (assets/rpgasset/ui/craft/bg_<TYPE>.png); this file
// draws only the dynamic parts: nickname, item name, qty seal, caption.

import (
        "fmt"
        "image"
        "image/color"
        "log"
        "os"
        "strings"

        "image-service/pkg/utils"

        "github.com/fogleman/gg"
        _ "image/png"
)

func isCraftType(t string) bool {
        return t == "CRAFT" || t == "BREW" || t == "COOK" || t == "FORGE"
}

func craftAsset(name string) string { return utils.GetAssetPath("rpgasset", "ui", "craft/"+name) }

func craftCinzel() string     { return craftAsset("Cinzel.ttf") }
func craftCinzelDec() string  { return craftAsset("CinzelDecBold.ttf") }
func craftMed() string        { return craftAsset("MedievalSharp.ttf") }

// drawRPGCraft renders the full RPG card onto dc (1000x600).
func drawRPGCraft(dc *gg.Context, req TransactionCardRequest, txType string) {
        bg, err := loadPNG(craftAsset("bg_" + txType + ".png"))
        if err != nil {
                log.Printf("craft bg missing (%v) — falling back to plain base", err)
                drawBase(dc)
        } else {
                dc.DrawImage(bg, 0, 0)
        }

        // nickname — gold Cinzel on the leather plate (plate is baked)
        name := sanitize(req.Nickname)
        if name == "" {
                name = "Adventurer"
        }
        loadFit(dc, craftCinzel(), 30, name, 240, 12)
        textLM(dc, name, 90, 84, rgb(214, 170, 82))

        // item name — dark ink Cinzel Decorative on the parchment
        item := sanitize(req.ItemName)
        if item == "" {
                item = "Unknown Item"
        }
        loadFit(dc, craftCinzelDec(), 52, item, 470, 16)
        textCenter(dc, item, 565, 298, rgb(52, 32, 16))

        // quantity wax seal (only for multi-crafts; seal is drawn, not baked)
        if req.Amount > 1 {
                cx, cy, r := 882.0, 96.0, 36.0
                dc.SetColor(color.NRGBA{R: 128, G: 28, B: 40, A: 245})
                dc.DrawCircle(cx, cy, r)
                dc.Fill()
                dc.SetColor(rgb(70, 12, 20))
                dc.SetLineWidth(3)
                dc.DrawCircle(cx, cy, r)
                dc.Stroke()
                dc.SetColor(rgb(220, 150, 90))
                dc.SetLineWidth(2)
                dc.DrawCircle(cx, cy, r-6)
                dc.Stroke()
                qs := fmt.Sprintf("X%d", int(req.Amount))
                loadFit(dc, craftCinzel(), 26, qs, 60, 12)
                textCenter(dc, qs, cx, cy-1, rgb(250, 210, 120))
        }

        // caption
        cap := sanitize(fmt.Sprintf(".j %s %s", strings.ToLower(txType), item))
        loadFit(dc, craftMed(), 22, cap, 800, 12)
        textCenter(dc, cap, 500, 552, rgb(176, 140, 96))
}

func loadPNG(path string) (image.Image, error) {
        f, err := os.Open(path)
        if err != nil {
                return nil, err
        }
        defer f.Close()
        img, _, err := image.Decode(f)
        return img, err
}
