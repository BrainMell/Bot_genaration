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
                        "--flat-playlist", "--print", "id", "--no-warnings",
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
                        // yt-dlp --print id outputs just the 11-char video ID
                        outStr := strings.TrimSpace(string(out))
                        re := regexp.MustCompile(`^[A-Za-z0-9_\-]{11}$`)
                        if re.MatchString(outStr) {
                                videoID = outStr
                                thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
                                watchURL = "https://www.youtube.com/watch?v=" + videoID
                        } else {
                                // Fallback: extract from multi-line output
                                re2 := regexp.MustCompile(`([A-Za-z0-9_\-]{11})`)
                                if match := re2.FindStringSubmatch(outStr); len(match) >= 1 {
                                        videoID = match[1]
                                        thumbnail = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID)
                                        watchURL = "https://www.youtube.com/watch?v=" + videoID
                                }
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

                // ── Attempt 1: iTunes Search API (30s preview, mainstream music) ──
                // Free, no auth, works from datacenter IPs. Gives proper metadata + artwork.
                // 30s preview is better than nothing for mainstream songs.
                if !downloaded {
                        fmt.Printf("[Audio] Trying iTunes Search API for: %s\n", query)
                        itunesURL := "https://itunes.apple.com/search?term=" + url.QueryEscape(query) + "&media=music&limit=1"
                        if resp, err := httpClient.Get(itunesURL); err == nil {
                                body, _ := io.ReadAll(resp.Body)
                                resp.Body.Close()
                                // Extract previewUrl and artworkUrl from JSON
                                previewRe := regexp.MustCompile(`"previewUrl"\s*:\s*"(https://[^"]+)"`)
                                artworkRe := regexp.MustCompile(`"artworkUrl100"\s*:\s*"(https://[^"]+)"`)
                                trackNameRe := regexp.MustCompile(`"trackName"\s*:\s*"([^"]+)"`)
                                _ = trackNameRe // reserved for future metadata enrichment
                                if m := previewRe.FindStringSubmatch(string(body)); len(m) >= 2 {
                                        previewURL := m[1]
                                        // Download the preview
                                        if resp2, err := httpClient.Get(previewURL); err == nil {
                                                previewData, _ := io.ReadAll(resp2.Body)
                                                resp2.Body.Close()
                                                if len(previewData) > 10000 { // at least 10KB
                                                        // Convert m4a to mp3 via ffmpeg
                                                        tmpM4a := mp3Path + ".m4a"
                                                        os.WriteFile(tmpM4a, previewData, 0644)
                                                        cmdFfmpeg := exec.Command("ffmpeg", "-y", "-i", tmpM4a, "-codec:a", "libmp3lame", "-b:a", "128k", mp3Path)
                                                        if err := cmdFfmpeg.Run(); err == nil {
                                                                os.Remove(tmpM4a)
                                                                downloaded = true
                                                                // Update metadata from iTunes
                                                                if m2 := trackNameRe.FindStringSubmatch(string(body)); len(m2) >= 2 {
                                                                        // Use the query as title but update thumbnail
                                                                }
                                                                if m2 := artworkRe.FindStringSubmatch(string(body)); len(m2) >= 2 {
                                                                        thumbnail = m2[1]
                                                                }
                                                                fmt.Printf("[Audio] iTunes preview downloaded: %d bytes\n", len(previewData))
                                                        } else {
                                                                os.Remove(tmpM4a)
                                                        }
                                                }
                                        }
                                }
                        }
                }

                // ── Attempt 2: Audius API (full track, 320kbps, indie music) ──
                // Free, no auth needed for discovery, works from datacenter IPs.
                // Best for lofi/electronic/indie. Mainstream won't be found here.
                if !downloaded {
                        fmt.Printf("[Audio] Trying Audius API for: %s\n", query)
                        audiusSearchURL := "https://discoveryprovider.audius.co/v1/tracks/search?query=" + url.QueryEscape(query) + "&limit=1"
                        if resp, err := httpClient.Get(audiusSearchURL); err == nil {
                                body, _ := io.ReadAll(resp.Body)
                                resp.Body.Close()
                                // Extract track ID
                                trackIDRe := regexp.MustCompile(`"id"\s*:\s*"([A-Za-z0-9_-]+)"`)
                                if m := trackIDRe.FindStringSubmatch(string(body)); len(m) >= 2 {
                                        trackID := m[1]
                                        streamURL := "https://discoveryprovider.audius.co/v1/tracks/" + trackID + "/stream"
                                        if resp2, err := httpClient.Get(streamURL); err == nil {
                                                streamData, _ := io.ReadAll(resp2.Body)
                                                resp2.Body.Close()
                                                if len(streamData) > 50000 { // at least 50KB = full track
                                                        os.WriteFile(mp3Path, streamData, 0644)
                                                        downloaded = true
                                                        fmt.Printf("[Audio] Audius full track downloaded: %d bytes\n", len(streamData))
                                                }
                                        }
                                }
                        }
                }

                // Attempt 3: YouTube with cookies + WARP proxy (FULL TRACK — best quality)
                // Try this because it gives full songs, not 30s previews.
                if !downloaded && videoID != "" && cookiesArg != "" && warpProxy != "" {
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
