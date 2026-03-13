package scraper

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// ScrapeAudio uses yt-dlp exclusively — no browser needed, works on HF Spaces.
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	fmt.Printf("[Audio] yt-dlp searching for: %s\n", query)

	// Step 1: Search YouTube via yt-dlp (no browser, no DNS issues)
	searchCmd := exec.Command(
		"yt-dlp",
		"ytsearch1:"+query,
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
	)
	searchOut, err := searchCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[Audio] Search failed: %v, output: %s\n", err, string(searchOut))
		c.JSON(500, gin.H{"error": "Search failed: " + err.Error()})
		return
	}

	line := strings.TrimSpace(string(searchOut))
	if line == "" {
		c.JSON(500, gin.H{"error": "No results returned"})
		return
	}

	var entry struct {
		ID        string  `json:"id"`
		Title     string  `json:"title"`
		Uploader  string  `json:"uploader"`
		Thumbnail string  `json:"thumbnail"`
		Duration  float64 `json:"duration"`
	}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		fmt.Printf("[Audio] Parse failed: %v, raw: %s\n", err, line[:min(len(line), 200)])
		c.JSON(500, gin.H{"error": "Failed to parse search result"})
		return
	}

	videoURL := "https://www.youtube.com/watch?v=" + entry.ID
	fmt.Printf("[Audio] Found video: %s (%s)\n", entry.Title, videoURL)

	// Step 2: Get direct audio stream URL via yt-dlp
	urlCmd := exec.Command(
		"yt-dlp",
		"-f", "bestaudio",
		"--get-url",
		"--no-warnings",
		videoURL,
	)
	directURLBytes, err := urlCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[Audio] Extract failed: %v, output: %s\n", err, string(directURLBytes))
		c.JSON(500, gin.H{"error": "Failed to extract audio stream: " + string(directURLBytes)})
		return
	}

	directURL := strings.TrimSpace(string(directURLBytes))
	if directURL == "" {
		c.JSON(500, gin.H{"error": "Empty audio URL returned"})
		return
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     entry.Title,
			Author:    entry.Uploader,
			Thumbnail: entry.Thumbnail,
			Duration:  fmt.Sprintf("%.0fs", entry.Duration),
			URL:       videoURL,
		},
		"audioURL": directURL,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}