package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"image-service/pkg/cards"
	"image-service/pkg/chess"
	"image-service/pkg/combat"
	"image-service/pkg/ludo"
	"image-service/pkg/scraper"
	"image-service/pkg/ttt"
)

// =============================================================================
// CONFIG
// =============================================================================

// MODE controls which routes this instance registers.
//
//	MODE=render    → lightweight image generation + built-in orchestrator for heavy routes
//	MODE=codespace → heavy scraping + card GIF only (Chrome required, no orchestrator)
//	MODE=full      → everything locally (dev/testing only)
var (
	mode         string
	codespaceName string
	githubToken  string
	codespaceURL string // Cloudflare tunnel URL — update in Render env after each wake
	upstashURL   string
	upstashToken string
	proxyClient  = &http.Client{Timeout: 120 * time.Second}
)

// Routes that must be forwarded to the Codespace when running in render mode.
// NOTE: /api/cards/gif is handled separately below to avoid duplicate
// registration in full mode.
var heavyRoutes = []string{
	"/api/scrape/pinterest",
	"/api/scrape/pornpics",
	"/api/scrape/audio",
	"/api/scrape/powerscale",
	"/api/scrape/anikai",
	"/api/scrape/news",
	"/api/scrape/rule34/deep",
}

// Cache TTLs in seconds
var cacheTTL = map[string]int{
	"pinterest":  3600,
	"pornpics":   3600,
	"rule34":     3600,
	"powerscale": 86400,
	"anikai":     86400,
	"news":       1800,
	"audio":      86400,
	"gif":        86400,
}

// =============================================================================
// REDIS (Upstash REST API — no extra driver needed)
// =============================================================================

type cacheEntry struct {
	ContentType string `json:"contentType"`
	Data        string `json:"data"`   // base64 for binary, raw string for JSON
	IsBinary    bool   `json:"binary"`
}

func cacheGet(key string) (*cacheEntry, error) {
	if upstashURL == "" {
		return nil, nil
	}
	req, _ := http.NewRequest("GET", upstashURL+"/get/"+url.PathEscape(key), nil)
	req.Header.Set("Authorization", "Bearer "+upstashToken)

	resp, err := proxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result *string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Result == nil {
		return nil, nil
	}

	var entry cacheEntry
	if err := json.Unmarshal([]byte(*result.Result), &entry); err != nil {
		return nil, nil
	}
	return &entry, nil
}

func cacheSet(key string, entry cacheEntry, ttlSecs int) {
	if upstashURL == "" {
		return
	}
	data, _ := json.Marshal(entry)
	reqURL := fmt.Sprintf("%s/set/%s?ex=%d", upstashURL, url.PathEscape(key), ttlSecs)

	req, _ := http.NewRequest("POST", reqURL, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+upstashToken)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := proxyClient.Do(req)
	if err != nil {
		fmt.Printf("[Cache] SET error: %v\n", err)
		return
	}
	resp.Body.Close()
}

func getCacheKey(path string, query url.Values, body []byte) string {
	key := "goservice:" + path + ":" + query.Encode() + ":" + string(body)
	if len(key) > 512 {
		key = key[:512]
	}
	return key
}

func getTTL(path string) int {
	for route, ttl := range cacheTTL {
		if strings.Contains(path, route) {
			return ttl
		}
	}
	return 3600
}

// =============================================================================
// CODESPACE ORCHESTRATION
// =============================================================================

func getCodespaceStatus() (string, error) {
	req, _ := http.NewRequest("GET",
		"https://api.github.com/user/codespaces/"+codespaceName, nil)
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := proxyClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		State string `json:"state"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.State, nil
}

func startCodespace() error {
	req, _ := http.NewRequest("POST",
		"https://api.github.com/user/codespaces/"+codespaceName+"/start", nil)
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := proxyClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func stopCodespace() {
	req, _ := http.NewRequest("POST",
		"https://api.github.com/user/codespaces/"+codespaceName+"/stop", nil)
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := proxyClient.Do(req)
	if err != nil {
		fmt.Printf("[Codespace] Stop error (non-fatal): %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("[Codespace] Stop signal sent.")
}

// wakeAndWait starts the Codespace if needed and polls until the
// Go service inside it responds healthy. Returns error if timeout.
func wakeAndWait() error {
	if codespaceURL == "" {
		return fmt.Errorf("CODESPACE_SERVICE_URL not set — update it in Render env after each Codespace wake")
	}

	status, err := getCodespaceStatus()
	if err != nil {
		return fmt.Errorf("could not get Codespace status: %v", err)
	}
	fmt.Printf("[Codespace] Status: %s\n", status)

	if status != "Available" {
		fmt.Println("[Codespace] Sending start signal...")
		if err := startCodespace(); err != nil {
			return fmt.Errorf("failed to start Codespace: %v", err)
		}
		// Poll up to 90s for Available state
		for i := 0; i < 45; i++ {
			time.Sleep(2 * time.Second)
			s, _ := getCodespaceStatus()
			fmt.Printf("[Codespace] Waiting... (%s)\n", s)
			if s == "Available" {
				break
			}
		}
	}

	// Poll Go service health up to 60s
	healthURL := codespaceURL + "/health"
	fmt.Printf("[Codespace] Waiting for service health at %s\n", healthURL)
	for i := 0; i < 30; i++ {
		resp, err := proxyClient.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			fmt.Println("[Codespace] ✅ Go service is healthy!")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("Codespace Go service did not become healthy in time")
}

// =============================================================================
// HEAVY REQUEST HANDLER
// =============================================================================

func handleHeavy(c *gin.Context) {
	path  := c.Request.URL.Path
	query := c.Request.URL.Query()
	method := c.Request.Method

	bodyBytes, _ := io.ReadAll(c.Request.Body)

	// 1. Cache check
	cacheKey := getCacheKey(path, query, bodyBytes)
	entry, _ := cacheGet(cacheKey)
	if entry != nil {
		fmt.Printf("[Cache] HIT for %s\n", path)
		c.Header("X-Cache", "HIT")
		if entry.IsBinary {
			decoded, err := base64.StdEncoding.DecodeString(entry.Data)
			if err == nil {
				c.Data(200, entry.ContentType, decoded)
				return
			}
		}
		c.Header("Content-Type", entry.ContentType)
		c.String(200, entry.Data)
		return
	}

	// 2. Cache miss — wake Codespace
	fmt.Printf("[Cache] MISS for %s — waking Codespace...\n", path)
	c.Header("X-Cache", "MISS")

	if err := wakeAndWait(); err != nil {
		c.JSON(503, gin.H{"error": "Codespace unavailable", "details": err.Error()})
		return
	}

	// 3. Forward request
	targetURL := codespaceURL + path
	if query.Encode() != "" {
		targetURL += "?" + query.Encode()
	}

	proxyReq, err := http.NewRequest(method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to build proxy request"})
		return
	}
	if ct := c.GetHeader("Content-Type"); ct != "" {
		proxyReq.Header.Set("Content-Type", ct)
	}

	resp, err := proxyClient.Do(proxyReq)
	if err != nil {
		c.JSON(502, gin.H{"error": "Codespace request failed", "details": err.Error()})
		go stopCodespace()
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "Failed to read Codespace response"})
		go stopCodespace()
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	// 4. Send response to client immediately
	c.Data(resp.StatusCode, contentType, respBody)

	// 5. Cache + stop Codespace in background (non-blocking)
	go func() {
		ttl := getTTL(path)
		isBinary := strings.HasPrefix(contentType, "image/") ||
			strings.HasPrefix(contentType, "video/") ||
			strings.HasPrefix(contentType, "audio/")

		if isBinary {
			cacheSet(cacheKey, cacheEntry{
				ContentType: contentType,
				Data:        base64.StdEncoding.EncodeToString(respBody),
				IsBinary:    true,
			}, ttl)
		} else {
			cacheSet(cacheKey, cacheEntry{
				ContentType: contentType,
				Data:        string(respBody),
				IsBinary:    false,
			}, ttl)
		}
		fmt.Printf("[Cache] Stored %s (ttl=%ds)\n", path, ttl)
		stopCodespace()
	}()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	mode          = os.Getenv("MODE")
	codespaceName = os.Getenv("CODESPACE_NAME")
	githubToken   = os.Getenv("GITHUB_TOKEN")
	codespaceURL  = strings.TrimRight(os.Getenv("CODESPACE_SERVICE_URL"), "/")
	upstashURL    = strings.TrimRight(os.Getenv("UPSTASH_REDIS_URL"), "/")
	upstashToken  = os.Getenv("UPSTASH_REDIS_TOKEN")

	if mode == "" {
		mode = "full"
	}

	fmt.Printf("🚀 Go Image & Scraper Service [MODE=%s]\n", mode)

	if mode == "codespace" || mode == "full" {
		fmt.Println("🚀 Chrome-Enhanced, API-driven scraping")
		scraper.InitBrowser()
		defer scraper.CloseBrowser()
	}

	if mode == "render" {
		fmt.Printf("🔗 Codespace : %s\n", codespaceName)
		fmt.Printf("🌐 Tunnel URL: %s\n", func() string {
			if codespaceURL != "" {
				return codespaceURL
			}
			return "(not set — update CODESPACE_SERVICE_URL after each Codespace wake)"
		}())
		fmt.Printf("💾 Redis     : %s\n", func() string {
			if upstashURL != "" {
				return "enabled"
			}
			return "disabled"
		}())
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

	// ── Root & Health ─────────────────────────────────────────────────────────
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "online",
			"service": "Go Image & Scraper Service",
			"version": "4.1.0",
			"mode":    mode,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": mode})
	})

	api := r.Group("/api")

	// ── RENDER routes — lightweight, instant ─────────────────────────────────
	if mode == "render" || mode == "full" {
		api.POST("/combat", combat.GenerateCombatImage)
		api.POST("/combat/endscreen", combat.GenerateEndScreen)
		api.POST("/ludo", ludo.RenderBoard)
		api.POST("/ttt", ttt.RenderBoard)
		api.POST("/ttt/leaderboard", ttt.RenderLeaderboard)
		api.POST("/chess", chess.RenderBoard)
		api.POST("/cards/burn", cards.GenerateBurnGif)
		api.POST("/cards/convert", cards.ConvertCard)
		api.GET("/scrape/stickers", scraper.SearchStickers)
		api.GET("/scrape/rule34", scraper.SearchRule34)

		// Heavy routes — proxied through built-in orchestrator
		// (cache check → wake Codespace → forward → cache result → stop Codespace)
		for _, route := range heavyRoutes {
			routePath := strings.TrimPrefix(route, "/api")
			api.GET(routePath, handleHeavy)
			api.POST(routePath, handleHeavy)
		}
		// cards/gif registered separately so it doesn't conflict with
		// the direct registration in codespace/full mode below
		if mode == "render" {
			api.POST("/cards/gif", handleHeavy)
		}
	}

	// ── CODESPACE routes — Chrome-powered ────────────────────────────────────
	if mode == "codespace" || mode == "full" {
		api.POST("/cards/gif", cards.GenerateCardGif)

		scrape := api.Group("/scrape")
		scrape.GET("/pinterest", scraper.ScrapePinterest)
		scrape.GET("/powerscale", scraper.ScrapePowerscale)
		scrape.GET("/powerscale/fetch", scraper.ScrapePowerscalePage)
		scrape.GET("/pornpics", scraper.ScrapePornPics)
		scrape.GET("/audio", scraper.ScrapeAudio)
		scrape.GET("/anikai", scraper.ScrapeAnikai)
		scrape.GET("/news", scraper.ScrapeAnimeNews)
		scrape.GET("/rule34/deep", scraper.ScrapeRule34)
	}

	log.Printf("🚀 Go Service starting on port %s [MODE=%s]", port, mode)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}