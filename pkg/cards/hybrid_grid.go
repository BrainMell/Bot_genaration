package cards

import (
        "fmt"
        "io"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
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
// This replaces the previous approach of compositing raw card images onto
// a plain black canvas, which was missing all the styling.

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
// exceeds this size. Real-world T6 GIFs on shoob.gg range from 1.5MB to 10MB;
// the 10MB ones blow up the render time + output size dramatically.
const maxAnimatedDownloadBytes = 5 * 1024 * 1024 // 5 MB cap

// isAnimatedURL detects whether a URL points to an animated format.
func isAnimatedURL(url string) bool {
        lower := strings.ToLower(url)
        return strings.HasSuffix(lower, ".gif") ||
                strings.HasSuffix(lower, ".webp") ||
                strings.HasSuffix(lower, ".webm")
}

// downloadFileWithLimit downloads a URL to dest, but aborts if the response body
// exceeds maxBytes. Prevents 10MB T6 GIFs from blowing up render time.
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

        // ── STEP 1: Download all cards sequentially ──────────────────────────────
        client := &http.Client{Timeout: 8 * time.Second}
        cards := make([]downloadedCard, len(req.Images))

        for i, input := range req.Images {
                filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", i))
                var downloadErr error
                if input.Animated || isAnimatedURL(input.URL) {
                        downloadErr = downloadFileWithLimit(client, input.URL, filePath, maxAnimatedDownloadBytes)
                } else {
                        downloadErr = downloadFile(client, input.URL, filePath)
                }
                if downloadErr != nil {
                        fmt.Printf("[HybridGrid] Download failed [%d]: %v\n", i, downloadErr)
                        cards[i] = downloadedCard{Path: "", Animated: false, Name: input.Name, Tier: input.Tier, Index: i}
                        continue
                }
                cards[i] = downloadedCard{
                        Path:     filePath,
                        Animated: input.Animated || isAnimatedURL(input.URL),
                        Name:     input.Name,
                        Tier:     input.Tier,
                        Index:    i,
                }
        }

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
        collInputs := make([]CollCardInput, len(cards))
        for i, card := range cards {
                collInputs[i] = CollCardInput{
                        URL:  "", // already downloaded locally — pass empty so renderer skips download
                        Name: card.Name,
                        Tier: card.Tier,
                }
                // If we have a local file, we need to pass it differently.
                // CollCardInput has URL not Path, so for downloaded cards we pass
                // the original URL and let the renderer re-download.
                // This is wasteful but keeps the code simple. The renderer's parallel
                // downloads are fast (~2s for 12 cards).
                // TODO: refactor renderCollectionGridPNG to accept pre-downloaded files.
        }

        // We need to pass the ORIGINAL URLs to the renderer so it can download them.
        // The local files we downloaded are just for ffmpeg overlay.
        collInputs = make([]CollCardInput, len(req.Images))
        for i, input := range req.Images {
                collInputs[i] = CollCardInput{
                        URL:  input.URL,
                        Name: input.Name,
                        Tier: input.Tier,
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
                "-crf", "23",            // 23 = high quality (was 32 which was too lossy)
                "-movflags", "+faststart",
        )

        outputPath := filepath.Join(tempDir, "output.mp4")
        args = append(args, outputPath)

        fmt.Printf("[HybridGrid] Rendering %d cards (%d animated) — static grid + ffmpeg overlay, %ds @ %dfps\n",
                len(cards), animCount, req.Duration, req.FPS)

        cmd := exec.Command("ffmpeg", args...)
        if output, err := cmd.CombinedOutput(); err != nil {
                fmt.Printf("[HybridGrid] FFmpeg error: %v\nOutput: %s\n", err, string(output))
                c.JSON(500, gin.H{"error": "FFmpeg render failed", "detail": string(output)})
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
