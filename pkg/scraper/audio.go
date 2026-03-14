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

var pipedAPIInstances = []string{
	"https://pipedapi.kavin.rocks",
	"https://pipedapi.tokhmi.xyz",
	"https://pipedapi.moomoo.me",
	"https://api.piped.projectsegfau.lt",
}

type pipedStreamsResp struct {
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	Uploader     string `json:"uploader"`
	Duration     int    `json:"duration"`
	AudioStreams  []struct {
		URL     string `json:"url"`
		Bitrate int    `json:"bitrate"`
	} `json:"audioStreams"`
}

func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}

	// ── Step 1: Search via r.jina.ai (unchanged, works on HF) ───────────────
	fmt.Printf("[Audio] Searching (via r.jina.ai) for: %s\n", query)
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
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	thumbnail := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
	watchURL := "https://www.youtube.com/watch?v=" + videoID

	// ── Step 2: Get stream URL from Piped API ────────────────────────────────
	// Piped hosts the stream — HF can reach pipedapi.* even if not youtube.com
	var streamURL, title, uploader string
	duration := 0

	for _, inst := range pipedAPIInstances {
		u := fmt.Sprintf("%s/streams/%s", inst, videoID)
		r, err := httpClient.Get(u)
		if err != nil {
			fmt.Printf("[Audio] %s error: %v\n", inst, err)
			continue
		}
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()

		if len(b) == 0 || b[0] == '<' {
			fmt.Printf("[Audio] %s returned HTML\n", inst)
			continue
		}

		var ps pipedStreamsResp
		if err := json.Unmarshal(b, &ps); err != nil || len(ps.AudioStreams) == 0 {
			fmt.Printf("[Audio] %s bad response: %v\n", inst, err)
			continue
		}

		best := ps.AudioStreams[0]
		for _, a := range ps.AudioStreams {
			if a.Bitrate > best.Bitrate {
				best = a
			}
		}
		streamURL = best.URL
		title = ps.Title
		uploader = ps.Uploader
		duration = ps.Duration
		if ps.ThumbnailUrl != "" {
			thumbnail = ps.ThumbnailUrl
		}
		fmt.Printf("[Audio] Got stream from %s\n", inst)
		break
	}

	if streamURL == "" {
		c.JSON(500, gin.H{"error": "could not get audio stream from any Piped instance"})
		return
	}

	// ── Step 3: Download stream bytes in Go, then ffmpeg converts locally ────
	// ffmpeg NEVER touches the internet — it only reads a local file
	_ = os.MkdirAll("downloads", 0755)
	mp3Path := fmt.Sprintf("downloads/%s.mp3", videoID)

	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		tmpPath := fmt.Sprintf("downloads/%s_raw", videoID)

		fmt.Printf("[Audio] Downloading stream bytes...\n")
		req, _ := http.NewRequest("GET", streamURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		dlResp, err := httpClient.Do(req)
		if err != nil {
			c.JSON(500, gin.H{"error": "download failed: " + err.Error()})
			return
		}
		f, _ := os.Create(tmpPath)
		io.Copy(f, dlResp.Body)
		f.Close()
		dlResp.Body.Close()

		fmt.Printf("[Audio] Converting to mp3 with ffmpeg...\n")
		cmd := exec.Command("ffmpeg", "-y",
			"-i", tmpPath,       // local file input — no network
			"-vn",
			"-acodec", "libmp3lame",
			"-q:a", "2",
			mp3Path,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Audio] ffmpeg failed: %v\n%s\n", err, string(out))
			os.Remove(tmpPath)
			c.JSON(500, gin.H{"error": "ffmpeg conversion failed"})
			return
		}
		os.Remove(tmpPath)
		fmt.Printf("[Audio] mp3 ready: %s\n", mp3Path)
	}

	host := os.Getenv("SERVICE_URL")
	if host == "" {
		host = "https://mellow2006-mellowbotbackend.hf.space"
	}

	c.JSON(200, gin.H{
		"metadata": AudioMetadata{
			Title:     title,
			Author:    uploader,
			Thumbnail: thumbnail,
			Duration:  fmt.Sprintf("%ds", duration),
			URL:       watchURL,
		},
		"audioURL": fmt.Sprintf("%s/downloads/%s.mp3", host, videoID),
	})
}