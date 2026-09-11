package economy

// renderer.go — /api/cards/economy (balance card), Kenney redesign.
// Owner-approved design: download/money_cards/MONEY1_balance_kenney.png

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	CARD_W = 1000
	CARD_H = 600
)

type EconomyCardRequest struct {
	Nickname   string  `json:"nickname"`
	Wallet     float64 `json:"wallet"`
	Bank       float64 `json:"bank"`
	Total      float64 `json:"total"`
	Frozen     float64 `json:"frozen"`
	ZeniSymbol string  `json:"zeniSymbol"`
	Rank       string  `json:"rank"`
	Level      int     `json:"level"`
	PfpUrl     string  `json:"pfpUrl"`
}

func GenerateEconomyCard(c *gin.Context) {
	var req EconomyCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.ZeniSymbol == "" {
		req.ZeniSymbol = "Z"
	}
	if req.Nickname == "" {
		req.Nickname = "Adventurer"
	}

	dc := gg.NewContext(CARD_W, CARD_H)
	drawBase(dc)

	name := sanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	badge := fmt.Sprintf("%s · LVL %d", strings.ToUpper(req.Rank), req.Level)
	drawHead(dc, name, badge, knYellow)

	// === total wealth block ===
	drawCoin(dc, 210, 326, 34, req.ZeniSymbol)
	loadFit(dc, fontFutureNarrow(), 22, "TOTAL WEALTH", 420, 12)
	textLM(dc, "TOTAL WEALTH", 264, 288, rgb(150, 158, 190))
	totalStr := formatFull(req.Total)
	loadFit(dc, fontFuture(), 62, totalStr, 620, 20)
	textLM(dc, totalStr, 264, 332, rgb(246, 214, 112))
	loadFit(dc, fontFutureNarrow(), 22, "ZENI", 200, 12)
	textLM(dc, "ZENI", 264, 380, rgb(200, 176, 110))

	// === wallet / bank panels ===
	kenButtonFlat(dc, 66, 408, 380, 74, knGreen)
	kvPill(dc, 66, 408, 446, 482, "WALLET", formatFull(req.Wallet),
		20, 28, rgb(222, 246, 218), color.RGBA{255, 255, 255, 255}, 30)
	kenButtonFlat(dc, 480, 408, 380, 74, knBlue)
	kvPill(dc, 480, 408, 860, 482, "BANK", formatFull(req.Bank),
		20, 28, rgb(216, 232, 250), color.RGBA{255, 255, 255, 255}, 30)

	// === distribution bar ===
	walletPct, bankPct := 0.0, 0.0
	if req.Total > 0 {
		walletPct = math.Min(1, math.Max(0, req.Wallet/req.Total))
		bankPct = math.Min(1-walletPct, math.Max(0, req.Bank/req.Total))
	}
	loadFit(dc, fontFutureNarrow(), 18, "0% WALLET", 300, 10)
	textLM(dc, fmt.Sprintf("%.0f%% WALLET", walletPct*100), 80, 502, rgb(140, 220, 150))
	textRM(dc, fmt.Sprintf("%.0f%% BANK", bankPct*100), cardW-80, 502, rgb(130, 170, 245))

	dc.SetColor(rgb(20, 22, 40))
	dc.DrawRoundedRectangle(66, 524, cardW-132, 28, 14)
	dc.Fill()
	dc.SetColor(rgb(72, 82, 140))
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(66, 524, cardW-132, 28, 14)
	dc.Stroke()

	innerW := cardW - 132
	fw := innerW * walletPct
	if walletPct > 0 && fw < 14 {
		fw = 14
	}
	if fw > 0 {
		dc.SetColor(rgb(94, 190, 110))
		dc.DrawRoundedRectangle(66, 524, fw, 28, 14)
		dc.Fill()
	}
	bw := innerW - fw
	if bw > 0 {
		dc.SetColor(rgb(86, 130, 230))
		dc.DrawRoundedRectangle(66+fw, 524, bw, 28, 14)
		dc.Fill()
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}
