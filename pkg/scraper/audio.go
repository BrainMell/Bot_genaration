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

        // Declare iTunes metadata variables at function scope (used in STEP 4 below)
        itunesPreviewURL := ""
        itunesTitle := ""
        itunesArtist := ""
        itunesArtwork := ""

        // Cookies file (optional — user must export from a browser)
        cookiesArg := ""
        if _, err := os.Stat("/etc/yt-dlp/cookies.txt"); err == nil {
                cookiesArg = "/etc/yt-dlp/cookies.txt"
        }

        if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
                downloaded := false

                // WARP proxy and cookies are already detected above (lines 46-89)
                // warpProxy and cookiesArg variables are in scope here.

                // ════════════════════════════════════════════════════════════════
                // STEP 1: Get metadata from iTunes (title, artist, artwork, preview URL)
                // Always do this first — gives us clean search term + thumbnail.
                // Don't download the preview yet — save it as last-resort fallback.
                // ════════════════════════════════════════════════════════════════

                fmt.Printf("[Audio] Fetching iTunes metadata for: %s\n", query)
                itunesURL := "https://itunes.apple.com/search?term=" + url.QueryEscape(query) + "&media=music&limit=1"
                if resp, err := httpClient.Get(itunesURL); err == nil {
                        body, _ := io.ReadAll(resp.Body)
                        resp.Body.Close()
                        bodyStr := string(body)
                        // Extract metadata
                        if m := regexp.MustCompile(`"trackName"\s*:\s*"([^"]+)"`).FindStringSubmatch(bodyStr); len(m) >= 2 {
                                itunesTitle = m[1]
                        }
                        if m := regexp.MustCompile(`"artistName"\s*:\s*"([^"]+)"`).FindStringSubmatch(bodyStr); len(m) >= 2 {
                                itunesArtist = m[1]
                        }
                        if m := regexp.MustCompile(`"artworkUrl100"\s*:\s*"(https://[^"]+)"`).FindStringSubmatch(bodyStr); len(m) >= 2 {
                                // Upgrade artwork to higher resolution (100x100 → 600x600)
                                itunesArtwork = strings.Replace(m[1], "100x100", "600x600", 1)
                        }
                        if m := regexp.MustCompile(`"previewUrl"\s*:\s*"(https://[^"]+)"`).FindStringSubmatch(bodyStr); len(m) >= 2 {
                                itunesPreviewURL = m[1]
                        }
                        fmt.Printf("[Audio] iTunes: title=%s, artist=%s, artwork=%s, preview=%s\n",
                                itunesTitle, itunesArtist, itunesArtwork, itunesPreviewURL != "")
                }

                // Use iTunes title as search query for better YouTube/SoundCloud results
                searchQuery := query
                if itunesTitle != "" && itunesArtist != "" {
                        searchQuery = itunesTitle + " " + itunesArtist
                }

                // ════════════════════════════════════════════════════════════════
                // STEP 2: Try FULL TRACK sources (in priority order)
                // ════════════════════════════════════════════════════════════════

                // ── Attempt 1: YouTube with cookies + WARP (FULL TRACK) ──
                if !downloaded && videoID != "" && cookiesArg != "" && warpProxy != "" {
                        fmt.Printf("[Audio] Trying YouTube (cookies+WARP) for: %s\n", videoID)
                        ytArgs := []string{
                                "--proxy", warpProxy,
                                "--cookies", cookiesArg,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-check-certificate",
                                "-o", mp3Path,
                                watchURL,
                        }
                        cmdYt := exec.Command("yt-dlp", ytArgs...)
                        if _, err := cmdYt.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] YouTube cookies+WARP failed: %v\n", err)
                        } else {
                                // Check file size — full tracks are > 500KB
                                if info, err := os.Stat(mp3Path); err == nil && info.Size() > 500000 {
                                        downloaded = true
                                        fmt.Printf("[Audio] YouTube FULL TRACK: %d bytes\n", info.Size())
                                } else if info != nil {
                                        fmt.Printf("[Audio] YouTube file too small (%d bytes), removing\n", info.Size())
                                        os.Remove(mp3Path)
                                }
                        }
                }

                // ── Attempt 2: YouTube default client + WARP (no cookies) ──
                if !downloaded && videoID != "" && warpProxy != "" {
                        fmt.Printf("[Audio] Trying YouTube (default+WARP, no cookies) for: %s\n", videoID)
                        ytArgs := []string{
                                "--proxy", warpProxy,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-check-certificate",
                                "-o", mp3Path,
                                watchURL,
                        }
                        cmdYt := exec.Command("yt-dlp", ytArgs...)
                        if _, err := cmdYt.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] YouTube default+WARP failed: %v\n", err)
                        } else {
                                if info, err := os.Stat(mp3Path); err == nil && info.Size() > 500000 {
                                        downloaded = true
                                        fmt.Printf("[Audio] YouTube (no cookies) FULL TRACK: %d bytes\n", info.Size())
                                } else if info != nil {
                                        os.Remove(mp3Path)
                                }
                        }
                }

                // ── Attempt 3: SoundCloud search (full track or preview) ──
                if !downloaded {
                        fmt.Printf("[Audio] Trying SoundCloud for: %s\n", searchQuery)
                        scArgs := []string{
                                "scsearch1:" + searchQuery,
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-playlist",
                                "-o", mp3Path,
                        }
                        if warpProxy != "" {
                                scArgs = append([]string{"--proxy", warpProxy}, scArgs...)
                        }
                        cmd := exec.Command("yt-dlp", scArgs...)
                        if _, err := cmd.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] SoundCloud failed: %v\n", err)
                        } else {
                                if info, err := os.Stat(mp3Path); err == nil {
                                        downloaded = true
                                        isFull := info.Size() > 500000
                                        fmt.Printf("[Audio] SoundCloud downloaded (%s): %d bytes\n",
                                                map[bool]string{true: "FULL", false: "preview"}[isFull], info.Size())
                                }
                        }
                }

                // ── Attempt 4: Audius API (full track, 320kbps, indie) ──
                if !downloaded {
                        fmt.Printf("[Audio] Trying Audius for: %s\n", searchQuery)
                        audiusSearchURL := "https://discoveryprovider.audius.co/v1/tracks/search?query=" + url.QueryEscape(searchQuery) + "&limit=1"
                        if resp, err := httpClient.Get(audiusSearchURL); err == nil {
                                body, _ := io.ReadAll(resp.Body)
                                resp.Body.Close()
                                trackIDRe := regexp.MustCompile(`"id"\s*:\s*"([A-Za-z0-9_-]+)"`)
                                if m := trackIDRe.FindStringSubmatch(string(body)); len(m) >= 2 {
                                        trackID := m[1]
                                        streamURL := "https://discoveryprovider.audius.co/v1/tracks/" + trackID + "/stream"
                                        if resp2, err := httpClient.Get(streamURL); err == nil {
                                                streamData, _ := io.ReadAll(resp2.Body)
                                                resp2.Body.Close()
                                                if len(streamData) > 50000 {
                                                        os.WriteFile(mp3Path, streamData, 0644)
                                                        downloaded = true
                                                        fmt.Printf("[Audio] Audius FULL TRACK: %d bytes\n", len(streamData))
                                                }
                                        }
                                }
                        }
                }

                // ── Attempt 5: YouTube TV client (no proxy — last resort for full track) ──
                if !downloaded && videoID != "" {
                        fmt.Printf("[Audio] Trying YouTube TV (no proxy) for: %s\n", videoID)
                        cmdYt := exec.Command("yt-dlp",
                                "-x", "--audio-format", "mp3", "--audio-quality", "0",
                                "--extractor-args", "youtube:player_client=tv",
                                "--no-check-certificate",
                                "-o", mp3Path,
                                watchURL,
                        )
                        if _, err := cmdYt.CombinedOutput(); err != nil {
                                fmt.Printf("[Audio] YouTube TV failed: %v\n", err)
                        } else {
                                if info, err := os.Stat(mp3Path); err == nil && info.Size() > 500000 {
                                        downloaded = true
                                        fmt.Printf("[Audio] YouTube TV FULL TRACK: %d bytes\n", info.Size())
                                } else if info != nil {
                                        os.Remove(mp3Path)
                                }
                        }
                }

                // ════════════════════════════════════════════════════════════════
                // STEP 3: LAST RESORT — iTunes 30s preview
                // Only if ALL full-track sources failed.
                // ════════════════════════════════════════════════════════════════
                if !downloaded && itunesPreviewURL != "" {
                        fmt.Printf("[Audio] Falling back to iTunes 30s preview\n")
                        if resp2, err := httpClient.Get(itunesPreviewURL); err == nil {
                                previewData, _ := io.ReadAll(resp2.Body)
                                resp2.Body.Close()
                                if len(previewData) > 10000 {
                                        tmpM4a := mp3Path + ".m4a"
                                        os.WriteFile(tmpM4a, previewData, 0644)
                                        cmdFfmpeg := exec.Command("ffmpeg", "-y", "-i", tmpM4a, "-codec:a", "libmp3lame", "-b:a", "128k", mp3Path)
                                        if err := cmdFfmpeg.Run(); err == nil {
                                                os.Remove(tmpM4a)
                                                downloaded = true
                                                fmt.Printf("[Audio] iTunes 30s preview downloaded: %d bytes\n", len(previewData))
                                        } else {
                                                os.Remove(tmpM4a)
                                        }
                                }
                        }
                }

                if !downloaded {
                        c.JSON(500, gin.H{"error": "all download attempts failed"})
                        return
                }
        }

        // ════════════════════════════════════════════════════════════════
        // STEP 4: Use iTunes metadata for title/thumbnail if available
        // (overrides the YouTube/SoundCloud metadata for consistency)
        // ════════════════════════════════════════════════════════════════
        if itunesTitle != "" {
                // Build a clean title: "Song Name - Artist"
                if itunesArtist != "" {
                        // Don't set the title field — let the Node bot use it as fileName
                }
        }
        if itunesArtwork != "" {
                thumbnail = itunesArtwork
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

        // Use iTunes title/artist if available, otherwise use the query
        displayTitle := query
        displayAuthor := "Audio"
        if itunesTitle != "" {
                displayTitle = itunesTitle
        }
        if itunesArtist != "" {
                displayAuthor = itunesArtist
        }

        c.JSON(200, gin.H{
                "metadata": AudioMetadata{
                        Title:     displayTitle,
                        Author:    displayAuthor,
                        Thumbnail: thumbnail,
                        Duration:  "",
                        URL:       watchURL,
                },
                "audioURL": fmt.Sprintf("%s/downloads/%s.mp3", baseURL, hash),
        })
}
