package cards

import (
        "fmt"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "strconv"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
)

// 💡 FIX 2026-10-09 (owner: "long gif animations show just one frame, e.g. the
// S-tier Broly vs Gogeta"): the cards DB stores cardr API URLs with
// ?size=400 (some even ?size=100). Shoob serves a STATIC 1-frame preview for
// any size <= 400 (measured: size=400 -> 144KB / 1 frame; size=500 -> 32MB /
// 349 frames / ~14s). Every animated card converted from size<=400 was
// therefore a static MP4. Upgrade those to size=500 (the smallest size that
// returns the full animation) and cap fps at 12 so the resulting 800x800 MP4
// stays well under WhatsApp's 16MB video ceiling (349-frame S-tier measured:
// 15.7MB at native fps vs 9.1MB at fps=12/crf24, full duration preserved).

type ConvertRequest struct {
        ImageUrl string `json:"imageUrl"`
}

var (
        cardrPathRe     = regexp.MustCompile(`api\.shoob\.gg/site/api/cardr/`)
        cardrSizePrmRe  = regexp.MustCompile(`\?size=\d+`)
)

// normalizeShoobCardrURL upgrades cardr API URLs to the full-animation size by
// replacing the size query param IN PLACE (scheme/host/other params untouched).
// CDN direct files (*.webm / *.gif under cdn.shoob.gg) are returned untouched.
func normalizeShoobCardrURL(raw string) string {
        if !cardrPathRe.MatchString(raw) {
                return raw
        }
        if m := cardrSizePrmRe.FindString(raw); m != "" {
                n, err := strconv.Atoi(strings.TrimPrefix(m, "?size="))
                if err == nil && n >= 500 {
                        return raw // already a full-animation size
                }
                return cardrSizePrmRe.ReplaceAllString(raw, "?size=500")
        }
        return raw + "?size=500"
}

func ConvertCard(c *gin.Context) {
        var req ConvertRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
        }

        imageUrl := normalizeShoobCardrURL(req.ImageUrl)
        if imageUrl != req.ImageUrl {
                fmt.Printf("[Convert] upgraded static-preview URL to full animation: %s\n", imageUrl)
        }

        tempDir, err := os.MkdirTemp("", "cardconv_*")
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create temp directory"})
                return
        }
        defer os.RemoveAll(tempDir)

        client := &http.Client{Timeout: 90 * time.Second} // 💡 30s starved the 32MB full-animation GIFs
        inputPath := filepath.Join(tempDir, "input")
        if err := downloadFile(client, imageUrl, inputPath); err != nil {
                c.JSON(500, gin.H{"error": "Failed to download image"})
                return
        }

        outputPath := filepath.Join(tempDir, "output.mp4")

        // FFmpeg: Scale to 800x800 square with padding to match .j coll behavior
        // This ensures the card shows up "large" in WhatsApp.
        // fps=12 first in the chain: fewer frames to scale AND encode (GIF loop
        // feel preserved, duration untouched).
        cmd := exec.Command("ffmpeg",
                "-i", inputPath,
                "-vf", "fps=12,scale=800:800:force_original_aspect_ratio=decrease,pad=800:800:(ow-iw)/2:(oh-ih)/2:color=black",
                "-c:v", "libx264",
                "-pix_fmt", "yuv420p",
                "-preset", "ultrafast",
                "-crf", "24",
                "-movflags", "+faststart",
                "-y", outputPath)

        if output, err := cmd.CombinedOutput(); err != nil {
                fmt.Printf("[Convert] FFmpeg error: %v\nOutput: %s\n", err, string(output))
                c.JSON(500, gin.H{"error": "Conversion failed"})
                return
        }

        data, err := os.ReadFile(outputPath)
        if err != nil {
                c.JSON(500, gin.H{"error": "Read failed"})
                return
        }

        c.Data(200, "video/mp4", data)
}

