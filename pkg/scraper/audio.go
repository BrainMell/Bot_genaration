package scraper

import (
	"bytes"
	"encoding/json"
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

// Free self-hosted cobalt instances that don't require auth
var cobaltInstances = []string{
	"https://co.wuk.sh",
	"https://cobalt.ggtyler.dev",
	"https://cobalt.api.timelessnesses.me",
	"https://cobalt-api.kkande.me",
}

func tryCobalt(httpClient *http.Client, watchURL string, mp3Path string) bool {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"url":          watchURL,
		"downloadMode": "audio",
		"audioFormat":  "mp3",
		"audioBitrate": "128",
	})

	for _, instance := range cobaltInstances {
		fmt.Printf("[Audio] Trying cobalt instance: %s\n", instance)
		req, _ := http.NewRequest("POST", instance+"/", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Printf("[Audio] %s error: %v\n", instance, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("[Audio] %s bad JSON\n", instance)
			continue
		}

		status, _ := result["status"].(string)
		if status != "tunnel" && status != "redirect" && status != "stream" {
			fmt.Printf("[Audio] %s status: %s\n", instance, status)
			continue
		}

		downloadURL, ok := result["url"].(string)
		if !ok || downloadURL == "" {
			continue
		}

		// Download the mp3
		dlReq, _ := http.NewRequest("GET", downloadURL, nil)
		dlReq.Header.Set("User-Agent", "Mozilla/5.0")
		dlResp, err := httpClient.Do(dlReq)
		if err != nil {
			fmt.Printf("[Audio] %s download error: %v\n", instance, err)
			continue
		}
		f, _ := os.Create(mp3Path)
		io.Copy(f, dlResp.Body)
		f.Close()
		dlResp.Body.Close()

		// Verify file has content
		if info, err := os.Stat(mp3Path); err == nil && info.Size() > 10000 {
			fmt.Printf("[Audio] Cobalt success via %s\n", instance)
			return true
		}
		os.Remove(mp3Path)
	}
	return false
}

func trySoundCloud(query string, mp3Path string) bool {
	fmt.Printf("[Audio] Trying SoundCloud via yt-dlp...\n")
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
		return false
	}
	if info, err := os.Stat(mp3Path); err == nil && info.Size() > 10000 {
		fmt.Printf("[Audio] SoundCloud success\n")
		return true
	}
	return false
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

	_ = os.MkdirAll("downloads", 0755)
	mp3Path := fmt.Sprintf("downloads/%s.mp3", videoID)

	// ── Step 2: Try cobalt first, fall back to SoundCloud ───────────────────
	source := "YouTube"
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		if !tryCobalt(httpClient, watchURL, mp3Path) {
			fmt.Printf("[Audio] Cobalt failed, falling back to SoundCloud...\n")
			if !trySoundCloud(query, mp3Path) {
				c.JSON(500, gin.H{"error": "all download methods failed"})
				return
			}
			source = "SoundCloud"
		}
	}

	host := os.Getenv("SERVICE_URL")
	if host == "" {
		host = "https://mellow2006-mellowbotbackend.hf.space"
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query,
			Author:    source,
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", host, videoID),
	})
}