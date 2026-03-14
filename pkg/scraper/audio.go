package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	// ── Step 1: Get YouTube video ID via r.jina.ai ───────────────────────────
	fmt.Printf("[Audio] Searching for: %s\n", query)
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search failed"})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	match := re.FindStringSubmatch(string(body))
	if len(match) < 2 {
		c.JSON(500, gin.H{"error": "no video found"})
		return
	}
	videoID := match[1]
	fmt.Printf("[Audio] Video ID: %s\n", videoID)

	thumbnail := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	watchURL := "https://www.youtube.com/watch?v=" + videoID

	// ── Step 2: Use cobalt.tools API to get MP3 download link ────────────────
	// cobalt.tools is a free open source YouTube converter — plain HTTPS API
	_ = os.MkdirAll("downloads", 0755)
	mp3Path := fmt.Sprintf("downloads/%s.mp3", videoID)

	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		fmt.Printf("[Audio] Requesting MP3 from cobalt.tools...\n")

		reqBody, _ := json.Marshal(map[string]interface{}{
			"url":           watchURL,
			"downloadMode":  "audio",
			"audioFormat":   "mp3",
			"audioBitrate":  "128",
		})

		req, _ := http.NewRequest("POST", "https://api.cobalt.tools/", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		cobaltResp, err := httpClient.Do(req)
		if err != nil {
			c.JSON(500, gin.H{"error": "cobalt request failed: " + err.Error()})
			return
		}
		cobaltBody, _ := io.ReadAll(cobaltResp.Body)
		cobaltResp.Body.Close()

		var cobaltResult map[string]interface{}
		if err := json.Unmarshal(cobaltBody, &cobaltResult); err != nil {
			fmt.Printf("[Audio] cobalt response: %s\n", string(cobaltBody))
			c.JSON(500, gin.H{"error": "cobalt response parse failed"})
			return
		}

		fmt.Printf("[Audio] Cobalt status: %v\n", cobaltResult["status"])

		downloadURL, ok := cobaltResult["url"].(string)
		if !ok || downloadURL == "" {
			fmt.Printf("[Audio] Cobalt full response: %s\n", string(cobaltBody))
			c.JSON(500, gin.H{"error": "no download URL from cobalt", "detail": cobaltResult})
			return
		}

		// ── Step 3: Download MP3 bytes locally ──────────────────────────────
		fmt.Printf("[Audio] Downloading MP3...\n")
		dlReq, _ := http.NewRequest("GET", downloadURL, nil)
		dlReq.Header.Set("User-Agent", "Mozilla/5.0")
		dlResp, err := httpClient.Do(dlReq)
		if err != nil {
			c.JSON(500, gin.H{"error": "mp3 download failed: " + err.Error()})
			return
		}
		f, _ := os.Create(mp3Path)
		io.Copy(f, dlResp.Body)
		f.Close()
		dlResp.Body.Close()
		fmt.Printf("[Audio] MP3 saved: %s\n", mp3Path)
	}

	host := os.Getenv("SERVICE_URL")
	if host == "" {
		host = "https://mellow2006-mellowbotbackend.hf.space"
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query,
			Author:    "YouTube",
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", host, videoID),
	})
}