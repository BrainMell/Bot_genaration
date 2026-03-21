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
	"sync"
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
//	MODE=render → lightweight image generation + built-in orchestrator for heavy routes
//	MODE=hf     → heavy scraping + card GIF only (Chrome required, runs on HF Space)
//	MODE=full   → everything locally (dev/testing only)
var (
	mode         string
	hfSpaceID    string // e.g. "username/space-name"
	hfToken      string // HF API Token with Write permissions
	hfSpaceURL   string // Permanent URL: https://username-space-name.hf.space
	upstashURL   string
	upstashToken string
	proxyClient  = &http.Client{Timeout: 120 * time.Second}
)

// Auto-pause timer — resets on every heavy request.
// After 5 minutes of no heavy requests, the HF Space is paused.
var (
	pauseTimer   *time.Timer
	pauseMu      sync.Mutex
	pauseDelay   = 5 * time.Minute
)

// Routes forwarded to the HF Space when running in render mode.
var heavyRoutes = []string{
	"/api/scrape/pinterest",
	"/api/scrape/pornpics",
	"/api/scrape/audio",
	"/api/scrape/powerscale",
	"/api/scrape/powerscale/fetch",
	"/api/scrape/anikai",
	"/api/scrape/news",
	"/api/scrape/rule34/deep",
}

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
// REDIS (Upstash REST API)
// =============================================================================

type cacheEntry struct {
	ContentType string `json:"contentType"`
	Data        string `json:"data"`
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
// HUGGING FACE SPACE CONTROL
// =============================================================================

// resumeSpace tells HF to wake the Space if it's paused/sleeping.
func resumeSpace() error {
	if hfSpaceID == "" || hfToken == "" {
		return fmt.Errorf("HF_SPACE_ID or HF_TOKEN not set")
	}

	apiURL := fmt.Sprintf("https://huggingface.co/api/spaces/%s/resume", hfSpaceID)
	req, _ := http.NewRequest("POST", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+hfToken)

	resp, err := proxyClient.Do(req)
	if err != nil {
		return fmt.Errorf("resume request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resume failed (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Println("[HF] Space resume signal sent.")
	return nil
}

// pauseSpace tells HF to pause the Space to save resources.
func pauseSpace() {
	if hfSpaceID == "" || hfToken == "" {
		return
	}

	apiURL := fmt.Sprintf("https://huggingface.co/api/spaces/%s/pause", hfSpaceID)
	req, _ := http.NewRequest("POST", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+hfToken)

	resp, err := proxyClient.Do(req)
	if err != nil {
		fmt.Printf("[HF] Pause error (non-fatal): %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("[HF] Space paused.")
}

// getSpaceStatus returns the current runtime status of the HF Space.
func getSpaceStatus() (string, error) {
	if hfSpaceID == "" {
		return "", fmt.Errorf("HF_SPACE_ID not set")
	}

	apiURL := fmt.Sprintf("https://huggingface.co/api/spaces/%s", hfSpaceID)
	req, _ := http.NewRequest("GET", apiURL, nil)
	if hfToken != "" {
		req.Header.Set("Authorization", "Bearer "+hfToken)
	}

	resp, err := proxyClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Runtime struct {
			Stage string `json:"stage"`
		} `json:"runtime"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Runtime.Stage, nil
}

// wakeAndWait resumes the HF Space and waits until /health responds OK.
func wakeAndWait() error {
	if hfSpaceURL == "" {
		return fmt.Errorf("HF_SPACE_URL not set")
	}

	// Check current status
	status, err := getSpaceStatus()
	if err != nil {
		fmt.Printf("[HF] Could not get status: %v — trying resume anyway\n", err)
	}
	fmt.Printf("[HF] Space status: %s\n", status)

	// Resume if not already running
	if strings.ToUpper(status) != "RUNNING" {
		fmt.Println("[HF] Sending resume signal...")
		if err := resumeSpace(); err != nil {
			return fmt.Errorf("failed to resume HF Space: %v", err)
		}

		// Wait up to 3 minutes for the space to become RUNNING
		for i := 0; i < 90; i++ {
			time.Sleep(2 * time.Second)
			s, _ := getSpaceStatus()
			fmt.Printf("[HF] Waiting for RUNNING... (%s)\n", s)
			if strings.ToUpper(s) == "RUNNING" {
				break
			}
		}
	}

	// Wait for the Go service health endpoint (up to 60s)
	healthURL := hfSpaceURL + "/health"
	fmt.Printf("[HF] Waiting for health at %s\n", healthURL)
	for i := 0; i < 30; i++ {
		resp, err := proxyClient.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			fmt.Println("[HF] ✅ Space is healthy!")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("HF Space did not become healthy in time")
}

// resetPauseTimer resets the auto-pause countdown.
// Every heavy request calls this — Space only pauses after 5 min of silence.
func resetPauseTimer() {
	pauseMu.Lock()
	defer pauseMu.Unlock()

	if pauseTimer != nil {
		pauseTimer.Stop()
	}
	pauseTimer = time.AfterFunc(pauseDelay, func() {
		fmt.Printf("[HF] No heavy requests for %v — pausing Space...\n", pauseDelay)
		pauseSpace()
	})
}

// =============================================================================
// HEAVY REQUEST HANDLER
// =============================================================================

func handleHeavy(c *gin.Context) {
	path   := c.Request.URL.Path
	query  := c.Request.URL.Query()
	method := c.Request.Method

	bodyBytes, _ := io.ReadAll(c.Request.Body)

	// 1. Check cache first
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

	// 2. Cache miss — wake HF Space
	fmt.Printf("[Cache] MISS for %s — waking HF Space...\n", path)
	c.Header("X-Cache", "MISS")

	if err := wakeAndWait(); err != nil {
		c.JSON(503, gin.H{"error": "HF Space unavailable", "details": err.Error()})
		return
	}

	// Reset the auto-pause timer — Space is active
	resetPauseTimer()

	// 3. Forward request to HF Space
	targetURL := hfSpaceURL + path
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
		c.JSON(502, gin.H{"error": "HF Space request failed", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "Failed to read HF Space response"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	// 4. Send to client immediately
	c.Data(resp.StatusCode, contentType, respBody)

	// 5. Cache + reset pause timer in background
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

		// Reset pause timer — give 5 more minutes before sleeping
		resetPauseTimer()
	}()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	mode         = os.Getenv("MODE")
	hfSpaceID    = os.Getenv("HF_SPACE_ID")
	hfToken      = os.Getenv("HF_TOKEN")
	hfSpaceURL   = strings.TrimRight(os.Getenv("HF_SPACE_URL"), "/")
	upstashURL   = strings.TrimRight(os.Getenv("UPSTASH_REDIS_URL"), "/")
	upstashToken = os.Getenv("UPSTASH_REDIS_TOKEN")

	if mode == "" {
		mode = "full"
	}

	fmt.Printf("🚀 Go Image & Scraper Service [MODE=%s]\n", mode)

	if mode == "hf" || mode == "full" {
		fmt.Println("🚀 Chrome-Enhanced, API-driven scraping")
		scraper.InitBrowser()
		defer scraper.CloseBrowser()
	}

	if mode == "render" {
		fmt.Printf("🤗 HF Space  : %s\n", hfSpaceID)
		fmt.Printf("🌐 Space URL : %s\n", func() string {
			if hfSpaceURL != "" {
				return hfSpaceURL
			}
			return "(not set)"
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
			"version": "5.0.0",
			"mode":    mode,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": mode})
	})

	api := r.Group("/api")

	// ── RENDER routes — lightweight, always instant ───────────────────────────
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

		// Heavy routes — proxied through HF Space (cache → resume → forward → cache → auto-pause)
		for _, route := range heavyRoutes {
			routePath := strings.TrimPrefix(route, "/api")
			api.GET(routePath, handleHeavy)
			api.POST(routePath, handleHeavy)
		}
		// cards/gif registered separately to avoid duplicate in full mode
		if mode == "render" {
			api.POST("/cards/gif", handleHeavy)
		}
	}

	// ── HF SPACE routes — Chrome-powered, runs on HF ─────────────────────────
	if mode == "hf" || mode == "full" {
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
