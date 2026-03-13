package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
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

// Use known API base URLs (pipedapi.*). Frontend domains often don't expose API endpoints.
var pipedInstances = []string{
	"https://pipedapi.kavin.rocks",
	"https://pipedapi-libre.kavin.rocks",
	"https://pipedapi.r4fo.com",
	// add more pipedapi.* instances (or fetch dynamic list from TeamPiped's instances list)
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
	AudioStreams []struct {
		URL     string `json:"url"`
		Quality string `json:"quality"`
		Bitrate int    `json:"bitrate"`
	} `json:"audioStreams"`
}

func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	fmt.Printf("[Audio] Piped search for: %s\n", query)

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// --- Step 1: Search ---
	videoID := ""
	title, thumbnail, uploader := "", "", ""
	duration := 0

	for _, instance := range pipedInstances {
		// Build safe URL
		base, err := url.Parse(instance)
		if err != nil {
			fmt.Printf("[Audio] invalid instance URL %s: %v\n", instance, err)
			continue
		}

		// Many setups use /api/v1/search?q=...&filter=music_songs
		// Construct path safely
		base.Path = path.Join(base.Path, "api", "v1", "search")
		base.RawQuery = "q=" + url.QueryEscape(query) + "&filter=music_songs"
		searchURL := base.String()

		resp, err := httpClient.Get(searchURL)
		if err != nil {
			fmt.Printf("[Audio] %s search error: %v\n", instance, err)
			continue
		}
		// read only on 200
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[Audio] %s search HTTP %d\n", instance, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result pipedSearchResult
		if err := json.Unmarshal(body, &result); err != nil || len(result.Items) == 0 {
			fmt.Printf("[Audio] %s returned no results or bad JSON\n", instance)
			continue
		}

		first := result.Items[0]
		parts := strings.Split(first.URL, "v=")
		if len(parts) < 2 {
			// some instances return "/watch?v=..." or "/watch?v=...&list=..."
			fmt.Printf("[Audio] malformed URL returned by %s: %s\n", instance, first.URL)
			continue
		}
		videoID = strings.Split(parts[1], "&")[0]
		title = first.Title
		thumbnail = first.Thumbnail
		uploader = first.UploaderName
		duration = first.Duration
		fmt.Printf("[Audio] Found: %s (ID: %s) via %s\n", title, videoID, instance)
		break
	}

	if videoID == "" {
		c.JSON(500, gin.H{"error": "No search results found"})
		return
	}

	// --- Step 2: Get streams ---
	audioURL := ""

	for _, instance := range pipedInstances {
		base, err := url.Parse(instance)
		if err != nil {
			fmt.Printf("[Audio] invalid instance URL %s: %v\n", instance, err)
			continue
		}

		base.Path = path.Join(base.Path, "streams", videoID)
		streamsURL := base.String()

		resp, err := httpClient.Get(streamsURL)
		if err != nil {
			fmt.Printf("[Audio] %s streams error: %v\n", instance, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[Audio] %s streams HTTP %d\n", instance, resp.StatusCode)
			resp.Body.Close()
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var streams pipedStreams
		if err := json.Unmarshal(body, &streams); err != nil || len(streams.AudioStreams) == 0 {
			fmt.Printf("[Audio] %s no streams or bad JSON\n", instance)
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

		fmt.Printf("[Audio] Stream ready for: %s via %s\n", title, instance)
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