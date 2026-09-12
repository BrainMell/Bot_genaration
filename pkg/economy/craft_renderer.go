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
	return t == "CRAFT" || t == "BREW" || t == "COOK" || t == "FORGE" || t == "FISH"
}

func craftAsset(name string) string { return utils.GetAssetPath("rpgasset", "ui", "craft/"+name) }

func craftCinzel() string    { return craftAsset("Cinzel.ttf") }
func craftCinzelDec() string { return craftAsset("CinzelDecBold.ttf") }
func craftMed() string       { return craftAsset("MedievalSharp.ttf") }

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

	// caption — req.Details override (FISH passes flavor), else ".j <type> <item>"
	cap := sanitize(req.Details)
	if cap == "" {
		cap = sanitize(fmt.Sprintf(".j %s %s", strings.ToLower(txType), item))
	}
	loadFit(dc, craftMed(), 22, cap, 800, 12)
	textCenter(dc, cap, 500, 552, rgb(176, 140, 96))
}

// drawRPGDecree renders the ROYAL DECREE rank-up card (bright parchment).
// Static art baked in bg_DECREE.png (build_hfd_bgs.py): frames, wood banner
// title, "LET IT BE KNOWN" header + divider, flavor line "by decree of the
// Adventurers' Guild". Dynamic: nickname plate, big rank name (500,300),
// old→new ledger with drawn arrow (500,396), big wax seal with rank letter
// (880,490) r48, caption (500,552).
func drawRPGDecree(dc *gg.Context, req TransactionCardRequest) {
	bg, err := loadPNG(craftAsset("bg_DECREE.png"))
	if err != nil {
		log.Printf("decree bg missing (%v) — falling back to plain base", err)
		drawBase(dc)
	} else {
		dc.DrawImage(bg, 0, 0)
	}

	// nickname — gold Cinzel on the leather plate
	name := sanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	loadFit(dc, craftCinzel(), 30, name, 240, 12)
	textLM(dc, name, 90, 84, rgb(214, 170, 82))

	// big rank name — dark ink Cinzel Decorative center
	rankName := sanitize(req.ItemName)
	if rankName == "" {
		rankName = "RANK UP"
	}
	loadFit(dc, craftCinzelDec(), 54, rankName, 640, 20)
	textCenter(dc, rankName, 500, 300, rgb(52, 32, 16))

	// ledger: old → new with a DRAWN arrow (Cinzel has no → glyph)
	ledger := sanitize(req.Details)
	if ledger != "" {
		parts := strings.Split(ledger, "->")
		if len(parts) == 2 {
			oldR, newR := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			loadFit(dc, craftCinzel(), 30, oldR, 200, 14)
			ow, _ := dc.MeasureString(oldR)
			loadFit(dc, craftCinzel(), 30, newR, 200, 14)
			nw, _ := dc.MeasureString(newR)
			const arrowHalf = 34.0
			total := ow + arrowHalf*2 + nw
			x0 := 500 - total/2
			textLM(dc, oldR, x0, 396, rgb(110, 78, 36))
			// arrow: shaft + chevron in dark gold
			ax0, ax1 := x0+ow+6, x0+ow+arrowHalf*2-6
			dc.SetColor(rgb(110, 78, 36))
			dc.SetLineWidth(3)
			dc.DrawLine(ax0, 396, ax1, 396)
			dc.Stroke()
			dc.DrawLine(ax1-9, 388, ax1, 396)
			dc.Stroke()
			dc.DrawLine(ax1-9, 404, ax1, 396)
			dc.Stroke()
			textLM(dc, newR, x0+ow+arrowHalf*2, 396, rgb(110, 78, 36))
		} else {
			loadFit(dc, craftCinzel(), 30, ledger, 640, 16)
			textCenter(dc, ledger, 500, 396, rgb(110, 78, 36))
		}
	}

	// big wax seal with the new rank letter (880,490) r48
	if seal := sanitize(req.SealText); seal != "" {
		cx, cy, r := 880.0, 490.0, 48.0
		dc.SetColor(color.NRGBA{R: 128, G: 28, B: 40, A: 245})
		dc.DrawCircle(cx, cy, r)
		dc.Fill()
		dc.SetColor(rgb(70, 12, 20))
		dc.SetLineWidth(4)
		dc.DrawCircle(cx, cy, r)
		dc.Stroke()
		dc.SetColor(rgb(220, 150, 90))
		dc.SetLineWidth(2)
		dc.DrawCircle(cx, cy, r-8)
		dc.Stroke()
		loadFit(dc, craftCinzel(), 40, seal, 70, 14)
		textCenter(dc, seal, cx, cy-1, rgb(250, 210, 120))
	}

	// caption (fixed flavor — the ledger already carries the ranks).
	// Dark ink: the decree bg is BRIGHT parchment — the wood-card tan vanishes.
	cap := "keep rising - the guild watches"
	loadFit(dc, craftMed(), 22, cap, 800, 12)
	textCenter(dc, cap, 500, 552, rgb(122, 88, 46))
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
