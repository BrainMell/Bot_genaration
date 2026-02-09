package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"image-service/pkg/combat"
	"image-service/pkg/ludo"
	"image-service/pkg/scraper"
	"image-service/pkg/ttt"
)

func main() {
	fmt.Println("🚀 Go Image & Scraper Service")
	fmt.Println("📌 Ultra-low RAM, API-driven scraping")

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// Global Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	// Root Endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "online",
			"service": "Go Image & Scraper Service",
			"version": "2.2.0",
			"features": []string{
				"DuckDuckGo image search",
				"Klipy v1 GIF/sticker API",
				"Robust VS Battles extraction",
				"Rule34 Web Fallback",
			},
		})
	})

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API Group
	api := r.Group("/api")
	{
		// Combat
		api.POST("/combat", combat.GenerateCombatImage)
		api.POST("/combat/endscreen", combat.GenerateEndScreen)

		// Games
		api.POST("/ludo", ludo.RenderBoard)
		api.POST("/ttt", ttt.RenderBoard)
		api.POST("/ttt/leaderboard", ttt.RenderLeaderboard)

		// Scrapers
		scrape := api.Group("/scrape")
		{
			scrape.GET("/pinterest", scraper.SearchPinterest)
			scrape.GET("/stickers", scraper.SearchStickers)
			scrape.GET("/vsbattles/search", scraper.SearchVSBattles)
			scrape.GET("/vsbattles/detail", scraper.GetVSBattlesDetail)
			scrape.GET("/rule34", scraper.SearchRule34)
		}
	}

	log.Printf("🚀 Go Service starting on port %s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}