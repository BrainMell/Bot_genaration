package scraper

import (
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

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Uses Jina AI reader proxy to fetch the YouTube search page HTML.
// Then uses YouTube oEmbed (no API key) for metadata.
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	fmt.Printf("[Audio] Searching (via r.jina.ai) for: %s\n", query)

	// 1) Fetch YouTube search HTML via Jina reader proxy (avoids needing JS)
	//    Example: https://r.jina.ai/http://www.youtube.com/results?search_query=babydoll
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search proxy failed", "detail": err.Error()})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	// 2) Extract first video id from HTML
	//    standard yt id pattern: 11 chars of [A-Za-z0-9_-]
	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		c.JSON(500, gin.H{"error": "no video id found"})
		return
	}
	videoID := matches[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	// 3) Get metadata via YouTube oEmbed (no API key needed)
	oembedURL := "https://www.youtube.com/oembed?url=" + url.QueryEscape("https://www.youtube.com/watch?v="+videoID) + "&format=json"

	resp2, err := httpClient.Get(oembedURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "oembed failed", "detail": err.Error()})
		return
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	var oembed struct {
		Title       string `json:"title"`
		AuthorName  string `json:"author_name"`
		ThumbnailURL string `json:"thumbnail_url"`
		// note: oEmbed doesn't return duration, so we'll leave duration blank here
	}
	if err := json.Unmarshal(body2, &oembed); err != nil {
		c.JSON(500, gin.H{"error": "invalid oembed response", "detail": err.Error()})
		return
	}

	// 4) Build response. audioURL will be the watch URL (or embed URL if you prefer)
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	embedURL := "https://www.youtube.com/embed/" + videoID

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     oembed.Title,
			Author:    oembed.AuthorName,
			Thumbnail: oembed.ThumbnailURL,
			Duration:  "", // oEmbed doesn't include duration; would need another source
			URL:       watchURL,
		},
		// Clients can either open "audioURL" directly in an <iframe>/<webview> or use watchURL
		"audioURL": embedURL,
	})
}