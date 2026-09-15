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
//   RANK — 2026-09-14, owner: "make the .rank an image card and include
//     your level and xp left to progress". baked bg_RANK.png — SAME
//     geometry as QUEST/TRIAL (banner "ADVENTURER", panel (55,208)-(545,832),
//     dividers y266/y492, lower label "THE STANDING" (85,520)). Top section
//     (y266..y492) paints the RECORD: big LEVEL line, rank pill, XP bar
//     (track (85,402)-(515,428), gold fill by percent, quarter ticks) and
//     under-bar strings xpNow / xpLeft. Lower section paints Standing rows
//     (y556 step 64, max 4, label/value). seal = rank letter, caption.
//     Node caller: progressionCommands.handleRankCommand.
//
//   END (victory/defeat) — 2026-09-14, owner: "update the victory and
//     defeat image cards with the style". Replaces the old gradient
//     GenerateEndScreen. baked bg_VICTORY.png / bg_DEFEAT.png clone the
//     DUEL geometry: banner VICTORY/DEFEATED, plate (player name), scene
//     window (55,208)-(545,642) (arena + player sprite LEFT + enemy sprite
//     RIGHT — vivid winner / faded loser), corner pills VICTOR+SLAIN /
//     FALLEN+VICTORIOUS, spoils ledger y746 step 44 (victory: zeni/xp/
//     spoils/depth — defeat: slain-by/rank/recovered), seal = rank letter
//     (victory) or fallen-at level (defeat), caption. HTTP handler:
//     renderer.go GenerateEndScreen -> WriteEndCard (this file).
//
// NOTE: no gg Clip() anywhere — clip state leaks in this gg version and
// erases later draws (same bug as hunt/boss QA rounds).

import (
        "fmt"
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

// portraitRow — generic label/value ledger row.
type portraitRow struct {
        Label string `json:"label"`
        Value string `json:"value"`
        Sub   string `json:"sub"` // optional right-of-middle tag (ALLOCATE gain, e.g. "+2/pt")
}

type portraitPlayerRow struct {
        Name string `json:"name"`
        XP   string `json:"xp"`
        Zeni string `json:"zeni"`
}

// portraitNode — one skill node on the SKILLTREE card (r4).
type portraitNode struct {
        Name  string `json:"name"`
        Cur   int    `json:"cur"`
        Max   int    `json:"max"`
        State string `json:"state"` // locked | open | learned | maxed
        Tier  int    `json:"tier"`
}

type portraitBranch struct {
        Name   string         `json:"name"`
        Skills []portraitNode `json:"skills"`
}

type portraitBuilding struct {
        Name  string `json:"name"`
        Level int    `json:"level"`
}

// r5 kinds (2026-09-14): SHOP + EQUIP + ABILITIES payloads.
// portraitEntry — one generic row (shop item, ability line).
type portraitEntry struct {
        Title string `json:"title"`
        Icon  string `json:"icon"`
        Sub   string `json:"sub"`
        Value string `json:"value"`
}

// portraitSlot — one equipment plate on the EQUIP armory board.
type portraitSlot struct {
        Slot      string  `json:"slot"`
        Icon      string  `json:"icon"`
        Name      string  `json:"name"`
        TierLabel string  `json:"tierLabel"`
        Tier      int     `json:"tier"`
        Dur       float64 `json:"dur"`
        DurMax    float64 `json:"durMax"`
        Empty     bool    `json:"empty"`
}

// portraitProgress — one PROGRESSION bar on the RANK card (r6, 2026-09-14):
// a requirement to progress, rendered as a filled progress bar.
type portraitProgress struct {
        Label     string `json:"label"`
        Cur       int    `json:"cur"`
        Max       int    `json:"max"`
        Done      bool   `json:"done"`
        ValueText string `json:"valueText"`
}

// portraitGroup — a titled section of ability rows (ABILITIES grimoire).
type portraitGroup struct {
        Name  string          `json:"name"`
        Sub   string          `json:"sub"`
        Items []portraitEntry `json:"items"`
}

// portraitRequest — payload for /api/cards/portrait (all kinds).
type portraitRequest struct {
        Kind        string              `json:"kind"`
        Nickname    string              `json:"nickname"`
        Caption     string              `json:"caption"`
        SealText    string              `json:"sealText"`
        Forfeit     bool                `json:"forfeit"` // DUEL only: "BY FORFEIT" stamp
        Ledger      []portraitRow       `json:"ledger"`
        Players     []portraitPlayerRow `json:"players"`
        WinnerClass string              `json:"winnerClass"`
        WinnerIndex int                 `json:"winnerIndex"`
        LoserClass  string              `json:"loserClass"`
        LoserIndex  int                 `json:"loserIndex"`
        // RANK kind (2026-09-14):
        Level          int                `json:"level"`
        RankLetter     string             `json:"rankLetter"`
        RankLine       string             `json:"rankLine"`
        XPNow          string             `json:"xpNow"`
        XPLeft         string             `json:"xpLeft"`
        XPPercent      int                `json:"xpPercent"`
        Standing       []portraitRow      `json:"standing"`
        // r6 (2026-09-14): rank requirements as PROGRESS BARS
        ProgressTitle  string             `json:"progressTitle"`
        Progress       []portraitProgress `json:"progress"`
        // r4 kinds (2026-09-14): QUESTSTART/RAID + GUILDINFO + SKILLTREE + SKILLUP
        Rows        []portraitRow      `json:"rows"`
        Branches    []portraitBranch   `json:"branches"`
        ClassName   string             `json:"className"`
        SkillPoints int                `json:"skillPoints"`
        SkillName   string             `json:"skillName"`
        SkillMax    int                `json:"skillMax"`
        Cur         int                `json:"cur"`
        Tier        int                `json:"tier"`
        Ascended    bool               `json:"ascended"`
        HexColor    string             `json:"hexColor"`
        Motto       string             `json:"motto"`
        Buildings   []portraitBuilding `json:"buildings"`
        Background  string             `json:"background"` // env asset filename
        PlayerClass string             `json:"playerClass"`
        PlayerIndex int                `json:"playerIndex"`
        PartyText   string             `json:"partyText"`
        // ALLOCATE (2026-09-15, owner: "use the style/assets with different
        // orientation used across the entire rpg") — stat-point allocation as
        // a bg_ALLOCATE portrait family card (600x1000, same bake geometry).
        PointsBig    string `json:"pointsBig"`
        Pill         string `json:"pill"`
        SpentPercent int    `json:"spentPercent"`
        SpentNow     string `json:"spentNow"`
        SpentLeft    string `json:"spentLeft"`
        // r5 kinds (2026-09-14): SHOP / EQUIP / ABILITIES
        Entries []portraitEntry `json:"entries"`
        Slots   []portraitSlot  `json:"slots"`
        Groups  []portraitGroup `json:"groups"`
}

// GeneratePortraitCard renders DUEL / QUEST portrait event cards (600x1000)
// plus the r4 kinds (QUESTSTART/RAID 1000x600, GUILDINFO 800x800,
// SKILLTREE 1200x800, SKILLUP 600x1000 — see eventcards.go).
func GeneratePortraitCard(c *gin.Context) {
        var req portraitRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        // r4 kinds have their own canvases + arrangements
        switch req.Kind {
        case "QUESTSTART", "RAID":
                renderQuestStartCard(c, &req, req.Kind == "RAID")
                return
        case "GUILDINFO":
                renderGuildInfoCard(c, &req)
                return
        case "SKILLTREE":
                renderSkillTreeCard(c, &req)
                return
        case "SKILLUP":
                renderSkillUpCard(c, &req)
                return
        // r5 kinds (2026-09-14): SHOP + EQUIP + ABILITIES — see eventcards.go
        case "SHOP":
                renderShopCard(c, &req)
                return
        case "EQUIP":
                renderEquipCard(c, &req)
                return
        case "ABILITIES":
                renderAbilitiesCard(c, &req)
                return
        }

        dc := gg.NewContext(int(portraitW), int(portraitH))
        bgFile := "bg_DUEL.png"
        if req.Kind == "QUEST" {
                bgFile = "bg_QUEST.png"
        } else if req.Kind == "TRIAL" {
                bgFile = "bg_TRIAL.png"
        } else if req.Kind == "RANK" {
                bgFile = "bg_RANK.png"
        } else if req.Kind == "ALLOCATE" {
                bgFile = "bg_ALLOCATE.png"
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

        if req.Kind == "RANK" {
                // ── THE RECORD (top section, y266..y492): big level + rank + XP bar ──
                lvl := fmt.Sprintf("LEVEL %d", req.Level)
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 56, lvl, 380, 30)
                dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                dc.DrawStringAnchored(lvl, 300, 306, 0.5, 0.5)

                rankLine := req.RankLine
                if rankLine == "" && req.RankLetter != "" {
                        rankLine = req.RankLetter + "-RANK ADVENTURER"
                }
                rankLine = strings.ToUpper(portraitSanitize(rankLine))
                if rankLine != "" {
                        portraitPillCentred(dc, rankLine, 300, 371,
                                portraitCol(96, 62, 24, 235), portraitCol(244, 214, 140, 255))
                }

                // XP bar: dark track + gold fill by percent + quarter ticks
                pct := req.XPPercent
                if pct < 0 {
                        pct = 0
                }
                if pct > 100 {
                        pct = 100
                }
                dc.SetColor(portraitCol(54, 36, 18, 255))
                dc.DrawRoundedRectangle(85, 402, 430, 26, 10)
                dc.Fill()
                fillW := 430.0 * float64(pct) / 100.0
                if fillW > 430 {
                        fillW = 430
                }
                if fillW > 4 {
                        dc.SetColor(portraitCol(214, 170, 82, 255))
                        dc.DrawRoundedRectangle(85, 402, fillW, 26, 10)
                        dc.Fill()
                        dc.SetColor(portraitCol(240, 205, 120, 90))
                        dc.DrawRoundedRectangle(85, 402, fillW, 10, 8)
                        dc.Fill()
                }
                dc.SetColor(portraitCol(170, 130, 60, 255))
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(85, 402, 430, 26, 10)
                dc.Stroke()
                dc.SetColor(portraitCol(120, 88, 40, 130))
                dc.SetLineWidth(1)
                for _, t := range []float64{0.25, 0.5, 0.75} {
                        tx := 85 + 430*t
                        dc.DrawLine(tx, 405, tx, 425)
                        dc.Stroke()
                }

                // under-bar strings: progress (left) · xp left to progress (right)
                if req.XPNow != "" {
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, portraitSanitize(req.XPNow), 200, 10)
                        dc.SetRGB(84.0/255.0, 96.0/255.0, 44.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(req.XPNow), 85, 452, 0, 0.5)
                }
                if req.XPLeft != "" {
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, portraitSanitize(req.XPLeft), 240, 10)
                        dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(req.XPLeft), 515, 452, 1, 0.5)
                }

                // ── THE STANDING (lower section) — 3×2 compact grid ──
                // 2026-09-14 r6 (owner: "add the requirements back … as a
                // progress bar / progress section"): the 8 single-column rows
                // became a 3×2 grid (XP|GP, totals, commands|achievements),
                // freeing the lower panel for the PROGRESSION bars below.
                for row := 0; row < 3; row++ {
                        yy := 550.0 + float64(row)*30
                        for col := 0; col < 2; col++ {
                                idx := row*2 + col
                                if idx >= len(req.Standing) {
                                        continue
                                }
                                sr := req.Standing[idx]
                                lx, vx := 85.0, 302.0
                                if col == 1 {
                                        lx, vx = 320.0, 515.0
                                }
                                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, portraitSanitize(sr.Label), 118, 9)
                                dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
                                dc.DrawStringAnchored(portraitSanitize(sr.Label), lx, yy, 0, 0.5)
                                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, portraitSanitize(sr.Value), vx-lx-12, 9)
                                dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                                dc.DrawStringAnchored(portraitSanitize(sr.Value), vx, yy, 1, 0.5)
                        }
                }

                // ── PROGRESSION — the requirements to advance, as bars ──
                // Owner r6: "make the requirements into a progress bar /
                // progress section so it clearly shows what you've completed
                // and what you still need to progress". Level / quests /
                // gate-mission objectives, each a gold fill (green when
                // complete) with cur/max text.
                title := strings.ToUpper(portraitSanitize(req.ProgressTitle))
                if title == "" {
                        title = "PROGRESSION"
                }
                dc.SetColor(portraitCol(170, 130, 60, 110))
                dc.SetLineWidth(1.5)
                dc.DrawLine(85, 646, 515, 646)
                dc.Stroke()
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, title, 360, 9)
                ctw, _ := dc.MeasureString(title)
                if ctw > 360 {
                        ctw = 360
                }
                dc.SetColor(portraitCol(96, 62, 24, 235))
                dc.DrawRoundedRectangle(85, 658, ctw+26, 26, 11)
                dc.Fill()
                dc.SetRGB(244.0/255.0, 214.0/255.0, 140.0/255.0)
                dc.DrawStringAnchored(title, 98, 671, 0, 0.5)

                barY := 698.0
                for i, p := range req.Progress {
                        if i >= 5 {
                                break
                        }
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 14, portraitSanitize(p.Label), 128, 9)
                        dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(p.Label), 85, barY, 0, 0.5)

                        pct := 0.0
                        if p.Max > 0 {
                                pct = float64(p.Cur) / float64(p.Max)
                        } else if p.Done {
                                pct = 1
                        }
                        if pct < 0 {
                                pct = 0
                        }
                        if pct > 1 {
                                pct = 1
                        }
                        trackX, trackW, trackH := 225.0, 220.0, 12.0
                        dc.SetColor(portraitCol(54, 36, 18, 255))
                        dc.DrawRoundedRectangle(trackX, barY-trackH/2, trackW, trackH, 6)
                        dc.Fill()
                        fillW := trackW * pct
                        fillCol := portraitCol(214, 170, 82, 255)
                        if p.Done {
                                fillCol = portraitCol(110, 160, 80, 255)
                        }
                        if fillW > 3 {
                                dc.SetColor(fillCol)
                                dc.DrawRoundedRectangle(trackX, barY-trackH/2, fillW, trackH, 6)
                                dc.Fill()
                                dc.SetColor(portraitCol(255, 255, 255, 34))
                                dc.DrawRoundedRectangle(trackX, barY-trackH/2, fillW, 5, 5)
                                dc.Fill()
                        }
                        dc.SetColor(portraitCol(170, 130, 60, 200))
                        dc.SetLineWidth(1.5)
                        dc.DrawRoundedRectangle(trackX, barY-trackH/2, trackW, trackH, 6)
                        dc.Stroke()

                        valTxt := portraitSanitize(p.ValueText)
                        if valTxt == "" {
                                valTxt = fmt.Sprintf("%d/%d", p.Cur, p.Max)
                        }
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 13, valTxt, 80, 8)
                        if p.Done {
                                dc.SetRGB(96.0/255.0, 140.0/255.0, 70.0/255.0)
                        } else {
                                dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                        }
                        dc.DrawStringAnchored(valTxt, 515, barY, 1, 0.5)
                        barY += 29
                }
                portraitWaxSeal(dc, portraitSanitize(req.SealText))
                portraitCaption(dc, portraitSanitize(req.Caption))
        } else if req.Kind == "ALLOCATE" {
                // ── THE POINTS (top section, y266..y492) — mirrors RANK's RECORD ──
                big := strings.ToUpper(portraitSanitize(req.PointsBig))
                if big == "" {
                        big = "0 POINTS"
                }
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 56, big, 380, 30)
                dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                dc.DrawStringAnchored(big, 300, 306, 0.5, 0.5)

                pillTxt := strings.ToUpper(portraitSanitize(req.Pill))
                if pillTxt != "" {
                        portraitPillCentred(dc, pillTxt, 300, 371,
                                portraitCol(96, 62, 24, 235), portraitCol(244, 214, 140, 255))
                }

                // spent bar: same geometry as RANK's XP bar (track (85,402) 430x26)
                pct := req.SpentPercent
                if pct < 0 {
                        pct = 0
                }
                if pct > 100 {
                        pct = 100
                }
                dc.SetColor(portraitCol(54, 36, 18, 255))
                dc.DrawRoundedRectangle(85, 402, 430, 26, 10)
                dc.Fill()
                fillW := 430.0 * float64(pct) / 100.0
                if fillW > 430 {
                        fillW = 430
                }
                if fillW > 4 {
                        dc.SetColor(portraitCol(214, 170, 82, 255))
                        dc.DrawRoundedRectangle(85, 402, fillW, 26, 10)
                        dc.Fill()
                        dc.SetColor(portraitCol(240, 205, 120, 90))
                        dc.DrawRoundedRectangle(85, 402, fillW, 10, 8)
                        dc.Fill()
                }
                dc.SetColor(portraitCol(170, 130, 60, 255))
                dc.SetLineWidth(2)
                dc.DrawRoundedRectangle(85, 402, 430, 26, 10)
                dc.Stroke()
                dc.SetColor(portraitCol(120, 88, 40, 130))
                dc.SetLineWidth(1)
                for _, t := range []float64{0.25, 0.5, 0.75} {
                        tx := 85 + 430*t
                        dc.DrawLine(tx, 405, tx, 425)
                        dc.Stroke()
                }
                if req.SpentNow != "" {
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, portraitSanitize(req.SpentNow), 200, 10)
                        dc.SetRGB(84.0/255.0, 96.0/255.0, 44.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(req.SpentNow), 85, 452, 0, 0.5)
                }
                if req.SpentLeft != "" {
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 18, portraitSanitize(req.SpentLeft), 240, 10)
                        dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(req.SpentLeft), 515, 452, 1, 0.5)
                }

                // ── THE ALLOCATION (lower section) — 7 stat rows ──
                rowY := 540.0
                for i, row := range req.Rows {
                        if i >= 7 {
                                break
                        }
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, portraitSanitize(row.Label), 200, 11)
                        dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
                        dc.DrawStringAnchored(portraitSanitize(row.Label), 85, rowY, 0, 0.5)
                        if row.Sub != "" {
                                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 16, portraitSanitize(row.Sub), 110, 9)
                                dc.SetRGB(84.0/255.0, 96.0/255.0, 44.0/255.0)
                                dc.DrawStringAnchored(portraitSanitize(row.Sub), 352, rowY, 1, 0.5)
                        }
                        val := portraitSanitize(row.Value)
                        if val == "" {
                                val = "0"
                        }
                        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 20, val, 130, 11)
                        dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                        dc.DrawStringAnchored(val, 515, rowY, 1, 0.5)
                        rowY += 38
                }
                portraitWaxSeal(dc, portraitSanitize(req.SealText))
                portraitCaption(dc, portraitSanitize(req.Caption))
        } else if req.Kind == "QUEST" || req.Kind == "TRIAL" {
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
                        dc.SetRGB(84.0/255.0, 96.0/255.0, 44.0/255.0)
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

        buf, ctype, err := utils.EncodeImageToBufferFormat(dc.Image(), c.Query("fmt"), 90)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode portrait card"})
                return
        }
        c.Data(200, ctype, buf)
}

// EndCardPayload — victory/defeat end screen, rendered by WriteEndCard.
// Legacy fields (text/victory/gold/xp/items) kept for old callers; the
// player/enemy/rank/floor/background/caption fields are optional enrichment
// sent by combatIntegration.renderCombatEnd. Everything degrades gracefully:
// no sprites -> bare arena, no rank -> no seal row, etc.
type EndCardPayload struct {
        Text        string `json:"text"`
        Victory     bool   `json:"victory"`
        Gold        int    `json:"gold"`
        XP          int    `json:"xp"`
        Items       string `json:"items"`
        PlayerName  string `json:"playerName"`
        PlayerClass string `json:"playerClass"`
        PlayerIndex int    `json:"playerIndex"`
        PlayerLevel int    `json:"playerLevel"`
        EnemyName   string `json:"enemyName"`
        EnemyLevel  int    `json:"enemyLevel"`
        EnemyIndex  int    `json:"enemyIndex"`
        EnemyIsBoss bool   `json:"enemyIsBoss"`
        Rank        string `json:"rank"`
        Floor       int    `json:"floor"`
        Background  string `json:"background"`
        Caption     string `json:"caption"`
}

// portraitEnemySprite resolves + trims + fits an enemy sprite for the END
// scene window. Tiny canvases (beholder-class sprites can be ~30px) are
// upscaled to at least 60% of the width budget so the enemy READS on the
// card (QA r1: ABYSS WARDEN rendered ~55px — invisible).
func portraitEnemySprite(name string, level, index int, isBoss bool, maxW, maxH int) image.Image {
        path := GetEnemySpritePath(name, level, index, isBoss, "assets")
        img, err := utils.LoadImage(path)
        if err != nil {
                return nil
        }
        img = trimTransparent(img, 8)
        b := img.Bounds()
        minW := maxW * 6 / 10
        if b.Dx() > 0 && b.Dx() < minW {
                scale := minW / b.Dx()
                if scale > 1 {
                        img = imaging.Resize(img, b.Dx()*scale, b.Dy()*scale, imaging.NearestNeighbor)
                }
        }
        return imaging.Fit(img, maxW, maxH, imaging.NearestNeighbor)
}

// WriteEndCard — 2026-09-14 redesign of the victory/defeat end screen
// (owner: "update the victory and defeat image cards with the style").
// Lamoot wood+gold+parchment portrait family on bg_VICTORY/bg_DEFEAT:
// scene window with vivid winner + faded loser, corner pills, spoils
// ledger, wax seal, caption.
func WriteEndCard(c *gin.Context, req EndCardPayload) {
        dc := gg.NewContext(int(portraitW), int(portraitH))
        bgFile := "bg_VICTORY.png"
        if !req.Victory {
                bgFile = "bg_DEFEAT.png"
        }
        if bgImg, err := utils.LoadImage(portraitAsset(bgFile)); err == nil {
                dc.DrawImage(bgImg, 0, 0)
        } else {
                dc.SetRGB(0.09, 0.06, 0.04)
                dc.DrawRectangle(0, 0, portraitW, portraitH)
                dc.Fill()
        }

        // ── name plate ──
        name := portraitSanitize(req.PlayerName)
        if name == "" {
                if req.Victory {
                        name = "The Party"
                } else {
                        name = "The Fallen"
                }
        }
        portraitFitText(dc, portraitAsset("Cinzel.ttf"), 28, name, 215, 12)
        dc.SetRGB(214.0/255.0, 170.0/255.0, 82.0/255.0)
        dc.DrawStringAnchored(name, 84, 159, 0, 0.5)

        // ── scene window: arena + player (left) + enemy (right) ──
        bg := filepath.Base(req.Background)
        if bg == "." || bg == "/" || bg == "" {
                bg = "spark_1.png"
        }
        if arena, err := utils.LoadImage(filepath.Join("assets", "rpgasset", "environment", bg)); err == nil {
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
        if pImg := portraitFighterSprite(req.PlayerClass, req.PlayerIndex, 270, 330, "RIGHT"); pImg != nil {
                if !req.Victory {
                        pImg = portraitFade(pImg, 0.45)
                }
                b := pImg.Bounds()
                px := duelSceneX + 44
                utils.DrawShadow(dc, px+float64(b.Dx())/2, sceneBottom-2, float64(b.Dx())*0.45, 0.5)
                dc.DrawImage(pImg, int(px), int(sceneBottom)-b.Dy())
        }
        if req.EnemyName != "" {
                maxW, maxH := 200, 220
                if !req.Victory {
                        maxW, maxH = 250, 270
                }
                if eImg := portraitEnemySprite(req.EnemyName, req.EnemyLevel, req.EnemyIndex, req.EnemyIsBoss, maxW, maxH); eImg != nil {
                        if req.Victory {
                                eImg = portraitFade(eImg, 0.5)
                        }
                        b := eImg.Bounds()
                        ax := duelSceneX + duelSceneW - 26 - float64(b.Dx())
                        eBottom := sceneBottom - 12
                        utils.DrawShadow(dc, ax+float64(b.Dx())/2, eBottom, float64(b.Dx())*0.42, 0.5)
                        dc.DrawImage(eImg, int(ax), int(eBottom)-b.Dy())
                }
        }

        // ── corner pills ──
        if req.Victory {
                portraitPill(dc, "VICTOR", duelSceneX+62, duelSceneY+26, false,
                        portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
                portraitPill(dc, "SLAIN", duelSceneX+duelSceneW-62, duelSceneY+26, true,
                        portraitCol(8, 8, 8, 150), portraitCol(232, 116, 97, 255))
        } else {
                portraitPill(dc, "FALLEN", duelSceneX+62, duelSceneY+26, false,
                        portraitCol(8, 8, 8, 150), portraitCol(232, 116, 97, 255))
                portraitPill(dc, "VICTORIOUS", duelSceneX+duelSceneW-62, duelSceneY+26, true,
                        portraitCol(8, 8, 8, 150), portraitCol(250, 210, 120, 255))
        }

        // ── spoils ledger (victory) / the fallen (defeat) ──
        type endRow struct {
                label string
                value string
        }
        rows := []endRow{}
        if req.Gold > 0 {
                rows = append(rows, endRow{"ZENI", fmt.Sprintf("+%d", req.Gold)})
        }
        if req.XP > 0 {
                rows = append(rows, endRow{"EXPERIENCE", fmt.Sprintf("+%d", req.XP)})
        }
        if req.Victory {
                if req.Items != "" {
                        rows = append(rows, endRow{"SPOILS", strings.ToUpper(portraitSanitize(req.Items))})
                }
                if req.Floor > 0 {
                        rows = append(rows, endRow{"DEPTH", fmt.Sprintf("FLOOR %d", req.Floor)})
                } else if req.Rank != "" {
                        rows = append(rows, endRow{"RANK", strings.ToUpper(portraitSanitize(req.Rank))})
                }
        } else {
                if req.EnemyName != "" {
                        rows = append(rows, endRow{"SLAIN BY", strings.ToUpper(portraitSanitize(req.EnemyName))})
                }
                if req.Rank != "" {
                        rows = append(rows, endRow{"RANK", strings.ToUpper(portraitSanitize(req.Rank))})
                }
                rows = append(rows, endRow{"RECOVERED", "NOTHING OF VALUE"})
        }
        y := duelLedgerY0
        for i, row := range rows {
                if i >= 4 {
                        break
                }
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 21, portraitSanitize(row.label), 190, 12)
                dc.SetRGB(120.0/255.0, 88.0/255.0, 40.0/255.0)
                dc.DrawStringAnchored(portraitSanitize(row.label), 85, y, 0, 0.5)
                portraitFitText(dc, portraitAsset("Cinzel.ttf"), 21, portraitSanitize(row.value), 300, 10)
                dc.SetRGB(52.0/255.0, 32.0/255.0, 16.0/255.0)
                dc.DrawStringAnchored(portraitSanitize(row.value), 515, y, 1, 0.5)
                y += duelLedgerStep
        }

        // ── seal: rank letter on victory, fallen-at level on defeat ──
        if req.Victory {
                portraitWaxSeal(dc, strings.ToUpper(portraitSanitize(req.Rank)))
        } else if req.PlayerLevel > 0 {
                portraitWaxSeal(dc, fmt.Sprintf("LV%d", req.PlayerLevel))
        }

        cap := req.Caption
        if cap == "" {
                if req.Victory {
                        cap = "the songs will remember this"
                } else {
                        cap = "the dungeon keeps its dead"
                }
        }
        portraitCaption(dc, portraitSanitize(cap))

        buf, ctype, err := utils.EncodeImageToBufferFormat(dc.Image(), c.Query("fmt"), 90)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode end card"})
                return
        }
        c.Data(200, ctype, buf)
}
