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
        "image-service/pkg/economy"
        "image-service/pkg/ludo"
        "image-service/pkg/profile"
        "image-service/pkg/scraper"
        "image-service/pkg/ttt"
)

// =============================================================================
// CONFIG (VDAP Core)
// =============================================================================

var (
        mode         string
        hfSpaceID    string
        hfToken      string
        hfSpaceURL   string
        upstashURL   string
        upstashToken string
        proxyClient  = &http.Client{Timeout: 120 * time.Second}

        // Auto-Pause logic
        pauseTimer *time.Timer
        pauseMu    sync.Mutex
        pauseDelay = 5 * time.Minute
)

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
// REDIS LAYER (Upstash REST API)
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
                fmt.Printf("[VDAP_CACHE] SET error: %v\n", err)
                return
        }
        resp.Body.Close()
}

func getCacheKey(path string, query url.Values, body []byte) string {
        key := "vdap_cache:" + path + ":" + query.Encode() + ":" + string(body)
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
// HF SPACE ORCHESTRATION (Neural-Resume)
// =============================================================================

func resumeSpace() error {
        if hfSpaceID == "" || hfToken == "" {
                return fmt.Errorf("HF_CREDENTIALS_MISSING")
        }

        apiURL := fmt.Sprintf("https://huggingface.co/api/spaces/%s/resume", hfSpaceID)
        req, _ := http.NewRequest("POST", apiURL, nil)
        req.Header.Set("Authorization", "Bearer "+hfToken)

        resp, err := proxyClient.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        return nil
}

func pauseSpace() {
        if hfSpaceID == "" || hfToken == "" {
                return
        }
        apiURL := fmt.Sprintf("https://huggingface.co/api/spaces/%s/pause", hfSpaceID)
        req, _ := http.NewRequest("POST", apiURL, nil)
        req.Header.Set("Authorization", "Bearer "+hfToken)
        resp, err := proxyClient.Do(req)
        if err == nil {
                resp.Body.Close()
        }
}

func getSpaceStatus() (string, error) {
        if hfSpaceID == "" {
                return "", fmt.Errorf("HF_SPACE_ID_MISSING")
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

func wakeAndWait() error {
        if hfSpaceURL == "" {
                return fmt.Errorf("HF_URL_MISSING")
        }

        status, err := getSpaceStatus()
        if err != nil {
                fmt.Printf("[VDAP_WAKE] Status check failed: %v\n", err)
        }

        if strings.ToUpper(status) != "RUNNING" {
                fmt.Printf("[VDAP_WAKE] Space state: %s. Initiating Neural-Resume...\n", status)
                resumeSpace()

                for i := 0; i < 90; i++ {
                        time.Sleep(2 * time.Second)
                        s, _ := getSpaceStatus()
                        if strings.ToUpper(s) == "RUNNING" {
                                break
                        }
                }
        }

        healthURL := hfSpaceURL + "/health"
        for i := 0; i < 30; i++ {
                resp, err := proxyClient.Get(healthURL)
                if err == nil && resp.StatusCode == 200 {
                        resp.Body.Close()
                        fmt.Println("[VDAP_WAKE] ✅ Remote Node Synchronized.")
                        return nil
                }
                if resp != nil {
                        resp.Body.Close()
                }
                time.Sleep(2 * time.Second)
        }

        return fmt.Errorf("RESOURCES_NOT_READY")
}

func resetPauseTimer() {
        pauseMu.Lock()
        defer pauseMu.Unlock()

        if pauseTimer != nil {
                pauseTimer.Stop()
        }
        pauseTimer = time.AfterFunc(pauseDelay, func() {
                fmt.Println("[VDAP_AUTO] Pipeline idle. Releasing remote resources...")
                pauseSpace()
        })
}

// =============================================================================
// INFERENCE HANDLER
// =============================================================================

func handleHeavy(c *gin.Context) {
        path := c.Request.URL.Path
        query := c.Request.URL.Query()
        method := c.Request.Method

        bodyBytes, _ := io.ReadAll(c.Request.Body)

        cacheKey := getCacheKey(path, query, bodyBytes)
        entry, _ := cacheGet(cacheKey)
        if entry != nil {
                fmt.Printf("[VDAP_CACHE] Cache HIT for kernel %s\n", path)
                c.Header("X-Neural-Cache", "HIT")
                if entry.IsBinary {
                        decoded, _ := base64.StdEncoding.DecodeString(entry.Data)
                        c.Data(200, entry.ContentType, decoded)
                        return
                }
                c.Header("Content-Type", entry.ContentType)
                c.String(200, entry.Data)
                return
        }

        if err := wakeAndWait(); err != nil {
                c.JSON(503, gin.H{"error": "NODE_UNAVAILABLE", "message": err.Error()})
                return
        }

        resetPauseTimer()

        targetURL := hfSpaceURL + path
        if query.Encode() != "" {
                targetURL += "?" + query.Encode()
        }

        proxyReq, _ := http.NewRequest(method, targetURL, bytes.NewReader(bodyBytes))
        if ct := c.GetHeader("Content-Type"); ct != "" {
                proxyReq.Header.Set("Content-Type", ct)
        }

        resp, err := proxyClient.Do(proxyReq)
        if err != nil {
                c.JSON(502, gin.H{"error": "FORWARD_ERROR"})
                return
        }
        defer resp.Body.Close()

        respBody, _ := io.ReadAll(resp.Body)
        contentType := resp.Header.Get("Content-Type")
        if contentType == "" {
                contentType = "application/json"
        }

        c.Data(resp.StatusCode, contentType, respBody)

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
                resetPauseTimer()
        }()
}

func proxyToScraper(c *gin.Context) {
        scraperPort := os.Getenv("SCRAPER_PORT")
        if scraperPort == "" {
                scraperPort = "7861"
        }

        targetURL := "http://localhost:" + scraperPort + c.Request.URL.Path
        if c.Request.URL.RawQuery != "" {
                targetURL += "?" + c.Request.URL.RawQuery
        }

        fmt.Printf("[PROXY] Forwarding to Puppeteer: %s\n", targetURL)

        req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create proxy request"})
                return
        }

        // Copy headers
        for k, v := range c.Request.Header {
                req.Header[k] = v
        }

        resp, err := proxyClient.Do(req)
        if err != nil {
                c.JSON(502, gin.H{"error": "Puppeteer scraper unreachable", "details": err.Error()})
                return
        }
        defer resp.Body.Close()

        body, _ := io.ReadAll(resp.Body)
        c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// =============================================================================
// MAIN ENGINE
// =============================================================================

func main() {
        // Add local bin directory to PATH so yt-dlp, ffmpeg, and ffprobe can be found locally
        if cwd, err := os.Getwd(); err == nil {
                path := os.Getenv("PATH")
                os.Setenv("PATH", cwd+"/bin:"+path)
        }

        mode = os.Getenv("MODE")
        hfSpaceID = os.Getenv("HF_SPACE_ID")
        hfToken = os.Getenv("HF_TOKEN")
        hfSpaceURL = strings.TrimRight(os.Getenv("HF_SPACE_URL"), "/")
        upstashURL = strings.TrimRight(os.Getenv("UPSTASH_REDIS_URL"), "/")
        upstashToken = os.Getenv("UPSTASH_REDIS_TOKEN")

        if mode == "" {
                mode = "full"
        }

        fmt.Printf("🧬 Vision-Data-Acquisition-Pipeline Core (VDAP) — Stage: %s\n", mode)

        if mode == "hf" || mode == "full" {
                fmt.Println("👁️  Loading Computer Vision Pre-processing Kernels...")
                // scraper.InitBrowser() // Disabled Go-Rod in favor of Puppeteer
                // defer scraper.CloseBrowser()
        }

        if mode == "render" {
                fmt.Printf("🛰️  Remote Synchronizer Node: %s\n", hfSpaceID)
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

        r.GET("/", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{
                        "status":  "active",
                        "engine":  "Vision-Data-Acquisition-Pipeline (VDAP)",
                        "version": "5.2.0-neural-core",
                        "mode":    mode,
                })
        })

        r.HEAD("/", func(c *gin.Context) {
                c.Status(http.StatusOK)
        })

        r.GET("/health", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"status": "ready", "engine": "synchronous"})
        })

        r.HEAD("/health", func(c *gin.Context) {
                c.Status(http.StatusOK)
        })

        api := r.Group("/api")

        // ── Non-scraper routes (all modes except hf-only) ──────────────────────
        if mode == "render" || mode == "full" {
                api.POST("/combat", combat.GenerateCombatImage)
                api.POST("/combat/endscreen", combat.GenerateEndScreen)
                api.POST("/combat/splash", combat.GenerateBossSplash)
                api.POST("/ludo", ludo.RenderBoard)
                api.POST("/ttt", ttt.RenderBoard)
                api.POST("/ttt/leaderboard", ttt.RenderLeaderboard)
                api.POST("/chess", chess.RenderBoard)
                api.POST("/cards/burn", cards.GenerateBurnGif)
                api.POST("/cards/convert", cards.ConvertCard)
                api.POST("/cards/eshop", cards.GenerateEShopDeck)
                api.POST("/cards/grid", cards.GenerateCollectionGrid)
                api.GET("/scrape/stickers", scraper.SearchStickers)
                api.POST("/cards/economy", economy.GenerateEconomyCard)
                api.POST("/cards/transaction", economy.GenerateTransactionCard)
                api.POST("/cards/profile", profile.GenerateProfileCard)
        }

        // ── Scraper routes: proxyToScraper (hf + full) ─────────────────────────
        // In full/hf mode, the Puppeteer sidecar on port 7861 handles browser work.
        if mode == "hf" || mode == "full" {
                api.POST("/cards/gif", cards.GenerateCardGif)
                scrape := api.Group("/scrape")
                scrape.GET("/rule34", proxyToScraper)
                scrape.GET("/rule34/deep", proxyToScraper)
                scrape.GET("/pinterest", proxyToScraper)
                scrape.GET("/powerscale", proxyToScraper)
                scrape.GET("/powerscale/fetch", proxyToScraper)
                scrape.GET("/pornpics", proxyToScraper)
                scrape.GET("/audio", scraper.ScrapeAudio) // yt-dlp stays in Go
                scrape.GET("/anikai", proxyToScraper)
                scrape.GET("/news", proxyToScraper)
        }

        // ── render mode: forward heavy routes to HF Space ──────────────────────
        // Only used when deploying a lightweight Render → HF Space architecture.
        if mode == "render" {
                api.POST("/cards/gif", handleHeavy)
                api.GET("/scrape/rule34", proxyToScraper)
                for _, route := range heavyRoutes {
                        routePath := strings.TrimPrefix(route, "/api")
                        api.GET(routePath, handleHeavy)
                        api.POST(routePath, handleHeavy)
                }
        }

        log.Printf("🚀 VDAP Inference Server starting on port %s [MODE=%s]", port, mode)
        if err := r.Run("0.0.0.0:" + port); err != nil {
                log.Fatal("Engine Startup Failed: ", err)
        }
}
