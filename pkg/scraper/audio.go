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

// Official maintained Invidious instances (from docs.invidious.io/instances)
var invidiousInstances = []string{
	"https://inv.nadeko.net",
	"https://yewtu.be",
	"https://invidious.nerdvpn.de",
}

type invidiousVideoResp struct {
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

func tryInvidious(httpClient *http.Client, videoID string, mp3Path string) (title, author, thumbnail string, ok bool) {
	for _, inst := range invidiousInstances {
		apiURL := fmt.Sprintf("%s/api/v1/videos/%s", inst, videoID)
		fmt.Printf("[Audio] Trying Invidious: %s\n", apiURL)

		resp, err := httpClient.Get(apiURL)
		if err != nil {
			fmt.Printf("[Audio] %s error: %v\n", inst, err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if len(b) == 0 || b[0] != '{' {
			fmt.Printf("[Audio] %s bad response\n", inst)
			continue
		}

		var v invidiousVideoResp
		if err := json.Unmarshal(b, &v); err != nil {
			fmt.Printf("[Audio] %s parse error: %v\n", inst, err)
			continue
		}

		// Find best audio-only stream
		var bestURL string
		bestBitrate := 0
		for _, f := range v.AdaptiveFormats {
			if len(f.Type) > 5 && f.Type[:5] == "audio" && f.Bitrate > bestBitrate {
				bestBitrate = f.Bitrate
				bestURL = f.URL
			}
		}

		if bestURL == "" {
			fmt.Printf("[Audio] %s no audio streams\n", inst)
			continue
		}

		// Get thumbnail
		thumb := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
		for _, t := range v.VideoThumbnails {
			if t.Quality == "high" || t.Quality == "medium" {
				thumb = t.URL
				break
			}
		}

		// Download stream bytes
		fmt.Printf("[Audio] Downloading stream from %s...\n", inst)
		tmpPath := mp3Path + "_raw"
		req, _ := http.NewRequest("GET", bestURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		dlResp, err := httpClient.Do(req)
		if err != nil {
			fmt.Printf("[Audio] Download failed: %v\n", err)
			continue
		}
		f, _ := os.Create(tmpPath)
		io.Copy(f, dlResp.Body)
		f.Close()
		dlResp.Body.Close()

		// Convert to mp3 with ffmpeg (local file only, no network)
		cmd := exec.Command("ffmpeg", "-y", "-i", tmpPath,
			"-vn", "-acodec", "libmp3lame", "-q:a", "2", mp3Path)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] ffmpeg error: %v\n%s\n", err, string(out))
			os.Remove(tmpPath)
			continue
		}
		os.Remove(tmpPath)

		if info, err := os.Stat(mp3Path); err != nil || info.Size() < 10000 {
			fmt.Printf("[Audio] mp3 too small, trying next instance\n")
			os.Remove(mp3Path)
			continue
		}

		fmt.Printf("[Audio] Invidious success via %s: %s\n", inst, v.Title)
		return v.Title, v.Author, thumb, true
	}
	return "", "", "", false
}

func trySoundCloud(query string, mp3Path string) bool {
	fmt.Printf("[Audio] Trying SoundCloud via yt-dlp...\n")
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

	title := query
	author := "YouTube"
	source := "YouTube"

	// ── Step 2: Try Invidious first, fall back to SoundCloud ─────────────────
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		if t, a, th, ok := tryInvidious(httpClient, videoID, mp3Path); ok {
			title, author, thumbnail = t, a, th
			source = "YouTube"
		} else {
			fmt.Printf("[Audio] Invidious failed, falling back to SoundCloud...\n")
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

	_ = source
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