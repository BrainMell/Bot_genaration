package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
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

var invidiousInstances = []string{
	"https://inv.nadeko.net",
	"https://yewtu.be",
	"https://invidious.nerdvpn.de",
}

type invidiousResp struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	LengthSeconds int    `json:"lengthSeconds"`
	VideoThumbnails []struct {
		Quality string `json:"quality"`
		URL     string `json:"url"`
	} `json:"videoThumbnails"`
	AdaptiveFormats []struct {
		Type    string `json:"type"`
		URL     string `json:"url"`
		Bitrate int    `json:"bitrate"`
	} `json:"adaptiveFormats"`
}

// tryJinaInvidious proxies the Invidious API call through r.jina.ai
// to bypass anti-bot protection on HF Spaces
func tryJinaInvidious(httpClient *http.Client, videoID string, mp3Path string) (title, author, thumbnail string, ok bool) {
	for _, inst := range invidiousInstances {
		apiURL := fmt.Sprintf("https://r.jina.ai/%s/api/v1/videos/%s", inst, videoID)
		fmt.Printf("[Audio] Trying Jina+Invidious: %s\n", inst)

		resp, err := httpClient.Get(apiURL)
		if err != nil {
			fmt.Printf("[Audio] %s jina error: %v\n", inst, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Jina wraps content — find the JSON object inside
		raw := string(body)
		start := strings.Index(raw, `{"title"`)
		if start == -1 {
			start = strings.Index(raw, `{"id"`)
		}
		if start == -1 {
			fmt.Printf("[Audio] %s no JSON found in Jina response\n", inst)
			continue
		}
		jsonStr := raw[start:]

		var v invidiousResp
		if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
			// Try trimming to first valid JSON object
			end := strings.LastIndex(jsonStr, "}")
			if end > 0 {
				json.Unmarshal([]byte(jsonStr[:end+1]), &v)
			}
		}

		if len(v.AdaptiveFormats) == 0 {
			fmt.Printf("[Audio] %s no adaptive formats\n", inst)
			continue
		}

		// Find best audio stream
		var bestURL string
		bestBitrate := 0
		for _, f := range v.AdaptiveFormats {
			if strings.HasPrefix(f.Type, "audio") && f.Bitrate > bestBitrate {
				bestBitrate = f.Bitrate
				bestURL = f.URL
			}
		}

		if bestURL == "" {
			fmt.Printf("[Audio] %s no audio streams\n", inst)
			continue
		}

		// Get best thumbnail
		thumb := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
		for _, t := range v.VideoThumbnails {
			if t.Quality == "high" || t.Quality == "medium" {
				thumb = t.URL
				break
			}
		}

		// Download stream bytes locally
		fmt.Printf("[Audio] Downloading stream...\n")
		tmpPath := mp3Path + "_raw"
		req, _ := http.NewRequest("GET", bestURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		dlResp, err := httpClient.Do(req)
		if err != nil {
			fmt.Printf("[Audio] stream download failed: %v\n", err)
			continue
		}
		f, _ := os.Create(tmpPath)
		io.Copy(f, dlResp.Body)
		f.Close()
		dlResp.Body.Close()

		// Convert to mp3 with ffmpeg (local file, no network)
		cmd := exec.Command("ffmpeg", "-y", "-i", tmpPath,
			"-vn", "-acodec", "libmp3lame", "-q:a", "2", mp3Path)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] ffmpeg error: %v\n%s\n", err, string(out))
			os.Remove(tmpPath)
			os.Remove(mp3Path)
			continue
		}
		os.Remove(tmpPath)

		if info, err := os.Stat(mp3Path); err != nil || info.Size() < 10000 {
			os.Remove(mp3Path)
			continue
		}

		fmt.Printf("[Audio] Jina+Invidious success: %s\n", v.Title)
		return v.Title, v.Author, thumb, true
	}
	return "", "", "", false
}

func trySoundCloud(query string, mp3Path string) bool {
	fmt.Printf("[Audio] Falling back to SoundCloud...\n")
	cmd := exec.Command("yt-dlp", "scsearch1:"+query,
		"-x", "--audio-format", "mp3", "--audio-quality", "0",
		"-o", mp3Path, "--no-playlist",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("[Audio] SoundCloud failed: %v\n%s\n", err, string(out))
		return false
	}
	if info, err := os.Stat(mp3Path); err == nil && info.Size() > 10000 {
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

	title := query
	author := "YouTube"

	// Step 2: Try Jina-proxied Invidious, fall back to SoundCloud
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		if t, a, th, ok := tryJinaInvidious(httpClient, videoID, mp3Path); ok {
			title, author, thumbnail = t, a, th
		} else {
			if !trySoundCloud(query, mp3Path) {
				c.JSON(500, gin.H{"error": "all download methods failed"})
				return
			}
			author = "SoundCloud"
		}
	}

	host := os.Getenv("SERVICE_URL")
	if host == "" {
		host = "https://mellow2006-mellowbotbackend.hf.space"
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     title,
			Author:    author,
			Thumbnail: thumbnail,
			Duration:  "",
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", host, videoID),
	})
}