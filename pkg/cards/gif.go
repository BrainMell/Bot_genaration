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
	if err := generateRandomGradient(bgPath, 500, 500); err != nil {
		fmt.Printf("[Cards] BG Gen failed: %v\n", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	
	type CardInput struct {
		Path       string
		IsAnimated bool
	}
	var localInputs []CardInput

	// Download images in parallel
	type downloadResult struct {
		index      int
		filePath   string
		isAnimated bool
		err        error
	}
	dlCh := make(chan downloadResult, len(req.Images))

	for i, urlStr := range req.Images {
		go func(index int, downloadUrl string) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", index))
			err := downloadFile(client, downloadUrl, filePath)
			dlCh <- downloadResult{
				index:      index,
				filePath:   filePath,
				isAnimated: isAnimated(downloadUrl),
				err:        err,
			}
		}(i, urlStr)
	}

	dlResults := make([]downloadResult, len(req.Images))
	for range req.Images {
		res := <-dlCh
		dlResults[res.index] = res
	}

	for _, res := range dlResults {
		if res.err == nil {
			localInputs = append(localInputs, CardInput{
				Path:       res.filePath,
				IsAnimated: res.isAnimated,
			})
		} else {
			fmt.Printf("[Cards] Failed to download card: %v\n", res.err)
		}
	}

	if len(localInputs) == 0 {
		c.JSON(500, gin.H{"error": "All image downloads failed"})
		return
	}

	type renderJob struct {
		isTransition bool
		index        int
		outputPath   string
		args         []string
	}
	var jobs []renderJob

	// 1. Create jobs for slides (1.5s duration, or 2.0s for the last one)
	for i, input := range localInputs {
		slidePath := filepath.Join(tempDir, fmt.Sprintf("slide_%d.mp4", i))
		duration := 1.5
		if i == len(localInputs)-1 {
			duration = 2.0
		}

		var args []string
		if input.IsAnimated {
			args = append(args, "-stream_loop", "-1", "-t", fmt.Sprintf("%.1f", duration), "-i", input.Path)
		} else {
			args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.1f", duration), "-i", input.Path)
		}
		args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.1f", duration), "-i", bgPath)
		
		filterStr := "[0:v]scale=460:460:force_original_aspect_ratio=decrease,pad=460:460:(460-iw)/2:(460-ih)/2:color=black@0[c];[1:v][c]overlay=20:20:shortest=1"
		
		args = append(args, 
			"-filter_complex", filterStr,
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
			"-r", "25",
			"-g", "50",
			"-preset", "ultrafast",
			"-y", slidePath,
		)

		jobs = append(jobs, renderJob{
			isTransition: false,
			index:        i,
			outputPath:   slidePath,
			args:         args,
		})
	}

	// 2. Create jobs for transitions (0.5s duration from i to i+1)
	for i := 0; i < len(localInputs)-1; i++ {
		transPath := filepath.Join(tempDir, fmt.Sprintf("trans_%d.mp4", i))
		input1 := localInputs[i]
		input2 := localInputs[i+1]

		var args []string
		if input1.IsAnimated {
			args = append(args, "-stream_loop", "-1", "-t", "0.5", "-i", input1.Path)
		} else {
			args = append(args, "-loop", "1", "-t", "0.5", "-i", input1.Path)
		}
		if input2.IsAnimated {
			args = append(args, "-stream_loop", "-1", "-t", "0.5", "-i", input2.Path)
		} else {
			args = append(args, "-loop", "1", "-t", "0.5", "-i", input2.Path)
		}
		args = append(args, "-loop", "1", "-t", "0.5", "-i", bgPath)

		filterStr := "[0:v]scale=460:460:force_original_aspect_ratio=decrease,pad=460:460:(460-iw)/2:(460-ih)/2:color=black@0[c1];[2:v][c1]overlay=20:20,format=yuv420p[s1];" +
			"[1:v]scale=460:460:force_original_aspect_ratio=decrease,pad=460:460:(460-iw)/2:(460-ih)/2:color=black@0[c2];[2:v][c2]overlay=20:20,format=yuv420p[s2];" +
			"[s1][s2]xfade=transition=slideright:duration=0.5:offset=0[out]"

		args = append(args, 
			"-filter_complex", filterStr,
			"-map", "[out]",
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
			"-r", "25",
			"-g", "50",
			"-preset", "ultrafast",
			"-y", transPath,
		)

		jobs = append(jobs, renderJob{
			isTransition: true,
			index:        i,
			outputPath:   transPath,
			args:         args,
		})
	}

	// 3. Run jobs sequentially to prevent memory spike and OOM crashes
	for _, job := range jobs {
		cmd := exec.Command("ffmpeg", job.args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Cards] FFmpeg error on job (isTrans=%t, idx=%d): %v\nOutput: %s\n", job.isTransition, job.index, err, string(output))
			c.JSON(500, gin.H{"error": "Failed to render showcase animation"})
			return
		}
	}

	// 4. Write concatenation text file in order: slide_0, trans_0, slide_1, trans_1, ..., slide_last
	concatTxtPath := filepath.Join(tempDir, "concat.txt")
	concatFile, err := os.Create(concatTxtPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create concat file"})
		return
	}
	for i := 0; i < len(localInputs); i++ {
		slidePath := filepath.Join(tempDir, fmt.Sprintf("slide_%d.mp4", i))
		escapedSlide := strings.ReplaceAll(slidePath, "'", "'\\''")
		fmt.Fprintf(concatFile, "file '%s'\n", escapedSlide)

		if i < len(localInputs)-1 {
			transPath := filepath.Join(tempDir, fmt.Sprintf("trans_%d.mp4", i))
			escapedTrans := strings.ReplaceAll(transPath, "'", "'\\''")
			fmt.Fprintf(concatFile, "file '%s'\n", escapedTrans)
		}
	}
	concatFile.Close()

	// 5. Concatenate slides
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
