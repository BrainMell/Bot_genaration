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

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Get YouTube video ID via r.jina.ai
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

	_ = os.MkdirAll("downloads", 0755)
	mp3Path := fmt.Sprintf("downloads/%s.mp3", videoID)

	// Step 2: Download via SoundCloud
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		fmt.Printf("[Audio] Downloading via SoundCloud...\n")
		cmd := exec.Command(
			"yt-dlp",
			"scsearch1:"+query,
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "0",
			"-o", mp3Path,
			"--no-playlist",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] SoundCloud failed: %v\n%s\n", err, string(out))
			c.JSON(500, gin.H{"error": "download failed"})
			return
		}
		fmt.Printf("[Audio] mp3 ready: %s\n", mp3Path)
	}

	host := os.Getenv("SERVICE_URL")
	if host == "" {
		host = "https://mellow2006-mellowbotbackend.hf.space"
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query,
			Author:    "SoundCloud",
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", host, videoID),
	})
}