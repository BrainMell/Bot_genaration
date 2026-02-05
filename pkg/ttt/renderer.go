package ttt

import (
	"fmt"
	"image/color"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

type TTTRequest struct {
	Board         []string `json:"board"` // "X", "O", or null/empty
	GridSize      int      `json:"gridSize"`
	LastMoveIndex int      `json:"lastMoveIndex"`
	WinPattern    []int    `json:"winPattern"`
}

var (
	BgColor    = utils.ParseHexColor("#ECF0F1")
	GridColor  = utils.ParseHexColor("#34495E")
	XColor     = utils.ParseHexColor("#E74C3C")
	OColor     = utils.ParseHexColor("#3498DB")
	Highlight  = utils.ParseHexColor("#F39C12")
)

func RenderBoard(c *gin.Context) {
	var req TTTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	size := 600
	cellSize := float64(size) / float64(req.GridSize)

	dc := gg.NewContext(size, size)
	dc.SetColor(BgColor)
	dc.Clear()

	// 1. Highlight Last Move
	if req.LastMoveIndex >= 0 && req.LastMoveIndex < len(req.Board) {
		row := req.LastMoveIndex / req.GridSize
		col := req.LastMoveIndex % req.GridSize
		dc.SetColor(Highlight)
		dc.DrawRectangle(float64(col)*cellSize, float64(row)*cellSize, cellSize, cellSize)
		dc.Fill() // Or stroke
	}

	// 2. Draw Grid
	dc.SetColor(GridColor)
	dc.SetLineWidth(4)
	for i := 1; i < req.GridSize; i++ {
		pos := float64(i) * cellSize
		// Vert
		dc.DrawLine(pos, 0, pos, float64(size))
		// Horiz
		dc.DrawLine(0, pos, float64(size), pos)
	}
	dc.Stroke()

	// 3. Draw Symbols
	for i, cell := range req.Board {
		if cell == "" {
			continue
		}
		
		row := i / req.GridSize
		col := i % req.GridSize
		cx := float64(col)*cellSize + cellSize/2
		cy := float64(row)*cellSize + cellSize/2
		radius := cellSize * 0.35

		dc.SetLineWidth(10)

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
	}

	// 4. Win Line
	if len(req.WinPattern) > 0 {
		dc.SetColor(Highlight)
		dc.SetLineWidth(15)
		
		// Draw line through start and end points
		start := req.WinPattern[0]
		end := req.WinPattern[len(req.WinPattern)-1]
		
		sx := (float64(start%req.GridSize) * cellSize) + cellSize/2
		sy := (float64(start/req.GridSize) * cellSize) + cellSize/2
		ex := (float64(end%req.GridSize) * cellSize) + cellSize/2
		ey := (float64(end/req.GridSize) * cellSize) + cellSize/2
		
		dc.DrawLine(sx, sy, ex, ey)
		dc.Stroke()
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "Encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

func RenderLeaderboard(c *gin.Context) {
    // Placeholder for leaderboard rendering
    // Logic: Load background, draw text list
    c.JSON(501, gin.H{"error": "Not implemented"})
}
