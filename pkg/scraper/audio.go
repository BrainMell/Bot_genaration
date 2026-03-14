package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

	thumbnail := fmt.Sprintf(
		"https://i.ytimg.com/vi/%s/hqdefault.jpg",
		videoID,
	)

	watchURL := "https://www.youtube.com/watch?v=" + videoID

	// Define download path
	downloadDir := "./downloads"
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		os.Mkdir(downloadDir, 0755)
	}

	filePath := filepath.Join(downloadDir, videoID+".mp3")

	// Check if file already exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("[Audio] Downloading %s...\n", videoID)
		// Download with yt-dlp
		cmd := exec.Command("yt-dlp",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "128K",
			"-o", filepath.Join(downloadDir, "%(id)s.%(ext)s"),
			watchURL,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("[Audio] Download failed: %v\nOutput: %s\n", err, string(output))
			c.JSON(500, gin.H{"error": "download failed"})
			return
		}

		// Schedule cleanup after 1 hour
		go func() {
			time.Sleep(1 * time.Hour)
			os.Remove(filePath)
			fmt.Printf("[Audio] Cleaned up %s\n", filePath)
		}()
	}

	// Construct download URL
	host := c.Request.Host // e.g., localhost:7860
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	audioDownloadURL := fmt.Sprintf("%s://%s/downloads/%s.mp3", scheme, host, videoID)

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query, // Ideally extract real title from yt-dlp output, but query is okay for now
			Author:    "YouTube",
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": audioDownloadURL,
	})
}
