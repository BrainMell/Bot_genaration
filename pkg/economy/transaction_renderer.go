package economy

// transaction_renderer.go — /api/cards/transaction, Kenney redesign.
// Owner-approved designs: MONEY2_transfer_kenney.png / MONEY3_withdraw_kenney.png
// Extended to DEPOSIT (from->to, blue), CRAFT/BREW/COOK/FORGE (item plate).

import (
	"fmt"
	"strings"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

const (
	TRANS_W = 1000
	TRANS_H = 600
)

type TransactionCardRequest struct {
	Nickname   string  `json:"nickname"`
	Type       string  `json:"type"` // "DEPOSIT", "WITHDRAW", "TRANSFER", "CRAFT", "BREW", "COOK", "FORGE", "FISH", "DECREE"
	Amount     float64 `json:"amount"`
	NewWallet  float64 `json:"newWallet"`
	NewBank    float64 `json:"newBank"`
	ZeniSymbol string  `json:"zeniSymbol"`
	PfpUrl     string  `json:"pfpUrl"`
	ItemName   string  `json:"itemName"`
	SealText   string  `json:"sealText"` // DECREE: rank letter for the wax seal
	Details    string  `json:"details"`  // DECREE ledger "F -> S"; FISH caption override
}

func GenerateTransactionCard(c *gin.Context) {
	var req TransactionCardRequest
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

	txType := strings.ToUpper(strings.TrimSpace(req.Type))
	if txType == "" {
		txType = "TRANSFER"
	}
	acc := txStyle(txType)

	// CRAFT / BREW / COOK / FORGE / FISH get their own RPG-style card (not Kenney)
	if isCraftType(txType) {
		dc := gg.NewContext(TRANS_W, TRANS_H)
		drawRPGCraft(dc, req, txType)
		buf, err := utils.EncodeImageToBuffer(dc.Image())
		if err != nil {
			c.JSON(500, gin.H{"error": "encode failed"})
			return
		}
		c.Data(200, "image/png", buf)
		return
	}

	// DECREE — bright-parchment rank-up card (royal decree family)
	if txType == "DECREE" {
		dc := gg.NewContext(TRANS_W, TRANS_H)
		drawRPGDecree(dc, req)
		buf, err := utils.EncodeImageToBuffer(dc.Image())
		if err != nil {
			c.JSON(500, gin.H{"error": "encode failed"})
			return
		}
		c.Data(200, "image/png", buf)
		return
	}

	dc := gg.NewContext(TRANS_W, TRANS_H)
	drawBase(dc)

	name := sanitize(req.Nickname)
	if name == "" {
		name = "Adventurer"
	}
	// badge: post-transaction total (payload carries no rank/level)
	badge := fmt.Sprintf("%s %s", req.ZeniSymbol, formatFull(req.NewWallet+req.NewBank))
	drawHead(dc, name, badge, acc.set)

	// title plate
	drawTitlePlate(dc, fmt.Sprintf("%s SUCCESSFUL", txType), acc.set, acc.titleCol)

	switch {
	case txType == "TRANSFER" || txType == "DEPOSIT":
		// big amount
		drawCoin(dc, cardW/2-180, 316, 40, req.ZeniSymbol)
		amount := formatFull(req.Amount)
		loadFit(dc, fontFuture(), 70, amount, 380, 20)
		aw, _ := dc.MeasureString(amount) // measure at the FINAL (70pt Future) face
		textLM(dc, amount, cardW/2-118, 318, acc.amountCol)
		loadFit(dc, fontFutureNarrow(), 26, "ZENI", 200, 12)
		textLM(dc, "ZENI", cardW/2-118+aw+28, 318, rgb(200, 176, 110))

		// from wallet -> to bank (values = balances AFTER the tx)
		fpw, fph := 330.0, 84.0
		kenButtonFlat(dc, 100, 416, fpw, fph, knGrey)
		kvPill(dc, 100, 416, 430, 500, "FROM · WALLET", formatFull(req.NewWallet),
			18, 28, rgb(128, 132, 152), rgb(56, 60, 84), 26)
		drawArrow(dc, cardW/2, 458)
		kenButtonFlat(dc, cardW-100-fpw, 416, fpw, fph, knBlue)
		kvPill(dc, cardW-100-fpw, 416, cardW-100, 500, "TO · BANK", formatFull(req.NewBank),
			18, 28, rgb(178, 208, 240), rgb(255, 255, 255), 26)

		drawCaption(dc, fmt.Sprintf(".j %s %s", strings.ToLower(txType), amount))

	case txType == "WITHDRAW":
		amount := "-" + formatFull(req.Amount)
		drawAmountLine(dc, amount, 318, acc.amountCol, 420)

		// rows show the balances AFTER the withdrawal
		rows := []struct {
			label, delta, value string
			set                 kenSet
		}{
			{"BANK", "-" + formatFull(req.Amount), formatFull(req.NewBank), knRed},
			{"WALLET", "+" + formatFull(req.Amount), formatFull(req.NewWallet), knGreen},
		}
		y := 420.0
		for _, row := range rows {
			kenButtonFlat(dc, 70, y, 860, 60, row.set)
			cy := faceCY(y, 60)
			loadFit(dc, fontFutureNarrow(), 22, row.label, 240, 10)
			textLM(dc, row.label, 100, cy, rgb(30, 32, 40))
			loadFit(dc, fontFuture(), 28, row.delta, 220, 12)
			textRM(dc, row.delta, 560, cy, rgb(30, 32, 40))
			loadFit(dc, fontFutureNarrow(), 16, "NEW", 120, 9)
			textLM(dc, "NEW", 600, cy, rgb(52, 56, 68))
			loadFit(dc, fontFuture(), 28, row.value, 260, 12)
			textRM(dc, row.value, 900, cy, rgb(20, 22, 30))
			y += 76
		}
		// no caption here — the approved mock ends with the WALLET row

	default: // CRAFT / BREW / COOK / FORGE and unknown types
		item := sanitize(req.ItemName)
		if item == "" {
			item = "Unknown Item"
		}
		loadFit(dc, fontFutureNarrow(), 22, "ITEM CREATED", 420, 12)
		textCenter(dc, "ITEM CREATED", cardW/2, 272, rgb(150, 158, 190))

		display := item
		if req.Amount > 1 {
			display = fmt.Sprintf("%s X%d", item, int(req.Amount))
		}
		loadFit(dc, fontFuture(), 70, display, 680, 20)
		textCenter(dc, display, cardW/2, 330, acc.amountCol)
		drawCaption(dc, fmt.Sprintf(".j %s %s", strings.ToLower(txType), item))
	}

	buf, err := utils.EncodeImageToBuffer(dc.Image())
	if err != nil {
		c.JSON(500, gin.H{"error": "encode failed"})
		return
	}
	c.Data(200, "image/png", buf)
}
