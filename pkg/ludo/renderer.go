package ludo

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"net/http"
	"strings"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	BOARD_SIZE = 900
	CELL_SIZE  = 60
)

// LudoRequest — POST /api/ludo. PlayerName / IsCurrentTurn are optional
// (2026-09-16 redesign); older payloads from the Node side still render.
type LudoRequest struct {
	Players []struct {
		JID           string `json:"jid"`
		Color         string `json:"color"`
		PfpURL        string `json:"pfpUrl"`
		PlayerName    string `json:"playerName"`
		IsCurrentTurn bool   `json:"isCurrentTurn"`
		Pieces        []struct {
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

// ── Colour system (2026-09-16 redesign) ──────────────────────────────────
// Root cause of the old "mismatched colours": alphaColor() built ILLEGAL
// alpha-premultiplied color.RGBA values (e.g. R=255 with A=89 — premultiplied
// R must be <= A). Go's compositor garbles those into random hues
// (red→cyan, green→orange, yellow→purple). All translucent fills now use
// color.NRGBA, which is the correct non-premultiplied type.
//
// Palette: one heraldic set shared by quadrant, home column, base ring and
// pieces so every player identity is coherent across the whole board.
var (
	// deep tones (pieces, rings, triangles, plate accents)
	Red    = color.RGBA{176, 58, 46, 255}   // heraldic red
	Green  = color.RGBA{30, 132, 73, 255}   // forest green
	Yellow = color.RGBA{183, 149, 11, 255}  // antique gold
	Blue   = color.RGBA{40, 116, 166, 255}  // royal blue
	// tints (quadrant fill, home column, slots)
	RedTint    = color.NRGBA{242, 215, 213, 255}
	GreenTint  = color.NRGBA{212, 239, 223, 255}
	YellowTint = color.NRGBA{252, 243, 207, 255}
	BlueTint   = color.NRGBA{214, 234, 248, 255}
	// board neutrals
	Parchment = color.RGBA{243, 233, 210, 255}
	CellCream = color.RGBA{250, 243, 227, 255}
	GridBrown = color.RGBA{90, 70, 50, 255}
	FrameDark = color.RGBA{62, 44, 28, 255}
	GoldLine  = color.RGBA{201, 162, 39, 255}
	InkDark   = color.RGBA{62, 44, 28, 255}
	White     = color.RGBA{255, 255, 255, 255}
	Black     = color.RGBA{0, 0, 0, 255}
)

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

var HomePaths = map[string][6][2]int{
	"red":    {{7, 1}, {7, 2}, {7, 3}, {7, 4}, {7, 5}, {7, 6}},
	"green":  {{1, 7}, {2, 7}, {3, 7}, {4, 7}, {5, 7}, {6, 7}},
	"yellow": {{7, 13}, {7, 12}, {7, 11}, {7, 10}, {7, 9}, {7, 8}},
	"blue":   {{13, 7}, {12, 7}, {11, 7}, {10, 7}, {9, 7}, {8, 7}},
}

var Bases = map[string][4][2]int{
	"red":    {{2, 2}, {2, 4}, {4, 2}, {4, 4}},
	"green":  {{2, 10}, {2, 12}, {4, 10}, {4, 12}},
	"yellow": {{10, 10}, {10, 12}, {12, 10}, {12, 12}},
	"blue":   {{10, 2}, {10, 4}, {12, 2}, {12, 4}},
}

// Quadrant anchor (top-left cell of each 6x6 base) per colour — red TL,
// green TR, yellow BR, blue BL (matches start order red→green→yellow→blue).
var QuadrantOrigin = map[string][2]int{
	"red":    {0, 0},
	"green":  {0, 9},
	"yellow": {9, 9},
	"blue":   {9, 0},
}

var PfpPositions = map[string]image.Point{
	"red":    {X: 180, Y: 96},
	"green":  {X: 720, Y: 96},
	"yellow": {X: 720, Y: 804},
	"blue":   {X: 180, Y: 804},
}

// Start cell index on MAIN_TRACK per colour (matches Node START_POSITIONS).
var StartIndex = map[string]int{"red": 0, "green": 13, "yellow": 26, "blue": 39}

// ── Entry point ───────────────────────────────────────────────────────────
func RenderBoard(c *gin.Context) {
	var req LudoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	dc := gg.NewContext(BOARD_SIZE, BOARD_SIZE)
	dc.SetColor(Parchment)
	dc.Clear()

	// soft vignette for depth (NRGBA — correct translucency)
	vig := gg.NewLinearGradient(0, 0, BOARD_SIZE, BOARD_SIZE)
	vig.AddColorStop(0, color.NRGBA{80, 55, 30, 26})
	vig.AddColorStop(0.5, color.NRGBA{80, 55, 30, 0})
	vig.AddColorStop(1, color.NRGBA{80, 55, 30, 26})
	dc.SetFillStyle(vig)
	dc.DrawRectangle(0, 0, BOARD_SIZE, BOARD_SIZE)
	dc.Fill()

	cornerSize := 6.0 * CELL_SIZE

	// 1. Quadrants: tint fill + deep inner ring (unified identity per colour)
	for _, col := range []string{"red", "green", "yellow", "blue"} {
		orig := QuadrantOrigin[col]
		x := float64(orig[1] * CELL_SIZE)
		y := float64(orig[0] * CELL_SIZE)
		dc.SetColor(tintOf(col))
		dc.DrawRectangle(x, y, cornerSize, cornerSize)
		dc.Fill()
		// deep-tone inset ring
		dc.SetColor(deepOf(col))
		dc.SetLineWidth(6)
		dc.DrawRectangle(x+7, y+7, cornerSize-14, cornerSize-14)
		dc.Stroke()
		// white play square
		inner := cornerSize * 0.7
		off := (cornerSize - inner) / 2
		dc.SetColor(White)
		dc.DrawRectangle(x+off, y+off, inner, inner)
		dc.Fill()
		dc.SetColor(GridBrown)
		dc.SetLineWidth(2)
		dc.DrawRectangle(x+off, y+off, inner, inner)
		dc.Stroke()
		// slot rings for the 4 base slots
		for _, slot := range Bases[col] {
			cx := float64(slot[1]*CELL_SIZE) + CELL_SIZE/2
			cy := float64(slot[0]*CELL_SIZE) + CELL_SIZE/2
			dc.SetColor(tintOf(col))
			dc.DrawCircle(cx, cy, 24)
			dc.Fill()
			dc.SetColor(deepOf(col))
			dc.SetLineWidth(3)
			dc.DrawCircle(cx, cy, 24)
			dc.Stroke()
		}
	}

	// 2. Track cells: cream fill over the cross shape
	for _, pos := range MainTrack {
		dc.SetColor(CellCream)
		dc.DrawRectangle(float64(pos[1]*CELL_SIZE)+1.5, float64(pos[0]*CELL_SIZE)+1.5, CELL_SIZE-3, CELL_SIZE-3)
		dc.Fill()
	}

	// 3. Home columns: tint fill + deep border on every cell (colour identity)
	for colorName, path := range HomePaths {
		for i, pos := range path {
			dc.SetColor(tintOf(colorName))
			dc.DrawRectangle(float64(pos[1]*CELL_SIZE)+2, float64(pos[0]*CELL_SIZE)+2, CELL_SIZE-4, CELL_SIZE-4)
			dc.Fill()
			if i == 0 {
				// entrance cell gets the deep tone border
				dc.SetColor(deepOf(colorName))
				dc.SetLineWidth(3.5)
				dc.DrawRectangle(float64(pos[1]*CELL_SIZE)+2, float64(pos[0]*CELL_SIZE)+2, CELL_SIZE-4, CELL_SIZE-4)
				dc.Stroke()
			}
		}
	}

	// 4. Grid
	dc.SetColor(GridBrown)
	dc.SetLineWidth(2)
	for i := 0; i <= 15; i++ {
		x := float64(i * CELL_SIZE)
		dc.DrawLine(x, 0, x, BOARD_SIZE)
		dc.DrawLine(0, x, BOARD_SIZE, x)
	}
	dc.Stroke()

	// 5. Safe squares = start cells, marked with the owner's star
	for colorName, idx := range StartIndex {
		pos := MainTrack[idx]
		drawStar(dc, float64(pos[1]*CELL_SIZE)+CELL_SIZE/2, float64(pos[0]*CELL_SIZE)+CELL_SIZE/2, 15, deepOf(colorName))
	}

	// 6. Center finish — triangle per side MATCHES each colour's home column
	//    (red left, green top, yellow right, blue bottom)
	center := 7.5 * CELL_SIZE
	t := CELL_SIZE * 1.5
	drawTriangle(dc, center, center, center-t, center, center, center-t, deepOf("red"))    // left
	drawTriangle(dc, center, center, center, center-t, center+t, center, deepOf("green"))  // top
	drawTriangle(dc, center, center, center+t, center, center, center+t, deepOf("yellow")) // right
	drawTriangle(dc, center, center, center, center+t, center-t, center, deepOf("blue"))   // bottom
	dc.SetColor(GoldLine)
	dc.SetLineWidth(4)
	dc.DrawCircle(center, center, 26)
	dc.Stroke()

	// 7. Pieces (deep fill, dark rim, white number badge)
	for _, p := range req.Players {
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
			drawPiece(dc, x, y, deepOf(p.Color), piece.ID)
		}
	}

	// 8. Player plates (pfp + name + home count) + turn glow
	for _, p := range req.Players {
		homeCount := 0
		for _, piece := range p.Pieces {
			if piece.InHome {
				homeCount++
			}
		}
		if p.IsCurrentTurn {
			orig := QuadrantOrigin[p.Color]
			x := float64(orig[1] * CELL_SIZE)
			y := float64(orig[0] * CELL_SIZE)
			dc.SetColor(GoldLine)
			dc.SetLineWidth(9)
			dc.DrawRectangle(x+3, y+3, cornerSize-6, cornerSize-6)
			dc.Stroke()
		}
		drawPlayerPlate(dc, p.Color, p.PfpURL, sanitizeName(p.PlayerName), homeCount, p.IsCurrentTurn)
	}

	// 9. Dice chip in the center (rounded, gold rim)
	if req.LastRoll > 0 {
		drawDice(dc, center, center, req.LastRoll)
	}

	// outer frame last (crisp border over everything)
	dc.SetColor(FrameDark)
	dc.SetLineWidth(10)
	dc.DrawRectangle(5, 5, BOARD_SIZE-10, BOARD_SIZE-10)
	dc.Stroke()
	dc.SetColor(GoldLine)
	dc.SetLineWidth(3)
	dc.DrawRectangle(14, 14, BOARD_SIZE-28, BOARD_SIZE-28)
	dc.Stroke()

	utils.RespondImage(c, dc.Image())
}

// ── Piece / UI drawing ────────────────────────────────────────────────────
func drawPiece(dc *gg.Context, x, y float64, col color.RGBA, id int) {
	// shadow
	dc.SetColor(color.NRGBA{40, 25, 12, 70})
	dc.DrawCircle(x+2, y+3, 17)
	dc.Fill()
	// body
	dc.SetColor(col)
	dc.DrawCircle(x, y, 17)
	dc.Fill()
	dc.SetColor(White)
	dc.SetLineWidth(2.5)
	dc.DrawCircle(x, y, 17)
	dc.Stroke()
	// number badge
	dc.SetColor(White)
	dc.DrawCircle(x, y, 10)
	dc.Fill()
	dc.SetColor(Black)
	if err := dc.LoadFontFace(utils.GetAssetPath("rpgasset", "ui", "craft/CinzelDecBold.ttf"), 13); err == nil {
		dc.DrawStringAnchored(fmt.Sprintf("%d", id), x, y, 0.5, 0.5)
	}
}

func drawPlayerPlate(dc *gg.Context, playerColor, pfpURL, name string, homeCount int, isTurn bool) {
	pos, ok := PfpPositions[playerColor]
	if !ok {
		return
	}
	px := float64(pos.X)
	py := float64(pos.Y)

	// plate panel
	pw, ph := 230.0, 60.0
	dc.SetColor(color.NRGBA{62, 44, 28, 225})
	dc.DrawRoundedRectangle(px-pw/2, py-ph/2, pw, ph, 12)
	dc.Fill()
	dc.SetColor(GoldLine)
	if isTurn {
		dc.SetLineWidth(4)
	} else {
		dc.SetLineWidth(2)
	}
	dc.DrawRoundedRectangle(px-pw/2, py-ph/2, pw, ph, 12)
	dc.Stroke()

	// pfp circle at plate left
	r := 22.0
	cxp := px - pw/2 + 34
	dc.SetColor(deepOf(playerColor))
	dc.DrawCircle(cxp, py, r+3)
	dc.Fill()

	var pfp image.Image
	var err error
	if len(pfpURL) > 4 && strings.HasPrefix(pfpURL, "http") {
		pfp, err = downloadImage(pfpURL)
	} else if pfpURL != "" {
		pfp, err = utils.LoadImage(pfpURL)
	}
	if err == nil && pfp != nil {
		pfp = imaging.Resize(pfp, int(r*2), int(r*2), imaging.Lanczos)
		pfp = makeCircular(pfp)
		dc.DrawImageAnchored(pfp, int(cxp), int(py), 0.5, 0.5)
	} else {
		// silhouette placeholder
		dc.SetColor(tintOf(playerColor))
		dc.DrawCircle(cxp, py, r)
		dc.Fill()
		dc.SetColor(deepOf(playerColor))
		dc.DrawCircle(cxp, py-6, 8)
		dc.Fill()
		dc.DrawCircle(cxp, py+12, 13)
		dc.Fill()
	}

	// name + home count
	if err := dc.LoadFontFace(utils.GetAssetPath("rpgasset", "ui", "craft/CinzelDecBold.ttf"), 17); err == nil {
		dc.SetColor(White)
		dc.DrawStringAnchored(name, px+18, py-8, 0, 0.5)
	}
	if err := dc.LoadFontFace(utils.GetAssetPath("rpgasset", "ui", "craft/Cinzel.ttf"), 13); err == nil {
		dc.SetColor(tintOf(playerColor))
		dc.DrawStringAnchored(fmt.Sprintf("HOME %d/4", homeCount), px+18, py+12, 0, 0.5)
	}
	if isTurn {
		if err := dc.LoadFontFace(utils.GetAssetPath("rpgasset", "ui", "craft/CinzelDecBold.ttf"), 12); err == nil {
			dc.SetColor(GoldLine)
			dc.DrawStringAnchored("★ TURN", px+pw/2-10, py-ph/2-2, 1, 0.5)
		}
	}
}

func drawDice(dc *gg.Context, x, y float64, val int) {
	s := 34.0
	dc.SetColor(color.NRGBA{40, 25, 12, 90})
	dc.DrawRoundedRectangle(x-s+3, y-s+4, s*2, s*2, 10)
	dc.Fill()
	dc.SetColor(White)
	dc.DrawRoundedRectangle(x-s, y-s, s*2, s*2, 10)
	dc.Fill()
	dc.SetColor(GoldLine)
	dc.SetLineWidth(3.5)
	dc.DrawRoundedRectangle(x-s, y-s, s*2, s*2, 10)
	dc.Stroke()
	drawDiceDots(dc, x, y, val)
}

func sanitizeName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "Player"
	}
	// strip control chars
	var b strings.Builder
	for _, r := range n {
		if r >= 32 && r != 127 {
			b.WriteRune(r)
		}
	}
	n = b.String()
	if len(n) > 16 {
		n = n[:15] + "…"
	}
	return n
}

// ── Helpers ───────────────────────────────────────────────────────────────
func deepOf(name string) color.RGBA {
	switch name {
	case "red":
		return Red
	case "green":
		return Green
	case "yellow":
		return Yellow
	case "blue":
		return Blue
	default:
		return Black
	}
}

func tintOf(name string) color.NRGBA {
	switch name {
	case "red":
		return RedTint
	case "green":
		return GreenTint
	case "yellow":
		return YellowTint
	case "blue":
		return BlueTint
	default:
		return color.NRGBA{200, 200, 200, 255}
	}
}

// translucent MUST return NRGBA — RGBA is premultiplied and illegal for
// arbitrary (r,g,b,a) combos (the original colour-mismatch bug).
func translucent(c color.RGBA, a float64) color.NRGBA {
	return color.NRGBA{c.R, c.G, c.B, uint8(a * 255)}
}

func makeCircular(img image.Image) image.Image {
	bounds := img.Bounds()
	size := bounds.Dx()
	radius := float64(size) / 2
	center := radius

	dst := image.NewRGBA(bounds)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= radius {
				dst.Set(x, y, img.At(x, y))
			}
		}
	}
	return dst
}

func downloadImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func drawStar(dc *gg.Context, x, y, size float64, c color.RGBA) {
	dc.SetColor(White)
	dc.SetLineWidth(2)
	drawStarPath(dc, x, y, size)
	dc.Stroke()
	dc.SetColor(c)
	drawStarPath(dc, x, y, size-2)
	dc.Fill()
}

func drawStarPath(dc *gg.Context, x, y, size float64) {
	for i := 0; i < 5; i++ {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		x1 := x + math.Cos(angle)*size
		y1 := y + math.Sin(angle)*size
		if i == 0 {
			dc.MoveTo(x1, y1)
		} else {
			dc.LineTo(x1, y1)
		}
		angle += math.Pi / 5
		dc.LineTo(x+math.Cos(angle)*size/2, y+math.Sin(angle)*size/2)
	}
	dc.ClosePath()
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
	spacing := 11.0
	dots := map[int][][2]float64{
		1: {{0, 0}},
		2: {{-spacing, -spacing}, {spacing, spacing}},
		3: {{-spacing, -spacing}, {0, 0}, {spacing, spacing}},
		4: {{-spacing, -spacing}, {-spacing, spacing}, {spacing, -spacing}, {spacing, spacing}},
		5: {{-spacing, -spacing}, {-spacing, spacing}, {0, 0}, {spacing, -spacing}, {spacing, spacing}},
		6: {{-spacing, -spacing}, {-spacing, 0}, {-spacing, spacing}, {spacing, -spacing}, {spacing, 0}, {spacing, spacing}},
	}
	dc.SetColor(InkDark)
	for _, dot := range dots[val] {
		dc.DrawCircle(x+dot[0], y+dot[1], 4.5)
		dc.Fill()
	}
}
