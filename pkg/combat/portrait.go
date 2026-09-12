package combat

// ============================================
// 🖼️ PORTRAIT EVENT CARDS — 2026-09-12
// ============================================
// Owner ask: "add more rpg image cards, with the same assets but a different
// orientation". Same lamoot wood+gold+parchment family as CRAFT/HUNT/FISH/
// DECREE/BOSS, but 600x1000 PORTRAIT (WhatsApp shows portrait art taller in
// chat). Two kinds:
//
//   DUEL  — PvP duel result (pvpSystem.finishDuel):
//     baked bg_DUEL.png: banner "DUEL RESOLVED", leather name plate
//       (60,132)-(330,186), scene window (55,208)-(545,642), spoils panel
//       (55,662)-(545,902) with label + divider y712.
//     we paint: winner name (84,159), spark_5 arena + winner sprite
//       (bottom-left, facing RIGHT) + faded loser sprite (bottom-right,
//       facing LEFT), VICTOR/DEFEATED pills, ledger rows y=746 step 44,
//       wax seal (70,935) r26 = winner level, caption (305,936).
//     2026-09-12: +forfeit flag -> a third "BY FORFEIT" pill centred between
//       VICTOR/DEFEATED (owner: "no card on forfeit" — forfeits now render
//       this card too, stamped).
//
//   QUEST — dungeon completion tally (guildAdventure.endAdventure):
//     baked bg_QUEST.png: banner "QUEST COMPLETE", name plate, tally panel
//       (55,208)-(545,832) with "THE TALLY" + dividers y266/y492 +
//       "PARTY SPOILS" (85,520).
//     we paint: party name (84,159), stat rows y=304 step 38, per-player
//       rows y=556 step 64, wax seal = dungeon rank, caption = narration.
//
//   TRIAL — evolution/ascension (2026-09-12, owner: same assets, different
//     orientation). baked bg_TRIAL.png — SAME 600x1000 geometry as QUEST
//     (plate (60,132)-(330,186), panel (55,208)-(545,832), dividers y266/
//     y492, lower label "THE REWARD" (85,520)) but banner "EVOLUTION",
//     panel label "THE ASCENSION". Renders through the SAME stat-row code
//     path as QUEST (Ledger = WAS/NOW/TIER, Players = reward rows),
//     seal = tier letter, caption = flavour.
//
// NOTE: no gg Clip() anywhere — clip state leaks in this gg version and
// erases later draws (same bug as hunt/boss QA rounds).

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

// portraitCol builds an NRGBA color (shorthand for the wax/pill palettes).
func portraitCol(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }

// geometry — MUST match build_portrait_bgs.py bake contract
const (
	portraitW          = 600.0
	portraitH          = 1000.0
	duelSceneX         = 55.0
	duelSceneY         = 208.0
	duelSceneW         = 490.0
	duelSceneH         = 434.0
	duelLedgerY0       = 746.0
	duelLedgerStep     = 44.0
	questStatY0        = 304.0
	questStatStep      = 38.0
	questPlayerY0      = 556.0
	questPlayerStep    = 64.0
	portraitSealX      = 70.0
	portraitSealY      = 935.0
	portraitSealR      = 26.0
	portraitCaptionX   = 305.0
	portraitCaptionY   = 936.0
	portraitCaptionMax = 340.0
)

func portraitAsset(name string) string { return utils.GetAssetPath("rpgasset", "ui", "craft/"+name) }

func portraitSanitize(s string) string { return huntSanitize(s) }

func portraitFitText(dc *gg.Context, path string, size int, s string, maxW float64, minSize int) {
	huntFitText(dc, path, size, s, maxW, minSize)
}

// portraitFade scales every pixel's alpha (loser ghost effect).
func portraitFade(img image.Image, factor float64) image.Image {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			i := out.PixOffset(x, y)
			out.Pix[i+3] = uint8(float64(out.Pix[i+3]) * factor)
		}
	}
	return out
}

// portraitFighterSprite resolves + trims + fits a class sprite for the duel
// window. facing "RIGHT" = as-is if native RIGHT else flipped; "LEFT" =
// mirrored logic. Returns nil when the class is unknown (window stays arena).
func portraitFighterSprite(class string, index int, maxW, maxH int, facing string) image.Image {
	file := ""
	// 2026-09-12 QA r1: callers send display-case class names ("Fighter",
	// "Scout") but the registry keys are UPPERCASE — normalize or sprites
	// silently vanish (window stays bare arena).
	key := strings.ToUpper(strings.TrimSpace(class))
	if sprites, ok := CharacterSprites[key]; ok && len(sprites) > 0 {
		file = sprites[index%len(sprites)]
	}
	if file == "" {
		return nil
	}
	img, err := utils.LoadImage(filepath.Join("assets", "rpgasset", "characters", file))
	if err != nil {
		return nil
	}
	img = trimTransparent(img, 8)
	b := img.Bounds()
	if b.Dx() < maxW/2 && b.Dy() < maxH/2 {
		// tiny canvas — integer upscale first so pixel art stays crisp
		scale := maxW / 2 / b.Dx()
		if s2 := maxH / 2 / b.Dy(); s2 < scale {
			scale = s2
		}
		if scale > 1 {
			img = imaging.Resize(img, b.Dx()*scale, b.Dy()*scale, imaging.NearestNeighbor)
		}
	}
	img = imaging.Fit(img, maxW, maxH, imaging.NearestNeighbor)
	native := GetSpriteFacing(file)
	wantFlip := (facing == "RIGHT" && native == "LEFT") || (facing == "LEFT" && native == "RIGHT")
	if wantFlip {
		img = imaging.FlipH(img)
	}
	return img
}

func portraitWaxSeal(dc *gg.Context, text string) {
	if text == "" {
		return
	}
	dc.SetColor(portraitCol(128, 28, 40, 245))
	dc.DrawCircle(portraitSealX, portraitSealY, portraitSealR)
	dc.Fill()
	dc.SetRGB(70.0/255.0, 12.0/255.0, 20.0/255.0)
	dc.SetLineWidth(3)
	dc.DrawCircle(portraitSealX, portraitSealY, portraitSealR)
	dc.Stroke()
	dc.SetRGB(220.0/255.0, 150.0/255.0, 90.0/255.0)
	dc.SetLineWidth(2)
	dc.DrawCircle(portraitSealX, portraitSealY, portraitSealR-6)
	dc.Stroke()
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, text, 40, 10)
	dc.SetRGB(250.0/255.0, 210.0/255.0, 120.0/255.0)
	dc.DrawStringAnchored(text, portraitSealX, portraitSealY-1, 0.5, 0.5)
}

func portraitCaption(dc *gg.Context, text string) {
	if text == "" {
		text = "the guild watches"
	}
	// QA round 3: at min font size a long narration still overran the wax
	// seal (fit-shrink gives up at minSize and draws anyway). Hard-truncate.
	runes := []rune(text)
	if len(runes) > 56 {
		text = string(runes[:53]) + "..."
	}
	portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 20, text, portraitCaptionMax, 12)
	dc.SetRGB(176.0/255.0, 140.0/255.0, 96.0/255.0)
	dc.DrawStringAnchored(text, portraitCaptionX, portraitCaptionY, 0.5, 0.5)
}

func portraitPill(dc *gg.Context, text string, x, y float64, rightAnchor bool, fill, txt color.NRGBA) {
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, text, 220, 12)
	tw, _ := dc.MeasureString(text)
	// pill rect must follow the SAME anchor as the text (QA round 1:
	// centered pill + edge-anchored text = text escaped the pill)
	px := x - 12
	ax := 0.0
	if rightAnchor {
		px = x - tw - 12
		ax = 1.0
	}
	dc.SetColor(fill)
	dc.DrawRoundedRectangle(px, y-14, tw+24, 28, 8)
	dc.Fill()
	dc.SetColor(txt)
	dc.DrawStringAnchored(text, x, y, ax, 0.5)
}

// portraitPillCentred — pill centred on x (used for the BY FORFEIT stamp).
func portraitPillCentred(dc *gg.Context, text string, x, y float64, fill, txt color.NRGBA) {
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, text, 220, 12)
	tw, _ := dc.MeasureString(text)
	dc.SetColor(fill)
	dc.DrawRoundedRectangle(x-tw/2-12, y-14, tw+24, 28, 8)
	dc.Fill()
	dc.SetColor(txt)
	dc.DrawStringAnchored(text, x, y, 0.5, 0.5)
}

// GeneratePortraitCard renders DUEL / QUEST portrait event cards (600x1000).
func GeneratePortraitCard(c *gin.Context) {
	var req struct {
		Kind     string `json:"kind"`
		Nickname string `json:"nickname"`
		Caption  string `json:"caption"`
		SealText string `json:"sealText"`
		Forfeit  bool   `json:"forfeit"` // DUEL only: "BY FORFEIT" stamp
		Ledger   []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"ledger"`
		Players []struct {
			Name string `json:"name"`
			XP   string `json:"xp"`
			Zeni string `json:"zeni"`
		} `json:"players"`
		WinnerClass string `json:"winnerClass"`
		WinnerIndex int    `json:"winnerIndex"`
		LoserClass  string `json:"loserClass"`
		LoserIndex  int    `json:"loserIndex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	dc := gg.NewContext(int(portraitW), int(portraitH))
	bgFile := "bg_DUEL.png"
	if req.Kind == "QUEST" {
		bgFile = "bg_QUEST.png"
	} else if req.Kind == "TRIAL" {
		bgFile = "bg_TRIAL.png"
	}
	if bgImg, err := utils.LoadImage(portraitAsset(bgFile)); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, portraitW, portraitH)
		dc.Fill()
	}

	// ── name plate (winner name / party name) ──
	name := portraitSanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 28, name, 215, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(name, 84, 159, 0, 0.5)

	if req.Kind == "QUEST" || req.Kind == "TRIAL" {
		// ── stat rows (label/value) ──
		y := questStatY0
		for i, row := range req.Ledger {
			if i >= 6 {
				break
			}
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, portraitSanitize(row.Label), 280, 12)
			dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
			dc.DrawStringAnchored(portraitSanitize(row.Label), 85, y, 0, 0.5)
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, portraitSanitize(row.Value), 280, 12)
			dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
			dc.DrawStringAnchored(portraitSanitize(row.Value), 515, y, 1, 0.5)
			y += questStatStep
		}
		// ── per-player spoils rows ──
		y = questPlayerY0
		for i, p := range req.Players {
			if i >= 4 {
				break
			}
			pn := portraitSanitize(p.Name)
			portraitFitText(dc, portraitAsset("CinzelDecBold.ttf"), 22, pn, 190, 10)
			dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
			dc.DrawStringAnchored(pn, 85, y, 0, 0.5)
			xp := portraitSanitize(p.XP)
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, xp, 150, 10)
			dc.SetRGB(96.0/255.0, 110.0/255.0, 50.0/255.0)
			dc.DrawStringAnchored(xp, 370, y, 1, 0.5)
			zeni := portraitSanitize(p.Zeni)
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, zeni, 130, 10)
			dc.SetRGB(150.0/255.0, 110.0/255.0, 30.0/255.0)
			dc.DrawStringAnchored(zeni, 515, y, 1, 0.5)
			y += questPlayerStep
		}
		portraitWaxSeal(dc, portraitSanitize(req.SealText))
		portraitCaption(dc, portraitSanitize(req.Caption))
	} else {
		// ── DUEL: arena + winner + faded loser ──
		if arena, err := utils.LoadImage(filepath.Join("assets", "rpgasset", "environment", "spark_5.png")); err == nil {
			arena = imaging.Fill(arena, int(duelSceneW), int(duelSceneH), imaging.Center, imaging.NearestNeighbor)
			dc.DrawImage(arena, int(duelSceneX), int(duelSceneY))
		} else {
			dc.SetColor(portraitCol(18, 22, 16, 255))
			dc.DrawRectangle(duelSceneX, duelSceneY, duelSceneW, duelSceneH)
			dc.Fill()
		}
		dc.SetColor(portraitCol(0, 0, 0, 46))
		dc.DrawRectangle(duelSceneX, duelSceneY, duelSceneW, duelSceneH)
		dc.Fill()

		sceneBottom := duelSceneY + duelSceneH - 6
		if wImg := portraitFighterSprite(req.WinnerClass, req.WinnerIndex, 270, 330, "RIGHT"); wImg != nil {
			b := wImg.Bounds()
			px := duelSceneX + 44
			utils.DrawShadow(dc, px+float64(b.Dx())/2, sceneBottom-2, float64(b.Dx())*0.45, 0.5)
			dc.DrawImage(wImg, int(px), int(sceneBottom)-b.Dy())
		}
		if lImg := portraitFighterSprite(req.LoserClass, req.LoserIndex, 150, 150, "LEFT"); lImg != nil {
			lImg = portraitFade(lImg, 0.5)
			b := lImg.Bounds()
			ax := duelSceneX + duelSceneW - 22 - float64(b.Dx())
			loserBottom := sceneBottom - 12 // lift off foreground clutter (QA r1)
			utils.DrawShadow(dc, ax+float64(b.Dx())/2, loserBottom, float64(b.Dx())*0.42, 0.5)
			dc.DrawImage(lImg, int(ax), int(loserBottom)-b.Dy())
		}
		// corner pills INSIDE the scene window (pixel-bounded, no Clip)
		portraitPill(dc, "VICTOR", duelSceneX+62, duelSceneY+26, false,
			portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
		portraitPill(dc, "DEFEATED", duelSceneX+duelSceneW-62, duelSceneY+26, true,
			portraitCol(8, 8, 8, 150), portraitCol(232, 116, 97, 255))
		if req.Forfeit {
			// 2026-09-12 owner: "no card on forfeit" — forfeit endings carry
			// the same card, stamped BY FORFEIT between the corner pills.
			portraitPillCentred(dc, "BY FORFEIT", duelSceneX+duelSceneW/2, duelSceneY+26,
				portraitCol(96, 18, 28, 175), portraitCol(250, 210, 120, 255))
		}

		// ── spoils ledger rows ──
		y := duelLedgerY0
		for i, row := range req.Ledger {
			if i >= 4 {
				break
			}
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 21, portraitSanitize(row.Label), 190, 12)
			dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
			dc.DrawStringAnchored(portraitSanitize(row.Label), 85, y, 0, 0.5)
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 21, portraitSanitize(row.Value), 300, 10)
			dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
			dc.DrawStringAnchored(portraitSanitize(row.Value), 515, y, 1, 0.5)
			y += duelLedgerStep
		}
		portraitWaxSeal(dc, portraitSanitize(req.SealText))
		portraitCaption(dc, portraitSanitize(req.Caption))
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to encode portrait card"})
		return
	}
	c.Data(200, "image/png", buf)
}
