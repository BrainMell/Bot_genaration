package ludo

import (
	"fmt"
	"image/color"
	"math"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	BOARD_SIZE = 900
	CELL_SIZE  = 60
)

type LudoRequest struct {
	Players []struct {
		JID    string `json:"jid"`
		Color  string `json:"color"`
		Pieces []struct {
			ID            int  `json:"id"`
			Position      int  `json:"position"`
			InBase        bool `json:"inBase"`
			InHome        bool `json:"inHome"`
			OnHomePath    bool `json:"onHomePath"`
			HomePathIndex int  `json:"homePathIndex"`
		} `json:"pieces"`
	} `json:"players"`
	LastRoll int `json:"lastRoll"`
}

var (
	Red       = utils.ParseHexColor("#FF4D4D")
	Green     = utils.ParseHexColor("#2ECC71")
	Yellow    = utils.ParseHexColor("#F1C40F")
	Blue      = utils.ParseHexColor("#3498DB")
	White     = color.White
	Black     = color.Black
	Gray      = utils.ParseHexColor("#CCCCCC")
	LightGray = utils.ParseHexColor("#F0F2F5")
	DarkGray  = utils.ParseHexColor("#333333")
)

// Coordinates from ludo.js [row, col]
var MainTrack = [][2]int{
	{6, 1}, {6, 2}, {6, 3}, {6, 4}, {6, 5},
	{5, 6}, {4, 6}, {3, 6}, {2, 6}, {1, 6}, {0, 6},
	{0, 7}, {0, 8},
	{1, 8}, {2, 8}, {3, 8}, {4, 8}, {5, 8},
	{6, 9}, {6, 10}, {6, 11}, {6, 12}, {6, 13}, {6, 14},
	{7, 14}, {8, 14},
	{8, 13}, {8, 12}, {8, 11}, {8, 10}, {8, 9},
	{9, 8}, {10, 8}, {11, 8}, {12, 8}, {13, 8}, {14, 8},
	{14, 7}, {14, 6},
	{13, 6}, {12, 6}, {11, 6}, {10, 6}, {9, 6},
	{8, 5}, {8, 4}, {8, 3}, {8, 2}, {8, 1}, {8, 0},
	{7, 0}, {6, 0},
}

var HomePaths = map[string][][2]int{
	"red":    {{7, 1}, {7, 2}, {7, 3}, {7, 4}, {7, 5}, {7, 6}},
	"green":  {{1, 7}, {2, 7}, {3, 7}, {4, 7}, {5, 7}, {6, 7}},
	"yellow": {{7, 13}, {7, 12}, {7, 11}, {7, 10}, {7, 9}, {7, 8}},
	"blue":   {{13, 7}, {12, 7}, {11, 7}, {10, 7}, {9, 7}, {8, 7}},
}

var Bases = map[string][][2]int{
	"red":    {{2, 2}, {2, 4}, {4, 2}, {4, 4}},
	"green":  {{2, 10}, {2, 12}, {4, 10}, {4, 12}},
	"yellow": {{10, 10}, {10, 12}, {12, 10}, {12, 12}},
	"blue":   {{10, 2}, {10, 4}, {12, 2}, {12, 4}},
}

var SafeSquares = []int{0, 13, 26, 39}

func RenderBoard(c *gin.Context) {
	var req LudoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	dc := gg.NewContext(BOARD_SIZE, BOARD_SIZE)
	dc.SetColor(LightGray)
	dc.Clear()

	// 1. Draw Background Areas
	cornerSize := 6.0 * CELL_SIZE
	drawArea(dc, 0, 0, cornerSize, cornerSize, Red, 0.35)
	drawArea(dc, BOARD_SIZE-cornerSize, 0, cornerSize, cornerSize, Green, 0.15)
	drawArea(dc, BOARD_SIZE-cornerSize, BOARD_SIZE-cornerSize, cornerSize, cornerSize, Yellow, 0.15)
	drawArea(dc, 0, BOARD_SIZE-cornerSize, cornerSize, cornerSize, Blue, 0.15)

	// Inner white areas
	innerSize := cornerSize * 0.7
	off := (cornerSize - innerSize) / 2
	dc.SetColor(White)
	dc.DrawRectangle(off, off, innerSize, innerSize)
	dc.DrawRectangle(BOARD_SIZE-cornerSize+off, off, innerSize, innerSize)
	dc.DrawRectangle(BOARD_SIZE-cornerSize+off, BOARD_SIZE-cornerSize+off, innerSize, innerSize)
	dc.DrawRectangle(off, BOARD_SIZE-cornerSize+off, innerSize, innerSize)
	dc.Fill()

	// 2. Home Paths Backgrounds
	for colorName, path := range HomePaths {
		dc.SetColor(alphaColor(getColor(colorName), 0.4))
		for _, pos := range path {
			dc.DrawRectangle(float64(pos[1]*CELL_SIZE)+2, float64(pos[0]*CELL_SIZE)+2, CELL_SIZE-4, CELL_SIZE-4)
			dc.Fill()
		}
	}

	// 3. Grid Lines
	dc.SetColor(DarkGray)
	dc.SetLineWidth(5)
	for i := 0; i <= 15; i++ {
		x := float64(i * CELL_SIZE)
		dc.DrawLine(x, 0, x, BOARD_SIZE)
		dc.DrawLine(0, x, BOARD_SIZE, x)
	}
	dc.Stroke()

	// 4. Safe Squares (Stars)
	for _, idx := range SafeSquares {
		pos := MainTrack[idx]
		drawStar(dc, float64(pos[1]*CELL_SIZE)+CELL_SIZE/2, float64(pos[0]*CELL_SIZE)+CELL_SIZE/2, 15, Yellow)
	}

	// 5. Center Finish (Triangles)
	center := 7.5 * CELL_SIZE
	triSize := CELL_SIZE * 1.5
	drawTriangle(dc, center, center, center-triSize, center, center, center-triSize, Red)
	drawTriangle(dc, center, center, center, center-triSize, center+triSize, center, Green)
	drawTriangle(dc, center, center, center+triSize, center, center, center+triSize, Yellow)
	drawTriangle(dc, center, center, center, center+triSize, center-triSize, center, Blue)

	// 6. Base Outlines
	for colorName, positions := range Bases {
		dc.SetColor(getColor(colorName))
		dc.SetLineWidth(3)
		for _, pos := range positions {
			dc.DrawCircle(float64(pos[1]*CELL_SIZE)+CELL_SIZE/2, float64(pos[0]*CELL_SIZE)+CELL_SIZE/2, 22)
			dc.Stroke()
		}
	}

	// 7. Pieces
	for _, p := range req.Players {
		c := getColor(p.Color)
		for _, piece := range p.Pieces {
			if piece.InHome {
				continue
			}

			var coords [2]int
			if piece.InBase {
				coords = Bases[p.Color][piece.ID-1]
			} else if piece.OnHomePath {
				coords = HomePaths[p.Color][piece.HomePathIndex]
			} else {
				coords = MainTrack[piece.Position]
			}

			x := float64(coords[1]*CELL_SIZE) + CELL_SIZE/2
			y := float64(coords[0]*CELL_SIZE) + CELL_SIZE/2

			// Draw Circle
			dc.SetColor(c)
			dc.DrawCircle(x, y, 18)
			dc.Fill()
			dc.SetColor(White)
			dc.SetLineWidth(2)
			dc.DrawCircle(x, y, 18)
			dc.Stroke()

			// Number
			dc.SetColor(Black)
			dc.DrawStringAnchored(fmt.Sprintf("%d", piece.ID), x, y, 0.5, 0.5)
		}
	}

	// 8. Dice
	if req.LastRoll > 0 {
		dc.SetColor(White)
		dc.DrawRectangle(center-30, center-30, 60, 60)
		dc.Fill()
		dc.SetColor(Black)
		dc.SetLineWidth(2)
		dc.DrawRectangle(center-30, center-30, 60, 60)
		dc.Stroke()
		drawDiceDots(dc, center, center, req.LastRoll)
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "Encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}

// Helpers
func drawArea(dc *gg.Context, x, y, w, h float64, c color.RGBA, alpha float64) {
	dc.SetColor(alphaColor(c, alpha))
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()
}

func alphaColor(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, uint8(a * 255)}
}

func getColor(name string) color.RGBA {
	switch name {
	case "red": return Red
	case "green": return Green
	case "yellow": return Yellow
	case "blue": return Blue
	default: return Black
	}
}

func drawStar(dc *gg.Context, x, y, size float64, c color.RGBA) {
	dc.SetColor(c)
	for i := 0; i < 5; i++ {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		x1 := x + math.Cos(angle)*size
		y1 := y + math.Sin(angle)*size
		if i == 0 {
			dc.MoveTo(x1, y1)
		} else {
			dc.LineTo(x1, y1)
		}
		// Inner points
		angle += math.Pi / 5
		dc.LineTo(x+math.Cos(angle)*size/2, y+math.Sin(angle)*size/2)
	}
	dc.ClosePath()
	dc.Stroke()
}

func drawTriangle(dc *gg.Context, x1, y1, x2, y2, x3, y3 float64, c color.RGBA) {
	dc.SetColor(c)
	dc.MoveTo(x1, y1)
	dc.LineTo(x2, y2)
	dc.LineTo(x3, y3)
	dc.ClosePath()
	dc.Fill()
}

func drawDiceDots(dc *gg.Context, x, y float64, val int) {
	spacing := 14.0
	dots := map[int][][2]float64{
		1: {{0, 0}},
		2: {{-spacing, -spacing}, {spacing, spacing}},
		3: {{-spacing, -spacing}, {0, 0}, {spacing, spacing}},
		4: {{-spacing, -spacing}, {-spacing, spacing}, {spacing, -spacing}, {spacing, spacing}},
		5: {{-spacing, -spacing}, {-spacing, spacing}, {0, 0}, {spacing, -spacing}, {spacing, spacing}},
		6: {{-spacing, -spacing}, {-spacing, 0}, {-spacing, spacing}, {spacing, -spacing}, {spacing, 0}, {spacing, spacing}},
	}
	dc.SetColor(Black)
	for _, dot := range dots[val] {
		dc.DrawCircle(x+dot[0], y+dot[1], 5)
		dc.Fill()
	}
}