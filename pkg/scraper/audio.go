package scraper

import (
	"crypto/md5"
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

	fmt.Printf("[Audio] Searching for: %s\n", query)

	_ = os.MkdirAll("downloads", 0755)

	// Use a stable filename based on query (so cache hits work)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(query)))[:12]
	mp3Path := fmt.Sprintf("downloads/%s.mp3", hash)

	// Get YouTube video ID for thumbnail (best effort)
	videoID := ""
	thumbnail := ""
	watchURL := ""
	httpClient := &http.Client{Timeout: 10 * time.Second}
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	if resp, err := httpClient.Get(searchURL); err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
		if match := re.FindStringSubmatch(string(body)); len(match) >= 2 {
			videoID = match[1]
			thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
			watchURL = "https://www.youtube.com/watch?v=" + videoID
		}
	}
	fmt.Printf("[Audio] Video ID: %s\n", videoID)

	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		downloaded := false

		// Attempt 1: SoundCloud search
		fmt.Printf("[Audio] Trying SoundCloud for: %s\n", query)
		cmd := exec.Command(
			"yt-dlp",
			"scsearch1:"+query,
			"-x", "--audio-format", "mp3", "--audio-quality", "0",
			"--no-playlist",
			"-o", mp3Path,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] SoundCloud failed: %v\n%s\n", err, string(out))
		} else {
			downloaded = true
		}

		// Attempt 2: YouTube with TV client (less blocked than android in datacenters)
		if !downloaded && videoID != "" {
			fmt.Printf("[Audio] Trying YouTube TV client for: %s\n", videoID)
			cmdYt := exec.Command(
				"yt-dlp",
				"-x", "--audio-format", "mp3", "--audio-quality", "0",
				"--extractor-args", "youtube:player_client=tv",
				"--no-check-certificate",
				"-o", mp3Path,
				watchURL,
			)
			if out, err := cmdYt.CombinedOutput(); err != nil {
				fmt.Printf("[Audio] YouTube TV failed: %v\n%s\n", err, string(out))
			} else {
				downloaded = true
			}
		}

		// Attempt 3: YouTube with web_embedded client
		if !downloaded && videoID != "" {
			fmt.Printf("[Audio] Trying YouTube web_embedded for: %s\n", videoID)
			cmdEmbed := exec.Command(
				"yt-dlp",
				"-x", "--audio-format", "mp3", "--audio-quality", "0",
				"--extractor-args", "youtube:player_client=web_embedded",
				"--no-check-certificate",
				"-o", mp3Path,
				watchURL,
			)
			if out, err := cmdEmbed.CombinedOutput(); err != nil {
				fmt.Printf("[Audio] YouTube web_embedded failed: %v\n%s\n", err, string(out))
			} else {
				downloaded = true
			}
		}

		// Attempt 4: yt-dlp YouTube search directly (bypasses video ID)
		if !downloaded {
			fmt.Printf("[Audio] Trying yt-dlp ytsearch for: %s\n", query)
			cmdSearch := exec.Command(
				"yt-dlp",
				"ytsearch1:"+query,
				"-x", "--audio-format", "mp3", "--audio-quality", "0",
				"--extractor-args", "youtube:player_client=tv",
				"--no-check-certificate",
				"--no-playlist",
				"-o", mp3Path,
			)
			if out, err := cmdSearch.CombinedOutput(); err != nil {
				fmt.Printf("[Audio] yt-dlp ytsearch failed: %v\n%s\n", err, string(out))
			} else {
				downloaded = true
			}
		}

		if !downloaded {
			c.JSON(500, gin.H{"error": "all download attempts failed"})
			return
		}
	}

	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		c.JSON(500, gin.H{"error": "audio file not generated"})
		return
	}

	fmt.Printf("[Audio] mp3 ready: %s\n", mp3Path)

	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     query,
			Author:    "SoundCloud/YouTube",
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", baseURL, hash),
	})
}
