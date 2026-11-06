package combat

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
)

type summonCardGIFRequest struct {
	CombatRequest
	CardID string `json:"cardId" binding:"required"`
}

func summonCardVFX(cardID string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(cardID)) {
	case "ifrit":
		return "Fire-bomb", true
	case "leviathan":
		return "Lightning", true
	case "shiva":
		return "Holy VFX 02", true
	default:
		return "", false
	}
}

// GenerateSummonCardGIF renders the already-resolved PvE combat action as a
// genuine animated GIF. Unlike /combat/animated, this endpoint never invokes
// ffmpeg and never falls back to MP4 or a static image.
func GenerateSummonCardGIF(c *gin.Context) {
	var req summonCardGIFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	vfx, ok := summonCardVFX(req.CardID)
	if !ok {
		c.JSON(400, gin.H{"error": "unknown summon card"})
		return
	}
	if req.Action == nil {
		c.JSON(400, gin.H{"error": "summon-card action is required"})
		return
	}
	if len(req.Players) == 0 || len(req.Enemies) == 0 {
		c.JSON(400, gin.H{"error": "combat players and enemies are required"})
		return
	}

	req.CombatType = "PVE"
	req.Action.Vfx = vfx
	frames, err := renderAnimationFrames(&req.CombatRequest)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("render summon-card frames: %v", err)})
		return
	}
	data, err := encodeSummonCardGIF(frames)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("encode summon-card GIF: %v", err)})
		return
	}
	c.Data(200, "image/gif", data)
}

// encodeSummonCardGIF keeps the combat animator's frame sequence but encodes
// it with Go's GIF encoder. The output is deliberately a real GIF stream.
func encodeSummonCardGIF(frames []image.Image) ([]byte, error) {
	if len(frames) < 2 {
		return nil, fmt.Errorf("at least two frames are required")
	}
	animated := &gif.GIF{LoopCount: 0}
	for i, frame := range frames {
		if frame == nil || frame.Bounds().Empty() {
			return nil, fmt.Errorf("frame %d is empty", i)
		}
		// WhatsApp-friendly dimensions and pixel-art scaling; this keeps
		// encoding bounded without involving a video codec or external process.
		scaled := imaging.Resize(frame, 768, 0, imaging.NearestNeighbor)
		paletted := image.NewPaletted(scaled.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), scaled, image.Point{})
		animated.Image = append(animated.Image, paletted)
		animated.Delay = append(animated.Delay, 8) // 80ms per frame
		animated.Disposal = append(animated.Disposal, gif.DisposalNone)
	}
	var out bytes.Buffer
	if err := gif.EncodeAll(&out, animated); err != nil {
		return nil, err
	}
	if out.Len() < 32 {
		return nil, fmt.Errorf("encoded GIF is unexpectedly small")
	}
	return out.Bytes(), nil
}
