package scraper

// Audio pipeline v3 (2026-09-14) - owner: ".j audio either finds a 30sec
// preview or the wrong song, and YouTube tooling is blocked".
//
// Root causes in v1: (a) YouTube search via an r.jina.ai text dump took the
// FIRST watch?v= match - often an unrelated video; (b) SoundCloud's public
// API now returns 30-SECOND PREVIEWS for most tracks (verified: the official
// "Starboy (feat. Daft Punk)" upload resolves to dur=30.0); (c) YouTube is
// bot-blocked on both datacenter IPs ("Sign in to confirm you're not a bot").
//
// v3 chain (verified end-to-end on 2026-09-14):
//  1. Deezer public API - authoritative match: official title, artist,
//     canonical duration, album art. Anchor for every later check.
//  2. JioSaavn - cascade + scored ranking; accepted only when the pick
//     matches the Deezer artist and duration (±12s). Excellent 320kbps
//     source for regional/catalog music, a minefield of covers for
//     international hits - the Deezer anchor is what filters those.
//  3. YouTube via WARP socks proxy (audio_proxy.conf / AUDIO_PROXY_URL):
//     ytsearch5 scored by duration (±15s), uploader match and title
//     tokens, downloaded with player_client=tv_simply or mweb (the
//     clients that still serve a usable audio format through WARP -
//     web_embedded went video-only in 2026, tv bot-checks). Top-3
//     candidates are attempted before giving up. ".audio <YouTube URL>"
//     bypasses search and downloads the exact linked video (v3.1).
//     Direct-IP YouTube is dead on both boxes.
//  4. SoundCloud - last resort, post-download duration guard rejects
//     30s previews.
//
// Every candidate is ffprobe-verified after download: rejected if shorter
// than 60s while the canonical track is ≥90s (preview), or shorter than 45%
// of canonical (wrong cut). Cache is salted v3 (v1/v2 cached wrong songs).

import (
        "context"
        "crypto/des"
        "crypto/md5"
        "encoding/base64"
        "encoding/json"
        "fmt"
        "html"
        "io"
        "math"
        "net/http"
        "net/url"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "sort"
        "strconv"
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

type deezerTrack struct {
        Title    string `json:"title"`
        Duration int    `json:"duration"`
        Artist   struct {
                Name string `json:"name"`
        } `json:"artist"`
        Album struct {
                Title    string `json:"title"`
                CoverXL  string `json:"cover_xl"`
                CoverBig string `json:"cover_big"`
        } `json:"album"`
        Link string `json:"link"`
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

func execCtx(d time.Duration) (context.Context, context.CancelFunc) {
        return context.WithTimeout(context.Background(), d)
}

// childEnv - pm2 sets NODE_CHANNEL_FD (and related vars) in the service
// environment. Leaked into child processes, they break deno (yt-dlp's JS
// challenge solver: "Failed to open IPC channel from NODE_CHANNEL_FD") and
// with it all PO-token-protected YouTube formats. Strip them for every
// external tool we spawn.
func childEnv() []string {
        env := os.Environ()
        clean := env[:0]
        for _, kv := range env {
                upper := kv
                if i := strings.Index(kv, "="); i > 0 {
                        upper = kv[:i]
                }
                if strings.HasPrefix(upper, "NODE_CHANNEL") || upper == "NODE_UNIQUE_ID" ||
                        strings.HasPrefix(upper, "PM2_") {
                        continue
                }
                clean = append(clean, kv)
        }
        return clean
}

func tailBytes(b []byte, n int) string {
        s := string(b)
        if len(s) > n {
                return s[len(s)-n:]
        }
        return s
}

func strOr(v interface{}, def string) string {
        if s, ok := v.(string); ok && s != "" {
                return s
        }
        return def
}

func numOr(v interface{}) float64 {
        if f, ok := v.(float64); ok {
                return f
        }
        return 0
}

func ffmpegToMP3(src, dst string) error {
        ctx, cancel := execCtx(90 * time.Second)
        defer cancel()
        cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", src,
                "-codec:a", "libmp3lame", "-q:a", "0", dst)
        cmd.Env = childEnv()
        if out, err := cmd.CombinedOutput(); err != nil {
                return fmt.Errorf("ffmpeg: %v: %s", err, tailBytes(out, 200))
        }
        return nil
}

func audioDuration(path string) float64 {
        ctx, cancel := execCtx(10 * time.Second)
        defer cancel()
        out, err := exec.CommandContext(ctx, "ffprobe", "-v", "quiet",
                "-show_entries", "format=duration", "-of", "csv=p=0", path).Output()
        if err != nil {
                return 0
        }
        d, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
        return d
}

// durationSane - reject previews and wildly wrong cuts.
func durationSane(got float64, want int) bool {
        if got < 1 {
                return false
        }
        if want >= 90 {
                return got >= 60 && got >= float64(want)*0.45
        }
        return got >= 30
}

var badTitleRe = regexp.MustCompile(`(?i)(\bkaraoke\b|\binstrumental\b|\bcover\b|\bremix\b|\bsped\b|\bslowed\b|\breverb\b|\bnightcore\b|\btribute\b|\blullaby\b|\b8\s*-?\s*d\b|\b(?:9|12|15|16)\s*-?\s*d\s*(?:audio|mix|version)?\b|\blofi\b|\blo-fi\b|\bringtone\b|\bmashup\b|\breaction\b|\blesson\b|\btutorial\b|\b1 hour\b|\b10 minutes\b)`)

// ---------------------------------------------------------------------------
// Deezer - authoritative match
// ---------------------------------------------------------------------------

var deezerHTTP = &http.Client{Timeout: 8 * time.Second}

// deezerClient - api.deezer.com is NOT reachable from every box's network;
// when a WARP proxy is configured (Box2), route Deezer through it too.
func deezerClient() *http.Client {
        if p := audioProxyURL(); p != "" {
                if pu, err := url.Parse(p); err == nil {
                        return &http.Client{Timeout: 15 * time.Second,
                                Transport: &http.Transport{Proxy: http.ProxyURL(pu)}}
                }
        }
        return deezerHTTP
}

func deezerMatch(query string) *deezerTrack {
        api := "https://api.deezer.com/search?q=" + url.QueryEscape(query) + "&limit=5"
        req, err := http.NewRequest("GET", api, nil)
        if err != nil {
                return nil
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
        resp, err := deezerClient().Do(req)
        if err != nil {
                fmt.Printf("[Audio v3] deezer unreachable: %v\n", err)
                return nil
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        var parsed struct {
                Data []deezerTrack `json:"data"`
        }
        if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Data) == 0 {
                fmt.Printf("[Audio v3] deezer empty/invalid (status %d, proxy=%v): %s\n",
                        resp.StatusCode, audioProxyURL() != "", tailBytes(body, 200))
                return nil
        }
        return &parsed.Data[0]
}

// ---------------------------------------------------------------------------
// JioSaavn
// ---------------------------------------------------------------------------

type saavnSearchResp struct {
        Results []saavnSong `json:"results"`
}

type saavnSong struct {
        ID        string `json:"id"`
        Title     string `json:"title"`
        Subtitle  string `json:"subtitle"`
        Image     string `json:"image"`
        PermaURL  string `json:"perma_url"`
        PlayCount string `json:"play_count"`
        MoreInfo  struct {
                Album             string `json:"album"`
                PrimaryArtists    string `json:"primary_artists"`
                Duration          string `json:"duration"`
                EncryptedMediaURL string `json:"encrypted_media_url"`
                Kbps320           string `json:"320kbps"`
                PlayCount         string `json:"play_count"`
                ArtistMap         struct {
                        PrimaryArtists []struct {
                                Name string `json:"name"`
                        } `json:"primary_artists"`
                } `json:"artistMap"`
        } `json:"more_info"`
}

var saavnHTTP = &http.Client{Timeout: 10 * time.Second}

func saavnSearch(q string, n int) []saavnSong {
        api := "https://www.jiosaavn.com/api.php?__call=search.getResults&q=" +
                url.QueryEscape(q) + "&_format=json&_marker=0&api_version=4&ctx=web6dot0" +
                "&n=" + strconv.Itoa(n) + "&p=1"
        req, err := http.NewRequest("GET", api, nil)
        if err != nil {
                return nil
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
        req.Header.Set("Accept", "application/json")
        resp, err := saavnHTTP.Do(req)
        if err != nil {
                return nil
        }
        defer resp.Body.Close()
        var out saavnSearchResp
        if json.NewDecoder(resp.Body).Decode(&out) != nil {
                return nil
        }
        return out.Results
}

// decryptSaavnURL - JioSaavn media URLs are DES-ECB encrypted with the
// well-known static key "38346591", base64 wrapped, then percent-encoded.
func decryptSaavnURL(enc string) (string, error) {
        block, err := des.NewCipher([]byte("38346591"))
        if err != nil {
                return "", err
        }
        data, err := base64.StdEncoding.DecodeString(enc)
        if err != nil || len(data) == 0 || len(data)%8 != 0 {
                return "", fmt.Errorf("bad encrypted media payload (len %d)", len(data))
        }
        out := make([]byte, len(data))
        for i := 0; i < len(data); i += 8 { // manual ECB
                block.Decrypt(out[i:i+8], data[i:i+8])
        }
        s := strings.TrimRight(string(out), "\x00\r\n ")
        if n := len(s); n > 0 {
                pad := int(s[n-1])
                if pad >= 1 && pad <= 8 && strings.HasSuffix(s, strings.Repeat(string(s[n-1]), pad)) {
                        s = s[:n-pad]
                }
        }
        return url.QueryUnescape(s)
}

func saavnArtists(s *saavnSong) string {
        names := make([]string, 0, 4)
        if s.MoreInfo.PrimaryArtists != "" {
                names = append(names, strings.ToLower(html.UnescapeString(s.MoreInfo.PrimaryArtists)))
        }
        for _, a := range s.MoreInfo.ArtistMap.PrimaryArtists {
                if a.Name != "" {
                        names = append(names, strings.ToLower(html.UnescapeString(a.Name)))
                }
        }
        return strings.Join(names, " ")
}

func saavnScore(s *saavnSong, query string, dz *deezerTrack) float64 {
        title := strings.ToLower(html.UnescapeString(strings.TrimSpace(s.Title)))
        artists := saavnArtists(s)
        text := title + " " + artists
        dur := 0
        fmt.Sscanf(s.MoreInfo.Duration, "%d", &dur)

        score := 0.0
        if badTitleRe.MatchString(title) {
                score -= 50
        }
        if badTitleRe.MatchString(text) {
                score -= 20
        }
        q := strings.ToLower(strings.TrimSpace(query))
        tokens := strings.Fields(q)
        cover := 0
        for _, t := range tokens {
                if len(t) > 2 && strings.Contains(text, t) {
                        cover++
                }
        }
        if len(tokens) > 0 {
                score += 30 * float64(cover) / float64(len(tokens))
        }
        if q != title && strings.HasPrefix(q, title) && len(title) >= 4 {
                score += 15
        }
        if q == title {
                score += 15
        }
        if dur >= 60 && dur <= 900 {
                score += 10
        } else if dur > 0 && dur < 60 {
                score -= 25
        }
        if s.MoreInfo.Kbps320 == "true" {
                score += 5
        }
        album := strings.ToLower(html.UnescapeString(s.MoreInfo.Album))
        if album != "" && (album == title || strings.HasPrefix(album, title)) {
                score += 8
        }
        pc := 0
        fmt.Sscanf(s.MoreInfo.PlayCount, "%d", &pc)
        if pc == 0 {
                fmt.Sscanf(s.PlayCount, "%d", &pc)
        }
        if pc > 0 {
                score += math.Min(10, float64(pc)/1_000_000)
        }
        // Deezer anchor: artist identity and duration proximity are decisive.
        if dz != nil {
                dzArtist := strings.ToLower(dz.Artist.Name)
                if dzArtist != "" && strings.Contains(artists, dzArtist) {
                        score += 20
                } else if dzArtist != "" && dur > 0 {
                        score -= 15 // JioSaavn hit exists but the artist doesn't match
                }
                if dz.Duration > 0 && dur > 0 {
                        d := dur - dz.Duration
                        if d < 0 {
                                d = -d
                        }
                        if d <= 12 {
                                score += 10
                        } else if d > 45 {
                                score -= 8
                        }
                }
        }
        return score
}

func saavnQueryCascade(query string, dz *deezerTrack) []string {
        add := func(list []string, q string) []string {
                q = strings.TrimSpace(q)
                if q == "" {
                        return list
                }
                for _, v := range list {
                        if strings.EqualFold(v, q) {
                                return list
                        }
                }
                return append(list, q)
        }
        list := []string{query}
        low := strings.ToLower(query)
        if i := strings.Index(low, " by "); i > 0 {
                list = add(list, query[:i])
        }
        if strings.Contains(query, " - ") {
                list = add(list, strings.SplitN(query, " - ", 2)[0])
        }
        words := strings.Fields(query)
        if len(words) > 3 {
                list = add(list, strings.Join(words[:3], " "))
        }
        if len(words) > 2 {
                list = add(list, strings.Join(words[:2], " "))
        }
        if len(words) > 1 && len(words[0]) >= 4 {
                list = add(list, words[0])
        }
        // Deezer anchor makes "<official title>" the strongest query.
        if dz != nil && dz.Title != "" {
                list = add(list, dz.Title+" "+dz.Artist.Name)
                list = add(list, dz.Title)
        }
        return list
}

func saavnPickBest(query string, dz *deezerTrack) *saavnSong {
        var best *saavnSong
        bestScore := -1e9
        for _, q := range saavnQueryCascade(query, dz) {
                songs := saavnSearch(q, 20)
                for i := range songs {
                        sc := saavnScore(&songs[i], query, dz)
                        if sc > bestScore {
                                bestScore = sc
                                best = &songs[i]
                        }
                }
                if best != nil && bestScore >= 60 {
                        break
                }
        }
        if best != nil && bestScore < 25 {
                return nil // nothing credible - don't waste a download on junk
        }
        return best
}

// saavnTrusted - a JioSaavn pick is trusted as the FIRST choice only when
// the credited artists match the Deezer anchor (or are unverifiable).
// JioSaavn's search is polluted with regional covers of international hits;
// this gate is what sends those queries to the YouTube/WARP source instead.
func saavnTrusted(s *saavnSong, dz *deezerTrack) bool {
        if dz == nil || dz.Artist.Name == "" {
                return true
        }
        artists := saavnArtists(s)
        if artists == "" {
                return true
        }
        return strings.Contains(artists, strings.ToLower(dz.Artist.Name))
}

func saavnArtistDisplay(s *saavnSong) string {
        if s.MoreInfo.PrimaryArtists != "" {
                return html.UnescapeString(s.MoreInfo.PrimaryArtists)
        }
        parts := make([]string, 0, 3)
        for i, a := range s.MoreInfo.ArtistMap.PrimaryArtists {
                if i >= 3 {
                        break
                }
                parts = append(parts, a.Name)
        }
        return html.UnescapeString(strings.Join(parts, ", "))
}

// ---------------------------------------------------------------------------
// yt-dlp: YouTube (via WARP) and SoundCloud
// ---------------------------------------------------------------------------

func audioProxyURL() string {
        if v := os.Getenv("AUDIO_PROXY_URL"); v != "" {
                return v
        }
        if b, err := os.ReadFile("audio_proxy.conf"); err == nil {
                if v := strings.TrimSpace(string(b)); v != "" {
                        return v
                }
        }
        return ""
}

func ytdlpJSON(args ...string) (map[string]interface{}, error) {
        ctx, cancel := execCtx(100 * time.Second)
        defer cancel()
        args = append(args, "--print-json", "--no-simulate", "--no-warnings",
                "--no-playlist", "-x", "--audio-format", "mp3", "--audio-quality", "0")
        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
        cmd.Env = childEnv()
        out, err := cmd.CombinedOutput()
        if err != nil {
                return nil, fmt.Errorf("%v: %s", err, tailBytes(out, 300))
        }
        for _, line := range strings.Split(string(out), "\n") {
                line = strings.TrimSpace(line)
                if strings.HasPrefix(line, "{") {
                        var m map[string]interface{}
                        if json.Unmarshal([]byte(line), &m) == nil {
                                return m, nil
                        }
                }
        }
        return nil, fmt.Errorf("no json metadata in yt-dlp output")
}

// ytProbeCandidates - resolve ytsearch results (metadata only, no download).
// Tolerates partial failures: yt-dlp may exit non-zero after printing some
// candidate JSON (shared-IP throttling) - parse whatever was printed.
func ytProbeCandidates(searchQuery string, n int, proxy string) []map[string]interface{} {
        ctx, cancel := execCtx(60 * time.Second)
        defer cancel()
        // --flat-playlist: search-API results only (id/title/uploader/duration)
        // - no per-video player calls, no challenge solving, ~5s total. This
        // keeps the download as the ONLY heavy YouTube interaction; probing
        // via full extraction used to burn ~15 innertube calls and throttle
        // the shared WARP IP right before the download.
        args := []string{"ytsearch" + strconv.Itoa(n) + ":" + searchQuery,
                "--flat-playlist", "--print-json", "--no-warnings", "--skip-download"}
        if proxy != "" {
                args = append([]string{"--proxy", proxy}, args...)
        }
        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
        cmd.Env = childEnv()
        out, err := cmd.CombinedOutput()
        var list []map[string]interface{}
        for _, line := range strings.Split(string(out), "\n") {
                line = strings.TrimSpace(line)
                if strings.HasPrefix(line, "{") {
                        var m map[string]interface{}
                        if json.Unmarshal([]byte(line), &m) == nil {
                                list = append(list, m)
                        }
                }
        }
        if len(list) == 0 {
                fmt.Printf("[Audio v3] yt probe failed: %v: %s\n", err, tailBytes(out, 200))
        }
        return list
}

func ytScoreCandidate(m map[string]interface{}, dz *deezerTrack, artistQuery string) float64 {
        title := strings.ToLower(strOr(m["title"], ""))
        uploader := strings.ToLower(strOr(m["uploader"], ""))
        dur := numOr(m["duration"])
        score := 0.0
        if badTitleRe.MatchString(title) {
                score -= 30
        }
        if dz != nil && dz.Artist.Name != "" && strings.Contains(uploader, strings.ToLower(dz.Artist.Name)) {
                score += 25 // official channel upload
        }
        if dz != nil && dz.Duration > 0 && dur > 0 {
                d := dur - float64(dz.Duration)
                if d < 0 {
                        d = -d
                }
                if d <= 15 {
                        score += 30
                } else if d <= 30 {
                        score += 10
                } else {
                        score -= 15
                }
        }
        for _, t := range strings.Fields(artistQuery) {
                if len(t) > 2 && (strings.Contains(title, t) || strings.Contains(uploader, t)) {
                        score += 5
                }
        }
        return score
}

func ytDownload(id string, mp3Path string, proxy string) error {
        // v3.1 client order (verified live 2026-09-14 via WARP): tv_simply
        // and mweb still expose a downloadable audio format (progressive
        // format 18); web_embedded now serves VIDEO-ONLY formats (yt-dlp
        // -x fails "Requested format is not available"); tv bot-checks
        // ("The page needs to be reloaded"). (Direct-IP youtube is
        // bot-checked on both boxes, so every attempt goes via the proxy.)
        // Overall budget stays under the Node-side 180s HTTP timeout.
        parent, cancelParent := execCtx(45 * time.Second) // v3.1: one candidate's full client loop
        defer cancelParent()
        clients := []string{"tv_simply", "mweb", "web_embedded", "tv"}
        var lastErr error
        for round := 0; round < 2; round++ {
                for _, cl := range clients {
                        if round > 0 && cl == "web_embedded" {
                                continue
                        }
                        if parent.Err() != nil {
                                return lastErr
                        }
                        if round > 0 {
                                time.Sleep(2 * time.Second)
                        }
                        ctx, cancel := context.WithTimeout(parent, 40*time.Second)
                        args := []string{"--proxy", proxy,
                                "https://www.youtube.com/watch?v=" + id,
                                "-x", "-f", "bestaudio/best", "--audio-format", "mp3", "--audio-quality", "0",
                                "--no-playlist", "--no-warnings",
                                "--sleep-requests", "1", "--retries", "2", "--extractor-retries", "2",
                                "--throttled-rate", "100K",
                                "--extractor-args", "youtube:player_client=" + cl,
                                "-o", mp3Path}
                        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
                        out, err := cmd.CombinedOutput()
                        cancel()
                        if err == nil {
                                return nil
                        }
                        os.WriteFile(fmt.Sprintf("/tmp/ytdl_dbg_%d.txt", time.Now().UnixMilli()), out, 0644)
                        lastErr = fmt.Errorf("%v: %s", err, tailBytes(out, 300))
                }
        }
        return lastErr
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// ytURLRe matches watch/shorts/embed/youtu.be links and captures the 11-char id.
var ytURLRe = regexp.MustCompile(`(?:youtube\.com/(?:watch\?(?:[^#]*&)?v=|shorts/|embed/)|youtu\.be/)([A-Za-z0-9_-]{11})`)

// ytProbeVideo - metadata-only extraction for one video id.
func ytProbeVideo(id string, proxy string) (map[string]interface{}, error) {
        ctx, cancel := execCtx(45 * time.Second)
        defer cancel()
        args := []string{"--proxy", proxy,
                "https://www.youtube.com/watch?v=" + id,
                "--print-json", "--no-warnings", "--skip-download", "--no-playlist"}
        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
        cmd.Env = childEnv()
        out, err := cmd.CombinedOutput()
        for _, line := range strings.Split(string(out), "\n") {
                line = strings.TrimSpace(line)
                if strings.HasPrefix(line, "{") {
                        var m map[string]interface{}
                        if json.Unmarshal([]byte(line), &m) == nil {
                                return m, nil
                        }
                }
        }
        if err != nil {
                return nil, fmt.Errorf("%v: %s", err, tailBytes(out, 200))
        }
        return nil, fmt.Errorf("no json metadata")
}

// scProbeCandidates - SoundCloud search, metadata only. SoundCloud is NOT
// bot-blocked on either box (no proxy needed), making it the independent
// fallback when YouTube/WARP is throttled. Note: some official uploads are
// HLS-gated and fail as "DRM protected" - hence try-multiple-candidates.
func scProbeCandidates(searchQuery string, n int) []map[string]interface{} {
        ctx, cancel := execCtx(45 * time.Second)
        defer cancel()
        args := []string{"scsearch" + strconv.Itoa(n) + ":" + searchQuery,
                "--flat-playlist", "--print-json", "--no-warnings", "--skip-download"}
        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
        cmd.Env = childEnv()
        out, err := cmd.CombinedOutput()
        var list []map[string]interface{}
        for _, line := range strings.Split(string(out), "\n") {
                line = strings.TrimSpace(line)
                if strings.HasPrefix(line, "{") {
                        var m map[string]interface{}
                        if json.Unmarshal([]byte(line), &m) == nil {
                                list = append(list, m)
                        }
                }
        }
        if len(list) == 0 {
                fmt.Printf("[Audio v3] sc probe failed: %v: %s\n", err, tailBytes(out, 200))
        }
        return list
}

// scDownload via the API URL (permalink URLs 404 when guessed).
func scDownload(id string, mp3Path string) error {
        ctx, cancel := execCtx(70 * time.Second)
        defer cancel()
        args := []string{"-f", "bestaudio/best", "-x", "--audio-format", "mp3", "--audio-quality", "0",
                "--no-playlist", "--no-warnings",
                "-o", mp3Path,
                "https://api.soundcloud.com/tracks/" + id}
        cmd := exec.CommandContext(ctx, "yt-dlp", args...)
        cmd.Env = childEnv()
        if out, err := cmd.CombinedOutput(); err != nil {
                return fmt.Errorf("%v: %s", err, tailBytes(out, 250))
        }
        return nil
}

func scScoreCandidate(m map[string]interface{}, dz *deezerTrack) float64 {
        title := strings.ToLower(strOr(m["title"], ""))
        uploader := strings.ToLower(strOr(m["uploader"], ""))
        dur := numOr(m["duration"])
        score := 0.0
        if badTitleRe.MatchString(title) {
                score -= 60
        }
        if dz != nil && dz.Artist.Name != "" && strings.Contains(uploader, strings.ToLower(dz.Artist.Name)) {
                score += 25 // official artist/label upload
        }
        if dz != nil && dz.Duration > 0 && dur > 0 {
                d := dur - float64(dz.Duration)
                if d < 0 {
                        d = -d
                }
                if d <= 8 {
                        score += 30
                } else if d <= 20 {
                        score += 10
                } else {
                        score -= 15
                }
        }
        return score
}

func extractYouTubeID(s string) string {
        if m := ytURLRe.FindStringSubmatch(strings.TrimSpace(s)); len(m) == 2 {
                return m[1]
        }
        return ""
}

// ytFormatsHealthy - YouTube strips media formats (leaving storyboards only)
// when it throttles an IP instead of returning an explicit error, so a
// doomed download only shows up as a 40s hang. Probe the top candidate's
// format list with two clients; any real mp4/webm/m4a row means the IP can
// still serve media. ~4-6s per client.
func ytFormatsHealthy(id string, proxy string) bool {
	for _, cl := range []string{"tv_simply", "mweb"} {
		ctx, cancel := execCtx(15 * time.Second)
		args := []string{"--proxy", proxy,
			"https://www.youtube.com/watch?v=" + id,
			"--list-formats", "--no-warnings",
			"--extractor-args", "youtube:player_client=" + cl}
		cmd := exec.CommandContext(ctx, "yt-dlp", args...)
		cmd.Env = childEnv()
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			continue
		}
		for _, ln := range strings.Split(string(out), "\n") {
			f := strings.Fields(ln)
			if len(f) < 2 {
				continue
			}
			if f[0] == "sb0" || f[0] == "sb1" || f[0] == "sb2" || f[0] == "sb3" {
				continue
			}
			if _, err := strconv.Atoi(f[0]); err == nil &&
				(f[1] == "mp4" || f[1] == "webm" || f[1] == "m4a") {
				return true
			}
		}
	}
	return false
}

// warpRotate - restart the wireproxy service that provides the WARP socks
// endpoint. Cloudflare issues a fresh edge IP on reconnect, which clears
// YouTube's per-IP format throttling (verified 2026-09-14: throttled IP
// served storyboards-only via tv_simply; after rotation format 18 returned).
// On Box2 the local pm2 owns warp-proxy; on Box1 the socks endpoint is an
// SSH tunnel into Box2, so rotation goes over the inter-box tunnel key.
func warpRotate() {
	type cmdSpec struct {
		name string
		argv []string
	}
	specs := []cmdSpec{
		{"local-pm2", []string{"pm2", "restart", "warp-proxy"}},
		{"ssh-box2", []string{"ssh", "-i", "/home/ubuntu/.ssh/id_ed25519_boxlink",
			"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes",
			"ubuntu@84.8.136.110", "pm2 restart warp-proxy"}},
	}
	for _, sp := range specs {
		ctx, cancel := execCtx(30 * time.Second)
		cmd := exec.CommandContext(ctx, sp.argv[0], sp.argv[1:]...)
		cmd.Env = childEnv()
		out, err := cmd.CombinedOutput()
		cancel()
		if err == nil {
			fmt.Printf("[Audio v3] WARP rotated via %s\n", sp.name)
			time.Sleep(6 * time.Second) // wireproxy handshake
			return
		}
		fmt.Printf("[Audio v3] rotate via %s failed: %v: %s\n", sp.name, err, tailBytes(out, 150))
	}
}

func ScrapeAudio(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "query required"})
                return
        }
        fmt.Printf("[Audio v3] Searching for: %s\n", query)
        // v3.1: direct URL mode - ".audio <YouTube URL>" downloads exactly the
        // linked video (no search, no Deezer/Saavn guessing).
        forceYtID := extractYouTubeID(query)
        if forceYtID != "" {
                query = "https://www.youtube.com/watch?v=" + forceYtID
                fmt.Printf("[Audio v3] URL mode, id %s\n", forceYtID)
        }
        reqStart := time.Now() // v3.1g: one budget for the whole chain (Node timeout is 180s)
        _ = os.MkdirAll("downloads", 0755)
        go pruneAudioCache()
        sum := fmt.Sprintf("%x", md5.Sum([]byte("v9|"+query)))
        hash := sum[:12]
        mp3Path := fmt.Sprintf("downloads/%s.mp3", hash)
        metaPath := fmt.Sprintf("downloads/%s.json", hash)
        var meta AudioMetadata
        source := "unknown"
        isPreview := false
        fallbackOnly := false // v3.1c: untrusted saavn fallback must not poison the cache
        if _, err := os.Stat(mp3Path); err == nil {
                if b, err := os.ReadFile(metaPath); err == nil {
                        if json.Unmarshal(b, &meta) == nil && meta.Title != "" {
                                source = "cache"
                        }
                }
        }
        if source != "cache" {
                var dz *deezerTrack
                if forceYtID == "" {
                        dz = deezerMatch(query)
                }
                if dz != nil {
                        fmt.Printf("[Audio v3] deezer anchor: %s - %s (%ds)\n", dz.Title, dz.Artist.Name, dz.Duration)
                } else if forceYtID == "" {
                        fmt.Printf("[Audio v3] no deezer anchor, matching on query alone\n")
                }
                // Attempt 1: JioSaavn (320kbps; Deezer-anchored scoring) - trusted
                // only when the credited artist matches the Deezer anchor. An
                // untrusted pick is kept as the last-resort fallback below.
                var saavnFallback *saavnSong
                if forceYtID == "" {
                        if song := saavnPickBest(query, dz); song != nil {
                                if saavnTrusted(song, dz) {
                                        if m, err := saavnDownloadAndConvert(song, mp3Path); err == nil {
                                                if durationSane(audioDuration(mp3Path), dzDur(dz)) {
                                                        meta = m
                                                        source = "JioSaavn"
                                                } else {
                                                        os.Remove(mp3Path)
                                                        fmt.Printf("[Audio v3] saavn pick rejected by duration guard\n")
                                                }
                                        } else {
                                                fmt.Printf("[Audio v3] saavn pipeline failed: %v\n", err)
                                        }
                                } else {
                                        saavnFallback = song
                                        fmt.Printf("[Audio v3] saavn artist mismatch (%s != %s) - deferring to YouTube\n",
                                                strings.TrimSpace(saavnArtistDisplay(song)), dz.Artist.Name)
                                }
                        }
                }
                // Attempt 2: YouTube via WARP proxy - v3.1: tv_simply/mweb clients
                // (web_embedded went video-only), explicit bestaudio selection, and
                // the TOP-3 scored candidates are attempted before giving up.
                // v3.1b: if every candidate fails (shared-IP throttle signature
                // - YouTube strips formats instead of erroring), rotate the
                // WARP exit IP and retry the whole branch once. Time-budgeted
                // so the Node-side 180s HTTP timeout is never hit.
                proxy := audioProxyURL()
                ytProbeHadCandidates := false
                ytThrottled := false
                ytStart := time.Now()
                if source == "unknown" && proxy != "" {
                        searchQ := query
                        artistQ := query
                        if dz != nil && dz.Title != "" {
                                searchQ = dz.Artist.Name + " " + dz.Title + " audio"
                                artistQ = dz.Artist.Name + " " + dz.Title
                        }
                        type scoredCand struct {
                                m  map[string]interface{}
                                sc float64
                        }
                        for attempt := 0; attempt < 2 && source == "unknown"; attempt++ {
                                if attempt > 0 {
                                        if time.Since(ytStart) > 95*time.Second {
                                                break
                                        }
                                        fmt.Printf("[Audio v3] rotating WARP exit and retrying yt branch\n")
                                        warpRotate()
                                }
                                fmt.Printf("[Audio v3] trying YouTube via proxy\n")
                                var ranked []scoredCand
                                if forceYtID != "" {
                                        if pr, perr := ytProbeVideo(forceYtID, proxy); perr == nil {
                                                ranked = append(ranked, scoredCand{m: pr})
                                        } else {
                                                fmt.Printf("[Audio v3] yt probe (url mode) failed: %v\n", perr)
                                                ranked = append(ranked, scoredCand{m: map[string]interface{}{"id": forceYtID}})
                                        }
                                } else {
                                        cands := ytProbeCandidates(searchQ, 5, proxy)
                                        if len(cands) > 0 {
                                                ytProbeHadCandidates = true
                                        }
                                        for _, m := range cands {
                                                ranked = append(ranked, scoredCand{m: m, sc: ytScoreCandidate(m, dz, artistQ)})
                                        }
                                        sort.Slice(ranked, func(a, b int) bool { return ranked[a].sc > ranked[b].sc })
                                        if len(ranked) > 3 {
                                                ranked = ranked[:3]
                                        }
                                }
                                // v3.1c: health gate - a throttled WARP IP serves
                                // storyboards-only format lists; rotating the WARP
                                // session lands a fresh edge IP. If even the fresh
                                // IP is degenerate, skip straight to the fallbacks
                                // instead of hanging on doomed downloads.
                                if len(ranked) > 0 {
                                        topID := strOr(ranked[0].m["id"], "")
                                        if topID != "" && !ytFormatsHealthy(topID, proxy) {
                                                fmt.Printf("[Audio v3] yt formats degenerate (throttled IP) - rotating\n")
                                                warpRotate()
                                                if !ytFormatsHealthy(topID, proxy) {
                                                        fmt.Printf("[Audio v3] yt still degenerate after rotation - skipping yt branch\n")
                                                        ytThrottled = true
                                                        break
                                                }
                                        }
                                }
                                for _, cand := range ranked {
                                        if source != "unknown" {
                                                break
                                        }
                                        if time.Since(reqStart) > 95*time.Second {
                                                break
                                        }
                                        id := strOr(cand.m["id"], "")
                                        if id == "" {
                                                continue
                                        }
                                        if err := ytDownload(id, mp3Path, proxy); err == nil {
                                                if durationSane(audioDuration(mp3Path), dzDur(dz)) {
                                                        meta = AudioMetadata{
                                                                Title:     strOr(cand.m["title"], dzTitle(dz, query)),
                                                                Author:    strOr(cand.m["uploader"], dzArtist(dz, "")),
                                                                Thumbnail: dzThumb(dz),
                                                                Duration:  fmt.Sprintf("%.0f", numOr(cand.m["duration"])),
                                                                URL:       "https://www.youtube.com/watch?v=" + id,
                                                        }
                                                        source = "YouTube"
                                                } else {
                                                        os.Remove(mp3Path)
                                                        fmt.Printf("[Audio v3] yt pick %s rejected by duration guard\n", id)
                                                }
                                        } else {
                                                fmt.Printf("[Audio v3] yt download failed for %s: %v\n", id, err)
                                        }
                                }
                        }
                }

                // Attempt 3: SoundCloud - post-download preview guard. Skipped
                // when the YouTube probe already resolved candidates (download
                // failures are transient; SoundCloud only adds preview junk).
                if source == "unknown" && forceYtID == "" && (!ytProbeHadCandidates || ytThrottled) {
                        fmt.Printf("[Audio v3] trying SoundCloud\n")
                        scQ := query
                        if dz != nil && dz.Title != "" {
                                scQ = dz.Artist.Name + " " + dz.Title
                        }
                        cands := scProbeCandidates(scQ, 6)
                        sort.Slice(cands, func(a, b int) bool {
                                return scScoreCandidate(cands[a], dz) > scScoreCandidate(cands[b], dz)
                        })
                        tried := 0
                        for _, cand := range cands {
                                if source == "SoundCloud" || tried >= 3 || time.Since(reqStart) > 160*time.Second {
                                        break
                                }
                                id := strOr(cand["id"], "")
                                if id == "" {
                                        continue
                                }
                                if badTitleRe.MatchString(strings.ToLower(strOr(cand["title"], ""))) {
                                        continue
                                }
                                tried++
                                if err := scDownload(id, mp3Path); err == nil {
                                        if durationSane(audioDuration(mp3Path), dzDur(dz)) {
                                                meta = AudioMetadata{
                                                        Title:     strOr(cand["title"], query),
                                                        Author:    strOr(cand["uploader"], "SoundCloud"),
                                                        Thumbnail: strOr(cand["thumbnail"], ""),
                                                        URL:       strOr(cand["webpage_url"], ""),
                                                        Duration:  fmt.Sprintf("%.0f", numOr(cand["duration"])),
                                                }
                                                source = "SoundCloud"
                                        } else {
                                                os.Remove(mp3Path)
                                                fmt.Printf("[Audio v3] sc pick %s rejected by duration guard\n", id)
                                        }
                                } else {
                                        fmt.Printf("[Audio v3] sc download failed for %s: %v\n", id, err)
                                }
                        }
                        if source != "SoundCloud" && tried > 0 {
                                isPreview = true // sc candidates were previews/gated
                        }
                }
                // Last resort: better an untrusted-but-plausible Saavn pick (cover
                // or regional variant, artist shown in the caption) than nothing.
                if source == "unknown" && forceYtID == "" && saavnFallback != nil && time.Since(reqStart) <= 165*time.Second {
                        fmt.Printf("[Audio v3] all primary sources failed - using saavn fallback (not cached)\n")
                        if m, err := saavnDownloadAndConvert(saavnFallback, mp3Path); err == nil {
                                meta = m
                                source = "JioSaavn"
                                isPreview = false
                                // v3.1c: never persist the untrusted fallback - the
                                // next identical query deserves a fresh shot at the
                                // primary sources instead of 48h of the same cover.
                                fallbackOnly = true
                        }
                }
                // Deezer metadata wins when the source is thin on details.
                if dz != nil {
                        if meta.Title == "" || meta.Title == query {
                                meta.Title = dz.Title
                        }
                        if meta.Author == "" {
                                meta.Author = dz.Artist.Name
                        }
                        if meta.Thumbnail == "" {
                                meta.Thumbnail = dzThumb(dz)
                        }
                        if meta.URL == "" {
                                meta.URL = dz.Link
                        }
                }
        }
        if _, err := os.Stat(mp3Path); err != nil {
                c.JSON(500, gin.H{"error": "all download attempts failed"})
                return
        }
        if meta.Title == "" {
                meta.Title = query
        }
        if source != "cache" && !fallbackOnly {
                if b, err := json.Marshal(meta); err == nil {
                        os.WriteFile(metaPath, b, 0644)
                }
        }
        fmt.Printf("[Audio v3] ready: %s | source=%s | %s - %s\n", mp3Path, source, meta.Title, meta.Author)
        scheme := "http"
        if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
                scheme = "https"
        }
        baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
        c.JSON(200, gin.H{
                "metadata":    meta,
                "audioURL":    fmt.Sprintf("%s/downloads/%s.mp3", baseURL, hash),
                "audioSource": source,
                "isPreview":   isPreview,
        })
}

func dzDur(dz *deezerTrack) int {
        if dz == nil {
                return 0
        }
        return dz.Duration
}
func dzTitle(dz *deezerTrack, def string) string {
        if dz == nil || dz.Title == "" {
                return def
        }
        return dz.Title
}
func dzArtist(dz *deezerTrack, def string) string {
        if dz == nil || dz.Artist.Name == "" {
                return def
        }
        return dz.Artist.Name
}
func dzThumb(dz *deezerTrack) string {
        if dz == nil {
                return ""
        }
        if dz.Album.CoverXL != "" {
                return dz.Album.CoverXL
        }
        return dz.Album.CoverBig
}

func saavnDownloadAndConvert(s *saavnSong, mp3Path string) (AudioMetadata, error) {
        mi := s.MoreInfo
        mediaURL, err := decryptSaavnURL(mi.EncryptedMediaURL)
        if err != nil {
                return AudioMetadata{}, err
        }
        if mi.Kbps320 == "true" {
                mediaURL = strings.Replace(mediaURL, "_96.mp4", "_320.mp4", 1)
        }
        rawPath := strings.TrimSuffix(mp3Path, ".mp3") + ".m4a"

        req, _ := http.NewRequest("GET", mediaURL, nil)
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
        dl, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
        if err != nil {
                return AudioMetadata{}, err
        }
        defer dl.Body.Close()
        if dl.StatusCode != 200 && dl.StatusCode != 206 {
                return AudioMetadata{}, fmt.Errorf("saavn media HTTP %d", dl.StatusCode)
        }
        raw, err := os.Create(rawPath)
        if err != nil {
                return AudioMetadata{}, err
        }
        n, err := io.Copy(raw, dl.Body)
        raw.Close()
        if err != nil || n < 10000 {
                os.Remove(rawPath)
                return AudioMetadata{}, fmt.Errorf("saavn download too small (%d bytes)", n)
        }

        if err := ffmpegToMP3(rawPath, mp3Path); err != nil {
                os.Remove(rawPath)
                return AudioMetadata{}, err
        }
        os.Remove(rawPath)

        thumb := strings.Replace(s.Image, "150x150", "500x500", 1)
        return AudioMetadata{
                Title:     html.UnescapeString(strings.TrimSpace(s.Title)),
                Author:    strings.TrimSpace(saavnArtistDisplay(s)),
                Thumbnail: thumb,
                Duration:  mi.Duration,
                URL:       s.PermaURL,
        }, nil
}

// pruneAudioCache - keep the downloads dir from growing forever: delete
// audio artifacts older than 48h. Best-effort; runs in background.
func pruneAudioCache() {
        entries, err := filepath.Glob("downloads/*")
        if err != nil {
                return
        }
        cutoff := time.Now().Add(-48 * time.Hour)
        for _, p := range entries {
                ext := strings.ToLower(filepath.Ext(p))
                if ext != ".mp3" && ext != ".m4a" && ext != ".json" {
                        continue
                }
                if st, err := os.Stat(p); err == nil && st.ModTime().Before(cutoff) {
                        os.Remove(p)
                }
        }
}
