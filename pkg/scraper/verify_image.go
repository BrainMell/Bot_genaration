// P26 (quiz box-offload): server-side image verification.
// quiz.js previously downloaded every candidate image on the bot box just to
// magic-byte-verify it (P13) and hash it. This endpoint performs the identical
// check server-side and returns only metadata, so the bot box skips the
// generation-time download entirely (the send-time fetch still happens once,
// through the quiz's asset cache).
package scraper

import (
        "crypto/sha1"
        "encoding/hex"
	"fmt"
        "io"
        "net/http"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
)

var verifyHTTP = &http.Client{Timeout: 20 * time.Second}

// VerifyImage GET /api/verify/image?url=<https url>
// Mirrors quizLore.downloadMedia("image") checks exactly:
//   - https-only URLs (plus loopback http for local QA)
//   - max 6MB, min 1KB
//   - magic bytes: jpeg / png / webp
//   - mime mapped with the same rule the Node code uses
//   - sha1 hex digest, first 16 chars (same length Node stores)
func VerifyImage(c *gin.Context) {
        u := c.Query("url")
        if u == "" {
                c.JSON(400, gin.H{"ok": false, "error": "url required"})
                return
        }
        if !strings.HasPrefix(u, "https://") &&
                !(strings.HasPrefix(u, "http://localhost:") || strings.HasPrefix(u, "http://127.0.0.1:")) {
                c.JSON(200, gin.H{"ok": false, "error": "unsupported_scheme"})
                return
        }
        req, err := http.NewRequest("GET", u, nil)
        if err != nil {
        	c.JSON(200, gin.H{"ok": false, "error": "bad_request"})
        	return
        }
        // same User-Agent the Node axios client sends (quizLore UA) - the
        // Fandom image CDN rejects the default Go client UA
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) WhatsAppQuizBot/1.0")
        req.Header.Set("Accept", "application/json")
        resp, err := verifyHTTP.Do(req)
        if err != nil {
                c.JSON(200, gin.H{"ok": false, "error": "fetch_failed"})
                return
        }
        defer resp.Body.Close()
        if resp.StatusCode < 200 || resp.StatusCode >= 300 {
                c.JSON(200, gin.H{"ok": false, "error": fmt.Sprintf("http_status_%d", resp.StatusCode)})
                return
        }
        buf, err := io.ReadAll(io.LimitReader(resp.Body, 6*1024*1024+1))
        if err != nil {
                c.JSON(200, gin.H{"ok": false, "error": "read_failed"})
                return
        }
        n := len(buf)
        if n < 1024 || n > 6*1024*1024 {
                c.JSON(200, gin.H{"ok": false, "error": "size_out_of_range", "size": n})
                return
        }
        ok := (buf[0] == 0xff && buf[1] == 0xd8) ||
                (buf[0] == 0x89 && buf[1] == 0x50) ||
                (string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP")
        if !ok {
                c.JSON(200, gin.H{"ok": false, "error": "magic_bytes"})
                return
        }
        mime := "image/jpeg"
        if buf[0] == 0x89 {
                mime = "image/png"
        } else if string(buf[8:12]) == "WEBP" {
                mime = "image/webp"
        }
        sum := sha1.Sum(buf)
        c.JSON(200, gin.H{
                "ok":   true,
                "mime": mime,
                "size": n,
                "sha1": hex.EncodeToString(sum[:])[:16],
        })
}
