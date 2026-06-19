package cards

import (
        "bytes"
        "encoding/base64"
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
// ESHOP DECK RENDERER — 4x4 grid of event card images
// =============================================================================

const (
        ESHOP_CARD_W      = 250 // width of each card slot in the grid
        ESHOP_CARD_H      = 350 // height of each card slot
        ESHOP_GRID_COLS   = 4
        ESHOP_GRID_ROWS   = 4
        ESHOP_PADDING     = 10  // padding between cards
        ESHOP_BORDER      = 2   // border around each card
        ESHOP_HEADER_H    = 60  // header banner height
        ESHOP_FOOTER_H    = 40  // footer height (for instructions)
        ESHOP_LABEL_H     = 30  // slot label area below each card
)

// EShopCard represents one card in the deck
type EShopCard struct {
        Slot     int    `json:"slot"`     // 1-16
        CardID   string `json:"cardId"`   // card ID for reference
        CardName string `json:"cardName"` // display name
        ImageURL string `json:"imageUrl"` // card image URL
        Price    int    `json:"price"`    // price in tokens
        Tier     string `json:"tier"`     // card tier (e.g. "E", "S", "6")
        Anime    string `json:"anime"`    // anime name
}

// EShopDeckRequest is the payload for the eShop deck render endpoint
type EShopDeckRequest struct {
        Title    string      `json:"title"`    // e.g. "EVENT SHOP — TOKEN EVENT"
        Cards    []EShopCard `json:"cards"`    // up to 16 cards
        Currency string      `json:"currency"` // e.g. "🎫 Tokens"
}

// GenerateEShopDeck renders a 4x4 grid of event card images as a static PNG.
// This is used by the `.j eshop` command to show all available event cards.
func GenerateEShopDeck(c *gin.Context) {
        var req EShopDeckRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        if req.Title == "" {
                req.Title = "🎁 EVENT SHOP"
        }
        if req.Currency == "" {
                req.Currency = "🎫 Tokens"
        }

        // Calculate canvas dimensions
        gridW := ESHOP_GRID_COLS*ESHOP_CARD_W + (ESHOP_GRID_COLS+1)*ESHOP_PADDING
        gridH := ESHOP_GRID_ROWS*(ESHOP_CARD_H+ESHOP_LABEL_H) + (ESHOP_GRID_ROWS+1)*ESHOP_PADDING
        totalW := gridW
        totalH := ESHOP_HEADER_H + gridH + ESHOP_FOOTER_H

        dc := gg.NewContext(totalW, totalH)

        // === Background ===
        dc.SetColor(color.RGBA{15, 16, 23, 255}) // dark navy
        dc.DrawRectangle(0, 0, float64(totalW), float64(totalH))
        dc.Fill()

        // === Header Banner ===
        fontBold := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
        fontMed := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")

        // Gradient header bar
        for x := 0; x < totalW; x++ {
                t := float64(x) / float64(totalW)
                r := uint8(20 + t*40)
                g := uint8(20 + t*20)
                b := uint8(60 + t*80)
                dc.SetColor(color.RGBA{r, g, b, 255})
                dc.SetPixel(x, 0)
                for y := 1; y < ESHOP_HEADER_H; y++ {
                        dc.SetPixel(x, y)
                }
        }

        if err := dc.LoadFontFace(fontBold, 28); err == nil {
                dc.SetColor(color.RGBA{255, 255, 255, 255})
                dc.DrawStringAnchored(req.Title, float64(totalW)/2, float64(ESHOP_HEADER_H)/2, 0.5, 0.5)
        }

        // Currency label (top-right)
        if err := dc.LoadFontFace(fontMed, 16); err == nil {
                dc.SetColor(color.RGBA{255, 215, 0, 255}) // gold
                dc.DrawStringAnchored(req.Currency, float64(totalW)-20, float64(ESHOP_HEADER_H)/2, 1.0, 0.5)
        }

        // === Card Grid ===
        client := &http.Client{Timeout: 15 * time.Second}

        for i, card := range req.Cards {
                if i >= ESHOP_GRID_COLS*ESHOP_GRID_ROWS {
                        break // max 16 cards
                }

                col := i % ESHOP_GRID_COLS
                row := i / ESHOP_GRID_COLS

                x := ESHOP_PADDING + col*(ESHOP_CARD_W+ESHOP_PADDING)
                y := ESHOP_HEADER_H + ESHOP_PADDING + row*(ESHOP_CARD_H+ESHOP_LABEL_H+ESHOP_PADDING)

                // Draw card slot background (dark card-shaped rectangle)
                dc.SetColor(color.RGBA{30, 32, 45, 255})
                dc.DrawRoundedRectangle(float64(x), float64(y), float64(ESHOP_CARD_W), float64(ESHOP_CARD_H), 8)
                dc.Fill()

                // Try to fetch and draw the card image
                if card.ImageURL != "" {
                        img, err := downloadAndResizeImage(client, card.ImageURL, ESHOP_CARD_W-2*ESHOP_BORDER, ESHOP_CARD_H-2*ESHOP_BORDER)
                        if err == nil {
                                // Draw the image inside the card slot with border
                                imgX := float64(x + ESHOP_BORDER)
                                imgY := float64(y + ESHOP_BORDER)
                                // Create a rounded-rectangle clip mask for the image
                                drawRoundedImage(dc, img, imgX, imgY, float64(ESHOP_CARD_W-2*ESHOP_BORDER), float64(ESHOP_CARD_H-2*ESHOP_BORDER), 6)
                        } else {
                                log.Printf("[eShop] Failed to fetch card image %s: %v", card.ImageURL, err)
                                // Draw placeholder text
                                if err := dc.LoadFontFace(fontMed, 14); err == nil {
                                        dc.SetColor(color.RGBA{120, 120, 140, 255})
                                        dc.DrawStringAnchored("No Image", float64(x+ESHOP_CARD_W/2), float64(y+ESHOP_CARD_H/2), 0.5, 0.5)
                                }
                        }
                }

                // Draw card border (tier-colored)
                tierColor := getTierColor(card.Tier)
                dc.SetColor(tierColor)
                dc.SetLineWidth(3)
                dc.DrawRoundedRectangle(float64(x), float64(y), float64(ESHOP_CARD_W), float64(ESHOP_CARD_H), 8)
                dc.Stroke()

                // Slot number (top-left corner)
                if err := dc.LoadFontFace(fontBold, 16); err == nil {
                        // Background circle for slot number
                        dc.SetColor(color.RGBA{0, 0, 0, 200})
                        dc.DrawCircle(float64(x+18), float64(y+18), 14)
                        dc.Fill()
                        dc.SetColor(color.RGBA{255, 255, 255, 255})
                        dc.DrawStringAnchored(fmt.Sprintf("%d", card.Slot), float64(x+18), float64(y+18), 0.5, 0.5)
                }

                // Price tag (bottom-right of card image)
                if card.Price > 0 {
                        priceText := fmt.Sprintf("🎫 %d", card.Price)
                        if err := dc.LoadFontFace(fontBold, 14); err == nil {
                                // Measure text width for background pill
                                w, h := dc.MeasureString(priceText)
                                pillX := float64(x+ESHOP_CARD_W) - w - 16
                                pillY := float64(y+ESHOP_CARD_H) - h - 16
                                // Draw pill background
                                dc.SetColor(color.RGBA{0, 0, 0, 220})
                                dc.DrawRoundedRectangle(pillX-6, pillY-4, w+12, h+8, 6)
                                dc.Fill()
                                // Draw price text
                                dc.SetColor(color.RGBA{255, 215, 0, 255}) // gold
                                dc.DrawString(priceText, pillX, pillY+h)
                        }
                }

                // Card name + anime (label area below card)
                labelY := y + ESHOP_CARD_H + 4
                if err := dc.LoadFontFace(fontMed, 12); err == nil {
                        // Truncate name if too long
                        nameStr := card.CardName
                        if len(nameStr) > 22 {
                                nameStr = nameStr[:20] + "…"
                        }
                        dc.SetColor(color.RGBA{220, 220, 230, 255})
                        dc.DrawStringAnchored(nameStr, float64(x+ESHOP_CARD_W/2), float64(labelY+8), 0.5, 0.5)
                }
        }

        // === Footer ===
        footerY := totalH - ESHOP_FOOTER_H
        if err := dc.LoadFontFace(fontMed, 14); err == nil {
                dc.SetColor(color.RGBA{160, 160, 180, 255})
                footerText := fmt.Sprintf("Use eshop buy <slot> to purchase • Currency: %s", req.Currency)
                dc.DrawStringAnchored(footerText, float64(totalW)/2, float64(footerY+ESHOP_FOOTER_H/2), 0.5, 0.5)
        }

        // Encode to PNG
        buf := &bytes.Buffer{}
        if err := dc.EncodePNG(buf); err != nil {
                c.JSON(500, gin.H{"error": "Failed to encode image"})
                return
        }

        c.Data(200, "image/png", buf.Bytes())
}

// downloadAndResizeImage fetches an image from a URL and resizes it to fit
// the target dimensions while maintaining aspect ratio (with cropping).
func downloadAndResizeImage(client *http.Client, url string, targetW, targetH int) (image.Image, error) {
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

        // Resize to fill (crop to exact dimensions)
        img = imaging.Fill(img, targetW, targetH, imaging.Center, imaging.Lanczos)
        return img, nil
}

// drawRoundedImage draws an image with rounded corners at the given position.
func drawRoundedImage(dc *gg.Context, img image.Image, x, y, w, h, radius float64) {
        // Create a mask with rounded rectangle
        mask := gg.NewContext(int(w), int(h))
        mask.SetColor(color.White)
        mask.DrawRoundedRectangle(0, 0, w, h, radius)
        mask.Fill()

        // Draw the image, then apply the mask
        dc.Push()
        // Create a temporary context for compositing
        tmp := gg.NewContext(int(w), int(h))
        tmp.DrawImage(img, 0, 0)
        // Apply mask: use the mask as alpha channel
        dst := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
        draw.DrawMask(dst, dst.Bounds(), tmp.Image(), image.Point{}, mask.Image(), image.Point{}, draw.Src)
        dc.DrawImage(dst, int(x), int(y))
        dc.Pop()
}

// getTierColor returns a border color for a card tier.
func getTierColor(tier string) color.RGBA {
        switch tier {
        case "S":
                return color.RGBA{255, 215, 0, 255} // gold
        case "6":
                return color.RGBA{186, 85, 211, 255} // medium orchid
        case "5":
                return color.RGBA{255, 140, 0, 255} // dark orange
        case "4":
                return color.RGBA{138, 43, 226, 255} // blue violet
        case "3":
                return color.RGBA{30, 144, 255, 255} // dodger blue
        case "2":
                return color.RGBA{50, 205, 50, 255} // lime green
        case "1":
                return color.RGBA{192, 192, 192, 255} // silver
        case "E":
                return color.RGBA{255, 105, 180, 255} // hot pink (event cards)
        default:
                return color.RGBA{120, 120, 140, 255} // grey
        }
}

// Base64EncodeImage is a helper that base64-encodes an image buffer (for
// embedding in JSON responses if needed).
func Base64EncodeImage(data []byte) string {
        return base64.StdEncoding.EncodeToString(data)
}
