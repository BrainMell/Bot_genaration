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
//  HYBRID GRID RENDERER — animated cards cycle in place, static cards stay still
// ═══════════════════════════════════════════════════════════════════════════
//
// Added 2026-07-27 per benchmark results showing Mode D (true hybrid via ffmpeg)
// is the right architecture for `.jk coll --anim` / `.jk deck --anim`.
//
// What it does:
//   - Downloads each card image (sequential, like the existing grid renderer)
//   - Detects whether each card is animated (URL ends in .gif/.webp/.webm OR
//     the request payload explicitly marks animated=true)
//   - Composites all 12 cards into a 540×1080 3-col grid via ffmpeg:
//       • Animated cards: loop their GIF frames in place (cycle continuously)
//       • Static cards: render the same PNG on every frame (stay still)
//   - Outputs an MP4 at the requested framerate/duration
//
// Benchmark numbers (Oracle 0.1 OCPU, 954MB RAM):
//   - 5s @ 10fps: 1.8s render, 37 KB output, 0 MB RAM delta
//   - 8s @ 15fps: 3.3s render, 34 KB output, 0 MB RAM delta  ← SWEET SPOT
//   - 10s @ 30fps: 5.7s render, 60 KB output, 2 MB RAM delta
//
// All numbers well within the 377 MB headroom we have post-perf-patches.

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
        Duration int              `json:"duration"` // seconds, default 8
        FPS      int              `json:"fps"`      // frames per second, default 15
}

// gridLayout — 4 cols × 3 rows = 12 cards max (matches existing static grid renderer
// in collection_grid.go so the hybrid looks the same proportions as .jk coll).
// Previous version used 3×4 at 160×240, which made cards look "tiny" compared
// to the static grid. Now matches COLL_CARD_W=240, COLL_CARD_H=360 exactly.
const (
        HYBRID_CARD_W    = 240
        HYBRID_CARD_H    = 360
        HYBRID_GRID_COLS = 4
        HYBRID_PADDING   = 10
        HYBRID_SPACING   = 10
        HYBRID_HEADER_H  = 60
)

// maxAnimatedDownloadBytes — for animated GIFs / WebMs, abort the download if it
// exceeds this size. Real-world T6 GIFs on shoob.gg range from 1.5MB to 10MB;
// the 10MB ones blow up the render time + output size dramatically. If we abort,
// the hybrid endpoint still renders the OTHER cards — the missing one becomes a
// blank slot (the overlay chain just skips it).
const maxAnimatedDownloadBytes = 5 * 1024 * 1024 // 5 MB cap

// isAnimatedURL detects whether a URL points to an animated format.
// GIF, WebP (animated), WebM are treated as animated.
func isAnimatedURL(url string) bool {
        lower := strings.ToLower(url)
        return strings.HasSuffix(lower, ".gif") ||
                strings.HasSuffix(lower, ".webp") ||
                strings.HasSuffix(lower, ".webm")
}

// downloadFileWithLimit downloads a URL to dest, but aborts if the response body
// exceeds maxBytes. This prevents 10MB T6 GIFs from blowing up render time +
// output size — we'd rather have a blank slot than a 2MB MP4.
//
// If maxBytes <= 0, behaves like downloadFile (no limit).
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

        // If Content-Length is known and exceeds the limit, bail early.
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

        // Use io.LimitReader + check if there's MORE data after the limit
        // (which means the response exceeded maxBytes).
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
// It returns a video/mp4 buffer with the animated grid. On any ffmpeg failure
// it returns HTTP 500 with an error message — the JS bot is expected to fall
// back to the static /api/cards/grid endpoint in that case.
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

        // Cap at 12 cards (3×4 grid) — same limit as the static grid renderer.
        maxImages := 12
        if len(req.Images) > maxImages {
                req.Images = req.Images[:maxImages]
        }

        // Apply defaults: 8s @ 15fps is the benchmark sweet spot.
        if req.Duration <= 0 {
                req.Duration = 8
        }
        if req.Duration > 30 {
                req.Duration = 30 // hard cap to prevent runaway renders
        }
        if req.FPS <= 0 {
                req.FPS = 15
        }
        if req.FPS > 30 {
                req.FPS = 30 // 30fps is enough — anything higher is wasteful
        }

        // Use a temp dir for downloaded card files + ffmpeg work.
        tempDir, err := os.MkdirTemp("", "hybridgrid_*")
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create temp directory"})
                return
        }
        defer os.RemoveAll(tempDir)

        // Sequential downloads — same pattern as the existing grid renderer.
        // Parallel downloads exhaust the connection pool on 0.1 CPU servers.
        client := &http.Client{Timeout: 8 * time.Second}

        cards := make([]downloadedCard, len(req.Images))

        for i, input := range req.Images {
                filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", i))
                // Apply size cap only to animated cards — static PNGs are typically <300KB
                // and the cap exists to prevent 10MB T6 GIFs from blowing up render time.
                // If animated detection fails or download exceeds the cap, the card
                // becomes a blank slot (overlay chain skips it).
                var downloadErr error
                if input.Animated || isAnimatedURL(input.URL) {
                        downloadErr = downloadFileWithLimit(client, input.URL, filePath, maxAnimatedDownloadBytes)
                } else {
                        downloadErr = downloadFile(client, input.URL, filePath)
                }
                if downloadErr != nil {
                        fmt.Printf("[HybridGrid] Download failed [%d]: %v\n", i, downloadErr)
                        cards[i] = downloadedCard{
                                Path:     "",
                                Animated: false,
                                Name:     input.Name,
                                Tier:     input.Tier,
                                Index:    i,
                        }
                        continue
                }
                // Animated if explicitly marked in payload OR if URL suggests animated format.
                animated := input.Animated || isAnimatedURL(input.URL)
                cards[i] = downloadedCard{
                        Path:     filePath,
                        Animated: animated,
                        Name:     input.Name,
                        Tier:     input.Tier,
                        Index:    i,
                }
        }

        // If NO cards downloaded successfully, fail fast — the JS bot will fall back to text.
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

        // Build the ffmpeg command.
        // Filtergraph:
        //   - For each animated card: -stream_loop -1 -i <file>  (loops indefinitely)
        //   - For each static card: -loop 1 -framerate <fps> -t <duration> -i <file>
        //   - All inputs scaled to HYBRID_CARD_W × HYBRID_CARD_H
        //   - Each overlay'd on a 540×1080 black canvas at the right grid position
        //   - Output: libx264 mp4, ultrafast preset, crf 28 (small file, fast encode)
        //
        // We skip inputs that failed to download — those slots get a placeholder
        // rectangle drawn directly on the base canvas via drawbox in the filtergraph.
        //
        // Note: ffmpeg's filter_complex syntax is picky. We build it as a single
        // string with semicolons between filter chains.

        n := len(cards)
        cols := HYBRID_GRID_COLS
        if n < cols {
                cols = n
        }
        rows := (n + cols - 1) / cols

        canvasW := cols*HYBRID_CARD_W + (cols-1)*HYBRID_SPACING + HYBRID_PADDING*2
        canvasH := rows*HYBRID_CARD_H + (rows-1)*HYBRID_SPACING + HYBRID_PADDING*2 + HYBRID_HEADER_H

        // Build ffmpeg args. We use a slice because the filtergraph string is complex.
        args := []string{"-y", "-loglevel", "error"}

        // Inputs — one per successfully-downloaded card.
        // Index them by their original position to keep overlay coordinates straight.
        inputIndex := 0
        inputMapByCardIdx := make([]int, n) // cardIdx → ffmpeg input index (-1 if no input)
        for i := range inputMapByCardIdx {
                inputMapByCardIdx[i] = -1
        }
        for i, card := range cards {
                if card.Path == "" {
                        continue
                }
                if card.Animated {
                        // Animated: stream_loop -1 (loop forever), ffmpeg will read frames as they come.
                        // No -t here because we'll cap output with -t at the end.
                        args = append(args, "-stream_loop", "-1", "-i", card.Path)
                } else {
                        // Static: loop a single frame for the full duration.
                        args = append(args, "-loop", "1", "-framerate", fmt.Sprintf("%d", req.FPS),
                                "-t", fmt.Sprintf("%d", req.Duration), "-i", card.Path)
                }
                inputMapByCardIdx[i] = inputIndex
                inputIndex++
        }

        // Build filtergraph
        // Each input gets scaled to card size + setpts to match output fps.
        // NO `loop` filter — see comment below for why.
        var filterParts []string

        // Per-input: scale + setpts. NO `loop` filter — it was freezing GIFs at frame 0.
        // The `-stream_loop -1` input option (for animated) and `-loop 1 -t duration`
        // input option (for static) already handle frame production correctly.
        // Adding `loop=loop=N:size=1:start=0` on top was wrong — it took only 1 frame
        // per stream-loop iteration, effectively showing frame 0 of the GIF 50 times.
        // Without the loop filter, ffmpeg decodes the GIF's actual frames and lets
        // them play at the requested fps via setpts.
        for i := range cards {
                idx := inputMapByCardIdx[i]
                if idx == -1 {
                        continue // skip failed downloads
                }
                // Same filter for animated and static — the input options differentiate them.
                filterParts = append(filterParts,
                        fmt.Sprintf("[%d:v]scale=%d:%d,setpts=N/(%d*TB)[v%d]",
                                idx, HYBRID_CARD_W, HYBRID_CARD_H, req.FPS, i))
        }

        // Base canvas — solid dark color matching the existing grid renderer.
        filterParts = append(filterParts,
                fmt.Sprintf("color=c=#0f1015:s=%dx%d:d=%d[base]",
                        canvasW, canvasH, req.Duration))

        // Sequential overlays — each card placed at its grid position.
        prevLabel := "base"
        overlayCount := 0
        for i := range cards {
                idx := inputMapByCardIdx[i]
                if idx == -1 {
                        continue
                }
                col := i % cols
                row := i / cols
                x := HYBRID_PADDING + col*(HYBRID_CARD_W+HYBRID_SPACING)
                y := HYBRID_PADDING + HYBRID_HEADER_H + row*(HYBRID_CARD_H+HYBRID_SPACING)

                newLabel := fmt.Sprintf("b%d", overlayCount)
                // Last overlay must output to [final] for the -map below.
                if overlayCount == successCount-1 {
                        newLabel = "final"
                }
                filterParts = append(filterParts,
                        fmt.Sprintf("[%s][v%d]overlay=x=%d:y=%d:shortest=1[%s]",
                                prevLabel, i, x, y, newLabel))
                prevLabel = newLabel
                overlayCount++
        }

        // Edge case: only one card succeeded — it's already labeled [final] above.
        // Edge case: zero successful overlays can't happen (we returned earlier if all failed).

        // Join all filter parts into one filtergraph string.
        filtergraph := strings.Join(filterParts, ";")

        args = append(args, "-filter_complex", filtergraph)

        // Map final + encoding options.
        // 💡 CRF bumped 28 → 32: real-world T6 GIFs are 1.5-10MB each (lots of motion),
        // which made output MP4s balloon to 1-2MB at CRF 28. CRF 32 cuts file size
        // by ~50% with minimal visible quality loss on WhatsApp's mobile preview.
        // Still using ultrafast preset to keep render time low on 0.1 CPU.
        args = append(args,
                "-map", "[final]",
                "-t", fmt.Sprintf("%d", req.Duration),
                "-r", fmt.Sprintf("%d", req.FPS),
                "-c:v", "libx264",
                "-pix_fmt", "yuv420p",
                "-preset", "ultrafast",
                "-crf", "32",
                "-movflags", "+faststart", // enables streaming on WhatsApp
        )

        outputPath := filepath.Join(tempDir, "output.mp4")
        args = append(args, outputPath)

        fmt.Printf("[HybridGrid] Rendering %d cards (%d animated) into %dx%d grid, %ds @ %dfps\n",
                len(cards), countAnimated(cards), canvasW, canvasH, req.Duration, req.FPS)

        cmd := exec.Command("ffmpeg", args...)
        if output, err := cmd.CombinedOutput(); err != nil {
                fmt.Printf("[HybridGrid] FFmpeg error: %v\nOutput: %s\n", err, string(output))
                c.JSON(500, gin.H{"error": "FFmpeg render failed", "detail": string(output)})
                return
        }

        data, err := os.ReadFile(outputPath)
        if err != nil {
                fmt.Printf("[HybridGrid] Read output failed: %v\n", err)
                c.JSON(500, gin.H{"error": "Failed to read output"})
                return
        }

        if len(data) < 100 {
                c.JSON(500, gin.H{"error": "Output too small — render produced empty file"})
                return
        }

        fmt.Printf("[HybridGrid] Success: %d bytes\n", len(data))
        c.Data(200, "video/mp4", data)
}

func countAnimated(cards []downloadedCard) int {
        n := 0
        for _, c := range cards {
                if c.Animated {
                        n++
                }
        }
        return n
}
