package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// Piped instances — falls back if one is down
var pipedInstances = []string{
	"https://piped.video",
	"https://piped.kavin.rocks",
	"https://piped.adminforge.de",
}

type pipedSearchResult struct {
	Items []struct {
		URL          string `json:"url"`
		Title        string `json:"title"`
		Thumbnail    string `json:"thumbnail"`
		UploaderName string `json:"uploaderName"`
		Duration     int    `json:"duration"`
	} `json:"items"`
}

type pipedStreams struct {
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	Uploader     string `json:"uploader"`
	Duration     int    `json:"duration"`
	AudioStreams  []struct {
		URL     string `json:"url"`
		Quality string `json:"quality"`
		Bitrate int    `json:"bitrate"`
	} `json:"audioStreams"`
}

// ScrapeAudio uses Piped API — pure HTTPS, no yt-dlp, no browser, works on HF Spaces.
func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	fmt.Printf("[Audio] Piped search for: %s\n", query)

	// --- Step 1: Search ---
	videoID := ""
	title, thumbnail, uploader := "", "", ""
	duration := 0

	for _, instance := range pipedInstances {
		searchURL := fmt.Sprintf("%s/api/v1/search?q=%s&filter=music_songs",
			instance, url.QueryEscape(query))

		resp, err := http.Get(searchURL)
		if err != nil {
			fmt.Printf("[Audio] %s search error: %v\n", instance, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result pipedSearchResult
		if err := json.Unmarshal(body, &result); err != nil || len(result.Items) == 0 {
			fmt.Printf("[Audio] %s returned no results\n", instance)
			continue
		}

		first := result.Items[0]
		// URL format: "/watch?v=VIDEO_ID"
		parts := strings.Split(first.URL, "v=")
		if len(parts) < 2 {
			continue
		}
		videoID = strings.Split(parts[1], "&")[0]
		title = first.Title
		thumbnail = first.Thumbnail
		uploader = first.UploaderName
		duration = first.Duration
		fmt.Printf("[Audio] Found: %s (ID: %s)\n", title, videoID)
		break
	}

	if videoID == "" {
		c.JSON(500, gin.H{"error": "No search results found"})
		return
	}

	// --- Step 2: Get streams ---
	audioURL := ""

	for _, instance := range pipedInstances {
		streamsURL := fmt.Sprintf("%s/streams/%s", instance, videoID)

		resp, err := http.Get(streamsURL)
		if err != nil {
			fmt.Printf("[Audio] %s streams error: %v\n", instance, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var streams pipedStreams
		if err := json.Unmarshal(body, &streams); err != nil || len(streams.AudioStreams) == 0 {
			fmt.Printf("[Audio] %s no streams\n", instance)
			continue
		}

		// Pick highest bitrate
		best := streams.AudioStreams[0]
		for _, s := range streams.AudioStreams {
			if s.Bitrate > best.Bitrate {
				best = s
			}
		}
		audioURL = best.URL

		// Prefer richer metadata from streams endpoint
		if streams.ThumbnailUrl != "" {
			thumbnail = streams.ThumbnailUrl
		}
		if streams.Uploader != "" {
			uploader = streams.Uploader
		}
		if streams.Title != "" {
			title = streams.Title
		}
		if streams.Duration > 0 {
			duration = streams.Duration
		}

		fmt.Printf("[Audio] Stream ready for: %s\n", title)
		break
	}

	if audioURL == "" {
		c.JSON(500, gin.H{"error": "Could not get audio stream from any Piped instance"})
		return
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     title,
			Author:    uploader,
			Thumbnail: thumbnail,
			Duration:  fmt.Sprintf("%ds", duration),
			URL:       "https://www.youtube.com/watch?v=" + videoID,
		},
		"audioURL": audioURL,
	})
}