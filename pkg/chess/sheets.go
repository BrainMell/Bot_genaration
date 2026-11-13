package chess

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const sampleFEN = "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4"

func loadFace(dc *gg.Context, name string, size float64) {
	if f, err := utils.LoadFont(utils.GetAssetPath("rpgasset", "ui", name), size); err == nil {
		dc.SetFontFace(f)
	}
}

func bgGradient(dc *gg.Context, w, h int, top, bottom string) {
	g := gg.NewLinearGradient(0, 0, 0, float64(h))
	g.AddColorStop(0, utils.ParseHexColor(top))
	g.AddColorStop(1, utils.ParseHexColor(bottom))
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()
}

// checker strip used as a header/footer accent
func checkerStrip(dc *gg.Context, x, y, w, h, sq float64, a, b color.Color) {
	n := int(w/sq) + 1
	rows := int(h / sq)
	for r := 0; r < rows; r++ {
		for i := 0; i < n; i++ {
			if (i+r)%2 == 0 {
				dc.SetColor(a)
			} else {
				dc.SetColor(b)
			}
			dc.DrawRectangle(x+float64(i)*sq, y+float64(r)*sq, sq, sq)
			dc.Fill()
		}
	}
}

func cleanName(n string) string {
	var b strings.Builder
	for _, r := range n {
		if r < 0x250 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' || r == '-' || r == '.' || r == '\'') {
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "Player"
	}
	if len(out) > 18 {
		out = out[:17] + ".."
	}
	return out
}

// RenderThemes — POST /api/chess/themes
//
//	{"kind":"boards","pieces":"fantasy","current":"dungeon","hint":".j chess theme <name>"}
//	{"kind":"pieces","board":"walnut","current":"classic","hint":".j chess pieces <name>"}
func RenderThemes(c *gin.Context) {
	var req struct {
		Kind    string `json:"kind"`
		Board   string `json:"board"`
		Pieces  string `json:"pieces"`
		Current string `json:"current"`
		Hint    string `json:"hint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Kind == "pieces" {
		renderPieceSheet(c, req.Board, req.Current, req.Hint)
		return
	}
	renderBoardSheet(c, req.Pieces, req.Current, req.Hint)
}

func renderBoardSheet(c *gin.Context, pieces, current, hint string) {
	const cell, cols, pad = 30, 4, 22
	fr := frameFor(cell)
	bw := cell*8 + fr*2
	labelH := 62
	rows := (len(themeOrder) + cols - 1) / cols
	W := cols*bw + (cols+1)*pad
	H := 96 + rows*(bw+labelH+pad) + 54
	dc := gg.NewContext(W, H)
	bgGradient(dc, W, H, "#1b1c26", "#0e0f15")
	checkerStrip(dc, 0, 0, float64(W), 24, 12, color.NRGBA{255, 255, 255, 14}, color.NRGBA{0, 0, 0, 0})
	loadFace(dc, "fantesy.ttf", 46)
	dc.SetColor(color.NRGBA{244, 214, 130, 255})
	centerText(dc, "BOARD THEMES", float64(W)/2, 74)

	curKey := GetTheme(current).Key
	for i, key := range themeOrder {
		t := themes[key]
		col, row := i%cols, i/cols
		x := float64(pad + col*(bw+pad))
		y := float64(96 + row*(bw+labelH+pad))
		img := renderBoardImage(ChessRequest{FEN: sampleFEN, LastMove: "g8f6", Board: key, Pieces: pieces}, cell)
		dc.DrawImage(img, int(x), int(y))
		if key == curKey {
			dc.SetColor(color.NRGBA{255, 214, 90, 255})
			dc.SetLineWidth(4)
			dc.DrawRectangle(x-3, y-3, float64(bw)+6, float64(bw)+6)
			dc.Stroke()
		}
		loadFace(dc, "Inter-Bold.ttf", 22)
		dc.SetColor(color.White)
		dc.DrawString(fmt.Sprintf("%d  %s", i+1, t.Label), x, y+float64(bw)+28)
		loadFace(dc, "Inter-Medium.ttf", 16)
		dc.SetColor(color.NRGBA{170, 175, 195, 255})
		sub := t.Blurb
		if key == curKey {
			sub = "ACTIVE  -  " + sub
		}
		dc.DrawString(sub, x, y+float64(bw)+50)
	}
	if hint != "" {
		loadFace(dc, "Inter-SemiBold.ttf", 22)
		dc.SetColor(color.NRGBA{244, 214, 130, 255})
		dc.DrawStringAnchored(hint, float64(W)/2, float64(H)-26, 0.5, 0.5)
	}
	utils.RespondImage(c, dc.Image())
}

func renderPieceSheet(c *gin.Context, board, current, hint string) {
	t := GetTheme(board)
	const sq, pad, labelW = 70, 24, 250
	W := pad*2 + labelW + sq*12
	rowH := sq + 22
	H := 100 + len(pieceOrder)*rowH + 56
	dc := gg.NewContext(W, H)
	bgGradient(dc, W, H, "#1b1c26", "#0e0f15")
	loadFace(dc, "fantesy.ttf", 46)
	dc.SetColor(color.NRGBA{244, 214, 130, 255})
	centerText(dc, "PIECE SETS", float64(W)/2, 68)
	order := []rune{'P', 'N', 'B', 'R', 'Q', 'K', 'p', 'n', 'b', 'r', 'q', 'k'}
	curKey := GetPieceSet(current).Key
	for i, key := range pieceOrder {
		set := pieceSets[key]
		y := float64(100 + i*rowH)
		for j, ch := range order {
			x := float64(pad + labelW + j*sq)
			if (i+j)%2 == 0 {
				dc.SetColor(t.Light.rgba())
			} else {
				dc.SetColor(t.Dark.rgba())
			}
			dc.DrawRectangle(x, y, sq, sq)
			dc.Fill()
			if p := getPiece(set, pieceFile(ch), sq-6, false); p != nil {
				dc.DrawImage(p, int(x)+3, int(y)+3)
			}
		}
		loadFace(dc, "Inter-Bold.ttf", 24)
		dc.SetColor(color.White)
		dc.DrawString(fmt.Sprintf("%d  %s", i+1, set.Label), float64(pad), y+30)
		loadFace(dc, "Inter-Medium.ttf", 15)
		sub := set.Credit
		dc.SetColor(color.NRGBA{170, 175, 195, 255})
		if key == curKey {
			sub = "ACTIVE  -  " + sub
			dc.SetColor(color.NRGBA{255, 214, 90, 255})
		}
		dc.DrawString(sub, float64(pad), y+54)
	}
	if hint != "" {
		loadFace(dc, "Inter-SemiBold.ttf", 22)
		dc.SetColor(color.NRGBA{244, 214, 130, 255})
		dc.DrawStringAnchored(hint, float64(W)/2, float64(H)-26, 0.5, 0.5)
	}
	utils.RespondImage(c, dc.Image())
}

// centerText draws on the baseline (the fantasy face reports odd metrics, so anchored drawing drifts)
func centerText(dc *gg.Context, str string, cx, baseline float64) {
	w, _ := dc.MeasureString(str)
	dc.DrawString(str, cx-w/2, baseline)
}
