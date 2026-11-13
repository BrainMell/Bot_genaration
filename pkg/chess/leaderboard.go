package chess

import (
	"fmt"
	"image/color"
	"math"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

type lbEntry struct {
	Name   string `json:"name"`
	JID    string `json:"jid"`
	Elo    int    `json:"elo"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Draws  int    `json:"draws"`
}

// RenderLeaderboard — POST /api/chess/leaderboard  {"title":"...","entries":[...]}  (already sorted by the bot)
func RenderLeaderboard(c *gin.Context) {
	var req struct {
		Title   string    `json:"title"`
		Entries []lbEntry `json:"entries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if len(req.Entries) > 10 {
		req.Entries = req.Entries[:10]
	}
	const W = 900
	rowH, top := 88, 190
	H := top + len(req.Entries)*rowH + 70
	if len(req.Entries) == 0 {
		H = top + 170
	}
	dc := gg.NewContext(W, H)
	bgGradient(dc, W, H, "#1e2230", "#0c0e14")

	// header: checker band + title
	checkerStrip(dc, 0, 0, W, 120, 40, color.NRGBA{255, 255, 255, 16}, color.NRGBA{0, 0, 0, 40})
	g := gg.NewLinearGradient(0, 0, 0, 150)
	g.AddColorStop(0, color.NRGBA{14, 15, 22, 0})
	g.AddColorStop(1, color.NRGBA{14, 15, 22, 255})
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, W, 150)
	dc.Fill()
	title := req.Title
	if title == "" {
		title = "CHESS LEADERBOARD"
	}
	loadFace(dc, "fantesy.ttf", 58)
	dc.SetColor(color.NRGBA{244, 214, 130, 255})
	centerText(dc, title, W/2, 86)
	loadFace(dc, "Inter-Medium.ttf", 20)
	dc.SetColor(color.NRGBA{160, 168, 190, 255})
	dc.DrawStringAnchored("ranked by Elo  -  everyone starts at 1200", W/2, 122, 0.5, 0.5)

	if len(req.Entries) == 0 {
		loadFace(dc, "Inter-SemiBold.ttf", 30)
		dc.SetColor(color.NRGBA{200, 205, 220, 255})
		dc.DrawStringAnchored("No ranked games yet - be the first!", W/2, float64(top)+60, 0.5, 0.5)
		utils.RespondImage(c, dc.Image())
		return
	}

	medal := []color.NRGBA{{236, 190, 60, 255}, {196, 202, 214, 255}, {205, 127, 70, 255}}
	maxElo, minElo := req.Entries[0].Elo, req.Entries[len(req.Entries)-1].Elo
	span := math.Max(1, float64(maxElo-minElo))
	for i, e := range req.Entries {
		y := float64(top + i*rowH)
		// row card
		if i%2 == 0 {
			dc.SetColor(color.NRGBA{255, 255, 255, 10})
		} else {
			dc.SetColor(color.NRGBA{255, 255, 255, 4})
		}
		dc.DrawRoundedRectangle(28, y, W-56, float64(rowH)-10, 14)
		dc.Fill()
		if i < 3 {
			dc.SetColor(color.NRGBA{medal[i].R, medal[i].G, medal[i].B, 255})
			dc.DrawRoundedRectangle(28, y, 7, float64(rowH)-10, 3)
			dc.Fill()
		}
		// rank badge
		cx, cy := 96.0, y+float64(rowH-10)/2
		if i < 3 {
			dc.SetColor(medal[i])
		} else {
			dc.SetColor(color.NRGBA{58, 64, 84, 255})
		}
		dc.DrawCircle(cx, cy, 29)
		dc.Fill()
		loadFace(dc, "Inter-Bold.ttf", 30)
		if i < 3 {
			dc.SetColor(color.NRGBA{30, 22, 5, 255})
		} else {
			dc.SetColor(color.White)
		}
		dc.DrawStringAnchored(fmt.Sprintf("%d", i+1), cx, cy, 0.5, 0.35)
		// name + record
		loadFace(dc, "Inter-Bold.ttf", 33)
		dc.SetColor(color.White)
		dc.DrawString(cleanName(e.Name), 148, y+37)
		games := e.Wins + e.Losses + e.Draws
		wr := 0
		if games > 0 {
			wr = int(math.Round(float64(e.Wins) * 100 / float64(games)))
		}
		loadFace(dc, "Inter-Medium.ttf", 20)
		dc.SetColor(color.NRGBA{150, 200, 150, 255})
		dc.DrawString(fmt.Sprintf("%dW", e.Wins), 148, y+66)
		dc.SetColor(color.NRGBA{215, 130, 130, 255})
		dc.DrawString(fmt.Sprintf("%dL", e.Losses), 205, y+66)
		dc.SetColor(color.NRGBA{170, 175, 195, 255})
		dc.DrawString(fmt.Sprintf("%dD   %d%% win", e.Draws, wr), 258, y+66)
		// elo bar + number
		bw := 170.0
		bx := float64(W) - 56 - 190 - bw
		dc.SetColor(color.NRGBA{255, 255, 255, 18})
		dc.DrawRoundedRectangle(bx, cy-6, bw, 12, 6)
		dc.Fill()
		frac := 0.25 + 0.75*float64(e.Elo-minElo)/span
		if len(req.Entries) == 1 {
			frac = 1
		}
		dc.SetColor(color.NRGBA{244, 214, 130, 255})
		dc.DrawRoundedRectangle(bx, cy-6, bw*frac, 12, 6)
		dc.Fill()
		loadFace(dc, "Inter-Bold.ttf", 40)
		dc.SetColor(color.NRGBA{244, 214, 130, 255})
		dc.DrawStringAnchored(fmt.Sprintf("%d", e.Elo), float64(W)-48, cy-6, 1, 0.35)
		loadFace(dc, "Inter-Medium.ttf", 16)
		dc.SetColor(color.NRGBA{150, 156, 178, 255})
		dc.DrawStringAnchored("ELO", float64(W)-48, cy+24, 1, 0.5)
	}
	utils.RespondImage(c, dc.Image())
}
