package scraper


import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
)

type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// ScrapeAudio handles YouTube searching (via Browser) and audio extraction (via yt-dlp)
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	fmt.Printf("[Audio] Processing browser-based search for: %s\n", query)

	var videoURL, title, thumb, author string

	// 1. Browser-based Search Phase (Bypasses yt-dlp DNS issues)
	err := WithPage(func(page *rod.Page) error {
		searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", url.QueryEscape(query))
		page.MustNavigate(searchURL).MustWaitLoad()

		// Wait for video results to appear
		wait := page.MustWaitIdle()
		wait()

		// Extract first video link and metadata
		// Selector for video title/link: ytd-video-renderer #video-title
		videoEl, err := page.Element("ytd-video-renderer #video-title")
		if err != nil {
			return fmt.Errorf("no video results found in browser")
		}

		href, _ := videoEl.Attribute("href")
		if href == nil {
			return fmt.Errorf("could not extract video URL")
		}

		videoURL = "https://www.youtube.com" + *href
		title = strings.TrimSpace(videoEl.MustText())

		// Try to get author and thumbnail (optional)
		if authEl, err := page.Element("ytd-video-renderer #channel-name a"); err == nil {
			author = strings.TrimSpace(authEl.MustText())
		}
		
		// Wait for image load
		time.Sleep(1 * time.Second)
		if imgEl, err := page.Element("ytd-video-renderer img"); err == nil {
			src, _ := imgEl.Attribute("src")
			if src != nil { thumb = *src }
		}

		return nil
	})

	if err != nil {
		fmt.Printf("[Audio] Browser search failed: %v\n", err)
		c.JSON(500, gin.H{"error": "Failed to find video via browser"})
		return
	}

	// 2. Extraction Phase (Direct URL to yt-dlp)
	// Now that we have a direct URL, yt-dlp usually succeeds even if search fails
	cmdURL := exec.Command("yt-dlp", "-f", "bestaudio", "--get-url", videoURL)
	directURLBytes, err := cmdURL.CombinedOutput()
	if err != nil {
		fmt.Printf("[Audio] yt-dlp extraction failed: %v, output: %s\n", err, string(directURLBytes))
		c.JSON(500, gin.H{"error": "Failed to extract audio stream"})
		return
	}
	directURL := strings.TrimSpace(string(directURLBytes))

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     title,
			Author:    author,
			Thumbnail: thumb,
			Duration:  "N/A", // Duration is harder to get quickly from search page
			URL:       videoURL,
		},
		"audioURL": directURL,
	})
}