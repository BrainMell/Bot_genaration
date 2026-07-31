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
        // Try jina.ai reader first, then fall back to direct YouTube search through WARP proxy
        videoID := ""
        thumbnail := ""
        watchURL := ""

        // WARP proxy detection (same as below)
        warpProxy := ""
        if _, err := os.Stat("/etc/wireproxy/warp.conf"); err == nil {
                warpProxy = "socks5://127.0.0.1:1080"
        }

        // Attempt 1: jina.ai reader (reads YouTube search page as text)
        httpClient := &http.Client{Timeout: 15 * time.Second}
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

        // Attempt 2: use yt-dlp itself to find the video ID (through WARP if available)
        // This is more reliable than jina.ai which can be rate-limited
        if videoID == "" {
                ytArgs := []string{
                        "--dump-json", "--no-download", "--no-warnings",
                        "ytsearch1:" + query,
                }
                if warpProxy != "" {
                        ytArgs = append([]string{"--proxy", warpProxy}, ytArgs...)
                }
                if _, err := os.Stat("/etc/yt-dlp/cookies.txt"); err == nil {
                        ytArgs = append([]string{"--cookies", "/etc/yt-dlp/cookies.txt"}, ytArgs...)
                }
                fmt.Printf("[Audio] Searching video ID via yt-dlp: yt-dlp %v\n", ytArgs)
                cmd := exec.Command("yt-dlp", ytArgs...)
                out, err := cmd.CombinedOutput()
                if err != nil {
                        outStr := string(out)
                        if len(outStr) > 300 {
                                outStr = outStr[:300]
                        }
                        fmt.Printf("[Audio] yt-dlp video ID search failed: %v\nOutput: %s\n", err, outStr)
                } else {
                        // yt-dlp --dump-json outputs a JSON line with "id" field
                        re := regexp.MustCompile(`"id"\s*:\s*"([A-Za-z0-9_\-]{11})"`)
                        if match := re.FindStringSubmatch(string(out)); len(match) >= 2 {
                                videoID = match[1]
                                thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
                                watchURL = "https://www.youtube.com/watch?v=" + videoID
                        }
                }
        }
        fmt.Printf("[Audio] Video ID: %s\n", videoID)

        if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
                downloaded := false

                // WARP proxy and cookies are already detected above (lines 46-89)
                // warpProxy and cookiesArg variables are in scope here.

                // Cookies file (optional — user must export from a browser)
                cookiesArg := ""
                if _, err := os.Stat("/etc/yt-dlp/cookies.txt"); err == nil {
                        cookiesArg = "/etc/yt-dlp/cookies.txt"
                }

                // Attempt 1: YouTube with cookies + WARP proxy (FULL TRACK — best quality)
                // Try this FIRST because it gives full songs, not 30s previews.
                if videoID != "" && cookiesArg != "" && warpProxy != "" {
                        fmt.Printf("[Audio] Trying YouTube with cookies + WARP for: %s\n", videoID)
                        ytArgs := []string{
                                "--proxy", warpProxy,
                                "--cookies", cookiesArg,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-check-certificate",
                                "-o", mp3Path,
                                watchURL,
                        }
                        cmdYt := exec.Command("yt-dlp", ytArgs...)
                        if out, err := cmdYt.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] YouTube cookies+WARP failed: %v\n%s\n", err, string(out))
                        } else {
                                // Verify the file is actually a full track (> 500KB = not a preview)
                                if info, err := os.Stat(mp3Path); err == nil && info.Size() > 500000 {
                                        downloaded = true
                                        fmt.Printf("[Audio] YouTube full track downloaded: %d bytes\n", info.Size())
                                } else if info != nil {
                                        fmt.Printf("[Audio] YouTube file too small (%d bytes), likely preview — trying next source\n", info.Size())
                                        os.Remove(mp3Path)
                                }
                        }
                }

                // Attempt 2: SoundCloud search (30s preview fallback)
                if !downloaded {
                        fmt.Printf("[Audio] Trying SoundCloud for: %s\n", query)
                        scArgs := []string{
                                "scsearch1:" + query,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-playlist",
                                "-o", mp3Path,
                        }
                        if warpProxy != "" {
                                scArgs = append([]string{"--proxy", warpProxy}, scArgs...)
                        }
                        cmd := exec.Command("yt-dlp", scArgs...)
                        if out, err := cmd.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] SoundCloud failed: %v\n%s\n", err, string(out))
                        } else {
                                downloaded = true
                        }
                }

                // Attempt 3: YouTube with default client + WARP proxy (no cookies)
                if !downloaded && videoID != "" && warpProxy != "" {
                        fmt.Printf("[Audio] Trying YouTube default client + WARP for: %s\n", videoID)
                        ytArgs := []string{
                                "--proxy", warpProxy,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-check-certificate",
                                "-o", mp3Path,
                                watchURL,
                        }
                        cmdYt := exec.Command("yt-dlp", ytArgs...)
                        if out, err := cmdYt.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] YouTube default+WARP failed: %v\n%s\n", err, string(out))
                        } else {
                                downloaded = true
                        }
                }

                // Attempt 4: YouTube TV client (no proxy — last resort)
                if !downloaded && videoID != "" {
                        fmt.Printf("[Audio] Trying YouTube TV client (no proxy) for: %s\n", videoID)
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

                if !downloaded {
                        c.JSON(500, gin.H{"error": "all download attempts failed (YouTube blocks datacenter IPs — need cookies file at /etc/yt-dlp/cookies.txt for full tracks)"})
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
