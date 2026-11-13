package chess

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

// ChessRequest — everything except `fen` is optional; an old-style {fen,lastMove}
// request still renders (default theme + default pieces), so a bot that has not
// been updated keeps working.
type ChessRequest struct {
	FEN      string   `json:"fen"`
	LastMove string   `json:"lastMove"`
	Board    string   `json:"board"`  // theme key (see themes.go)
	Pieces   string   `json:"pieces"` // piece-set key
	Flip     bool     `json:"flip"`   // view from Black's side
	Check    string   `json:"check"`  // square of a king in check, e.g. "e1"
	Moves    []string `json:"moves"`  // legal destination squares to dot (for "show me where this piece can go")
}

// ── caches ──────────────────────────────────────────────────────────────────

var (
	layerMu    sync.Mutex
	layerCache = map[string]*image.RGBA{}
	pieceMu    sync.Mutex
	pieceCache = map[string]image.Image{}
)

func frameFor(cell int) int { return int(math.Round(float64(cell) * 0.44)) }

// staticLayer = frame + textured squares + coordinates (+vignette). It never changes
// for a given (theme, cell, flip), so it is built once and copied per request.
func staticLayer(t *Theme, cell int, flip bool) *image.RGBA {
	key := fmt.Sprintf("%s|%d|%v", t.Key, cell, flip)
	layerMu.Lock()
	if l, ok := layerCache[key]; ok {
		layerMu.Unlock()
		return l
	}
	layerMu.Unlock()

	fr := frameFor(cell)
	bs := cell * 8
	total := bs + fr*2
	img := image.NewRGBA(image.Rect(0, 0, total, total))
	seed := uint32(len(t.Key))*977 + uint32(t.Key[0])

	// frame fill + faint grain
	for y := 0; y < total; y++ {
		for x := 0; x < total; x++ {
			g := 0.96 + 0.08*vnoise(float64(x)*0.6, float64(y)*0.6, seed+55)
			if t.Key == "walnut" {
				g = 0.88 + 0.22*(0.5+0.5*math.Sin(float64(y)*0.21+fbm(float64(x)*0.01, float64(y)*0.004, 3, seed)*12))
			}
			img.SetRGBA(x, y, t.Frame.mul(g).rgba())
		}
	}

	// squares
	for y := 0; y < bs; y++ {
		for x := 0; x < bs; x++ {
			col, row := x/cell, y/cell
			s := shadeCtx{
				x: float64(x) * 100 / float64(cell), y: float64(y) * 100 / float64(cell), // texture space is always the 100px design grid
				lx: float64(x%cell) * 100 / float64(cell), ly: float64(y%cell) * 100 / float64(cell), cell: 100,
				file: col, rank: row, light: (row+col)%2 == 0, seed: seed,
			}
			c := t.square(s)
			if t.Vignette > 0 {
				dx, dy := (float64(x)/float64(bs))-0.5, (float64(y)/float64(bs))-0.5
				d := math.Sqrt(dx*dx+dy*dy) / 0.7071
				k := 1 - t.Vignette*math.Pow(d, 2.4)
				c = c.mul(k) // torch-lit centre, dark corners
			}
			img.SetRGBA(fr+x, fr+y, c.rgba())
		}
	}

	dc := gg.NewContextForRGBA(img)
	drawFrame(dc, t, fr, bs, total)
	drawCoords(dc, t, cell, fr, bs, flip)

	layerMu.Lock()
	layerCache[key] = img
	layerMu.Unlock()
	return img
}

func rgbaA(c rgb, a uint8) color.NRGBA { r := c.rgba(); return color.NRGBA{r.R, r.G, r.B, a} }

func drawFrame(dc *gg.Context, t *Theme, fr, bs, total int) {
	f := float64(fr)
	line := t.FrameLine
	box := func(inset, w float64, c color.Color) {
		dc.SetColor(c)
		dc.SetLineWidth(w)
		dc.DrawRectangle(f-inset, f-inset, float64(bs)+inset*2, float64(bs)+inset*2)
		dc.Stroke()
	}
	scale := f / 44
	switch t.FrameStyle {
	case "gold":
		box(2*scale, 2.5*scale, rgbaA(line, 230))
		box(7*scale, 1.2*scale, rgbaA(line, 150))
	case "studs":
		box(2*scale, 3*scale, rgbaA(line, 235))
		box(8*scale, 1.4*scale, rgbaA(line, 140))
		for _, p := range [][2]float64{{0.5, 0.5}, {float64(total)/f - 0.5, 0.5}, {0.5, float64(total)/f - 0.5}, {float64(total)/f - 0.5, float64(total)/f - 0.5}} {
			cx, cy := p[0]*f, p[1]*f
			dc.SetColor(rgbaA(line, 255))
			dc.DrawCircle(cx, cy, 7*scale)
			dc.Fill()
			dc.SetColor(color.NRGBA{60, 35, 10, 255})
			dc.DrawCircle(cx, cy, 3.2*scale)
			dc.Fill()
		}
		// iron rivets on the mid-edges
		mid := float64(total) / 2
		for _, p := range [][2]float64{{mid, f * 0.5}, {mid, float64(total) - f*0.5}, {f * 0.5, mid}, {float64(total) - f*0.5, mid}} {
			dc.SetColor(color.NRGBA{150, 120, 70, 255})
			dc.DrawCircle(p[0], p[1], 3.6*scale)
			dc.Fill()
		}
	case "ember":
		for i, a := range []uint8{40, 70, 120, 230} {
			box(float64(8-2*i)*scale, (6-1.4*float64(i))*scale, rgbaA(line, a))
		}
	case "neon":
		for i, a := range []uint8{25, 45, 80, 240} {
			box(float64(7-2*i)*scale, (7-1.7*float64(i))*scale, rgbaA(line, a))
		}
	default: // bevel
		box(2*scale, 2*scale, rgbaA(line, 210))
		box(0, 1.2*scale, color.NRGBA{0, 0, 0, 90})
	}
}

func fontPath(t *Theme) string {
	if t.UseFantasy {
		return utils.GetAssetPath("rpgasset", "ui", "fantesy.ttf")
	}
	return utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
}

func drawCoords(dc *gg.Context, t *Theme, cell, fr, bs int, flip bool) {
	face, err := utils.LoadFont(fontPath(t), math.Max(9, float64(cell)*0.22))
	if err != nil {
		return
	}
	dc.SetFontFace(face)
	dc.SetColor(rgbaA(t.CoordColor, 235))
	f := float64(fr)
	for i := 0; i < 8; i++ {
		file, rank := string(rune('a'+i)), fmt.Sprintf("%d", 8-i)
		if flip {
			file, rank = string(rune('a'+(7-i))), fmt.Sprintf("%d", i+1)
		}
		cx := f + float64(i*cell) + float64(cell)/2
		cy := f + float64(i*cell) + float64(cell)/2
		dc.DrawStringAnchored(file, cx, f*0.5, 0.5, 0.35)
		dc.DrawStringAnchored(file, cx, float64(fr+bs)+f*0.5, 0.5, 0.35)
		dc.DrawStringAnchored(rank, f*0.5, cy, 0.5, 0.35)
		dc.DrawStringAnchored(rank, float64(fr+bs)+f*0.5, cy, 0.5, 0.35)
	}
}

// ── pieces ─────────────────────────────────────────────────────────────────

func pieceFile(ch rune) string {
	m := map[rune]string{'P': "white-pawn", 'R': "white-rook", 'N': "white-knight", 'B': "white-bishop", 'Q': "white-queen", 'K': "white-king",
		'p': "black-pawn", 'r': "black-rook", 'n': "black-knight", 'b': "black-bishop", 'q': "black-queen", 'k': "black-king"}
	return m[ch]
}

func piecePath(set PieceSet, name string) string {
	if set.Dir == "" {
		return utils.GetAssetPath("chess", name+".png")
	}
	return utils.GetAssetPath("chess", "sets", set.Dir, name+".png")
}

// scaled piece (and its soft drop-shadow) cached per (set, piece, size)
func getPiece(set PieceSet, name string, size int, shadow bool) image.Image {
	key := fmt.Sprintf("%s|%s|%d|%v", set.Key, name, size, shadow)
	pieceMu.Lock()
	if v, ok := pieceCache[key]; ok {
		pieceMu.Unlock()
		return v
	}
	pieceMu.Unlock()
	src, err := utils.LoadImage(piecePath(set, name))
	if err != nil {
		fmt.Printf("[Chess] missing piece %s/%s: %v\n", set.Key, name, err)
		return nil
	}
	img := imaging.Resize(src, size, size, imaging.Lanczos)
	var out image.Image = img
	if shadow {
		sh := image.NewNRGBA(img.Bounds())
		for i := 3; i < len(img.Pix); i += 4 {
			sh.Pix[i] = uint8(float64(img.Pix[i]) * 0.42) // black, alpha-only silhouette
		}
		out = imaging.Blur(sh, math.Max(1.2, float64(size)/28))
	}
	pieceMu.Lock()
	pieceCache[key] = out
	pieceMu.Unlock()
	return out
}

// ── board render ────────────────────────────────────────────────────────────

func sqToColRow(sq string, flip bool) (int, int, bool) {
	if len(sq) != 2 || sq[0] < 'a' || sq[0] > 'h' || sq[1] < '1' || sq[1] > '8' {
		return 0, 0, false
	}
	f, r := int(sq[0]-'a'), 8-int(sq[1]-'0')
	if flip {
		return 7 - f, 7 - r, true
	}
	return f, r, true
}

func parseFEN(fen string) map[[2]int]rune { // [file, rankIdx(0=rank 8)] -> piece
	out := map[[2]int]rune{}
	parts := strings.Split(fen, " ")
	for r, rowStr := range strings.Split(parts[0], "/") {
		f := 0
		for _, ch := range rowStr {
			if ch >= '1' && ch <= '8' {
				f += int(ch - '0')
			} else {
				out[[2]int{f, r}] = ch
				f++
			}
		}
	}
	return out
}

func renderBoardImage(req ChessRequest, cell int) *image.RGBA {
	t := GetTheme(req.Board)
	set := GetPieceSet(req.Pieces)
	layer := staticLayer(t, cell, req.Flip)
	img := image.NewRGBA(layer.Bounds())
	draw.Draw(img, img.Bounds(), layer, image.Point{}, draw.Src)
	dc := gg.NewContextForRGBA(img)
	fr := float64(frameFor(cell))
	c := float64(cell)
	pos := func(col, row int) (float64, float64) { return fr + float64(col)*c, fr + float64(row)*c }
	pieces := parseFEN(req.FEN)

	// last move
	if len(req.LastMove) >= 4 {
		for _, sq := range []string{req.LastMove[0:2], req.LastMove[2:4]} {
			if col, row, ok := sqToColRow(sq, req.Flip); ok {
				x, y := pos(col, row)
				dc.SetColor(t.LastMove)
				dc.DrawRectangle(x, y, c, c)
				dc.Fill()
			}
		}
	}
	// check glow
	if col, row, ok := sqToColRow(req.Check, req.Flip); ok {
		x, y := pos(col, row)
		g := gg.NewRadialGradient(x+c/2, y+c/2, c*0.08, x+c/2, y+c/2, c*0.62)
		g.AddColorStop(0, color.NRGBA{255, 40, 40, 235})
		g.AddColorStop(0.55, color.NRGBA{230, 30, 30, 140})
		g.AddColorStop(1, color.NRGBA{200, 0, 0, 0})
		dc.SetFillStyle(g)
		dc.DrawRectangle(x, y, c, c)
		dc.Fill()
	}

	// pieces (shadow first so neighbours never overpaint a piece)
	size := int(c * 0.94)
	off := (c - float64(size)) / 2
	for k, ch := range pieces {
		col, row := k[0], k[1]
		if req.Flip {
			col, row = 7-col, 7-row
		}
		x, y := pos(col, row)
		name := pieceFile(ch)
		if name == "" {
			continue
		}
		if sh := getPiece(set, name, size, true); sh != nil {
			dc.DrawImage(sh, int(x+off+c*0.025), int(y+off+c*0.045))
		}
	}
	for k, ch := range pieces {
		col, row := k[0], k[1]
		if req.Flip {
			col, row = 7-col, 7-row
		}
		x, y := pos(col, row)
		name := pieceFile(ch)
		if name == "" {
			continue
		}
		if p := getPiece(set, name, size, false); p != nil {
			dc.DrawImage(p, int(x+off), int(y+off))
		}
	}

	// legal-move hints on top (dot = empty square, ring = capture)
	for _, sq := range req.Moves {
		col, row, ok := sqToColRow(sq, req.Flip)
		if !ok {
			continue
		}
		x, y := pos(col, row)
		f, r := int(sq[0]-'a'), 8-int(sq[1]-'0')
		dc.SetColor(t.Dot)
		if _, occ := pieces[[2]int{f, r}]; occ {
			dc.SetLineWidth(c * 0.075)
			dc.DrawCircle(x+c/2, y+c/2, c*0.43)
			dc.Stroke()
		} else {
			dc.DrawCircle(x+c/2, y+c/2, c*0.16)
			dc.Fill()
		}
	}
	return img
}

// RenderBoard — POST /api/chess
func RenderBoard(c *gin.Context) {
	var req ChessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.FEN) == "" {
		c.JSON(400, gin.H{"error": "fen required"})
		return
	}
	utils.RespondImage(c, renderBoardImage(req, 100))
}
