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

// AudioMetadata is the response metadata shape
type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// ScrapeAudio - single-file handler that:
// 1) fetches YouTube search HTML via r.jina.ai proxy
// 2) extracts first video ID
// 3) fetches oEmbed for title/author/thumbnail (no API key)
// 4) returns metadata + embed URL as "audioURL"
//
// NOTE: This returns an embed/watch URL (playable in webviews/iframes).
// If you need raw audio streams (mp3/webm) you'll need a proxy like yt-dlp.
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}

	// local client to avoid package-level redeclare issues
	httpClient := &http.Client{Timeout: 12 * time.Second}

	fmt.Printf("[Audio] Searching (via r.jina.ai) for: %s\n", query)

	// 1) Fetch YouTube search HTML via Jina reader proxy (server-side HTML)
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search proxy failed", "detail": err.Error()})
		return
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed reading search response", "detail": err.Error()})
		return
	}
	html := string(body)

	// 2) Extract first video id from HTML
	// YouTube id pattern: 11 chars of [A-Za-z0-9_-]
	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no video id found"})
		return
	}
	videoID := matches[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	// 3) Get metadata via YouTube oEmbed (no API key needed)
	oembedURL := "https://www.youtube.com/oembed?url=" + url.QueryEscape("https://www.youtube.com/watch?v="+videoID) + "&format=json"

	resp2, err := httpClient.Get(oembedURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oembed failed", "detail": err.Error()})
		return
	}
	body2, err := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed reading oembed", "detail": err.Error()})
		return
	}

	var oembed struct {
		Title        string `json:"title"`
		AuthorName   string `json:"author_name"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.Unmarshal(body2, &oembed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid oembed response", "detail": err.Error()})
		return
	}

	// Build response
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	embedURL := "https://www.youtube.com/embed/" + videoID

	c.JSON(http.StatusOK, gin.H{
		"metadata": AudioMetadata{
			Title:     oembed.Title,
			Author:    oembed.AuthorName,
			Thumbnail: oembed.ThumbnailURL,
			Duration:  "", // oEmbed doesn't return duration
			URL:       watchURL,
		},
		"audioURL": embedURL,
	})
}