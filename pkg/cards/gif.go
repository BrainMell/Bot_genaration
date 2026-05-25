package cards

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"image-service/pkg/utils"

	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

type GifRequest struct {
	Images []string `json:"images"`
	Title  string   `json:"title"`
}

func isAnimated(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, ".webm") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".webp")
}

func GenerateCardGif(c *gin.Context) {
	var req GifRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if len(req.Images) == 0 {
		c.JSON(400, gin.H{"error": "No images provided"})
		return
	}

	maxImages := 15
	if len(req.Images) > maxImages {
	        req.Images = req.Images[:maxImages]
	}
	tempDir, err := os.MkdirTemp("", "cardvid_*")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.RemoveAll(tempDir)

	bgPath := filepath.Join(tempDir, "bg.png")
	if err := generateRandomGradient(bgPath, 800, 800); err != nil {
		fmt.Printf("[Cards] BG Gen failed: %v\n", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	
	type CardInput struct {
		Path       string
		IsAnimated bool
	}
	var localInputs []CardInput

	for i, url := range req.Images {
		filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", i))
		if err := downloadFile(client, url, filePath); err == nil {
			localInputs = append(localInputs, CardInput{
				Path:       filePath,
				IsAnimated: isAnimated(url),
			})
		} else {
			fmt.Printf("[Cards] Failed to download %s: %v\n", url, err)
		}
	}

	if len(localInputs) == 0 {
		c.JSON(500, gin.H{"error": "All image downloads failed"})
		return
	}

	// Generate slide videos for each card
	var slidePaths []string
	for i, input := range localInputs {
		slidePath := filepath.Join(tempDir, fmt.Sprintf("slide_%d.mp4", i))
		var slideArgs []string
		
		if input.IsAnimated {
			slideArgs = append(slideArgs, "-stream_loop", "-1", "-t", "2.0", "-i", input.Path)
		} else {
			slideArgs = append(slideArgs, "-loop", "1", "-t", "2.0", "-i", input.Path)
		}
		slideArgs = append(slideArgs, "-loop", "1", "-t", "2.0", "-i", bgPath)
		
		filterStr := "[0:v]scale=740:740:force_original_aspect_ratio=decrease,pad=740:740:(740-iw)/2:(740-ih)/2:color=black@0[c];[1:v][c]overlay=30:30:shortest=1"
		
		slideArgs = append(slideArgs, 
			"-filter_complex", filterStr,
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
			"-r", "25",
			"-g", "50",
			"-preset", "ultrafast",
			"-y", slidePath,
		)
		
		cmd := exec.Command("ffmpeg", slideArgs...)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Cards] FFmpeg slide_%d error: %v\nOutput: %s\n", i, err, string(output))
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate slide %d", i)})
			return
		}
		slidePaths = append(slidePaths, slidePath)
	}

	// Write concatenation text file
	concatTxtPath := filepath.Join(tempDir, "concat.txt")
	concatFile, err := os.Create(concatTxtPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create concat file"})
		return
	}
	for _, path := range slidePaths {
		escapedPath := strings.ReplaceAll(path, "'", "'\\''")
		fmt.Fprintf(concatFile, "file '%s'\n", escapedPath)
	}
	concatFile.Close()

	// Concatenate slides
	outputPath := filepath.Join(tempDir, "output.mp4")
	concatCmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatTxtPath, "-c", "copy", "-y", outputPath)
	if output, err := concatCmd.CombinedOutput(); err != nil {
		fmt.Printf("[Cards] FFmpeg concat error: %v\nOutput: %s\n", err, string(output))
		c.JSON(500, gin.H{"error": "Video concatenation failed"})
		return
	}

	vidData, err := os.ReadFile(outputPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Read failed"})
		return
	}

	c.Data(200, "video/mp4", vidData)
}

func generateRandomGradient(path string, w, h int) error {
	dc := gg.NewContext(w, h)
	r1, g1, b1 := rand.Float64(), rand.Float64(), rand.Float64()
	r2, g2, b2 := rand.Float64(), rand.Float64(), rand.Float64()
	grad := gg.NewLinearGradient(0, 0, float64(w), float64(h))
	grad.AddColorStop(0, utils.ParseHexColor(fmt.Sprintf("#%02x%02x%02x", int(r1*255), int(g1*255), int(b1*255))))
	grad.AddColorStop(1, utils.ParseHexColor(fmt.Sprintf("#%02x%02x%02x", int(r2*255), int(g2*255), int(b2*255))))
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()
	return dc.SavePNG(path)
}

