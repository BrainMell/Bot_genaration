package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// Stable Invidious instance
const invidiousInstance = "https://inv.nadeko.net"

type searchResult struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	VideoID   string `json:"videoId"`
	LengthSec int    `json:"lengthSeconds"`
}

type videoStreams struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	VideoID      string `json:"videoId"`
	LengthSec    int    `json:"lengthSeconds"`
	AdaptiveFormats []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
		Bitrate int `json:"bitrate"`
	} `json:"adaptiveFormats"`
}

func ScrapeAudio(c *gin.Context) {

	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	fmt.Println("[Audio] Searching:", query)

	// --- STEP 1: SEARCH ---
	searchURL := fmt.Sprintf("%s/api/v1/search?q=%s&type=video",
		invidiousInstance, query)

	resp, err := http.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search request failed"})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var results []searchResult
	json.Unmarshal(body, &results)

	if len(results) == 0 {
		c.JSON(500, gin.H{"error": "no results"})
		return
	}

	video := results[0]
	videoID := video.VideoID

	fmt.Println("[Audio] Found video:", video.Title)

	// --- STEP 2: GET STREAMS ---
	streamURL := fmt.Sprintf("%s/api/v1/videos/%s",
		invidiousInstance, videoID)

	resp2, err := http.Get(streamURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "stream request failed"})
		return
	}

	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	var streams videoStreams
	json.Unmarshal(body2, &streams)

	audioURL := ""

	bestBitrate := 0

	for _, f := range streams.AdaptiveFormats {

		if f.Type == "audio/webm" || f.Type == "audio/mp4" {

			if f.Bitrate > bestBitrate {

				bestBitrate = f.Bitrate
				audioURL = f.URL
			}
		}
	}

	if audioURL == "" {
		c.JSON(500, gin.H{"error": "no audio stream"})
		return
	}

	thumbnail := fmt.Sprintf(
		"https://i.ytimg.com/vi/%s/hqdefault.jpg",
		videoID,
	)

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     streams.Title,
			Author:    streams.Author,
			Thumbnail: thumbnail,
			Duration:  fmt.Sprintf("%ds", streams.LengthSec),
			URL:       "https://youtube.com/watch?v=" + videoID,
		},
		"audioURL": audioURL,
	})
}