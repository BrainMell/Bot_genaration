package cardstyle

import (
	"image"
	"math"

	"github.com/fogleman/gg"
)

// FitEmblem draws img scaled to FIT inside the (maxW x maxH) box centred at
// (cx, cy), preserving aspect ratio. The caller owns clipping/framing so the
// emblem sits inside each style's own crest geometry (shield, ring, stele
// medallion, HUD panel...). No-op on nil image.
func FitEmblem(dc *gg.Context, img image.Image, cx, cy, maxW, maxH float64) {
	if img == nil {
		return
	}
	iw, ih := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	if iw <= 0 || ih <= 0 {
		return
	}
	scale := math.Min(maxW/iw, maxH/ih)
	dc.Push()
	dc.Translate(cx, cy)
	dc.Scale(scale, scale)
	dc.DrawImageAnchored(img, 0, 0, 0.5, 0.5)
	dc.Pop()
}
