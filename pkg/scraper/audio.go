package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
		Timeout: 10 * time.Second,
	}

	fmt.Printf("[Audio] Searching (via r.jina.ai) for: %s\n", query)
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search failed"})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		c.JSON(500, gin.H{"error": "no video found"})
		return
	}
	videoID := match[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	thumbnail := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	embedURL := "https://www.youtube.com/embed/" + videoID

	// ── Convert to mp3 with ffmpeg ──────────────────────────────────────────
	_ = os.MkdirAll("downloads", 0755)
	mp3Path := fmt.Sprintf("downloads/%s.mp3", videoID)

	// Only convert if not already cached
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		fmt.Printf("[Audio] Converting %s to mp3...\n", videoID)
		cmd := exec.Command("ffmpeg", "-y",
			"-i", embedURL,
			"-vn",
			"-acodec", "libmp3lame",
			"-q:a", "2",
			mp3Path,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] ffmpeg failed: %v\n%s\n", err, string(out))
			// Fall back to embed URL if ffmpeg fails
			c.JSON(200, gin.H{
				"metadata": AudioMetadata{
					Title:     query,
					Author:    "YouTube",
					Thumbnail: thumbnail,
					Duration:  "",
					URL:       watchURL,
				},
				"audioURL": embedURL,
			})
			return
		}
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