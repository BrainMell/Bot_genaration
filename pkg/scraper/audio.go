package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// audio response shape (keeps compatibility)
type AudioMetadata struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	Duration  string `json:"duration"`
	URL       string `json:"url"`
}

// Piped instance candidates (try these; order matters)
var pipedInstances = []string{
	"https://piped.video",            // official front, sometimes proxied
	"https://piped.kavin.rocks",     // community instance
	"https://piped.adminforge.de",   // another instance
	// add/remove as you discover reliable ones for your region
}

// pipedStreams matches common Piped streams JSON layout
type pipedStreams struct {
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	Uploader     string `json:"uploader"`
	Duration     int    `json:"duration"`
	AudioStreams []struct {
		URL     string `json:"url"`
		Quality string `json:"quality"`
		Bitrate int    `json:"bitrate"`
		Mime    string `json:"mimeType,omitempty"`
	} `json:"audioStreams"`
}

func ScrapeAudio(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query required"})
		return
	}

	// local client avoids package-level redeclare issues
	httpClient := &http.Client{Timeout: 15 * time.Second}

	fmt.Printf("[Audio] Searching for: %s\n", query)

	// 1) Search via Jina proxy (works in Spaces)
	searchURL := "https://r.jina.ai/http://www.youtube.com/results?search_query=" + url.QueryEscape(query)
	resp, err := httpClient.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search failed", "detail": err.Error()})
		return
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		c.JSON(500, gin.H{"error": "read search failed", "detail": err.Error()})
		return
	}
	html := string(body)

	// 2) Extract first video id
	re := regexp.MustCompile(`watch\?v=([A-Za-z0-9_\-]{11})`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		c.JSON(500, gin.H{"error": "no video id found"})
		return
	}
	videoID := matches[1]
	fmt.Printf("[Audio] Found video id: %s\n", videoID)

	// Ensure downloads dir exists
	if err := os.MkdirAll("downloads", 0755); err != nil {
		c.JSON(500, gin.H{"error": "mkdir failed", "detail": err.Error()})
		return
	}
	outPath := fmt.Sprintf("downloads/%s.mp3", videoID)

	// if already downloaded, return quickly
	if _, err := os.Stat(outPath); err == nil {
		audioURL := "/downloads/" + videoID + ".mp3"
		c.JSON(200, gin.H{"audioURL": audioURL, "videoID": videoID})
		return
	}

	// 3) Try Piped instances for a direct audio stream (avoids contacting youtube.com)
	fmt.Printf("[Audio] Trying Piped instances for %s\n", videoID)
	var streamURL string
	for _, inst := range pipedInstances {
		// try common endpoints: /api/v1/streams/ID first, then /streams/ID
		tryPaths := []string{
			path.Join("api", "v1", "streams", videoID),
			path.Join("streams", videoID),
		}
		for _, p := range tryPaths {
			u, err := url.Parse(inst)
			if err != nil {
				continue
			}
			u.Path = path.Join(u.Path, p)
			// make request
			resp, err := httpClient.Get(u.String())
			if err != nil {
				fmt.Printf("[Audio] piped %s request error: %v\n", u.String(), err)
				continue
			}
			if resp.StatusCode != 200 {
				resp.Body.Close()
				fmt.Printf("[Audio] piped %s http %d\n", u.String(), resp.StatusCode)
				continue
			}
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var ps pipedStreams
			if err := json.Unmarshal(b, &ps); err != nil {
				fmt.Printf("[Audio] piped %s bad json: %v\n", u.String(), err)
				continue
			}
			if len(ps.AudioStreams) == 0 {
				fmt.Printf("[Audio] piped %s no audio streams\n", u.String())
				continue
			}
			// pick highest bitrate
			best := ps.AudioStreams[0]
			for _, a := range ps.AudioStreams {
				if a.Bitrate > best.Bitrate {
					best = a
				}
			}
			streamURL = best.URL
			fmt.Printf("[Audio] got piped stream from %s\n", inst)
			break
		}
		if streamURL != "" {
			break
		}
	}

	// Helper to download a URL to a temp file
	downloadToFile := func(src string, dest string) error {
		req, _ := http.NewRequest("GET", src, nil)
		// some streams want a browser UA
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; scraper/1.0)")
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return fmt.Errorf("http %d", resp.StatusCode)
		}
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, resp.Body)
		return err
	}

	// 4) If we got a streamURL from Piped, download it and convert to mp3
	if streamURL != "" {
		tmpPath := fmt.Sprintf("downloads/%s_stream", videoID)
		// try to infer extension
		ext := ".webm"
		if strings.Contains(streamURL, ".m4a") || strings.Contains(streamURL, "audio/mp4") {
			ext = ".m4a"
		}
		tmpPath = tmpPath + ext

		fmt.Printf("[Audio] Downloading piped stream (%s) to %s\n", streamURL, tmpPath)
		if err := downloadToFile(streamURL, tmpPath); err != nil {
			fmt.Printf("[Audio] failed downloading piped stream: %v\n", err)
			// fall through to yt-dlp fallback
		} else {
			// convert to mp3 using ffmpeg
			fmt.Printf("[Audio] converting %s -> %s\n", tmpPath, outPath)
			// remove any existing outPath
			_ = os.Remove(outPath)
			cmd := exec.Command("ffmpeg", "-y", "-i", tmpPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", outPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("[Audio] ffmpeg failed: %v\n", err)
				// attempt simple move if formats already mp3
				_ = os.Remove(outPath)
			} else {
				// done: remove temp
				_ = os.Remove(tmpPath)
				audioURL := "/downloads/" + videoID + ".mp3"
				c.JSON(200, gin.H{"audioURL": audioURL, "videoID": videoID})
				return
			}
		}
	}

	// 5) Fallback: run yt-dlp (existing fallback). This may fail if DNS to youtube.com is blocked.
	fmt.Println("[Audio] Falling back to yt-dlp (may fail if yt blocked)")
	ytOut := outPath
	// yt-dlp output template must point to full path; ensure extension .mp3
	cmd := exec.Command("yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"-o", ytOut,
		"https://www.youtube.com/watch?v="+videoID,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("[Audio] yt-dlp error: %v\n", err)
		c.JSON(500, gin.H{"error": "all download methods failed", "detail": err.Error()})
		return
	}
	// confirm
	if _, err := os.Stat(outPath); err != nil {
		c.JSON(500, gin.H{"error": "output missing after yt-dlp"})
		return
	}
	audioURL := "/downloads/" + videoID + ".mp3"
	c.JSON(200, gin.H{"audioURL": audioURL, "videoID": videoID})
}