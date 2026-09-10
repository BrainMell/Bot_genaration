package cards

import (
        "fmt"
        "io"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "sync"
        "time"

        "github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════════════
//  HYBRID GRID RENDERER — static grid styling + animated cards on top
// ═══════════════════════════════════════════════════════════════════════════
//
// Added 2026-07-27. Rewritten 2026-07-27 to match static grid styling exactly.
//
// How it works:
//   1. Download all 12 cards (with size cap for animated GIFs/WebMs)
//   2. Call the EXISTING static grid renderer (GenerateCardGridImage) to
//      produce a fully-styled PNG: header, footer, tier borders, card names,
//      tier badges, gradient header bar. This PNG is the BACKGROUND.
//      Animated GIFs show their first frame in this static render.
//   3. Use ffmpeg to overlay each ANIMATED card's GIF on top of its slot
//      in the background PNG. Static cards stay as part of the background.
//   4. Output as MP4.
//
// Result: the hybrid grid looks EXACTLY like the static grid, but animated
// cards (T6/S/E) cycle through their GIF frames in place.
//
// 💡 FIX 2026-09-10 — "animated cards render as stills in .jk coll / .jk deck":
//   - The old 5MB download cap rejected MOST real shoob GIFs (probe of 14
//     T6/S GIFs: 700KB–39.6MB, median ~15MB). Rejected cards silently fell
//     back to the static background frame → still images. Cap raised to 45MB.
//   - The shared 8s HTTP client timed out on big GIFs → animated path now
//     uses a dedicated 30s client.
//   - The static grid renderer re-downloaded every card (double fetch).
//     STEP 1 files are now reused via CollCardInput.LocalPath.
//   - One bad overlay input aborted the whole ffmpeg render (500 → bot fell
//     back to a fully static grid). Files are now magic-byte validated
//     (GIF8/RIFF/EBML only), and if ffmpeg still fails we return the styled
//     background PNG instead of a 500.

// HybridCardInput — same as CollCardInput but the Animated field is honored.
type HybridCardInput struct {
        URL      string `json:"url" binding:"required"`
        Name     string `json:"name"`
        Tier     string `json:"tier"`
        Animated bool   `json:"animated"`
}

// downloadedCard — internal struct tracking each card's local file + animation flag.
type downloadedCard struct {
        Path     string
        Animated bool
        Name     string
        Tier     string
        Index    int
}

// HybridGridRequest is the payload for the hybrid grid endpoint.
type HybridGridRequest struct {
        Images   []HybridCardInput `json:"images" binding:"required"`
        Title    string           `json:"title"`
        Duration int              `json:"duration"` // seconds, default 5
        FPS      int              `json:"fps"`      // frames per second, default 10
}

// Layout constants — MUST match collection_grid.go exactly so the hybrid grid
// looks identical to the static grid. These are referenced for overlay positioning.
const (
        HYBRID_CARD_W    = 240 // = COLL_CARD_W
        HYBRID_CARD_H    = 360 // = COLL_CARD_H
        HYBRID_GRID_COLS = 4   // = COLL_GRID_COLS
        HYBRID_PADDING   = 10  // = COLL_PADDING
        HYBRID_HEADER_H  = 60  // = COLL_HEADER_H
        HYBRID_LABEL_H   = 25  // = COLL_LABEL_H
)

// maxAnimatedDownloadBytes — for animated GIFs / WebMs, abort the download if it
// exceeds this size.
//
// 💡 RAISED 2026-09-10 (was 5MB): the 5MB cap was set when T6 GIFs were
// believed to be "1.5MB to 10MB", but a real sample of 14 shoob.gg T6/S GIFs
// probed at 712KB / 2.2MB / 2.5MB / 5.7MB / 9.4MB / 11.4MB / 14.7MB / 18.9MB /
// 20.3MB / 25MB / 31.9MB / 39.6MB — i.e. MOST animated cards blew the old cap.
// Every rejected card rendered as a still image in .jk coll / .jk deck.
// 45MB covers the full observed range with headroom.
//
// Render cost stays bounded regardless of GIF size: the overlay filtergraph
// only decodes frames the output timeline consumes (background ends at
// `duration` seconds, overlay uses shortest=1), so a 40MB GIF costs the same
// decode as the first few seconds of its animation.
const maxAnimatedDownloadBytes = 45 * 1024 * 1024

// isAnimatedURL detects whether a URL points to an animated format.
func isAnimatedURL(url string) bool {
        lower := strings.ToLower(url)
        return strings.HasSuffix(lower, ".gif") ||
                strings.HasSuffix(lower, ".webp") ||
                strings.HasSuffix(lower, ".webm")
}

// hasAnimatedMagic sniffs a downloaded file's header and reports whether it is
// actually an animated container:
//   - GIF:  "GIF8"
//   - WebP: "RIFF"...."WEBP"
//   - WebM: EBML header 0x1A45DFA3
//
// Files that are really JPEG/PNG (tier-heuristic "animated" cards with static
// bodies) return false and are rendered as static — keeping them out of the
// ffmpeg overlay chain, where one undecodable input aborts the whole render.
func hasAnimatedMagic(path string) bool {
        f, err := os.Open(path)
        if err != nil {
                return false
        }
        defer f.Close()

        head := make([]byte, 12)
        n, err := f.Read(head)
        if err != nil && n < 4 {
                return false
        }
        if n >= 4 && string(head[:4]) == "GIF8" {
                return true
        }
        if n >= 4 && string(head[:4]) == "RIFF" {
                return true
        }
        if n >= 4 && head[0] == 0x1A && head[1] == 0x45 && head[2] == 0xDF && head[3] == 0xA3 {
                return true
        }
        return false
}

// downloadFileWithLimit downloads a URL to dest, but aborts if the response body
// exceeds maxBytes. Prevents pathological downloads from filling the disk.
func downloadFileWithLimit(client *http.Client, url string, dest string, maxBytes int64) error {
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
        req.Header.Set("Referer", "https://www.pinterest.com/")
        req.Header.Set("Accept", "image/webp,image/png,image/jpeg,image/gif")
        req.Header.Set("Accept-Language", "en-US,en;q=0.9")
        resp, err := client.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("bad status: %s", resp.Status)
        }

        if maxBytes > 0 && resp.ContentLength > maxBytes {
                return fmt.Errorf("content too large: %d bytes (max %d)", resp.ContentLength, maxBytes)
        }

        out, err := os.Create(dest)
        if err != nil {
                return err
        }
        defer out.Close()

        if maxBytes <= 0 {
                _, err = io.Copy(out, resp.Body)
                return err
        }

        limited := io.LimitReader(resp.Body, maxBytes+1)
        n, err := io.Copy(out, limited)
        if err != nil {
                return err
        }
        if n > maxBytes {
                return fmt.Errorf("content exceeded maxBytes=%d (got at least %d)", maxBytes, n)
        }
        return nil
}

// GenerateHybridGrid is the gin handler for POST /api/cards/hybrid-grid.
//
// It returns a video/mp4 buffer. The video looks EXACTLY like the static grid
// (same styling, same dimensions, same card names + tier badges) but animated
// cards cycle through their GIF frames in place.
func GenerateHybridGrid(c *gin.Context) {
        var req HybridGridRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }
        if len(req.Images) == 0 {
                c.JSON(400, gin.H{"error": "No images provided"})
                return
        }

        maxImages := 12
        if len(req.Images) > maxImages {
                req.Images = req.Images[:maxImages]
        }

        if req.Duration <= 0 {
                req.Duration = 5
        }
        if req.Duration > 30 {
                req.Duration = 30
        }
        if req.FPS <= 0 {
                req.FPS = 10
        }
        if req.FPS > 30 {
                req.FPS = 30
        }

        tempDir, err := os.MkdirTemp("", "hybridgrid_*")
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create temp directory"})
                return
        }
        defer os.RemoveAll(tempDir)

        // ── STEP 1: Download all cards (4-way parallel) ──────────────────────────
        // 💡 PERF FIX 2026-09-10: downloads used to be strictly sequential — a
        // deck holding ~200MB of GIFs spent 60s+ downloading alone and blew
        // the bot's HTTP timeout (→ static fallback again). 4 concurrent
        // workers cut wall time ~4x. Cards land in distinct slots, so the
        // parallel writes are index-safe.
        client := &http.Client{Timeout: 8 * time.Second}
        // 💡 FIX 2026-09-10: animated GIFs run up to ~40MB — the shared 8s client
        // timed out on them. Dedicated 30s client for the animated path.
        animClient := &http.Client{Timeout: 30 * time.Second}
        cards := make([]downloadedCard, len(req.Images))

        sem := make(chan struct{}, 4)
        var dlWg sync.WaitGroup
        for i, input := range req.Images {
                dlWg.Add(1)
                go func(i int, input HybridCardInput) {
                        defer dlWg.Done()
                        sem <- struct{}{}
                        defer func() { <-sem }()

                        filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", i))
                        var downloadErr error
                        if input.Animated || isAnimatedURL(input.URL) {
                                downloadErr = downloadFileWithLimit(animClient, input.URL, filePath, maxAnimatedDownloadBytes)
                        } else {
                                downloadErr = downloadFile(client, input.URL, filePath)
                        }
                        if downloadErr != nil {
                                fmt.Printf("[HybridGrid] Download failed [%d]: %v\n", i, downloadErr)
                                cards[i] = downloadedCard{Path: "", Animated: false, Name: input.Name, Tier: input.Tier, Index: i}
                                return
                        }
                        // 💡 FIX 2026-09-10: only keep truly animated files in the overlay
                        // chain. An "animated-flagged" card whose body is a plain JPEG/PNG
                        // used to go straight into ffmpeg; one undecodable input aborted the
                        // whole render (500) and the bot fell back to a fully static grid.
                        isAnim := input.Animated || isAnimatedURL(input.URL)
                        if isAnim && !hasAnimatedMagic(filePath) {
                                isAnim = false
                        }
                        cards[i] = downloadedCard{
                                Path:     filePath,
                                Animated: isAnim,
                                Name:     input.Name,
                                Tier:     input.Tier,
                                Index:    i,
                        }
                }(i, input)
        }
        dlWg.Wait()

        successCount := 0
        for _, c := range cards {
                if c.Path != "" {
                        successCount++
                }
        }
        if successCount == 0 {
                c.JSON(500, gin.H{"error": "All card downloads failed"})
                return
        }

        // ── STEP 2: Generate styled static grid PNG (with ALL cards) ────────────
        // This gives us: header, footer, tier borders, card names, tier badges,
        // gradient header bar — everything the static grid has.
        // Animated GIFs will show their first frame in this static render.
        //
        // 💡 FIX 2026-09-10: pass the STEP 1 local files through CollCardInput.
        // LocalPath so the renderer does NOT re-download every card (previously
        // each card was fetched twice — brutal for decks holding 10-40MB GIFs).
        collInputs := make([]CollCardInput, len(req.Images))
        for i, input := range req.Images {
                collInputs[i] = CollCardInput{
                        URL:       input.URL,
                        LocalPath: cards[i].Path,
                        Name:      input.Name,
                        Tier:      input.Tier,
                }
        }

        bgPNG, bgW, bgH, err := renderCollectionGridPNG(collInputs, req.Title)
        if err != nil {
                fmt.Printf("[HybridGrid] Static grid generation failed: %v\n", err)
                c.JSON(500, gin.H{"error": "Static grid generation failed"})
                return
        }

        // Save the static PNG to disk — ffmpeg will read it as the background input.
        bgPath := filepath.Join(tempDir, "background.png")
        if err := os.WriteFile(bgPath, bgPNG, 0644); err != nil {
                c.JSON(500, gin.H{"error": "Failed to write background PNG"})
                return
        }
        fmt.Printf("[HybridGrid] Static grid: %dx%d (%d bytes)\n", bgW, bgH, len(bgPNG))

        // ── STEP 3: Check if any cards are animated ─────────────────────────────
        // If none are animated, just return the static PNG — no need for ffmpeg.
        // The bot will send it as an image (we return image/png content-type).
        animCount := 0
        for _, card := range cards {
                if card.Path != "" && card.Animated {
                        animCount++
                }
        }

        if animCount == 0 {
                fmt.Printf("[HybridGrid] No animated cards — returning static PNG (%d bytes, %dx%d)\n", len(bgPNG), bgW, bgH)
                c.Data(200, "image/png", bgPNG)
                return
        }

        // ── STEP 4: Build ffmpeg command ────────────────────────────────────────
        // Input 0: the styled static PNG (looped for full duration)
        // Inputs 1..N: animated GIFs (stream_loop -1, loop forever)
        // Filtergraph: scale each animated input to card size, overlay at grid position
        args := []string{"-y", "-loglevel", "error"}

        // Background input — loop the static PNG for the full duration at the target fps
        args = append(args, "-loop", "1", "-framerate", fmt.Sprintf("%d", req.FPS),
                "-t", fmt.Sprintf("%d", req.Duration), "-i", bgPath)

        // Animated card inputs
        type animInput struct {
                CardIdx  int // index in cards[]
                InputIdx int // index in ffmpeg args (0 = background, 1+ = animated)
        }
        var animInputs []animInput
        inputIdx := 1
        for i, card := range cards {
                if card.Path == "" || !card.Animated {
                        continue
                }
                args = append(args, "-stream_loop", "-1", "-i", card.Path)
                animInputs = append(animInputs, animInput{CardIdx: i, InputIdx: inputIdx})
                inputIdx++
        }

        // Build filtergraph
        var filterParts []string

        // Scale each animated input to card size + setpts to match output fps
        for _, ai := range animInputs {
                filterParts = append(filterParts,
                        fmt.Sprintf("[%d:v]scale=%d:%d,setpts=N/(%d*TB)[v%d]",
                                ai.InputIdx, HYBRID_CARD_W, HYBRID_CARD_H, req.FPS, ai.CardIdx))
        }

        // Overlay chain: start with background (0:v), overlay each animated card
        // at its grid position. Position formula matches collection_grid.go:
        //   x = PADDING + col*(CARD_W+PADDING)
        //   y = HEADER_H + PADDING + row*(CARD_H+LABEL_H+PADDING)
        prevLabel := "0:v"
        for i, ai := range animInputs {
                col := ai.CardIdx % HYBRID_GRID_COLS
                row := ai.CardIdx / HYBRID_GRID_COLS
                x := HYBRID_PADDING + col*(HYBRID_CARD_W+HYBRID_PADDING)
                y := HYBRID_HEADER_H + HYBRID_PADDING + row*(HYBRID_CARD_H+HYBRID_LABEL_H+HYBRID_PADDING)

                newLabel := fmt.Sprintf("b%d", i)
                if i == len(animInputs)-1 {
                        newLabel = "final"
                }
                filterParts = append(filterParts,
                        fmt.Sprintf("[%s][v%d]overlay=x=%d:y=%d:shortest=1[%s]",
                                prevLabel, ai.CardIdx, x, y, newLabel))
                prevLabel = newLabel
        }

        filtergraph := strings.Join(filterParts, ";")
        args = append(args, "-filter_complex", filtergraph)

        // Encoding options
        args = append(args,
                "-map", "[final]",
                "-t", fmt.Sprintf("%d", req.Duration),
                "-r", fmt.Sprintf("%d", req.FPS),
                "-c:v", "libx264",
                "-pix_fmt", "yuv420p",
                "-preset", "ultrafast",
                "-crf", "23", // 23 = high quality (was 32 which was too lossy)
                "-movflags", "+faststart",
        )

        outputPath := filepath.Join(tempDir, "output.mp4")
        args = append(args, outputPath)

        fmt.Printf("[HybridGrid] Rendering %d cards (%d animated) — static grid + ffmpeg overlay, %ds @ %dfps\n",
                len(cards), animCount, req.Duration, req.FPS)

        cmd := exec.Command("ffmpeg", args...)
        if output, err := cmd.CombinedOutput(); err != nil {
                fmt.Printf("[HybridGrid] FFmpeg error: %v\nOutput: %s\n", err, string(output))
                // 💡 FIX 2026-09-10: don't 500 — the styled background PNG is already
                // rendered and paid for. Returning it lets the bot send a proper
                // static grid instantly instead of triggering a second full render.
                c.Data(200, "image/png", bgPNG)
                return
        }

        data, err := os.ReadFile(outputPath)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to read output"})
                return
        }

        if len(data) < 100 {
                c.JSON(500, gin.H{"error": "Output too small — render produced empty file"})
                return
        }

        fmt.Printf("[HybridGrid] Success: %d bytes (%d animated cards)\n", len(data), animCount)
        c.Data(200, "video/mp4", data)
}
