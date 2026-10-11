package combat

// skillicons.go - skill/rune artwork loader for the Go card renderers
// (2026-10-11 owner asset overhaul: "first 2 images are runes all next
// are skills"). The Node repo slices the reference sheets into
// whatsapp-bot/core/rpgasset/{skills,runes}/<name>.png with manifest maps
// (commit e8c371f) and sends basenames in portrait payloads:
//   portraitNode.Icon        - SKILLTREE per-node art
//   portraitRequest.Icon     - SKILLUP medallion art
// The Go service reads those files cross-repo so both sides share one
// asset truth (no duplicated slices). Contract: missing file, unsafe
// name or unreadable PNG -> nil -> callers keep their initials/medallion
// fallback, so old payloads and renamed files degrade gracefully.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"image"
	"image/color"

	"github.com/fogleman/gg"
	"image-service/pkg/utils"
)

var (
	skillIconOnce sync.Once
	skillIconDirs []string
	// basenames only - the manifests ship flat lowercase_snake.png files
	skillIconNameRe = regexp.MustCompile(`^[a-z0-9_]+\.[pP][nN][gG]$`)
)

// skillIconDirsResolved - where to look for skill/rune art.
// SKILL_ICON_DIRS="dir1:dir2" overrides; default is the whatsapp-bot repo
// sitting next to this service on the box (pm2 cwd /home/ubuntu), then
// this repo's own assets/rpgasset as a secondary.
func skillIconDirsResolved() []string {
	skillIconOnce.Do(func() {
		if env := os.Getenv("SKILL_ICON_DIRS"); env != "" {
			for _, d := range strings.Split(env, ":") {
				if d != "" {
					skillIconDirs = append(skillIconDirs, d)
				}
			}
		}
		if len(skillIconDirs) == 0 {
			skillIconDirs = []string{
				"/home/ubuntu/whatsapp-bot/core/rpgasset",
				"assets/rpgasset",
			}
		}
	})
	return skillIconDirs
}

// skillIconPath validates an icon basename and resolves it under
// <base>/<kindDir>/; returns "" when the name is unsafe or not found.
func skillIconPath(kindDir, name string) string {
	name = strings.TrimSpace(name)
	if name == "" || !skillIconNameRe.MatchString(name) {
		return ""
	}
	for _, base := range skillIconDirsResolved() {
		p := filepath.Join(base, kindDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadSkillIcon loads skill art (utils.LoadImage caches by path for the
// process lifetime, so repeated tree renders never re-read the disk).
func LoadSkillIcon(name string) image.Image {
	p := skillIconPath("skills", name)
	if p == "" {
		return nil
	}
	img, err := utils.LoadImage(p)
	if err != nil {
		return nil
	}
	return img
}

// LoadRuneIcon loads rune art the same way.
func LoadRuneIcon(name string) image.Image {
	p := skillIconPath("runes", name)
	if p == "" {
		return nil
	}
	img, err := utils.LoadImage(p)
	if err != nil {
		return nil
	}
	return img
}

// DrawIconDisc paints icon art inside a circular disc at (x,y,r) using a
// cover-fit (scaled to fill, centre-cropped). The slices are x3
// nearest-neighbour upsamples of the source sheets, so downscaling keeps
// smoothing on (gg default) for clean edges. An optional flat tint is
// composited over the disc (state dimming for locked/open nodes).
// Returns false when img is nil so callers fall back to initials.
// NOTE: the disc is composed OFFSCREEN - see the comment in the body.
func DrawIconDisc(dc *gg.Context, img image.Image, x, y, r float64, tint *color.NRGBA) bool {
	if dc == nil || img == nil || r <= 0 {
		return false
	}
	b := img.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		return false
	}
	// Compose in a small OFFSCREEN context and blit once: gg's Clip mask
	// must never be set on the caller's context - Push/Pop does not
	// contain the clip in the gg version pinned here (it leaked and
	// swallowed every draw that followed).
	W := int(r*2 + 1)
	if W < 2 {
		return false
	}
	disc := gg.NewContext(W, W)
	cx, cy := float64(W)/2, float64(W)/2
	disc.DrawCircle(cx, cy, r)
	disc.Clip()
	scale := (2 * r) / sw
	if (2*r)/sh > scale {
		scale = (2 * r) / sh
	}
	w, h := sw*scale, sh*scale
	disc.Push()
	disc.Translate(cx-w/2, cy-h/2)
	disc.Scale(scale, scale)
	disc.DrawImage(img, 0, 0)
	disc.Pop()
	if tint != nil {
		disc.SetColor(*tint)
		disc.DrawCircle(cx, cy, r)
		disc.Fill()
	}
	dc.DrawImage(disc.Image(), int(x-float64(W)/2), int(y-float64(W)/2))
	return true
}

// ── styled-variant wiring (2026-10-11, styles 01-09) ─────────────────────
// nodeArtTint mirrors the r6 default-style state language: locked dims
// heavy, open dims light, learned/maxed stay full bright (nil tint).
func nodeArtTint(state string) *color.NRGBA {
	switch state {
	case "locked":
		c := color.NRGBA{R: 10, G: 6, B: 4, A: 150}
		return &c
	case "open":
		c := color.NRGBA{R: 12, G: 8, B: 6, A: 60}
		return &c
	}
	return nil
}

// DrawNodeArtTile cover-fits skill art into a rectangular node tile with a
// readability scrim so the style's own name/level text stays legible over
// busy art. Composed OFFSCREEN - never Clip the caller's context (gg leak).
func DrawNodeArtTile(dc *gg.Context, art image.Image, x, y, w, h float64, state string) bool {
	if dc == nil || art == nil || w < 10 || h < 10 {
		return false
	}
	W, H := int(w+1), int(h+1)
	if W < 2 || H < 2 {
		return false
	}
	b := art.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		return false
	}
	tile := gg.NewContext(W, H)
	scale := w / sw
	if h/sh > scale {
		scale = h / sh
	}
	tw, th := sw*scale, sh*scale
	tile.Push()
	tile.Translate((w-tw)/2, (h-th)/2)
	tile.Scale(scale, scale)
	tile.DrawImage(art, 0, 0)
	tile.Pop()
	tile.SetColor(color.NRGBA{R: 10, G: 6, B: 4, A: 92})
	tile.DrawRectangle(0, 0, w, h)
	tile.Fill()
	g := gg.NewLinearGradient(0, h*0.26, 0, h)
	g.AddColorStop(0, color.NRGBA{R: 10, G: 6, B: 4, A: 0})
	g.AddColorStop(1, color.NRGBA{R: 10, G: 6, B: 4, A: 185})
	tile.SetFillStyle(g)
	tile.DrawRectangle(0, h*0.26, w, h*0.74)
	tile.Fill()
	if tint := nodeArtTint(state); tint != nil {
		tile.SetColor(*tint)
		tile.DrawRectangle(0, 0, w, h)
		tile.Fill()
	}
	dc.DrawImage(tile.Image(), int(x), int(y))
	return true
}

// DrawArtChip - small dark circular chip with centered text (the r6
// progress-chip pattern), keeps station labels readable on top of node art.
func DrawArtChip(dc *gg.Context, x, y, r float64, text string) {
	dc.SetColor(portraitCol(16, 10, 5, 225))
	dc.DrawCircle(x, y, r)
	dc.Fill()
	dc.SetColor(portraitCol(226, 182, 96, 255))
	portraitFitText(dc, portraitAsset("Cinzel.ttf"), 12, text, r*1.7, 7)
	dc.DrawStringAnchored(text, x, y, 0.5, 0.5)
}
