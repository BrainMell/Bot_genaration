package main

import (
        "bytes"
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
        upstashURL   string
        upstashToken string
        proxyClient  = &http.Client{Timeout: 120 * time.Second}
)

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

func proxyToScraper(c *gin.Context) {
        scraperPort := os.Getenv("SCRAPER_PORT")
        if scraperPort == "" {
                scraperPort = "7861"
        }

        targetURL := "http://localhost:" + scraperPort + c.Request.URL.Path
        if c.Request.URL.RawQuery != "" {
                targetURL += "?" + c.Request.URL.RawQuery
        }

        fmt.Printf("[PROXY] Forwarding to Scraper: %s\n", targetURL)

        req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create proxy request"})
                return
        }

        for k, v := range c.Request.Header {
                req.Header[k] = v
        }

        resp, err := proxyClient.Do(req)
        if err != nil {
                c.JSON(502, gin.H{"error": "Scraper unreachable", "details": err.Error()})
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
        if cwd, err := os.Getwd(); err == nil {
                path := os.Getenv("PATH")
                os.Setenv("PATH", cwd+"/bin:"+path)
        }

        mode = os.Getenv("MODE")
        upstashURL = strings.TrimRight(os.Getenv("UPSTASH_REDIS_URL"), "/")
        upstashToken = os.Getenv("UPSTASH_REDIS_TOKEN")

        if mode == "" {
                mode = "full"
        }

        fmt.Printf("🧬 Image Generation Microservice (Go Core) — Mode: %s\n", mode)

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
                        "engine":  "Image Generation REST Microservice",
                        "version": "5.2.0-core",
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

        // Core Image Rendering Endpoints
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
        api.POST("/cards/hybrid-grid", cards.GenerateHybridGrid) // NEW 2026-07-27: animated hybrid grid MP4
        api.GET("/scrape/stickers", scraper.SearchStickers)
        api.POST("/cards/economy", economy.GenerateEconomyCard)
        api.POST("/cards/transaction", economy.GenerateTransactionCard)
        api.POST("/cards/profile", profile.GenerateProfileCard)
        api.POST("/cards/gif", cards.GenerateCardGif)

        // Scraper API Proxy Endpoints
        scrape := api.Group("/scrape")
        scrape.GET("/rule34", proxyToScraper)
        scrape.GET("/rule34/deep", proxyToScraper)
        scrape.GET("/pinterest", proxyToScraper)
        scrape.GET("/powerscale", proxyToScraper)
        scrape.GET("/powerscale/fetch", proxyToScraper)
        scrape.GET("/pornpics", proxyToScraper)
        scrape.GET("/audio", scraper.ScrapeAudio)
        scrape.GET("/anikai", proxyToScraper)
        scrape.GET("/news", proxyToScraper)

        log.Printf("🚀 Microservice starting on port %s [MODE=%s]", port, mode)
        if err := r.Run("0.0.0.0:" + port); err != nil {
                log.Fatal("Engine Startup Failed: ", err)
        }
}
