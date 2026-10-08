package cards

import (
        "fmt"
        "image"
        "image/color"
        "image/draw"
        "image/gif"
        "image/png"
        "io"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "sync"
        "time"

        "github.com/disintegration/imaging"
        "github.com/gin-gonic/gin"

        "image-service/pkg/utils"
)

/*
 * Card Burn Animation — "fire placed on the card"
 *
 * 💡 FIX 2026-10-08 (owner: "the card burn animation lol it never worked"):
 * The old implementation downloaded a fire overlay from a GitHub URL that was
 * 404 (the file never existed in the repo) with a dead giphy fallback, so EVERY
 * request 500'd, the Node caller got null and users only ever saw the text
 * fallback. Even if the overlay HAD downloaded, the ffmpeg filter used
 * `blend=all_mode=screen` on a PAL8 GIF stream, which corrupts colors on some
 * ffmpeg builds (whole frame washed magenta).
 *
 * New pipeline (deterministic, zero network dependency):
 *   1. Download card image, decode (png/jpeg/gif-first-frame/webp), cover-fill
 *      an 800x800 canvas on black.
 *   2. Load the BUNDLED fire overlay GIF (assets/vfx/fire_overlay.gif) — a
 *      procedurally generated seamless loop of flames on pure black.
 *   3. Per fire frame, true ADDITIVE screen blend in Go:
 *         out = 255 - (255-base)*(255-fire)/255     (per channel)
 *      Black fire pixels leave the card untouched; bright flames glow on top.
 *   4. Write the blended frames as PNGs and let ffmpeg assemble the MP4 with
 *      NO filters at all (the one ffmpeg step that never breaks).
 *   5. MP4 (H.264 yuv420p + faststart) -> WhatsApp plays it as a looping GIF.
 *
 * Results are cached per card URL (burn preview + accept share one render).
 */

type BurnRequest struct {
        ImageUrl string `json:"imageUrl"`
}

// Remote fallback for the fire asset, only consulted when the bundled file is
// missing (e.g. fresh clone where assets weren't synced). The same file lives
// in the whatsapp-bot repo at this exact path.
const FireGifUrl = "https://raw.githubusercontent.com/BrainMell/whatsapp-bot/main/core/rpgasset/ui/fire_overlay.gif"

const (
        // Box2's CPU runs at a burstable baseline (~1/6 nominal), so x264 cost is
        // dominated by pixel count x frames. 640x640 @ 12fps keeps a fresh render
        // in single-digit seconds while still filling WhatsApp's media frame.
        burnCanvas   = 640 // output canvas (square)
        burnFPS      = 12
        burnFmt      = "video/mp4"
        burnCacheMax = 30 // max cached burn renders
        burnCacheTTL = 30 * time.Minute
)

type burnCacheEntry struct {
        data    []byte
        created time.Time
}

var (
        burnCache   = make(map[string]burnCacheEntry)
        burnCacheMu sync.Mutex
)

// loadFireFrames decodes the bundled fire overlay GIF into RGBA frames.
// Decoded per request (~100ms, burns are rare) to keep idle memory at zero.
func loadFireFrames() ([]*image.RGBA, error) {
        path := utils.GetAssetPath("vfx", "fire_overlay.gif")
        f, err := os.Open(path)
        if err != nil {
                // Bundled asset missing -> try the remote copy once and cache it.
                if dlErr := downloadFireAsset(path); dlErr != nil {
                        return nil, fmt.Errorf("fire overlay not found locally (%s) and remote fetch failed: %v", path, dlErr)
                }
                f, err = os.Open(path)
                if err != nil {
                        return nil, err
                }
        }
        defer f.Close()

        g, err := gif.DecodeAll(f)
        if err != nil {
                return nil, fmt.Errorf("fire overlay decode failed: %v", err)
        }
        frames := make([]*image.RGBA, 0, len(g.Image))
        for _, fr := range g.Image {
                // Scale to canvas (512 -> 800, pure upscale; Fill centers identically)
                scaled := imaging.Fill(fr, burnCanvas, burnCanvas, imaging.Center, imaging.Linear)
                rgba := image.NewRGBA(image.Rect(0, 0, burnCanvas, burnCanvas))
                draw.Draw(rgba, rgba.Bounds(), scaled, image.Point{}, draw.Src)
                frames = append(frames, rgba)
        }
        if len(frames) == 0 {
                return nil, fmt.Errorf("fire overlay has no frames")
        }
        return frames, nil
}

// downloadFireAsset fetches the remote fire GIF and stores it at path.
func downloadFireAsset(path string) error {
        client := &http.Client{Timeout: 20 * time.Second}
        resp, err := client.Get(FireGifUrl)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("fire asset download: bad status %s", resp.Status)
        }
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
                return err
        }
        tmp := path + ".tmp"
        out, err := os.Create(tmp)
        if err != nil {
                return err
        }
        if _, err := io.Copy(out, resp.Body); err != nil {
                out.Close()
                os.Remove(tmp)
                return err
        }
        out.Close()
        return os.Rename(tmp, path)
}

// flattenCard renders the card image cover-cropped onto a black canvas
// (transparency flattened — the burn plays on a black stage).
func flattenCard(img image.Image) *image.RGBA {
        filled := imaging.Fill(img, burnCanvas, burnCanvas, imaging.Center, imaging.Lanczos)
        canvas := image.NewRGBA(image.Rect(0, 0, burnCanvas, burnCanvas))
        draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
        draw.Draw(canvas, canvas.Bounds(), filled, image.Point{}, draw.Over)
        return canvas
}

// screenBlend: out = 255 - (255-base)*(255-fire)/255, per channel.
// Black fire pixels are a no-op -> flames appear additively ON the card.
func screenBlend(base *image.RGBA, fire *image.RGBA) *image.RGBA {
        out := image.NewRGBA(base.Bounds())
        b := base.Bounds()
        w, h := b.Dx(), b.Dy()
        for y := 0; y < h; y++ {
                rowO := out.Pix[y*out.Stride : y*out.Stride+w*4]
                rowB := base.Pix[y*base.Stride : y*base.Stride+w*4]
                rowF := fire.Pix[y*fire.Stride : y*fire.Stride+w*4]
                for x := 0; x < w*4; x += 4 {
                        rowO[x+0] = screenByte(rowB[x+0], rowF[x+0])
                        rowO[x+1] = screenByte(rowB[x+1], rowF[x+1])
                        rowO[x+2] = screenByte(rowB[x+2], rowF[x+2])
                        rowO[x+3] = 255
                }
        }
        return out
}

func screenByte(a, b uint8) uint8 {
        av, bv := int(a), int(b)
        return uint8(255 - (255-av)*(255-bv)/255)
}

func getCachedBurn(url string) ([]byte, bool) {
        burnCacheMu.Lock()
        defer burnCacheMu.Unlock()
        e, ok := burnCache[url]
        if !ok || time.Since(e.created) > burnCacheTTL {
                return nil, false
        }
        return e.data, true
}

func setCachedBurn(url string, data []byte) {
        burnCacheMu.Lock()
        defer burnCacheMu.Unlock()
        // simple size-cap eviction (oldest first)
        for len(burnCache) >= burnCacheMax {
                var oldestKey string
                var oldest time.Time
                first := true
                for k, v := range burnCache {
                        if first || v.created.Before(oldest) {
                                oldestKey, oldest, first = k, v.created, false
                        }
                }
                delete(burnCache, oldestKey)
        }
        burnCache[url] = burnCacheEntry{data: data, created: time.Now()}
}

func GenerateBurnGif(c *gin.Context) {
        var req BurnRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }
        if req.ImageUrl == "" {
                c.JSON(400, gin.H{"error": "imageUrl required"})
                return
        }

        if data, ok := getCachedBurn(req.ImageUrl); ok {
                c.Data(200, burnFmt, data)
                return
        }

        tempDir, err := os.MkdirTemp("", "cardburn_*")
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create temp directory"})
                return
        }
        defer os.RemoveAll(tempDir)

        // 1. Card image -> 800x800 flattened RGBA
        // (bounded 45s download + local decode: utils.DownloadImage uses a raw
        // http.Get with NO timeout — a stalled fetch would hang a worker forever)
        cardImg, err := downloadAndDecodeCard(req.ImageUrl, tempDir)
        if err != nil {
                // Fallback for exotic formats: download raw + ffmpeg-extract frame 1
                cardImg, err = downloadAndExtractFirstFrame(req.ImageUrl, tempDir)
                if err != nil {
                        fmt.Printf("[Burn] card download/decode failed (%s): %v\n", req.ImageUrl, err)
                        c.JSON(500, gin.H{"error": "Failed to download card image"})
                        return
                }
        }
        card := flattenCard(cardImg)
        tCard := time.Now()

        // 2. Fire overlay frames (bundled asset)
        frames, err := loadFireFrames()
        if err != nil {
                fmt.Printf("[Burn] fire overlay load failed: %v\n", err)
                c.JSON(500, gin.H{"error": "Fire overlay unavailable"})
                return
        }
        fmt.Printf("[Burn] timings: card+fire %s\n", time.Since(tCard).Round(time.Millisecond))

        // 3. Additive screen blend per frame
        tBlend := time.Now()
        blended := make([]*image.RGBA, len(frames))
        for i, fr := range frames {
                blended[i] = screenBlend(card, fr)
        }

        // 4. Raw RGBA frames piped straight into ffmpeg (no intermediate files)
        outputPath := filepath.Join(tempDir, "burn.mp4")
        asm := exec.Command("ffmpeg",
                "-f", "rawvideo",
                "-pix_fmt", "rgba",
                "-video_size", fmt.Sprintf("%dx%d", burnCanvas, burnCanvas),
                "-framerate", fmt.Sprint(burnFPS),
                "-i", "pipe:0",
                "-c:v", "libx264",
                "-pix_fmt", "yuv420p",
                "-preset", "ultrafast",
                "-crf", "26",
                "-movflags", "+faststart",
                "-y", outputPath)
        stdin, err := asm.StdinPipe()
        if err != nil {
                c.JSON(500, gin.H{"error": "ffmpeg pipe failed"})
                return
        }
        var asmErr error
        var wg sync.WaitGroup
        wg.Add(1)
        go func() {
                defer wg.Done()
                for _, fr := range blended {
                        if _, err := stdin.Write(fr.Pix); err != nil {
                                asmErr = err
                                return
                        }
                }
                stdin.Close()
        }()
        if output, err := asm.CombinedOutput(); err != nil {
                fmt.Printf("[Burn] FFmpeg assembly error: %v\nOutput: %s\n", err, string(output))
                c.JSON(500, gin.H{"error": "FFmpeg assembly failed"})
                return
        }
        wg.Wait()
        if asmErr != nil {
                fmt.Printf("[Burn] frame pipe error: %v\n", asmErr)
        }
        fmt.Printf("[Burn] timings: blend+pipe %s\n", time.Since(tBlend).Round(time.Millisecond))

        data, err := os.ReadFile(outputPath)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to read output"})
                return
        }

        setCachedBurn(req.ImageUrl, data)
        c.Data(200, burnFmt, data)
}

// downloadAndDecodeCard fetches the card (45s cap) and decodes it with the
// globally-registered codecs (png/jpeg/gif-first-frame/webp via gif.go imports).
func downloadAndDecodeCard(url, tempDir string) (image.Image, error) {
        client := &http.Client{Timeout: 45 * time.Second}
        raw := filepath.Join(tempDir, "card_raw")
        if err := downloadFile(client, url, raw); err != nil {
                return nil, err
        }
        fh, err := os.Open(raw)
        if err != nil {
                return nil, err
        }
        defer fh.Close()
        img, _, err := image.Decode(fh)
        return img, err
}

// downloadAndExtractFirstFrame handles card formats Go can't decode natively:
// download to disk, let ffmpeg pull frame 1 as PNG, decode that.
func downloadAndExtractFirstFrame(url, tempDir string) (image.Image, error) {
        client := &http.Client{Timeout: 30 * time.Second}
        resp, err := client.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return nil, fmt.Errorf("bad status: %s", resp.Status)
        }
        raw := filepath.Join(tempDir, "card_raw")
        out, err := os.Create(raw)
        if err != nil {
                return nil, err
        }
        if _, err := io.Copy(out, resp.Body); err != nil {
                out.Close()
                return nil, err
        }
        out.Close()

        first := filepath.Join(tempDir, "card_first.png")
        cmd := exec.Command("ffmpeg",
                "-i", raw,
                "-frames:v", "1",
                "-y", first)
        if output, err := cmd.CombinedOutput(); err != nil {
                return nil, fmt.Errorf("ffmpeg first-frame extract failed: %v: %s", err, string(output)[:min(len(output), 300)])
        }
        fh, err := os.Open(first)
        if err != nil {
                return nil, err
        }
        defer fh.Close()
        return png.Decode(fh)
}

func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// downloadFile fetches url to dest — shared helper (also used by convert.go / gif.go)
func downloadFile(client *http.Client, url string, dest string) error {
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

        out, err := os.Create(dest)
        if err != nil {
                return err
        }
        defer out.Close()

        _, err = io.Copy(out, resp.Body)
        return err
}
