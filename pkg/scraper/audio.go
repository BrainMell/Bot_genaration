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

                // WARP SOCKS5 proxy (Cloudflare WARP via wireproxy on port 1080)
                // YouTube blocks datacenter IPs; WARP routes through Cloudflare IPs.
                // If wireproxy isn't running, this flag is empty (no proxy).
                warpProxy := ""
                if _, err := os.Stat("/etc/wireproxy/warp.conf"); err == nil {
                        warpProxy = "socks5://127.0.0.1:1080"
                }

                // Cookies file (optional — user must export from a browser)
                cookiesArg := ""
                if _, err := os.Stat("/etc/yt-dlp/cookies.txt"); err == nil {
                        cookiesArg = "/etc/yt-dlp/cookies.txt"
                }

                // Attempt 1: SoundCloud search (works without proxy, gives 30s previews)
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

                // Attempt 2: YouTube with cookies + WARP proxy (if cookies file exists)
                if !downloaded && videoID != "" && cookiesArg != "" {
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
