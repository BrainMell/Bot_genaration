package combat

// styles_common.go - shared helpers for the 9 rebuilt style renderers
// (2026-09-16 v2 card redesign). Compositions live in style01..10.go.
// Style 7 (Royal Decree) is the baked baseline and never routes here.

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"

	"image-service/pkg/cardstyle"
	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

// styledKinds - kinds the rebuilt systems render (battle-active kinds are
// excluded by owner rule and keep the baked combat art).
func styledKinds() map[string]bool {
	return map[string]bool{
		"RANK": true, "ALLOCATE": true, "SKILLUP": true, "ABILITIES": true, "EVOLVE": true,
		"SKILLTREE": true, "EQUIP": true, "SHOP": true, "GUILDINFO": true,
	}
}

// renderStyledImage resolves the per-style composition for kind.
// Returns nil when the kind/style has no rebuilt composition.
func renderStyledImage(style int, kind string, req *portraitRequest) image.Image {
	if style <= 0 || style == 7 || style > 10 || !styledKinds()[kind] {
		return nil
	}
	switch style {
	case 1:
		return renderStyle01(kind, req)
	case 2:
		return renderStyle02(kind, req)
	case 3:
		return renderStyle03(kind, req)
	case 4:
		return renderStyle04(kind, req)
	case 5:
		return renderStyle05(kind, req)
	case 6:
		return renderStyle06(kind, req)
	case 8:
		return renderStyle08(kind, req)
	case 9:
		return renderStyle09(kind, req)
	case 10:
		return renderStyle10(kind, req)
	}
	return nil
}

// renderStyledCard renders kind for a rebuilt style (1-6, 8-10).
// Returns false when the kind/style has no rebuilt composition, so the
// caller falls back to the canonical baked paths.
func renderStyledCard(c *gin.Context, req *portraitRequest) bool {
	if req == nil {
		return false
	}
	img := renderStyledImage(req.Style, req.Kind, req)
	if img == nil {
		return false
	}
	utils.RespondImage(c, img)
	return true
}

// QAStyledRender renders a styled card from a raw JSON payload
// (identical wire format as /api/cards/portrait). QA harness only.
func QAStyledRender(payload []byte) image.Image {
	var req portraitRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil
	}
	return renderStyledImage(req.Style, req.Kind, &req)
}

// ── shared payload helpers ────────────────────────────────────────────

func styledName(req *portraitRequest) string {
	n := cardstyle.Sanitize(req.Nickname)
	if n == "" {
		n = "Adventurer"
	}
	return n
}

// heroImage returns the player class sprite (already trimmed/fitted).
func heroImage(req *portraitRequest, maxW, maxH int) image.Image {
	img := portraitFighterSprite(req.PlayerClass, req.PlayerIndex, maxW, maxH, "RIGHT")
	return img
}

// drawHeroFitted centers a sprite (or initials fallback) in a box.
func drawHeroFitted(dc *gg.Context, req *portraitRequest, cx, cy, maxW, maxH float64, ring cardstyle.Palette) {
	img := heroImage(req, int(maxW), int(maxH))
	if img != nil {
		b := img.Bounds()
		dc.DrawImage(img, int(cx-float64(b.Dx())/2), int(cy-float64(b.Dy())/2))
		return
	}
	// initials medallion fallback
	initials := initialLetters(styledName(req))
	cardstyle.Ring(dc, cx, cy, maxW*0.42, ring.Accent, 2)
	dc.SetColor(cardstyle.WithA(ring.Panel, 200))
	dc.DrawCircle(cx, cy, maxW*0.42-2)
	dc.Fill()
	cardstyle.Fit(dc, cardstyle.FtCinzel, 64, initials, maxW*0.6, 20)
	dc.SetColor(ring.Ink)
	dc.DrawStringAnchored(initials, cx, cy, 0.5, 0.5)
}

// abilitiesH mirrors the canonical dynamic height so pagination never clips.
func abilitiesH(req *portraitRequest) int {
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
	return int(H)
}

// totalAbilityRows counts rendered rows (for numbering).
func totalAbilityRows(req *portraitRequest) int {
	total := 0
	for _, g := range req.Groups {
		total += len(g.Items)
	}
	return total
}

// parseLeadingInt extracts the first integer in s ("5 POINTS" -> 5).
func parseLeadingInt(s string, fallback int) int {
	num := ""
	for _, r := range s {
		if r >= '0' && r <= '9' {
			num += string(r)
		} else if num != "" {
			break
		}
	}
	if num == "" {
		return fallback
	}
	v := 0
	for _, d := range num {
		v = v*10 + int(d-'0')
	}
	return v
}

// gainFromSub converts "+15/pt" -> "+15".
func gainFromSub(sub string) string {
	s := cardstyle.Sanitize(sub)
	if i := strings.Index(s, "/"); i > 0 {
		s = s[:i]
	}
	if s == "" {
		return "+1"
	}
	return s
}

// fmtProgressText renders "cur / max".
func fmtProgressText(cur, max int, vt string) string {
	if vt != "" {
		return vt
	}
	return fmt.Sprintf("%d / %d", cur, max)
}

// firstRuneUpper returns the first rune of s uppercased (crest initials).
func firstRuneUpper(s string) string {
	r := []rune(strings.ToUpper(cardstyle.Sanitize(s)))
	if len(r) == 0 {
		return "G"
	}
	return string(r[:1])
}
