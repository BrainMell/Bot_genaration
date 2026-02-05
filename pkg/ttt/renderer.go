package ttt

import (
	"fmt"  // ← ADDED
	"image/color"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

type TTTRequest struct {
	Board         []string `json:"board"`
	GridSize      int      `json:"gridSize"`
	LastMoveIndex int      `json:"lastMoveIndex"`
	WinPattern    []int    `json:"winPattern"`
}

var (
	BgColor   = utils.ParseHexColor("#ECF0F1")
	GridColor = utils.ParseHexColor("#34495E")
	XColor    = utils.ParseHexColor("#E74C3C")
	OColor    = utils.ParseHexColor("#3498DB")
	Highlight = utils.ParseHexColor("#F39C12")
)

func RenderBoard(c *gin.Context) {
	var req TTTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	size := 600.0
	grid := float64(req.GridSize)
	cellSize := size / grid

	dc := gg.NewContext(int(size), int(size))
	dc.SetColor(BgColor)
	dc.Clear()

	// 1. Highlight Cells
	for i, cell := range req.Board {
		row, col := i/req.GridSize, i%req.GridSize
		x, y := float64(col)*cellSize, float64(row)*cellSize

		// Win Pattern Highlight
		isWinCell := false
		for _, w := range req.WinPattern {
			if w == i {
				isWinCell = true
				break
			}
		}

		if isWinCell {
			dc.SetColor(color.RGBA{Highlight.R, Highlight.G, Highlight.B, 50}) // 20% alpha
			dc.DrawRectangle(x+2, y+2, cellSize-4, cellSize-4)
			dc.Fill()
		} else if i == req.LastMoveIndex {
			// Last Move Highlight (Outline)
			dc.SetColor(Highlight)
			dc.SetLineWidth(3)
			dc.DrawRectangle(x+10, y+10, cellSize-20, cellSize-20)
			dc.Stroke()
		}

		// Symbols
		cx, cy := x+cellSize/2, y+cellSize/2
		radius := cellSize * 0.35
		dc.SetLineWidth(gridLineWidth(req.GridSize))

		if cell == "X" {
			dc.SetColor(XColor)
			dc.DrawLine(cx-radius, cy-radius, cx+radius, cy+radius)
			dc.DrawLine(cx+radius, cy-radius, cx-radius, cy+radius)
			dc.Stroke()
		} else if cell == "O" {
			dc.SetColor(OColor)
			dc.DrawCircle(cx, cy, radius)
			dc.Stroke()
		}

		// Numbers for empty cells
		if cell == "" {
			fontPath := utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
			fontSize := fontSize(req.GridSize)
			face, err := utils.LoadFont(fontPath, fontSize)
			if err == nil {
				dc.SetFontFace(face)
				dc.SetColor(color.RGBA{0, 0, 0, 100})
				dc.DrawStringAnchored(fmt.Sprintf("%d", i), cx, cy, 0.5, 0.5)
			}
		}
	}

	// 2. Grid Lines
	dc.SetColor(GridColor)
	dc.SetLineWidth(4)
	for i := 1; i < req.GridSize; i++ {
		pos := float64(i) * cellSize
		dc.DrawLine(pos, 0, pos, size)
		dc.DrawLine(0, pos, size, pos)
	}
	dc.Stroke()

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "Encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func gridLineWidth(grid int) float64 {
	if grid <= 3 { return 10 }
	if grid <= 8 { return 5 }
	return 3
}

func fontSize(grid int) float64 {
	if grid <= 3 { return 40 }
	if grid <= 8 { return 20 }
	return 12
}
