package combat

// ============================================
// 🎴 EVENT CARDS R4 — 2026-09-14
// ============================================
// Owner: "add a starting image card to quest starting … add an image card to
// the guild info command, make the skill tree an image card too, and make
// the skill upgrade an image card as well. All of them should have unique
// orientations and arrangements, but use the same assets we've been using
// for the RPG's UI."
//
// Four new kinds on the lamoot wood+gold+parchment family, each with a
// DIFFERENT canvas + arrangement:
//
//   QUESTSTART / RAID — 1000x600 landscape (bg_QSTART / bg_RAID).
//     Split arrangement: LEFT scene window (55,190)-(505,530) with the
//     dungeon environment + hero sprite; RIGHT contract panel with 5 rows.
//     seal (70,556) r24, caption (620,556).
//
//   GUILDINFO — 800x800 SQUARE (bg_GUILD). Crest ring (400,215) r100 with a
//     heraldic shield painted inside (emblem color + guild initial), plate
//     (230,330)-(570,384), motto line y410, charter panel (60,425)-(740,690)
//     rows y512 step 46, level pill + XP bar y658-676, three building rings
//     (baked, cx 220/400/580 cy 722) with name + L## painted inside.
//     seal (70,748) r24, caption (620,748).
//
//   SKILLTREE — 1200x800 ULTRAWIDE (bg_TREE). Full parchment inset
//     (40,190)-(1160,705) hosts a real RPG skill tree: branch headers,
//     trunk connectors down to a class root medallion, circular skill
//     medallions in 4 states (maxed/learned/open/locked), cur/max pips
//     inside each node. Class plate (85,120)-(455,174); skill-point pill
//     top-right. seal (70,742) r24, caption (640,745).
//
//   SKILLUP — 600x1000 PORTRAIT (bg_SKILLUP). Giant skill medallion in the
//     baked ring (300,385) r150 with tier-colored accents, skill name y566,
//     level pips y604, THE PATH panel rows y724 step 46.
//     seal (70,935) r26 (portrait standard), caption (305,936).
//
// All four respond through utils.RespondImage so ?fmt=jpeg&q=NN works.

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

// sealAt — wax seal at an arbitrary anchor (portraitWaxSeal is pinned to
// the portrait consts; these cards have their own anchors).
func sealAt(dc *gg.Context, text string, x, y, r float64) {
	if text == "" {
		return
	}
	dc.SetColor(portraitCol(128, 28, 40, 245))
	dc.DrawCircle(x, y, r)
	dc.Fill()
	dc.SetRGB(70.0/255.0, 12.0/255.0, 20.0/255.0)
	dc.SetLineWidth(3)
	dc.DrawCircle(x, y, r)
	dc.Stroke()
	dc.SetRGB(220.0/255.0, 150.0/255.0, 90.0/255.0)
	dc.SetLineWidth(2)
	dc.DrawCircle(x, y, r-6)
	dc.Stroke()
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, text, r*1.55, 9)
	dc.SetRGB(250.0/255.0, 210.0/255.0, 120.0/255.0)
	dc.DrawStringAnchored(text, x, y-1, 0.5, 0.5)
}

// captionAt — flavour caption at an arbitrary anchor. Same truncation rule
// as portraitCaption (fit-shrink gives up at minSize and would still draw).
func captionAt(dc *gg.Context, text string, x, y, maxW float64) {
	if text == "" {
		text = "the guild watches"
	}
	runes := []rune(text)
	if len(runes) > 52 {
		text = string(runes[:49]) + "..."
	}
	portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 20, text, maxW, 12)
	dc.SetRGB(176.0/255.0, 140.0/255.0, 96.0/255.0)
	dc.DrawStringAnchored(text, x, y, 0.5, 0.5)
}

// darkPill — the DUEL corner-pill styling, usable anywhere.
func darkPill(dc *gg.Context, text string, x, y float64, rightAnchor bool) {
	portraitPill(dc, text, x, y, rightAnchor,
		portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
}

// drawRows — generic label/value ledger rows (panel ink palette).
// 2026-09-14 QA: a long value ("WHISPERING FOREST") collided with a long
// label ("ENVIRONMENT") — the value now fits into the width LEFT over
// after the measured label, instead of a fixed 300px box.
func drawRows(dc *gg.Context, rows []portraitRow, xLabel, xValue, y0, step float64, max int, size int) {
	y := y0
	for i, row := range rows {
		if i >= max {
			break
		}
		label := portraitSanitize(row.Label)
		value := portraitSanitize(row.Value)
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), size, label, 230, 11)
		dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
		dc.DrawStringAnchored(label, xLabel, y, 0, 0.5)
		lw, _ := dc.MeasureString(label)
		avail := xValue - xLabel - lw - 24
		if avail < 150 {
			avail = 150
		}
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), size, value, avail, 10)
		dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
		dc.DrawStringAnchored(value, xValue, y, 1, 0.5)
		y += step
	}
}

// tierAccent — accent color for skill tiers (medallion ring + pips).
func tierAccent(tier int, ascended bool) color.NRGBA {
	if ascended {
		return portraitCol(200, 60, 80, 255)
	}
	switch tier {
	case 1:
		return portraitCol(170, 120, 62, 255) // bronze
	case 2:
		return portraitCol(172, 178, 190, 255) // silver
	case 3:
		return portraitCol(214, 170, 82, 255) // gold
	default:
		return portraitCol(158, 88, 208, 255) // arcane purple
	}
}

// initialLetters — up to 2 uppercase initials from a name ("Power Slash" -> PS).
func initialLetters(name string) string {
	letters := ""
	lastWasBreak := true
	for _, r := range strings.ToUpper(portraitSanitize(name)) {
		if r == ' ' {
			lastWasBreak = true
			continue
		}
		if lastWasBreak && r >= 'A' && r <= 'Z' {
			letters += string(r)
			if len(letters) >= 2 {
				break
			}
		}
		lastWasBreak = false
	}
	if letters == "" {
		letters = "?"
	}
	return letters
}

// drawSceneWindow — fill (x,y,w,h) with an environment asset + mood overlay.
func drawSceneWindow(dc *gg.Context, envFile string, x, y, w, h float64) {
	base := filepath.Base(envFile)
	if base == "." || base == "/" || base == "" {
		base = "spark_1.png"
	}
	if scene, err := utils.LoadImage(filepath.Join("assets", "rpgasset", "environment", base)); err == nil {
		scene = imaging.Fill(scene, int(w), int(h), imaging.Center, imaging.NearestNeighbor)
		dc.DrawImage(scene, int(x), int(y))
	} else {
		dc.SetColor(portraitCol(18, 22, 16, 255))
		dc.DrawRectangle(x, y, w, h)
		dc.Fill()
	}
	dc.SetColor(portraitCol(0, 0, 0, 46))
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()
}

// ─────────────────────────────────────────────────────────────────
// QUESTSTART / RAID — 1000x600 landscape
// ─────────────────────────────────────────────────────────────────
func renderQuestStartCard(c *gin.Context, req *portraitRequest, raid bool) {
	dc := gg.NewContext(1000, 600)
	bgFile := "bg_QSTART.png"
	if raid {
		bgFile = "bg_RAID.png"
	}
	if bgImg, err := utils.LoadImage(portraitAsset(bgFile)); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, 1000, 600)
		dc.Fill()
	}

	// hero name on the plate (55,118)-(330,172)
	name := portraitSanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, name, 220, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(name, 70, 145, 0, 0.5)

	// LEFT scene window: dungeon environment + hero sprite
	drawSceneWindow(dc, req.Background, 55, 190, 450, 340)
	sceneBottom := 524.0
	if !raid {
		if pImg := portraitFighterSprite(req.PlayerClass, req.PlayerIndex, 200, 235, "RIGHT"); pImg != nil {
			b := pImg.Bounds()
			px := 95.0
			utils.DrawShadow(dc, px+float64(b.Dx())/2, sceneBottom-2, float64(b.Dx())*0.45, 0.5)
			dc.DrawImage(pImg, int(px), int(sceneBottom)-b.Dy())
		}
	} else if req.PartyText != "" {
		darkPill(dc, portraitSanitize(req.PartyText), 78, 220, false)
	}

	// RIGHT contract rows (panel 535,190 - 945,530)
	drawRows(dc, req.Rows, 565, 915, 280, 48, 5, 21)

	sealAt(dc, portraitSanitize(req.SealText), 70, 556, 24)
	cap := req.Caption
	if cap == "" {
		if raid {
			cap = "the horns sound — assemble"
		} else {
			cap = "the gate groans open"
		}
	}
	captionAt(dc, portraitSanitize(cap), 620, 556, 300)

	utils.RespondImage(c, dc.Image())
}

// ─────────────────────────────────────────────────────────────────
// GUILDINFO — 800x800 square
// ─────────────────────────────────────────────────────────────────
func renderGuildInfoCard(c *gin.Context, req *portraitRequest) {
	dc := gg.NewContext(800, 800)
	if bgImg, err := utils.LoadImage(portraitAsset("bg_GUILD.png")); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, 800, 800)
		dc.Fill()
	}

	name := portraitSanitize(req.Nickname)
	if name == "" {
		name = "The Guild"
	}

	// heraldic shield inside the crest ring (400,215) r100
	accent := color.RGBA{R: 128, G: 28, B: 40, A: 255}
	if req.HexColor != "" {
		accent = utils.ParseHexColor(req.HexColor)
	}
	shield := func() {
		dc.MoveTo(352, 150)
		dc.LineTo(448, 150)
		dc.LineTo(448, 208)
		dc.CubicTo(448, 252, 424, 274, 400, 286)
		dc.CubicTo(376, 274, 352, 252, 352, 208)
		dc.LineTo(352, 150)
		dc.ClosePath()
	}
	dc.SetColor(accent)
	shield()
	dc.Fill()
	dc.SetColor(portraitCol(214, 170, 82, 255))
	dc.SetLineWidth(4)
	shield()
	dc.Stroke()
	// initial letter with dark drop (gg has no stroked text — paint twice)
	initial := initialLetters(name)
	if len(initial) > 1 {
		initial = initial[:1]
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 58, initial, 70, 24)
	dc.SetColor(portraitCol(30, 14, 6, 255))
	dc.DrawStringAnchored(initial, 402, 213, 0.5, 0.5)
	dc.SetColor(portraitCol(244, 214, 140, 255))
	dc.DrawStringAnchored(initial, 400, 210, 0.5, 0.5)

	// guild name on the plate (230,330)-(570,384)
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, name, 290, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(name, 245, 357, 0, 0.5)

	// motto
	if req.Motto != "" {
		motto := portraitSanitize(req.Motto)
		runes := []rune(motto)
		if len(runes) > 60 {
			motto = string(runes[:57]) + "..."
		}
		portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 19, "\""+motto+"\"", 620, 12)
		dc.SetRGB(196.0/255.0, 160.0/255.0, 106.0/255.0)
		dc.DrawStringAnchored("\""+motto+"\"", 400, 410, 0.5, 0.5)
	}

	// charter rows (panel 60,425 - 740,690) — QA r1: pulled up to
	// y498/step 44 so row 4 clears the XP bar at y658.
	drawRows(dc, req.Rows, 90, 710, 498, 44, 4, 21)

	// level pill + XP bar (658-676)
	lvlPill := fmt.Sprintf("LVL %d", req.Level)
	if req.Level <= 0 {
		lvlPill = "LVL 1"
	}
	darkPill(dc, lvlPill, 92, 667, false)
	pct := req.XPPercent
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	dc.SetColor(portraitCol(54, 36, 18, 255))
	dc.DrawRoundedRectangle(185, 658, 525, 18, 8)
	dc.Fill()
	fillW := 525.0 * float64(pct) / 100.0
	if fillW > 525 {
		fillW = 525
	}
	if fillW > 3 {
		dc.SetColor(portraitCol(214, 170, 82, 255))
		dc.DrawRoundedRectangle(185, 658, fillW, 18, 8)
		dc.Fill()
		dc.SetColor(portraitCol(240, 205, 120, 90))
		dc.DrawRoundedRectangle(185, 658, fillW, 7, 6)
		dc.Fill()
	}
	dc.SetColor(portraitCol(170, 130, 60, 255))
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(185, 658, 525, 18, 8)
	dc.Stroke()

	// building rings (baked at cx 220/400/580, cy 722): QA r1 — the
	// ring interiors sit on DARK wood, so the dark parchment ink was
	// unreadable. Dark backing + light gold text now.
	ringX := [3]float64{220, 400, 580}
	for i, b := range req.Buildings {
		if i >= 3 {
			break
		}
		cx := ringX[i]
		dc.SetColor(portraitCol(24, 16, 10, 170))
		dc.DrawCircle(cx, 722, 31)
		dc.Fill()
		bname := portraitSanitize(b.Name)
		if bname == "" {
			bname = "?"
		}
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 12, bname, 54, 8)
		dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
		dc.DrawStringAnchored(bname, cx, 710, 0.5, 0.5)
		lvlTxt := fmt.Sprintf("L%d", b.Level)
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 25, lvlTxt, 54, 12)
		dc.SetRGB(240.0/255.0, 205.0/255.0, 120.0/255.0)
		dc.DrawStringAnchored(lvlTxt, cx, 740, 0.5, 0.5)
	}

	sealAt(dc, portraitSanitize(req.SealText), 70, 748, 24)
	// QA r1: bottom-right is building-ring territory — only draw a
	// caption when the caller explicitly provides one.
	if req.Caption != "" {
		captionAt(dc, portraitSanitize(req.Caption), 620, 748, 230)
	}

	utils.RespondImage(c, dc.Image())
}

// ─────────────────────────────────────────────────────────────────
// SKILLTREE — 1200x800 ultrawide
// ─────────────────────────────────────────────────────────────────

// treeStateStyle — medallion fill/ring/ink per node state.
type treeStateStyle struct {
	fill color.NRGBA
	ring color.NRGBA
	ink  color.NRGBA
	glow bool
}

func treeStyleFor(state string) treeStateStyle {
	switch state {
	case "maxed":
		return treeStateStyle{portraitCol(214, 170, 82, 255), portraitCol(240, 205, 120, 255), portraitCol(52, 32, 16, 255), true}
	case "learned":
		return treeStateStyle{portraitCol(96, 62, 24, 255), portraitCol(214, 170, 82, 255), portraitCol(244, 214, 140, 255), false}
	case "open":
		return treeStateStyle{portraitCol(232, 214, 170, 255), portraitCol(170, 130, 60, 255), portraitCol(52, 32, 16, 255), true}
	default: // locked
		return treeStateStyle{portraitCol(58, 52, 42, 255), portraitCol(120, 112, 100, 255), portraitCol(150, 140, 120, 255), false}
	}
}

func drawTreeMedallion(dc *gg.Context, x, y, r float64, st treeStateStyle, initials, progress string) {
	if st.glow {
		dc.SetColor(portraitCol(214, 170, 82, 70))
		dc.DrawCircle(x, y, r+6)
		dc.Fill()
	}
	dc.SetColor(st.fill)
	dc.DrawCircle(x, y, r)
	dc.Fill()
	dc.SetColor(st.ring)
	dc.SetLineWidth(3)
	dc.DrawCircle(x, y, r)
	dc.Stroke()
	dc.SetColor(portraitCol(0, 0, 0, 60))
	dc.SetLineWidth(1)
	dc.DrawCircle(x, y, r-4)
	dc.Stroke()

	if progress != "" {
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, initials, r*1.5, 12)
		dc.SetColor(st.ink)
		dc.DrawStringAnchored(initials, x, y-8, 0.5, 0.5)
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, progress, r*1.3, 8)
		dc.DrawStringAnchored(progress, x, y+16, 0.5, 0.5)
	} else {
		portraitFitText(dc, portraitAsset("Cinzel.ttf"), 30, initials, r*1.5, 12)
		dc.SetColor(st.ink)
		dc.DrawStringAnchored(initials, x, y, 0.5, 0.5)
	}
}

func renderSkillTreeCard(c *gin.Context, req *portraitRequest) {
	dc := gg.NewContext(1200, 800)
	if bgImg, err := utils.LoadImage(portraitAsset("bg_TREE.png")); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, 1200, 800)
		dc.Fill()
	}

	// class plate (85,120)-(455,174)
	classLine := portraitSanitize(req.ClassName)
	if classLine == "" {
		classLine = "Adventurer"
	}
	if req.Level > 0 {
		classLine = fmt.Sprintf("%s - LV %d", classLine, req.Level)
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, classLine, 330, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(classLine, 115, 147, 0, 0.5)

	// skill points pill top-right
	pts := req.SkillPoints
	ptsText := "NO POINTS"
	if pts == 1 {
		ptsText = "1 POINT"
	} else if pts > 1 {
		ptsText = fmt.Sprintf("%d POINTS", pts)
	}
	darkPill(dc, ptsText, 1130, 147, true)

	// ── tree layout inside the parchment inset (40,190)-(1160,705) ──
	n := len(req.Branches)
	if n > 0 {
		bx := make([]float64, n)
		for i := range bx {
			if n == 1 {
				bx[i] = 600
			} else {
				bx[i] = 220 + float64(i)*(760.0/float64(n-1))
			}
		}
		const nodeR = 30.0
		// tier rows: t1 bottom y585, t4 top y300
		maxTier := 1
		for _, br := range req.Branches {
			for _, s := range br.Skills {
				if s.Tier > maxTier {
					maxTier = s.Tier
				}
			}
		}
		if maxTier > 4 {
			maxTier = 4
		}
		yFor := func(tier int) float64 {
			if tier < 1 {
				tier = 1
			}
			if tier > maxTier {
				tier = maxTier
			}
			if maxTier == 1 {
				return 585
			}
			step := (585.0 - 300.0) / float64(maxTier-1)
			return 585 - float64(tier-1)*step
		}
		clampX := func(x float64) float64 {
			if x < 80 {
				return 80
			}
			if x > 1120 {
				return 1120
			}
			return x
		}
		nodePosX := func(bxc float64, slot, count int) float64 {
			if count <= 1 {
				return bxc
			}
			return clampX(bxc + (float64(slot)-float64(count-1)/2.0)*110.0)
		}

		// organise skills into ordered per-tier slots
		type slotNode struct {
			node portraitNode
			x, y float64
		}
		branchSlots := make([][]slotNode, n)
		for i, br := range req.Branches {
			byTier := map[int][]portraitNode{}
			order := []int{}
			for _, s := range br.Skills {
				t := s.Tier
				if t < 1 {
					t = 1
				}
				if t > maxTier {
					t = maxTier
				}
				if _, ok := byTier[t]; !ok {
					order = append(order, t)
				}
				byTier[t] = append(byTier[t], s)
			}
			slots := []slotNode{}
			for _, t := range order {
				for j, s := range byTier[t] {
					slots = append(slots, slotNode{node: s, x: nodePosX(bx[i], j, len(byTier[t])), y: yFor(t)})
				}
			}
			branchSlots[i] = slots
		}

		// branch headers
		for i, br := range req.Branches {
			hname := portraitSanitize(br.Name)
			portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, hname, 300, 12)
			dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
			dc.DrawStringAnchored(hname, bx[i], 235, 0.5, 0.5)
		}

		// connectors first (under the medallions): every node links
		// DOWN its branch centre; tier-1 nodes link to the class root.
		for i := range req.Branches {
			for _, sn := range branchSlots[i] {
				locked := sn.node.State == "locked"
				dc.SetColor(portraitCol(150, 120, 70, 210))
				dc.SetLineWidth(3)
				if locked {
					dc.SetColor(portraitCol(170, 160, 140, 150))
					dc.SetLineWidth(2)
				}
				if sn.node.Tier <= 1 {
					dc.DrawLine(bx[i], sn.y+nodeR-4, 600, 645)
				} else {
					dc.DrawLine(bx[i], sn.y+nodeR-4, bx[i], sn.y-nodeR+8)
				}
				dc.Stroke()
			}
		}

		// class root medallion
		rootInitial := initialLetters(classLine)
		if len(rootInitial) > 1 {
			rootInitial = rootInitial[:1]
		}
		drawTreeMedallion(dc, 600, 665, 24, treeStyleFor("learned"), rootInitial, "")

		// medallions + names
		for i := range req.Branches {
			for _, sn := range branchSlots[i] {
				s := sn.node
				st := treeStyleFor(s.State)
				initials := initialLetters(s.Name)
				progress := ""
				if s.Cur > 0 {
					progress = fmt.Sprintf("%d/%d", s.Cur, s.Max)
				}
				drawTreeMedallion(dc, sn.x, sn.y, nodeR, st, initials, progress)
				nname := portraitSanitize(s.Name)
				portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, nname, 106, 9)
				ink := st.ink
				if s.State == "locked" {
					ink = portraitCol(140, 128, 105, 255)
				}
				dc.SetColor(ink)
				dc.DrawStringAnchored(nname, sn.x, sn.y+nodeR+16, 0.5, 0.5)
			}
		}
	}

	sealAt(dc, portraitSanitize(req.SealText), 70, 742, 24)
	cap := req.Caption
	if cap == "" {
		cap = "spend points with .skill up <name>"
	}
	captionAt(dc, portraitSanitize(cap), 640, 745, 320)

	utils.RespondImage(c, dc.Image())
}

// ─────────────────────────────────────────────────────────────────
// SKILLUP — 600x1000 portrait
// ─────────────────────────────────────────────────────────────────
func renderSkillUpCard(c *gin.Context, req *portraitRequest) {
	dc := gg.NewContext(600, 1000)
	if bgImg, err := utils.LoadImage(portraitAsset("bg_SKILLUP.png")); err == nil {
		dc.DrawImage(bgImg, 0, 0)
	} else {
		dc.SetRGB(0.09, 0.06, 0.04)
		dc.DrawRectangle(0, 0, 600, 1000)
		dc.Fill()
	}

	// hero name on the plate
	name := portraitSanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, name, 215, 12)
	dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
	dc.DrawStringAnchored(name, 84, 159, 0, 0.5)

	accent := tierAccent(req.Tier, req.Ascended)

	// medallion inside the baked ring (300,385) r150
	dc.SetColor(portraitCol(24, 16, 10, 255))
	dc.DrawCircle(300, 385, 141)
	dc.Fill()
	dc.SetColor(accent)
	dc.SetLineWidth(5)
	dc.DrawCircle(300, 385, 141)
	dc.Stroke()
	dc.SetColor(portraitCol(0, 0, 0, 80))
	dc.SetLineWidth(1)
	dc.DrawCircle(300, 385, 128)
	dc.Stroke()

	initials := initialLetters(req.SkillName)
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 76, initials, 170, 30)
	dc.SetColor(portraitCol(18, 10, 4, 255))
	dc.DrawStringAnchored(initials, 302, 368, 0.5, 0.5)
	dc.SetColor(accent)
	dc.DrawStringAnchored(initials, 300, 365, 0.5, 0.5)

	tierLabel := fmt.Sprintf("TIER %d", req.Tier)
	if req.Ascended {
		tierLabel = "ASCENDED"
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, tierLabel, 160, 10)
	dc.SetRGB(176.0/255.0, 140.0/255.0, 96.0/255.0)
	dc.DrawStringAnchored(tierLabel, 300, 448, 0.5, 0.5)

	// skill name between ring and panel
	sname := portraitSanitize(req.SkillName)
	if sname == "" {
		sname = "Skill"
	}
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, sname, 430, 12)
	dc.SetRGB(240.0/255.0, 205.0/255.0, 120.0/255.0)
	dc.DrawStringAnchored(sname, 300, 566, 0.5, 0.5)

	// level pips y604 (dots filled to cur)
	maxPips := req.SkillMax
	if maxPips < 1 {
		maxPips = 1
	}
	if maxPips > 7 {
		maxPips = 7
	}
	curPips := req.Cur
	if curPips > maxPips {
		curPips = maxPips
	}
	totalW := float64(maxPips-1) * 22
	startX := 300 - totalW/2
	for i := 0; i < maxPips; i++ {
		px := startX + float64(i)*22
		if i < curPips {
			dc.SetColor(accent)
			dc.DrawCircle(px, 604, 7)
			dc.Fill()
			dc.SetColor(portraitCol(0, 0, 0, 90))
			dc.SetLineWidth(1.5)
			dc.DrawCircle(px, 604, 7)
			dc.Stroke()
		} else {
			dc.SetColor(portraitCol(24, 16, 10, 200))
			dc.DrawCircle(px, 604, 6)
			dc.Fill()
			dc.SetColor(portraitCol(170, 130, 60, 255))
			dc.SetLineWidth(2)
			dc.DrawCircle(px, 604, 6)
			dc.Stroke()
		}
	}

	// THE PATH panel rows (panel 55,640 - 545,880)
	drawRows(dc, req.Rows, 85, 515, 724, 46, 3, 20)

	sealAt(dc, portraitSanitize(req.SealText), 70, 935, 26)
	cap := req.Caption
	if cap == "" {
		if req.SkillMax > 0 && req.Cur >= req.SkillMax {
			cap = "the path is fully mastered"
		} else {
			cap = "the path sharpens"
		}
	}
	portraitCaption(dc, portraitSanitize(cap))

	utils.RespondImage(c, dc.Image())
}
