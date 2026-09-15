package utils

import (
        "bytes"
        "fmt"
        "image"
        "image/color"
        "image/draw"
        "image/jpeg"
        "image/png"
        "io"
        "net/http"
        "os"
        "path/filepath"
        "strconv"
        "strings"
        "sync"
        "time"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
        "golang.org/x/image/font"
        "golang.org/x/image/font/opentype"
)

// Asset Cache to avoid re-reading from disk
var (
        imageCache = make(map[string]image.Image)
        mutex      sync.RWMutex
)

// LoadImage loads an image from disk or cache
func LoadImage(path string) (image.Image, error) {
        mutex.RLock()
        if img, ok := imageCache[path]; ok {
                mutex.RUnlock()
                return img, nil
        }
        mutex.RUnlock()

        // Check if file exists
        if _, err := os.Stat(path); os.IsNotExist(err) {
                return nil, fmt.Errorf("file not found: %s", path)
        }

        img, err := imaging.Open(path)
        if err != nil {
                return nil, err
        }

        mutex.Lock()
        imageCache[path] = img
        mutex.Unlock()

        return img, nil
}

// DownloadImage fetches an image from a URL
func DownloadImage(url string) (image.Image, error) {
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()

        img, _, err := image.Decode(resp.Body)
        if err != nil {
                return nil, err
        }
        return img, nil
}

// LoadFont loads a TTF font face at the given size.
// 2026-09-14 PERF: the TTF bytes were re-read + re-PARSED on every call —
// a single card render loads fonts 15-30x (huntFitText re-loads while
// shrinking), making parse the dominant CPU cost. opentype.Font is
// immutable after Parse, so the parsed font is cached per path and shared
// across requests; the FACE (glyph cache state) is still created per call,
// keeping concurrent requests safe.
var (
        fontMutex sync.RWMutex
        fontCache = make(map[string]*opentype.Font)
)

func loadParsedFont(path string) (*opentype.Font, error) {
        fontMutex.RLock()
        if ft, ok := fontCache[path]; ok {
                fontMutex.RUnlock()
                return ft, nil
        }
        fontMutex.RUnlock()

        fontBytes, err := os.ReadFile(path)
        if err != nil {
                return nil, err
        }
        ft, err := opentype.Parse(fontBytes)
        if err != nil {
                return nil, err
        }

        fontMutex.Lock()
        fontCache[path] = ft
        fontMutex.Unlock()
        return ft, nil
}

func LoadFont(path string, size float64) (font.Face, error) {
        ft, err := loadParsedFont(path)
        if err != nil {
                return nil, err
        }

        return opentype.NewFace(ft, &opentype.FaceOptions{
                Size:    size,
                DPI:     72,
                Hinting: font.HintingFull,
        })
}

// ParseHexColor converts hex string to color.RGBA
func ParseHexColor(s string) color.RGBA {
        c := color.RGBA{0, 0, 0, 255}
        switch len(s) {
        case 7:
                fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
        case 9:
                fmt.Sscanf(s, "#%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A)
        }
        return c
}

// DrawShadow draws a radial shadow
func DrawShadow(dc *gg.Context, x, y, radius float64, alpha float64) {
        // 💡 FIX 2026-08-07: Changed shadow from pure black to dark warm brown
        // so it's visible against the dark overlay background. Pure black
        // shadows were invisible on the 31% black overlay, making dark-colored
        // sprites (wolf, bat) appear "floating" to viewers.
        grad := gg.NewRadialGradient(x, y, 0, x, y, radius)
        grad.AddColorStop(0, color.RGBA{20, 10, 0, uint8(alpha * 255)})
        grad.AddColorStop(0.5, color.RGBA{15, 8, 0, uint8(alpha * 128)})
        grad.AddColorStop(1, color.RGBA{0, 0, 0, 0})
        dc.SetFillStyle(grad)
        dc.DrawCircle(x, y, radius)
        dc.Fill()
}

// DrawShadowEllipse draws a wide, flat elliptical shadow (ground shadow).
// rx = horizontal radius (wide), ry = vertical radius (short).
// Used for PvP where a circular shadow would overlap the UI panel below.
func DrawShadowEllipse(dc *gg.Context, x, y, rx, ry, alpha float64) {
        grad := gg.NewRadialGradient(x, y, 0, x, y, rx)
        grad.AddColorStop(0, color.RGBA{20, 10, 0, uint8(alpha * 255)})
        grad.AddColorStop(0.5, color.RGBA{15, 8, 0, uint8(alpha * 128)})
        grad.AddColorStop(1, color.RGBA{0, 0, 0, 0})
        dc.SetFillStyle(grad)
        dc.DrawEllipse(x, y, rx, ry)
        dc.Fill()
}

// TintImage adds a red overlay to an image (for dead units)
func TintImage(img image.Image, tint color.RGBA) image.Image {
        bounds := img.Bounds()
        dst := image.NewRGBA(bounds)

        // Copy original image
        draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

        // Apply tint manually to non-transparent pixels
        for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
                for x := bounds.Min.X; x < bounds.Max.X; x++ {
                        originalColor := dst.At(x, y)
                        _, _, _, a := originalColor.RGBA()

                        if a > 0 {
                                // Blend the tint color
                                blended := Blend(originalColor, tint)
                                dst.Set(x, y, blended)
                        }
                }
        }

        return dst
}

// Blend blends two colors (simplified for red tint)
func Blend(base color.Color, tint color.RGBA) color.Color {
        r1, g1, b1, a1 := base.RGBA()

        // Convert to 8-bit
        r1 >>= 8
        g1 >>= 8
        b1 >>= 8
        a1 >>= 8

        // Blend logic: increase red, decrease others based on tint alpha
        alpha := float64(tint.A) / 255.0

        r := uint8(float64(r1)*(1-alpha) + float64(tint.R)*alpha)
        g := uint8(float64(g1)*(1-alpha) + float64(tint.G)*alpha)
        b := uint8(float64(b1)*(1-alpha) + float64(tint.B)*alpha)

        return color.RGBA{r, g, b, uint8(a1)}
}

// EncodeImageToBuffer returns PNG bytes
func EncodeImageToBuffer(img image.Image) ([]byte, error) {
        buf := new(bytes.Buffer)
        if err := png.Encode(buf, img); err != nil {
                return nil, err
        }
        return buf.Bytes(), nil
}

// EncodeImageToBufferFormat encodes PNG (default) or JPEG (?fmt=jpeg).
// 2026-09-14 PERF: the parchment family renders ~800KB-1MB PNGs; the same
// art at JPEG q90 is 4-6x smaller, which directly cuts the bot's WhatsApp
// media upload time (the dominant chunk of perceived render latency).
// Cards are fully opaque (baked wood/parchment), so JPEG's no-alpha rule
// is a non-issue. Returns (bytes, contentType, error).
func EncodeImageToBufferFormat(img image.Image, format string, quality int) ([]byte, string, error) {
        switch strings.ToLower(strings.TrimSpace(format)) {
        case "jpeg", "jpg":
                if quality <= 0 {
                        quality = 90
                }
                if quality > 100 {
                        quality = 100
                }
                buf := new(bytes.Buffer)
                if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
                        return nil, "", err
                }
                return buf.Bytes(), "image/jpeg", nil
        default:
                b, err := EncodeImageToBuffer(img)
                return b, "image/png", err
        }
}

// RespondImage writes the rendered image, honoring ?fmt=jpeg&q=NN.
// Default stays PNG so every existing caller is unchanged.
func RespondImage(c *gin.Context, img image.Image) {
        quality := 90
        if qs := c.Query("q"); qs != "" {
                if n, err := strconv.Atoi(qs); err == nil && n >= 40 && n <= 100 {
                        quality = n
                }
        }
        buf, ctype, err := EncodeImageToBufferFormat(img, c.Query("fmt"), quality)
        if err != nil {
                c.JSON(500, gin.H{"error": "encode failed"})
                return
        }
        c.Data(200, ctype, buf)
}

// ─────────────────────────────────────────────────────────────────────────────
// ThumbHandler — POST /api/thumb
// 2026-09-15 PERF (owner: "make images get sent faster"):
// The bot's Node side built every WhatsApp preview thumbnail (jpegThumbnail)
// with jimp — a pure-JS FULL-image decode — measured at 400-900ms per image
// on Box 1's CPU, paid inline on EVERY image send BEFORE the media upload
// even starts. Go decodes the same bytes in single-digit ms and scales with
// imaging.CatmullRom, so this endpoint turns that fixed cost into ~10-20ms
// of localhost work.
// Body  = raw image bytes (PNG or JPEG, what every renderer returns).
// Reply = aspect-fit ≤120px JPEG q60 (same size/quality band as the old jimp
//         output, ~2-3KB) + X-Thumb-Ms timing header.
// Node falls back to jimp automatically if this endpoint errors.
func ThumbHandler(c *gin.Context) {
        start := time.Now()
        body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 24<<20))
        if err != nil || len(body) < 64 {
                c.JSON(400, gin.H{"error": "bad or empty image body"})
                return
        }
        src, _, err := image.Decode(bytes.NewReader(body))
        if err != nil {
                c.JSON(400, gin.H{"error": "unsupported image bytes"})
                return
        }
        sb := src.Bounds()
        var out image.Image = src
        if sb.Dx() > 120 || sb.Dy() > 120 {
                out = imaging.Fit(src, 120, 120, imaging.CatmullRom)
        }
        buf := new(bytes.Buffer)
        if err := jpeg.Encode(buf, out, &jpeg.Options{Quality: 60}); err != nil {
                c.JSON(500, gin.H{"error": "thumb encode failed"})
                return
        }
        c.Header("X-Thumb-Ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
        c.Data(200, "image/jpeg", buf.Bytes())
}

func GetAssetPath(parts ...string) string {
        // Resolve assets relative to the executable's directory.
        var base string
        if exePath, err := os.Executable(); err == nil {
                base = filepath.Dir(exePath)
        } else {
                // Fallback to current working directory if Exec path fails
                cwd, _ := os.Getwd()
                base = cwd
        }
        // Build full path to assets directory
        fullPath := filepath.Join(append([]string{base, "assets"}, parts...)...)

        // Check if the asset exists in the executable folder; if not, look in working directory
        if _, err := os.Stat(fullPath); os.IsNotExist(err) {
                cwd, _ := os.Getwd()
                fallbackPath := filepath.Join(append([]string{cwd, "assets"}, parts...)...)
                if _, err := os.Stat(fallbackPath); err == nil {
                        return fallbackPath
                }
        }

        return fullPath
}
