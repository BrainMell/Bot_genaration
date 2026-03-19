package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"image-service/pkg/cards"
	"image-service/pkg/chess"
	"image-service/pkg/combat"
	"image-service/pkg/ludo"
	"image-service/pkg/scraper"
	"image-service/pkg/ttt"
)

func main() {
	// MODE controls which routes this instance registers.
	// MODE=render    → lightweight image generation only (no Chrome)
	// MODE=codespace → heavy scraping + card GIF only (Chrome required)
	// MODE=full      → everything (default, backwards compatible)
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "full"
	}

	fmt.Printf("🚀 Go Image & Scraper Service [MODE=%s]\n", mode)

	// Only init the browser when we actually need it
	if mode == "codespace" || mode == "full" {
		fmt.Println("🚀 Chrome-Enhanced, API-driven scraping")
		scraper.InitBrowser()
		defer scraper.CloseBrowser()
	}

	if err := os.MkdirAll("downloads", 0755); err != nil {
		log.Fatalf("failed to create downloads dir: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	port := os.Getenv("PORT")
	if port == "" {
		port = "7860"
	}

	r := gin.Default()
	r.Static("/downloads", "./downloads")

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	// ── Root & Health (always available) ──────────────────────────────────────
	r.GET("/", func(c *gin.Context) {
		features := []string{}
		if mode == "render" || mode == "full" {
			features = append(features,
				"Combat Image Generation",
				"Game Boards (Ludo, TTT, Chess)",
				"Card Burn & Convert (FFmpeg)",
			)
		}
		if mode == "codespace" || mode == "full" {
			features = append(features,
				"Browser-based Pinterest & PornPics",
				"Deep Rule34 Scrape",
				"Rod-powered VS Battles Powerscaling",
				"YouTube Audio Search & DL",
				"Anime Corner News & Anikai Resolver",
				"Animated Card GIFs",
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "online",
			"service":  "Go Image & Scraper Service",
			"version":  "4.0.0",
			"mode":     mode,
			"features": features,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": mode})
	})

	api := r.Group("/api")

	// ── RENDER routes (lightweight — no Chrome) ───────────────────────────────
	if mode == "render" || mode == "full" {
		api.POST("/combat", combat.GenerateCombatImage)
		api.POST("/combat/endscreen", combat.GenerateEndScreen)
		api.POST("/ludo", ludo.RenderBoard)
		api.POST("/ttt", ttt.RenderBoard)
		api.POST("/ttt/leaderboard", ttt.RenderLeaderboard)
		api.POST("/chess", chess.RenderBoard)
		api.POST("/cards/burn", cards.GenerateBurnGif)
		api.POST("/cards/convert", cards.ConvertCard)

		// Lightweight scrapers (no Chrome — API/HTTP only)
		api.GET("/scrape/stickers", scraper.SearchStickers)
		api.GET("/scrape/rule34", scraper.SearchRule34) // uses Gelbooru DAPI
	}

	// ── CODESPACE routes (heavy — Chrome required) ────────────────────────────
	if mode == "codespace" || mode == "full" {
		api.POST("/cards/gif", cards.GenerateCardGif)

		scrape := api.Group("/scrape")
		{
			scrape.GET("/pinterest", scraper.ScrapePinterest)
			scrape.GET("/powerscale", scraper.ScrapePowerscale)
			scrape.GET("/powerscale/fetch", scraper.ScrapePowerscalePage)
			scrape.GET("/pornpics", scraper.ScrapePornPics)
			scrape.GET("/audio", scraper.ScrapeAudio)
			scrape.GET("/anikai", scraper.ScrapeAnikai)
			scrape.GET("/news", scraper.ScrapeAnimeNews)
			// Deep Chrome rule34 — overrides the Gelbooru-only version when
			// running in codespace/full mode (registered last so it wins)
			scrape.GET("/rule34/deep", scraper.ScrapeRule34)
		}
	}

	log.Printf("🚀 Go Service starting on port %s [MODE=%s]", port, mode)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}