package cards

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"

	"image-service/pkg/utils"
)

// =============================================================================
// COLLECTION/DECK GRID RENDERER — 4x4 grid (same style as eShop)
// =============================================================================
// Uses the SAME rendering approach as eshop.go:
//   - 4 columns × 4 rows (16 card slots max)
//   - Sequential image downloads (no parallel goroutines)
//   - Tier-colored borders
//   - Card name below each card
//   - PNG output
//   - Sends as { image: buffer, caption } — no GIF/MP4 involved
//
// This replaces the old GenerateCardGif path which tried Cloudinary
// slideshow (always failed on 500MB) then fell back to a broken grid.

const (
	COLL_CARD_W    = 200 // width of each card
	COLL_CARD_H    = 300 // height of each card
	COLL_GRID_COLS = 4
	COLL_PADDING   = 10
	COLL_BORDER    = 2
	COLL_HEADER_H  = 60
	COLL_LABEL_H   = 25  // name area below card
	COLL_FOOTER_H  = 35
)

// CollCardInput represents one card in the collection/deck grid
type CollCardInput struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	Tier     string `json:"tier"`
	Animated bool   `json:"animated"` // ignored — we render static PNG
}

// CollGridRequest is the payload for the collection grid endpoint
type CollGridRequest struct {
	Images []CollCardInput `json:"images"`
	Title  string          `json:"title"`
}

// GenerateCollectionGrid renders a 4×4 grid of card images as a static PNG.
// Same style as eShop deck — full-width image, no GIF/MP4.
func GenerateCollectionGrid(c *gin.Context) {
	var req CollGridRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if len(req.Images) == 0 {
		c.JSON(400, gin.H{"error": "No images provided"})
		return
	}

	// Max 16 cards (4×4)
	maxCards := 16
	if len(req.Images) > maxCards {
		req.Images = req.Images[:maxCards]
	}

	if req.Title == "" {
		req.Title = "COLLECTION"
	}

	// Calculate rows based on actual card count
	cardCount := len(req.Images)
	rows := (cardCount + COLL_GRID_COLS - 1) / COLL_GRID_COLS
	if rows < 1 {
		rows = 1
	}

	gridW := COLL_GRID_COLS*COLL_CARD_W + (COLL_GRID_COLS+1)*COLL_PADDING
	gridH := rows*(COLL_CARD_H+COLL_LABEL_H) + (rows+1)*COLL_PADDING
	totalW := gridW
	totalH := COLL_HEADER_H + gridH + COLL_FOOTER_H

	dc := gg.NewContext(totalW, totalH)

	// === Background ===
	dc.SetColor(color.RGBA{15, 16, 23, 255})
	dc.DrawRectangle(0, 0, float64(totalW), float64(totalH))
	dc.Fill()

	fontBold := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
	fontMed := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")

	// === Header (gradient bar) ===
	for x := 0; x < totalW; x++ {
		t := float64(x) / float64(totalW)
		r := uint8(20 + t*40)
		g := uint8(20 + t*20)
		b := uint8(60 + t*80)
		dc.SetColor(color.RGBA{r, g, b, 255})
		dc.SetPixel(x, 0)
		for y := 1; y < COLL_HEADER_H; y++ {
			dc.SetPixel(x, y)
		}
	}

	if err := dc.LoadFontFace(fontBold, 24); err == nil {
		dc.SetColor(color.RGBA{255, 255, 255, 255})
		dc.DrawStringAnchored(req.Title, float64(totalW)/2, float64(COLL_HEADER_H)/2, 0.5, 0.5)
	}

	// === Card Grid (sequential downloads) ===
	client := &http.Client{Timeout: 8 * time.Second}

	for i, card := range req.Images {
		col := i % COLL_GRID_COLS
		row := i / COLL_GRID_COLS

		x := COLL_PADDING + col*(COLL_CARD_W+COLL_PADDING)
		y := COLL_HEADER_H + COLL_PADDING + row*(COLL_CARD_H+COLL_LABEL_H+COLL_PADDING)

		// Card slot background
		dc.SetColor(color.RGBA{30, 32, 45, 255})
		dc.DrawRoundedRectangle(float64(x), float64(y), float64(COLL_CARD_W), float64(COLL_CARD_H), 8)
		dc.Fill()

		// Download and draw card image
		if card.URL != "" {
			img, err := downloadAndResizeCollImage(client, card.URL, COLL_CARD_W-2*COLL_BORDER, COLL_CARD_H-2*COLL_BORDER)
			if err == nil {
				imgX := float64(x + COLL_BORDER)
				imgY := float64(y + COLL_BORDER)
				drawRoundedCollImage(dc, img, imgX, imgY, float64(COLL_CARD_W-2*COLL_BORDER), float64(COLL_CARD_H-2*COLL_BORDER), 6)
			} else {
				log.Printf("[CollGrid] Failed to fetch card %d: %v", i, err)
				if err := dc.LoadFontFace(fontMed, 14); err == nil {
					dc.SetColor(color.RGBA{120, 120, 140, 255})
					dc.DrawStringAnchored("?", float64(x+COLL_CARD_W/2), float64(y+COLL_CARD_H/2), 0.5, 0.5)
				}
			}
		}

		// Tier-colored border
		tierColor := getTierColor(card.Tier)
		dc.SetColor(tierColor)
		dc.SetLineWidth(3)
		dc.DrawRoundedRectangle(float64(x), float64(y), float64(COLL_CARD_W), float64(COLL_CARD_H), 8)
		dc.Stroke()

		// Tier badge (top-left)
		tierLabel := getTierLabel(card.Tier)
		if tierLabel != "" {
			if err := dc.LoadFontFace(fontBold, 12); err == nil {
				dc.SetColor(color.RGBA{0, 0, 0, 200})
				dc.DrawRoundedRectangle(float64(x+4), float64(y+4), 30, 18, 4)
				dc.Fill()
				dc.SetColor(tierColor)
				dc.DrawStringAnchored(tierLabel, float64(x+4+15), float64(y+4+9), 0.5, 0.5)
			}
		}

		// Card name (below card)
		labelY := y + COLL_CARD_H + 4
		if err := dc.LoadFontFace(fontMed, 12); err == nil {
			nameStr := card.Name
			if len(nameStr) > 20 {
				nameStr = nameStr[:18] + "…"
			}
			dc.SetColor(color.RGBA{220, 220, 230, 255})
			dc.DrawStringAnchored(nameStr, float64(x+COLL_CARD_W/2), float64(labelY+8), 0.5, 0.5)
		}
	}

	// === Footer ===
	footerY := totalH - COLL_FOOTER_H
	if err := dc.LoadFontFace(fontMed, 13); err == nil {
		dc.SetColor(color.RGBA{160, 160, 180, 255})
		footerText := fmt.Sprintf("Showing %d cards", len(req.Images))
		dc.DrawStringAnchored(footerText, float64(totalW)/2, float64(footerY+COLL_FOOTER_H/2), 0.5, 0.5)
	}

	// Encode to PNG
	buf := &bytes.Buffer{}
	if err := dc.EncodePNG(buf); err != nil {
		c.JSON(500, gin.H{"error": "Failed to encode image"})
		return
	}

	c.Data(200, "image/png", buf.Bytes())
}

// downloadAndResizeCollImage fetches and resizes a card image
func downloadAndResizeCollImage(client *http.Client, url string, targetW, targetH int) (image.Image, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// NearestNeighbor for speed on 0.1 CPU (Lanczos is 10x slower)
	img = imaging.Fill(img, targetW, targetH, imaging.Center, imaging.NearestNeighbor)
	return img, nil
}

// drawRoundedCollImage draws an image with rounded corners
func drawRoundedCollImage(dc *gg.Context, img image.Image, x, y, w, h, radius float64) {
	mask := gg.NewContext(int(w), int(h))
	mask.SetColor(color.White)
	mask.DrawRoundedRectangle(0, 0, w, h, radius)
	mask.Fill()

	dc.Push()
	tmp := gg.NewContext(int(w), int(h))
	tmp.DrawImage(img, 0, 0)
	dst := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	draw.DrawMask(dst, dst.Bounds(), tmp.Image(), image.Point{}, mask.Image(), image.Point{}, draw.Src)
	dc.DrawImage(dst, int(x), int(y))
	dc.Pop()
}

// getTierLabel returns a short badge label for a tier
func getTierLabel(tier string) string {
	switch tier {
	case "S":
		return "TS"
	case "E":
		return "EV"
	case "1", "2", "3", "4", "5", "6":
		return "T" + tier
	default:
		return ""
	}
}
