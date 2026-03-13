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

// ScrapeAudio downloads YouTube audio via yt-dlp -> mp3 and returns a direct URL.
// Requires yt-dlp + ffmpeg installed in the runtime image.
// Saves files to ./downloads/<videoID>.mp3 which is served by main.go static route.
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}

	httpClient := &http.Client{Timeout: 12 * time.Second}
	fmt.Printf("[Audio] Searching for: %s\n", query)

	// Use Jina proxy to fetch static search HTML (no JS)
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed", "detail": err.Error()})
		return
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read search failed", "detail": err.Error()})
		return
	}
	html := string(body)

	// Extract first video id
	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no video id found"})
		return
	}
	videoID := matches[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	// Ensure downloads dir exists
	if err := os.MkdirAll("downloads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir failed", "detail": err.Error()})
		return
	}

	// Local target file (explicit .mp3 extension)
	outPath := fmt.Sprintf("downloads/%s.mp3", videoID)

	// If file already exists, return it immediately (avoid re-downloading)
	if _, err := os.Stat(outPath); err == nil {
		audioURL := "/downloads/" + videoID + ".mp3"
		c.JSON(http.StatusOK, gin.H{"audioURL": audioURL, "videoID": videoID})
		return
	}

	// Run yt-dlp to extract audio and ffmpeg to convert to mp3.
	// Using output template that forces mp3 extension.
	cmd := exec.Command(
		"yt-dlp",
		"-x",                      // extract audio
		"--audio-format", "mp3",   // convert to mp3
		"-o", outPath,             // output path
		"https://www.youtube.com/watch?v="+videoID,
	)
	// stdout/stderr printed to container logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("[Audio] running yt-dlp, this may take a few seconds...")
	if err := cmd.Run(); err != nil {
		fmt.Printf("[Audio] yt-dlp error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "yt-dlp failed", "detail": err.Error()})
		return
	}

	// Confirm file exists
	if _, err := os.Stat(outPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "output missing after yt-dlp"})
		return
	}

	audioURL := "/downloads/" + videoID + ".mp3"
	c.JSON(http.StatusOK, gin.H{"audioURL": audioURL, "videoID": videoID})
}