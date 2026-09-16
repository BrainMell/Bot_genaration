package combat

// ============================================
// 🎴 R6 CARD REDESIGNS - 2026-09-14 (owner feedback round)
// ============================================
// Owner: equipment card "showing dual" (the r5 two-plate grid) → completely
// fresh design, same family/assets, different orientation+layout; skill tree
// still "trash" → a layout that actually fits an RPG skill-tree UI; abilities
// needs its own dedicated card instead of the shared r5 board.
//
//   SKILLTREE (r6) - 1200x1000 on bg_TREE2.png (bg_TREE extended 200px).
//     Dark "constellation chart" panel inside the parchment inset: class
//     root medallion at the bottom, branch spines fanning up-left/up/up-
//     right through tier rows, quadratic-bezier links that GLOW along the
//     learned path, medallion nodes in 4 states, branch header chips, tier
//     badges, star-field backdrop, legend. PoE-style, but wood+gold family.
//
//   EQUIP (r6) - 1500x1000 LANDSCAPE armory wall. Left: hero panel with the
//     player's class sprite on a shadowed stage + SLOTS/AVG TIER/DAMAGED
//     chips. Right: single-column gear rack - one wide horizontal strip per
//     slot (tier-ringed medallion, slot label, item, tier pill, durability
//     bar). No grid of plates anywhere.
//
//   ABILITIES (r6) - 1000x1400 arc-scroll grimoire. Wooden scroll rods top
//     + bottom, unrolled parchment body, swallowtail ribbon group banners,
//     wax-seal number medallions, level pips beside every ability name,
//     cost/CD line under it, effect column right. Shares nothing with the
//     r5 board family except fonts/colors.

import (
        "fmt"
        "image/color"
        "strings"

        "image-service/pkg/utils"

        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════════
// SKILLTREE R6-1200x1000 constellation tree
// ═══════════════════════════════════════════════════════════════════════

// treeStyleR6 - node styling tuned for the DARK panel (the r5 palette was
// built for parchment; pale inks vanished on it).
type treeStyleR6 struct {
        fill color.NRGBA
        ring color.NRGBA
        ink  color.NRGBA
        name color.NRGBA
        glow int // 0 none · 1 soft · 2 strong
}

func treeStyleForR6(state string) treeStyleR6 {
        switch state {
        case "maxed":
                return treeStyleR6{portraitCol(36, 25, 11, 255), portraitCol(242, 206, 122, 255), portraitCol(250, 221, 142, 255), portraitCol(250, 221, 142, 255), 2}
        case "learned":
                return treeStyleR6{portraitCol(28, 21, 12, 255), portraitCol(214, 170, 82, 255), portraitCol(244, 214, 140, 255), portraitCol(232, 214, 170, 255), 1}
        case "open":
                return treeStyleR6{portraitCol(33, 28, 19, 255), portraitCol(226, 208, 164, 255), portraitCol(236, 222, 182, 255), portraitCol(226, 208, 164, 255), 1}
        default: // locked
                return treeStyleR6{portraitCol(19, 17, 14, 255), portraitCol(108, 101, 87, 255), portraitCol(132, 124, 108, 255), portraitCol(124, 116, 100, 255), 0}
        }
}

// r6segState classifies a connector between two nodes.
func r6segEnergized(a, b string) bool {
        return (a == "learned" || a == "maxed") && (b == "learned" || b == "maxed")
}
func r6segLocked(a, b string) bool {
        return a == "locked" || b == "locked"
}

// r6quadLink - quadratic-bezier connector with casing + core + midpoint
// diamond when energized.
func r6quadLink(dc *gg.Context, x0, y0, x2, y2, bias float64, energized, locked bool) {
        cx := (x0+x2)/2 + bias
        cy := (y0+y2)/2 + 26
        switch {
        case energized:
                dc.SetColor(portraitCol(0, 0, 0, 120))
                dc.SetLineWidth(6)
        case locked:
                dc.SetColor(portraitCol(120, 112, 96, 70))
                dc.SetLineWidth(1.5)
                dc.SetDash(5, 7)
        default:
                dc.SetColor(portraitCol(152, 140, 114, 100))
                dc.SetLineWidth(2)
        }
        dc.MoveTo(x0, y0)
        dc.QuadraticTo(cx, cy, x2, y2)
        dc.Stroke()
        if locked {
                dc.SetDash()
        }
        if energized {
                dc.SetColor(portraitCol(220, 178, 94, 240))
                dc.SetLineWidth(2.5)
                dc.MoveTo(x0, y0)
                dc.QuadraticTo(cx, cy, x2, y2)
                dc.Stroke()
                // midpoint diamond (bezier point at t=0.5)
                mx := 0.25*x0 + 0.5*cx + 0.25*x2
                my := 0.25*y0 + 0.5*cy + 0.25*y2
                dc.MoveTo(mx, my-5)
                dc.LineTo(mx+4.5, my)
                dc.LineTo(mx, my+5)
                dc.LineTo(mx-4.5, my)
                dc.ClosePath()
                dc.SetColor(portraitCol(242, 206, 122, 255))
                dc.Fill()
        }
}

// r6medallion - a skill node on the dark panel.
func r6medallion(dc *gg.Context, x, y, r float64, st treeStyleR6, initials, progress string) {
        switch st.glow {
        case 2:
                for _, g := range []struct {
                        dr float64
                        a  uint8
                }{{4, 60}, {9, 30}, {15, 13}} {
                        dc.SetColor(portraitCol(226, 182, 96, g.a))
                        dc.DrawCircle(x, y, r+g.dr)
                        dc.Fill()
                }
        case 1:
                dc.SetColor(portraitCol(214, 176, 100, 30))
                dc.DrawCircle(x, y, r+4)
                dc.Fill()
        }
        dc.SetColor(st.fill)
        dc.DrawCircle(x, y, r)
        dc.Fill()
        dc.SetColor(st.ring)
        lw := 2.0
        if st.glow == 2 {
                lw = 3.5
        }
        dc.SetLineWidth(lw)
        dc.DrawCircle(x, y, r)
        dc.Stroke()
        dc.SetColor(portraitCol(0, 0, 0, 90))
        dc.SetLineWidth(1)
        dc.DrawCircle(x, y, r-4)
        dc.Stroke()

        if progress != "" {
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 25, initials, r*1.5, 11)
                dc.SetColor(st.ink)
                dc.DrawStringAnchored(initials, x, y-8, 0.5, 0.5)
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, progress, r*1.35, 8)
                dc.DrawStringAnchored(progress, x, y+15, 0.5, 0.5)
        } else {
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 28, initials, r*1.5, 12)
                dc.SetColor(st.ink)
                dc.DrawStringAnchored(initials, x, y, 0.5, 0.5)
        }
}

func renderSkillTreeCard(c *gin.Context, req *portraitRequest) {
        // R6-2026-09-14 (owner: "current one is trash … find a style/layout
        // that actually fits an RPG skill-tree"). Rooted constellation tree on a
        // dark panel: class root at the bottom, branch spines fan upward, links
        // glow along the learned path. Canvas grew to 1200x1000 (bg_TREE2) so
        // four tier rows + branch chips + root + legend all breathe.
        dc := gg.NewContext(1200, 1000)
        if bgImg, err := utils.LoadImage(portraitAsset("bg_TREE2.png")); err == nil {
                dc.DrawImage(bgImg, 0, 0)
        } else {
                // fallback: wood family painted procedurally
                dc.SetRGB(0.086, 0.058, 0.035)
                dc.DrawRectangle(0, 0, 1200, 1000)
                dc.Fill()
                dc.SetColor(portraitCol(214, 170, 82, 255))
                dc.SetLineWidth(4)
                dc.DrawRoundedRectangle(14, 14, 1172, 972, 14)
                dc.Stroke()
                dc.SetColor(portraitCol(232, 214, 170, 245))
                dc.DrawRoundedRectangle(340, 48, 520, 64, 12)
                dc.Fill()
        }

        // class plate (baked leather plate 85,120-455,174)
        classLine := portraitSanitize(req.ClassName)
        if classLine == "" {
                classLine = "Adventurer"
        }
        if req.Level > 0 {
                classLine = fmt.Sprintf("%s - LV %d", classLine, req.Level)
        }
        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 26, classLine, 330, 12)
        dc.SetRGB(214.0 / 255.0, 170.0 / 255.0, 82.0 / 255.0)
        dc.DrawStringAnchored(classLine, 115, 147, 0, 0.5)

        // skill points pill (top-right, on the wood)
        pts := req.SkillPoints
        ptsText := "NO POINTS"
        if pts == 1 {
                ptsText = "1 POINT"
        } else if pts > 1 {
                ptsText = fmt.Sprintf("%d POINTS", pts)
        }
        darkPill(dc, ptsText, 1130, 147, true)

        // ── dark constellation panel over the parchment inset ──
        dc.SetColor(portraitCol(17, 13, 9, 255))
        dc.DrawRoundedRectangle(54, 200, 1092, 708, 18)
        dc.Fill()
        // depth washes
        dc.SetColor(portraitCol(214, 170, 82, 8))
        dc.DrawCircle(250, 380, 330)
        dc.Fill()
        dc.SetColor(portraitCol(214, 170, 82, 7))
        dc.DrawCircle(950, 730, 360)
        dc.Fill()
        dc.SetColor(portraitCol(170, 130, 60, 80))
        dc.SetLineWidth(1.5)
        dc.DrawRoundedRectangle(66, 212, 1068, 684, 14)
        dc.Stroke()

        n := len(req.Branches)
        type r6node struct {
                node    portraitNode
                x, y    float64
                br      int
                tier    int
                slot    int
                count   int
                apex    float64
        }
        if n > 0 {
                // ── tier geometry ──
                maxTier := 1
                for _, br := range req.Branches {
                        for _, s := range br.Skills {
                                if s.Tier > maxTier {
                                        maxTier = s.Tier
                                }
                        }
                }
                if maxTier > 5 {
                        maxTier = 5
                }
                tierBottom, tierTop := 762.0, 340.0
                step := 150.0
                if maxTier > 1 {
                        step = (tierBottom - tierTop) / float64(maxTier-1)
                        if step < 96 {
                                step = 96
                        }
                        if step > 150 {
                                step = 150
                        }
                }
                yFor := func(tier int) float64 {
                        t := tier
                        if t < 1 {
                                t = 1
                        }
                        if t > maxTier {
                                t = maxTier
                        }
                        return tierBottom - float64(t-1)*step
                }

                // ── branch spread ──
                spreadTop := 300.0
                if n > 3 {
                        spreadTop = 620.0 / float64(n-1)
                        if spreadTop > 300 {
                                spreadTop = 300
                        }
                }
                if n == 1 {
                        spreadTop = 0
                }
                apexOf := func(i int) float64 {
                        theta := (float64(i) - float64(n-1)/2) / 1.0
                        if n == 1 {
                                return 600
                        }
                        return 600 + theta*spreadTop
                }
                baseSpread := 160.0
                yTopRow := yFor(maxTier)

                // ── positions ──
                allNodes := []r6node{}
                branchNodes := make([][]r6node, n)
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
                        apex := apexOf(i)
                        theta := 0.0
                        if n > 1 {
                                theta = (float64(i) - float64(n-1)/2)
                        }
                        nodes := []r6node{}
                        for _, t := range order {
                                cnt := len(byTier[t])
                                for j, s := range byTier[t] {
                                        sf := 0.0
                                        if tierBottom > yTopRow {
                                                        sf = (tierBottom - yFor(t)) / (tierBottom - yTopRow)
                                        }
                                        if sf < 0 {
                                                sf = 0
                                        }
                                        if sf > 1 {
                                                sf = 1
                                        }
                                        sp := baseSpread + (spreadTop-baseSpread)*sf
                                        x := 600 + theta*sp + (float64(j)-float64(cnt-1)/2)*52
                                        if x < 96 {
                                                x = 96
                                        }
                                        if x > 1104 {
                                                x = 1104
                                        }
                                        nodes = append(nodes, r6node{node: s, x: x, y: yFor(t), br: i, tier: t, slot: j, count: cnt, apex: apex})
                                }
                        }
                        branchNodes[i] = nodes
                        allNodes = append(allNodes, nodes...)
                }

                // ── star field (deterministic; keeps clear of nodes) ──
                seed := uint32(20260914)
                rnd := func() float64 {
                        seed = seed*1664525 + 1013904223
                        return float64(seed>>8&0xFFFFFF) / float64(0xFFFFFF)
                }
                for d := 0; d < 60; d++ {
                        dx := 92 + rnd()*1016
                        dy := 292 + rnd()*580
                        clear := true
                        for _, nd := range allNodes {
                                ndx, ndy := nd.x-dx, nd.y-dy
                                if ndx*ndx+ndy*ndy < 62*62 {
                                        clear = false
                                        break
                                }
                        }
                        if clear {
                                rdx, rdy := 600-dx, 848-dy
                                if rdx*rdx+rdy*rdy < 72*72 {
                                        clear = false
                                }
                        }
                        if clear {
                                dc.SetColor(portraitCol(226, 190, 120, uint8(18+rnd()*22)))
                                dc.DrawCircle(dx, dy, 0.8+rnd()*1.1)
                                dc.Fill()
                        }
                }

                // ── tier rails + T badges ──
                for t := 1; t <= maxTier; t++ {
                        yy := yFor(t)
                        dc.SetColor(portraitCol(214, 170, 82, 26))
                        dc.SetLineWidth(1)
                        dc.DrawLine(84, yy, 1116, yy)
                        dc.Stroke()
                        badge := fmt.Sprintf("T%d", t)
                        dc.SetColor(portraitCol(24, 18, 10, 235))
                        dc.DrawRoundedRectangle(62, yy-11, 38, 22, 7)
                        dc.Fill()
                        dc.SetColor(portraitCol(170, 130, 60, 220))
                        dc.SetLineWidth(1)
                        dc.DrawRoundedRectangle(62, yy-11, 38, 22, 7)
                        dc.Stroke()
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, badge, 30, 8)
                        dc.SetRGB(244.0 / 255.0, 214.0 / 255.0, 140.0 / 255.0)
                        dc.DrawStringAnchored(badge, 81, yy, 0.5, 0.5)
                }

                // ── branch header chips (top of each spine) ──
                chipW := 264.0
                if n == 1 {
                        chipW = 380.0
                }
                for i, br := range req.Branches {
                        hname := portraitSanitize(br.Name)
                        if hname == "" {
                                hname = "Path"
                        }
                        apex := apexOf(i)
                        dc.SetColor(portraitCol(232, 214, 170, 242))
                        dc.DrawRoundedRectangle(apex-chipW/2, 226, chipW, 40, 10)
                        dc.Fill()
                        dc.SetColor(portraitCol(170, 130, 60, 255))
                        dc.SetLineWidth(2)
                        dc.DrawRoundedRectangle(apex-chipW/2, 226, chipW, 40, 10)
                        dc.Stroke()
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, hname, chipW-36, 11)
                        dc.SetRGB(52.0 / 255.0, 32.0 / 255.0, 16.0 / 255.0)
                        dc.DrawStringAnchored(hname, apex, 246, 0.5, 0.5)
                }

                // ── connectors: root → tier1 → … per branch, siblings chained ──
                for i := range req.Branches {
                        nodes := branchNodes[i]
                        if len(nodes) == 0 {
                                continue
                        }
                        theta := 0.0
                        if n > 1 {
                                theta = (float64(i) - float64(n-1)/2)
                        }
                        bias := theta * 26
                        prevX, prevY, prevSt := 600.0, 848.0, "learned" // class root
                        byTier := map[int][]r6node{}
                        for _, nd := range nodes {
                                byTier[nd.tier] = append(byTier[nd.tier], nd)
                        }
                        tiers := []int{}
                        for _, nd := range nodes {
                                found := false
                                for _, t := range tiers {
                                        if t == nd.tier {
                                                found = true
                                        }
                                }
                                if !found {
                                        tiers = append(tiers, nd.tier)
                                }
                        }
                        for _, t := range tiers {
                                row := byTier[t]
                                first := row[0]
                                r6quadLink(dc, prevX, prevY, first.x, first.y, bias,
                                        r6segEnergized(prevSt, first.node.State), r6segLocked(prevSt, first.node.State))
                                for j := 1; j < len(row); j++ {
                                        r6quadLink(dc, row[j-1].x, row[j-1].y, row[j].x, row[j].y, bias,
                                                r6segEnergized(row[j-1].node.State, row[j].node.State),
                                                r6segLocked(row[j-1].node.State, row[j].node.State))
                                }
                                prevX, prevY, prevSt = first.x, first.y, first.node.State
                        }
                }

                // ── class root medallion ──
                rootInitial := initialLetters(classLine)
                if len(rootInitial) > 1 {
                        rootInitial = rootInitial[:1]
                }
                r6medallion(dc, 600, 848, 24, treeStyleForR6("learned"), rootInitial, "")

                // ── nodes + names ──
                for i := range req.Branches {
                        for _, nd := range branchNodes[i] {
                                st := treeStyleForR6(nd.node.State)
                                initials := initialLetters(nd.node.Name)
                                progress := ""
                                if nd.node.Cur > 0 {
                                        progress = fmt.Sprintf("%d/%d", nd.node.Cur, nd.node.Max)
                                }
                                r6medallion(dc, nd.x, nd.y, 27, st, initials, progress)
                                nname := portraitSanitize(nd.node.Name)
                                nameW := 156.0
                                if nd.count > 1 {
                                        nameW = 116.0
                                }
                                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, nname, nameW, 9)
                                dc.SetColor(st.name)
                                dc.DrawStringAnchored(nname, nd.x, nd.y+27+16, 0.5, 0.5)
                        }
                }
        } else {
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 22, "NO SKILL TREE DATA", 400, 12)
                dc.SetColor(portraitCol(226, 208, 164, 255))
                dc.DrawStringAnchored("NO SKILL TREE DATA", 600, 550, 0.5, 0.5)
        }

        // ── legend (bottom-left of the panel; root sits centre, no collision) ──
        legend := []struct {
                label string
                st    treeStyleR6
        }{
                {"MAXED", treeStyleForR6("maxed")},
                {"LEARNED", treeStyleForR6("learned")},
                {"AVAILABLE", treeStyleForR6("open")},
                {"LOCKED", treeStyleForR6("locked")},
        }
        lx := 88.0
        for _, lg := range legend {
                dc.SetColor(lg.st.fill)
                dc.DrawCircle(lx, 872, 7)
                dc.Fill()
                dc.SetColor(lg.st.ring)
                dc.SetLineWidth(2)
                dc.DrawCircle(lx, 872, 7)
                dc.Stroke()
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, lg.label, 90, 8)
                dc.SetColor(portraitCol(226, 208, 164, 255))
                dc.DrawStringAnchored(lg.label, lx+13, 872, 0, 0.5)
                lx += 13 + float64(len(lg.label))*7.4 + 26
        }

        sealAt(dc, portraitSanitize(req.SealText), 70, 952, 24)
        cap := req.Caption
        if cap == "" {
                cap = "spend points with .skill up <name>"
        }
        captionAt(dc, portraitSanitize(cap), 640, 954, 320)

        utils.RespondImage(c, dc.Image())
}

// ═══════════════════════════════════════════════════════════════════════
// EQUIP R6-1500x1000 landscape armory wall
// ═══════════════════════════════════════════════════════════════════════

func renderEquipCard(c *gin.Context, req *portraitRequest) {
        dc := gg.NewContext(1500, 1000)
        // phase 8: themed armory board (style 0/7 = decree baked look)
        _eqTh := resolveTheme(req.Style)
        if _eqTh != nil {
                paintRpgBoardThemed(dc, 1500, 1000, "THE ARMORY", req.Nickname, _eqTh)
        } else {
                paintRpgBoard(dc, 1500, 1000, "THE ARMORY", req.Nickname)
        }

        slots := req.Slots
        filled, tierSum, damaged := 0, 0, 0
        for _, s := range slots {
                if !s.Empty {
                        filled++
                        tierSum += s.Tier
                        if s.DurMax > 0 && s.Dur/s.DurMax <= 0.35 {
                                damaged++
                        }
                }
        }
        avgTier := 0.0
        if filled > 0 {
                avgTier = float64(tierSum) / float64(filled)
        }

        // ── LEFT hero panel ── (phase 8: themed plate + accent)
        if _eqTh != nil {
                dc.SetColor(darken(_eqTh.Plate, 20))
                dc.DrawRoundedRectangle(60, 200, 410, 730, 16)
                dc.Fill()
                dc.SetColor(_eqTh.Gold)
                dc.SetLineWidth(2.5)
                dc.DrawRoundedRectangle(60, 200, 410, 730, 16)
                dc.Stroke()
        } else {
                dc.SetColor(portraitCol(22, 15, 9, 252))
                dc.DrawRoundedRectangle(60, 200, 410, 730, 16)
                dc.Fill()
                dc.SetColor(portraitCol(214, 170, 82, 255))
                dc.SetLineWidth(2.5)
                dc.DrawRoundedRectangle(60, 200, 410, 730, 16)
                dc.Stroke()
        }
        dc.SetColor(portraitCol(0, 0, 0, 80))
        dc.SetLineWidth(1)
        dc.DrawRoundedRectangle(70, 210, 390, 710, 12)
        dc.Stroke()

        classLabel := strings.ToUpper(portraitSanitize(req.PlayerClass))
        if classLabel == "" {
                classLabel = "ADVENTURER"
        }
        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 21, classLabel, 350, 12)
        dc.SetRGB(214.0 / 255.0, 170.0 / 255.0, 82.0 / 255.0)
        dc.DrawStringAnchored(classLabel, 265, 236, 0.5, 0.5)

        // stage shadow + class sprite (same asset the QUESTSTART card uses)
        utils.DrawShadow(dc, 265, 612, 110, 0.55)
        spriteOK := false
        if pImg := portraitFighterSprite(req.PlayerClass, req.PlayerIndex, 250, 330, "RIGHT"); pImg != nil {
                b := pImg.Bounds()
                dc.DrawImage(pImg, 265-b.Dx()/2, 608-b.Dy())
                spriteOK = true
        }
        if !spriteOK {
                initialMedallion(dc, 265, 430, 72, req.Nickname, portraitCol(214, 170, 82, 255))
        }

        portraitPillCentred(dc, fmt.Sprintf("SLOTS %d/%d", filled, len(slots)), 265, 668,
                portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
        portraitPillCentred(dc, fmt.Sprintf("AVG TIER %.1f", avgTier), 265, 712,
                portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
        if damaged > 0 {
                portraitPillCentred(dc, fmt.Sprintf("%d DAMAGED", damaged), 265, 756,
                        portraitCol(96, 18, 28, 185), portraitCol(250, 210, 120, 255))
        } else {
                portraitPillCentred(dc, "ALL STURDY", 265, 756,
                        portraitCol(26, 46, 22, 185), portraitCol(196, 224, 156, 255))
        }
        portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 16, "repair at .j blacksmith", 330, 10)
        dc.SetRGB(176.0 / 255.0, 140.0 / 255.0, 96.0 / 255.0)
        dc.DrawStringAnchored("repair at .j blacksmith", 265, 806, 0.5, 0.5)

        // ── RIGHT gear rack - one wide strip per slot ──
        rowH, gap := 78.0, 2.5
        y := 202.0
        for i, s := range slots {
                if i >= 9 {
                        break
                }
                // strip
                if s.Empty {
                        dc.SetColor(portraitCol(232, 214, 170, 110))
                        dc.DrawRoundedRectangle(500, y, 940, rowH, 10)
                        dc.Fill()
                        dc.SetColor(portraitCol(170, 130, 60, 110))
                        dc.SetLineWidth(1.5)
                        dc.DrawRoundedRectangle(500, y, 940, rowH, 10)
                        dc.Stroke()
                } else {
                        if _eqTh != nil {
                                dc.SetColor(alphaN(_eqTh.Panel, 238))
                        } else {
                                dc.SetColor(portraitCol(232, 214, 170, 238))
                        }
                        dc.DrawRoundedRectangle(500, y, 940, rowH, 10)
                        dc.Fill()
                        if _eqTh != nil {
                                dc.SetColor(_eqTh.Gold)
                        } else {
                                dc.SetColor(portraitCol(170, 130, 60, 255))
                        }
                        dc.SetLineWidth(2)
                        dc.DrawRoundedRectangle(500, y, 940, rowH, 10)
                        dc.Stroke()
                }

                slotLabel := portraitSanitize(s.Slot)
                if s.Empty {
                        // dashed medallion
                        dc.SetColor(portraitCol(58, 52, 42, 180))
                        dc.DrawCircle(546, y+rowH/2, 26)
                        dc.Fill()
                        dc.SetColor(portraitCol(150, 140, 120, 170))
                        dc.SetLineWidth(2)
                        dc.SetDash(4, 5)
                        dc.DrawCircle(546, y+rowH/2, 26)
                        dc.Stroke()
                        dc.SetDash()
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 12, slotLabel, 130, 8)
                        dc.SetRGB(140.0 / 255.0, 120.0 / 255.0, 90.0 / 255.0)
                        dc.DrawStringAnchored(slotLabel, 590, y+19, 0, 0.5)
                        emptyTxt := "NOT EQUIPPED - use .j equip"
                        portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 15, emptyTxt, 400, 10)
                        dc.SetRGB(140.0 / 255.0, 120.0 / 255.0, 90.0 / 255.0)
                        dc.DrawStringAnchored(emptyTxt, 590, y+44, 0, 0.5)
                        y += rowH + gap
                        continue
                }

                // tier-ringed medallion with slot initials
                dc.SetColor(portraitCol(24, 16, 10, 235))
                dc.DrawCircle(546, y+rowH/2, 26)
                dc.Fill()
                dc.SetColor(tierPillCol(s.Tier))
                dc.SetLineWidth(3)
                dc.DrawCircle(546, y+rowH/2, 26)
                dc.Stroke()
                dc.SetColor(portraitCol(0, 0, 0, 90))
                dc.SetLineWidth(1)
                dc.DrawCircle(546, y+rowH/2, 22)
                dc.Stroke()
                init := initialLetters(slotLabel)
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, init, 40, 8)
                dc.SetRGB(244.0 / 255.0, 214.0 / 255.0, 140.0 / 255.0)
                dc.DrawStringAnchored(init, 546, y+rowH/2-1, 0.5, 0.5)

                // slot label + item name
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 12, slotLabel, 200, 8)
                dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
                dc.DrawStringAnchored(slotLabel, 590, y+16, 0, 0.5)
                nameTxt := portraitSanitize(s.Name)
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, nameTxt, 500, 10)
                dc.SetRGB(52.0 / 255.0, 32.0 / 255.0, 16.0 / 255.0)
                dc.DrawStringAnchored(nameTxt, 590, y+40, 0, 0.5)

                // tier pill (top-right)
                tierPill := fmt.Sprintf("T%d", s.Tier)
                if s.TierLabel != "" {
                        tierPill = fmt.Sprintf("T%d %s", s.Tier, portraitSanitize(s.TierLabel))
                }
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, tierPill, 150, 8)
                tw, _ := dc.MeasureString(tierPill)
                pillW := tw + 24
                if pillW < 64 {
                        pillW = 64
                }
                dc.SetColor(portraitCol(24, 16, 10, 215))
                dc.DrawRoundedRectangle(1424-pillW, y+14, pillW, 24, 11)
                dc.Fill()
                dc.SetColor(tierPillCol(s.Tier))
                dc.DrawStringAnchored(tierPill, 1424-pillW/2, y+26, 0.5, 0.5)

                // durability bar + numbers
                if s.DurMax > 0 {
                        pct := s.Dur / s.DurMax
                        if pct < 0 {
                                pct = 0
                        }
                        if pct > 1 {
                                pct = 1
                        }
                        barX, barY, barW, barH := 590.0, y+58, 380.0, 11.0
                        dc.SetColor(portraitCol(54, 36, 18, 255))
                        dc.DrawRoundedRectangle(barX, barY, barW, barH, 6)
                        dc.Fill()
                        fillW := barW * pct
                        barCol := portraitCol(110, 160, 80, 255)
                        if pct <= 0.15 {
                                barCol = portraitCol(200, 60, 60, 255)
                        } else if pct <= 0.35 {
                                barCol = portraitCol(214, 170, 82, 255)
                        }
                        if fillW > 3 {
                                dc.SetColor(barCol)
                                dc.DrawRoundedRectangle(barX, barY, fillW, barH, 6)
                                dc.Fill()
                                dc.SetColor(portraitCol(255, 255, 255, 34))
                                dc.DrawRoundedRectangle(barX, barY, fillW, 5, 5)
                                dc.Fill()
                        }
                        dc.SetColor(portraitCol(170, 130, 60, 255))
                        dc.SetLineWidth(1.5)
                        dc.DrawRoundedRectangle(barX, barY, barW, barH, 6)
                        dc.Stroke()

                        durTxt := fmt.Sprintf("%s/%s", trimFloat(s.Dur), trimFloat(s.DurMax))
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, durTxt, 90, 8)
                        dc.SetRGB(52.0 / 255.0, 32.0 / 255.0, 16.0 / 255.0)
                        dc.DrawStringAnchored(durTxt, 978, y+63, 0, 0.5)

                        pctTxt := fmt.Sprintf("%d%%", int(pct*100+0.5))
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, pctTxt, 70, 8)
                        dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
                        dc.DrawStringAnchored(pctTxt, 1424, y+63, 1, 0.5)
                } else {
                        noDur := "NO WEAR"
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 12, noDur, 120, 8)
                        dc.SetRGB(150.0 / 255.0, 118.0 / 255.0, 60.0 / 255.0)
                        dc.DrawStringAnchored(noDur, 590, y+63, 0, 0.5)
                }

                y += rowH + gap
        }

        sealAt(dc, portraitSanitize(req.SealText), 70, 962, 24)
        cap := req.Caption
        if cap == "" {
                cap = "repair at the blacksmith"
        }
        captionAt(dc, portraitSanitize(cap), 780, 946, 320)

        utils.RespondImage(c, dc.Image())
}

// ═══════════════════════════════════════════════════════════════════════
// ABILITIES R6-1000x1400 arc-scroll grimoire
// ═══════════════════════════════════════════════════════════════════════

// r6rod - wooden scroll rod with gold end caps.
func r6rod(dc *gg.Context, cy float64) {
        dc.SetColor(portraitCol(86, 54, 28, 255))
        dc.DrawRoundedRectangle(40, cy-28, 920, 56, 26)
        dc.Fill()
        dc.SetColor(portraitCol(126, 86, 48, 130))
        dc.DrawRoundedRectangle(52, cy-22, 896, 12, 6)
        dc.Fill()
        dc.SetColor(portraitCol(44, 27, 14, 150))
        dc.DrawRoundedRectangle(52, cy+8, 896, 12, 6)
        dc.Fill()
        for _, cx := range []float64{54, 946} {
                dc.SetColor(portraitCol(58, 36, 18, 255))
                dc.DrawCircle(cx, cy, 36)
                dc.Fill()
                dc.SetColor(portraitCol(214, 170, 82, 255))
                dc.SetLineWidth(3)
                dc.DrawCircle(cx, cy, 36)
                dc.Stroke()
                dc.SetColor(portraitCol(214, 170, 82, 255))
                dc.DrawCircle(cx, cy, 9)
                dc.Fill()
        }
}

// r6parseLv - pull "Lv.4/5" off the front of a sub line; returns cur, max
// and the remainder ("· ⚡18 · CD2" with the separator trimmed).
func r6parseLv(sub string) (int, int, string) {
        if !strings.HasPrefix(sub, "Lv.") {
                return 0, 0, sub
        }
        rest := sub[3:]
        parts := strings.SplitN(rest, " ", 2)
        cur, mx := 0, 0
        fmt.Sscanf(parts[0], "%d/%d", &cur, &mx)
        tail := ""
        if len(parts) > 1 {
                tail = parts[1]
                tail = strings.TrimPrefix(tail, "· ")
                tail = strings.TrimSpace(tail)
        }
        return cur, mx, tail
}

// r6diamond - small rotated-square ornament.
func r6diamond(dc *gg.Context, x, y, r float64, col color.NRGBA) {
        dc.MoveTo(x, y-r)
        dc.LineTo(x+r, y)
        dc.LineTo(x, y+r)
        dc.LineTo(x-r, y)
        dc.ClosePath()
        dc.SetColor(col)
        dc.Fill()
}

func renderAbilitiesCard(c *gin.Context, req *portraitRequest) {
        // QA r2 (owner: "fix the alignment issues with all the cards"):
        // the scroll was a fixed 1000x1400 - a 4-ability grimoire left
        // ~70% dead parchment. Height now follows the content.
        natH := 246.0
        if strings.TrimSpace(req.DocQuote) != "" {
                natH += 26
        }
        for gi, g := range req.Groups {
                if gi >= 6 {
                        break
                }
                natH += 46 + float64(len(g.Items))*58 + 10
        }
        H := natH + 170.0
        if H < 700 {
                H = 700
        }
        if H > 1400 {
                H = 1400
        }
        dc := gg.NewContext(1000, int(H))

        // wood backdrop
        dc.SetRGB(0.075, 0.05, 0.03)
        dc.DrawRectangle(0, 0, 1000, H)
        dc.Fill()
        dc.SetColor(portraitCol(255, 220, 160, 10))
        for wy := 60.0; wy < H; wy += 60 {
                dc.SetLineWidth(1)
                dc.DrawLine(0, wy, 1000, wy)
                dc.Stroke()
        }

        // parchment body between the rods
        dc.SetColor(portraitCol(233, 215, 171, 255))
        dc.DrawRectangle(70, 92, 860, H-184)
        dc.Fill()
        // edge shading
        dc.SetColor(portraitCol(90, 70, 40, 26))
        dc.DrawRectangle(70, 92, 14, H-184)
        dc.Fill()
        dc.DrawRectangle(916, 92, 14, H-184)
        dc.Fill()
        // fiber lines
        dc.SetColor(portraitCol(90, 70, 40, 7))
        for fy := 118.0; fy < H-100; fy += 26 {
                dc.SetLineWidth(1)
                dc.DrawLine(84, fy, 916, fy)
                dc.Stroke()
        }
        // gold hairline frame
        dc.SetColor(portraitCol(170, 130, 60, 150))
        dc.SetLineWidth(1.5)
        dc.DrawRoundedRectangle(82, 102, 836, H-194, 6)
        dc.Stroke()

        // rods (over the parchment edges)
        r6rod(dc, 64)
        r6rod(dc, H-64)

        // ── header ──
        total := 0
        for _, g := range req.Groups {
                total += len(g.Items)
        }
        title := strings.TrimSpace(req.DocTitle)
        if title == "" {
                title = "ABILITY GRIMOIRE"
        }
        portraitFitText(dc, portraitAsset("CinzelDecBold.ttf"), 30, title, 620, 20)
        dc.SetRGB(52.0 / 255.0, 32.0 / 255.0, 16.0 / 255.0)
        dc.DrawStringAnchored(title, 500, 148, 0.5, 0.5)
        tw, _ := dc.MeasureString(title)
        dc.SetColor(portraitCol(170, 130, 60, 255))
        r6diamond(dc, 500-tw/2-34, 146, 6, portraitCol(170, 130, 60, 255))
        r6diamond(dc, 500+tw/2+34, 146, 6, portraitCol(170, 130, 60, 255))

        if pageLabel := strings.TrimSpace(req.PageLabel); pageLabel != "" {
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, pageLabel, 150, 10)
                dc.SetColor(portraitCol(170, 130, 60, 230))
                dc.DrawStringAnchored(pageLabel, 916, 118, 1, 0.5)
        }

        classLine := strings.ToUpper(portraitSanitize(req.ClassName))
        if classLine == "" {
                classLine = "ADVENTURER"
        }
        sub := fmt.Sprintf("%s", classLine)
        if req.Level > 0 {
                sub = fmt.Sprintf("%s · LV %d", sub, req.Level)
        }
        sub = fmt.Sprintf("%s · %d ABILITIES", sub, total)
        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 17, sub, 700, 10)
        dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
        dc.DrawStringAnchored(sub, 500, 190, 0.5, 0.5)

        quoteY := 0.0
        if quote := strings.TrimSpace(req.DocQuote); quote != "" {
                quoteY = 1
                q := portraitSanitize(quote)
                portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 17, q, 780, 11)
                dc.SetRGB(0.47, 0.34, 0.16)
                dc.DrawStringAnchored(q, 500, 212, 0.5, 0.5)
        }

        divY := 214.0
        groupStart := 246.0
        if quoteY == 1 {
                divY = 240.0
                groupStart = 272.0
        }
        dc.SetColor(portraitCol(170, 130, 60, 190))
        dc.SetLineWidth(1.5)
        dc.DrawLine(150, divY, 850, divY)
        dc.Stroke()
        dc.SetLineWidth(1)
        dc.DrawLine(150, divY+5, 850, divY+5)
        dc.Stroke()
        r6diamond(dc, 500, divY+2, 4.5, portraitCol(170, 130, 60, 220))

        // ── groups ──
        y := groupStart
        abilityNo := 0
        if req.StartNumber > 0 {
                abilityNo = req.StartNumber - 1
        }
        for gi, g := range req.Groups {
                if gi >= 6 || y > H-220 {
                        break
                }
                current := strings.EqualFold(portraitSanitize(g.Sub), "CURRENT")
                gname := portraitSanitize(g.Name)
                if gname == "" {
                        gname = "SKILLS"
                }
                if g.Sub != "" {
                        gname = fmt.Sprintf("%s · %s", gname, portraitSanitize(g.Sub))
                }
                // swallowtail ribbon
                ribH := 34.0
                ribW := 470.0
                dc.MoveTo(128, y)
                dc.LineTo(128+ribW, y)
                dc.LineTo(128+ribW, y+ribH)
                dc.LineTo(128, y+ribH)
                dc.LineTo(96, y+ribH/2)
                dc.ClosePath()
                if current {
                        dc.SetColor(portraitCol(96, 62, 24, 245))
                } else {
                        dc.SetColor(portraitCol(52, 40, 24, 238))
                }
                dc.Fill()
                dc.SetColor(portraitCol(170, 130, 60, 210))
                dc.SetLineWidth(1.5)
                dc.MoveTo(128, y)
                dc.LineTo(128+ribW, y)
                dc.LineTo(128+ribW, y+ribH)
                dc.LineTo(128, y+ribH)
                dc.LineTo(96, y+ribH/2)
                dc.ClosePath()
                dc.Stroke()
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, gname, ribW-56, 10)
                if current {
                        dc.SetRGB(244.0 / 255.0, 214.0 / 255.0, 140.0 / 255.0)
                } else {
                        dc.SetColor(portraitCol(226, 208, 164, 255))
                }
                dc.DrawStringAnchored(gname, 158, y+ribH/2, 0, 0.5)
                y += ribH + 12

                for _, it := range g.Items {
                        if y > H-164 {
                                break
                        }
                        abilityNo++
                        // wax-seal number medallion
                        dc.SetColor(portraitCol(128, 28, 40, 242))
                        dc.DrawCircle(150, y+27, 17)
                        dc.Fill()
                        dc.SetColor(portraitCol(70, 12, 20, 255))
                        dc.SetLineWidth(2.5)
                        dc.DrawCircle(150, y+27, 17)
                        dc.Stroke()
                        dc.SetColor(portraitCol(220, 150, 90, 130))
                        dc.SetLineWidth(1)
                        dc.DrawCircle(150, y+27, 13)
                        dc.Stroke()
                        numTxt := fmt.Sprintf("%d", abilityNo)
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 15, numTxt, 26, 8)
                        dc.SetRGB(250.0 / 255.0, 210.0 / 255.0, 120.0 / 255.0)
                        dc.DrawStringAnchored(numTxt, 150, y+26, 0.5, 0.5)

                        // name + level pips
                        nameTxt := strings.TrimSpace(strings.TrimPrefix(portraitSanitize(it.Title), "🪞"))
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, nameTxt, 330, 9)
                        dc.SetRGB(52.0 / 255.0, 32.0 / 255.0, 16.0 / 255.0)
                        dc.DrawStringAnchored(nameTxt, 186, y+17, 0, 0.5)
                        nw, _ := dc.MeasureString(nameTxt)
                        cur, mx, tail := r6parseLv(portraitSanitize(it.Sub))
                        pipX := 186 + nw + 18
                        if mx > 0 && mx <= 8 {
                                for p := 0; p < mx; p++ {
                                        if p < cur {
                                                dc.SetColor(portraitCol(96, 62, 24, 255))
                                                dc.DrawCircle(pipX, y+17, 3.2)
                                                dc.Fill()
                                        } else {
                                                dc.SetColor(portraitCol(120, 88, 40, 140))
                                                dc.SetLineWidth(1.2)
                                                dc.DrawCircle(pipX, y+17, 3.2)
                                                dc.Stroke()
                                        }
                                        pipX += 11
                                }
                        }
                        // cost / CD line
                        tailX := 186.0
                        if tail != "" {
                                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, tail, 560, 8)
                                dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
                                dc.DrawStringAnchored(tail, 186, y+41, 0, 0.5)
                                if w, _ := dc.MeasureString(tail); w > 0 {
                                        tailX = 186 + w + 14
                                }
                        }
                        // effect runes (phases 4-6): DejaVu glyphs only - the
                        // Cinzel/MedievalSharp faces have no symbol coverage.
                        if it.Runes != "" {
                                if face, ferr := utils.LoadFont("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 16); ferr == nil {
                                        dc.SetFontFace(face)
                                        dc.SetRGB(0.58, 0.42, 0.15)
                                        dc.DrawStringAnchored(it.Runes, tailX, y+41, 0, 0.5)
                                }
                        }

                        // effect column (right)
                        val := portraitSanitize(it.Value)
                        if val != "" {
                                portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 15, val, 300, 9)
                                dc.SetRGB(96.0 / 255.0, 70.0 / 255.0, 30.0 / 255.0)
                                dc.DrawStringAnchored(val, 900, y+28, 1, 0.5)
                        }

                        y += 58
                }
                y += 10
        }
        if abilityNo == 0 {
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, "NO ABILITIES", 400, 10)
                dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
                dc.DrawStringAnchored("NO ABILITIES", 500, 600, 0.5, 0.5)
        }

        // ── footer on the parchment, above the bottom rod ──
        sealAt(dc, portraitSanitize(req.SealText), 112, H-138, 24)
        cap := req.Caption
        if cap == "" {
                cap = "cast with .combat ability <num>"
        }
        runes := []rune(cap)
        if len(runes) > 52 {
                cap = string(runes[:49]) + "..."
        }
        portraitFitText(dc, portraitAsset("MedievalSharp.ttf"), 18, cap, 620, 11)
        dc.SetRGB(120.0 / 255.0, 88.0 / 255.0, 40.0 / 255.0)
        dc.DrawStringAnchored(cap, 540, H-137, 0.5, 0.5)

        utils.RespondImage(c, dc.Image())
}
