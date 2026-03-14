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

// More instances — some are more reliable than others
var pipedInstances = []string{
	"https://piped.kavin.rocks",
	"https://piped.smnz.de",
	"https://piped.tokhmi.xyz",
	"https://piped.mha.fi",
	"https://waterdrop.ducks.party",
}

type pipedStreams struct {
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	Uploader     string `json:"uploader"`
	Duration     int    `json:"duration"`
	AudioStreams  []struct {
		URL     string `json:"url"`
		Bitrate int    `json:"bitrate"`
	} `json:"audioStreams"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	fmt.Printf("[Audio] Searching for: %s\n", query)

	// Step 1: Search via Jina proxy (proxies YouTube search, works on HF)
	searchURL := "https://r.jina.ai/https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search request failed: " + err.Error()})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		c.JSON(500, gin.H{"error": "no video id found in search results"})
		return
	}
	videoID := matches[1]
	fmt.Printf("[Audio] Video ID: %s\n", videoID)

	// Step 2: Try each Piped instance for streams
	// IMPORTANT: build URLs with fmt.Sprintf, NOT path.Join (path.Join breaks https://)
	for _, instance := range pipedInstances {
		streamsURL := fmt.Sprintf("%s/streams/%s", instance, videoID)
		fmt.Printf("[Audio] Trying: %s\n", streamsURL)

		resp, err := httpClient.Get(streamsURL)
		if err != nil {
			fmt.Printf("[Audio] %s error: %v\n", instance, err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check for HTML response (means instance is down or blocked)
		if len(b) > 0 && b[0] == '<' {
			fmt.Printf("[Audio] %s returned HTML (down or blocked)\n", instance)
			continue
		}

		var streams pipedStreams
		if err := json.Unmarshal(b, &streams); err != nil {
			fmt.Printf("[Audio] %s bad JSON: %v\n", instance, err)
			continue
		}
		if len(streams.AudioStreams) == 0 {
			fmt.Printf("[Audio] %s no audio streams\n", instance)
			continue
		}

		// Pick highest bitrate
		best := streams.AudioStreams[0]
		for _, a := range streams.AudioStreams {
			if a.Bitrate > best.Bitrate {
				best = a
			}
		}

		fmt.Printf("[Audio] Success via %s: %s\n", instance, streams.Title)
		c.JSON(200, gin.H{
			"metadata": AudioMetadata{
				Title:     streams.Title,
				Author:    streams.Uploader,
				Thumbnail: streams.ThumbnailUrl,
				Duration:  fmt.Sprintf("%ds", streams.Duration),
				URL:       "https://www.youtube.com/watch?v=" + videoID,
			},
			"audioURL": best.URL,
		})
		return
	}

	c.JSON(500, gin.H{
		"error":   "all Piped instances failed",
		"videoID": videoID,
	})
}