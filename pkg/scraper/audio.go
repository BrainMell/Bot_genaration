package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
	}

	fmt.Printf("[Audio] Searching (via r.jina.ai) for: %s\n", query)

	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		fmt.Printf("[Audio] Search request failed: %v\n", err)
		c.JSON(500, gin.H{"error": "search failed"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	match := re.FindStringSubmatch(html)

	if len(match) < 2 {
		fmt.Printf("[Audio] No video ID found in response\n")
		c.JSON(500, gin.H{"error": "no video found"})
		return
	}

	videoID := match[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	thumbnail := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	watchURL := "https://www.youtube.com/watch?v=" + videoID

	// Call Cobalt API directly from Go
	fmt.Printf("[Audio] Calling Cobalt API for: %s\n", watchURL)
	
	cobaltReqBody, _ := json.Marshal(map[string]interface{}{
		"url":           watchURL,
		"videoQuality":  "720", // unused for audio but required by some instances
		"audioFormat":   "mp3",
		"downloadMode":  "audio",
		"filenameStyle": "pretty",
	})

	req, _ := http.NewRequest("POST", "https://api.cobalt.tools/api/json", bytes.NewBuffer(cobaltReqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	cobaltResp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("[Audio] Cobalt API request failed: %v\n", err)
		c.JSON(500, gin.H{"error": "failed to get audio stream"})
		return
	}
	defer cobaltResp.Body.Close()

	var result struct {
		Status string `json:"status"`
		URL    string `json:"url"`
		Text   string `json:"text"` // for errors
	}

	if err := json.NewDecoder(cobaltResp.Body).Decode(&result); err != nil {
		fmt.Printf("[Audio] Failed to decode Cobalt response: %v\n", err)
		c.JSON(500, gin.H{"error": "service communication error"})
		return
	}

	if result.Status != "stream" && result.Status != "redirect" && result.Status != "success" {
		fmt.Printf("[Audio] Cobalt returned error status: %s (Text: %s)\n", result.Status, result.Text)
		c.JSON(500, gin.H{"error": "audio extraction failed"})
		return
	}

	fmt.Printf("[Audio] Extraction successful: %s\n", result.URL)

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query,
			Author:    "YouTube",
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": result.URL,
	})
}
