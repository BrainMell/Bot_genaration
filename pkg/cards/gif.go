package cards

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
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

type CloudinaryUploadResponse struct {
	PublicID  string `json:"public_id"`
	SecureURL string `json:"secure_url"`
	Error     struct {
		Message string `json:"message"`
	} `json:"error"`
}

func generateSignature(publicID, tags, timestamp, apiSecret string) string {
	rawStr := fmt.Sprintf("public_id=%s&tags=%s&timestamp=%s%s", publicID, tags, timestamp, apiSecret)
	h := sha1.New()
	h.Write([]byte(rawStr))
	return hex.EncodeToString(h.Sum(nil))
}

func uploadToCloudinary(client *http.Client, cloudName, apiKey, apiSecret, fileURL, publicID, tag string) (string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := generateSignature(publicID, tag, timestamp, apiSecret)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("file", fileURL)
	_ = w.WriteField("public_id", publicID)
	_ = w.WriteField("tags", tag)
	_ = w.WriteField("timestamp", timestamp)
	_ = w.WriteField("api_key", apiKey)
	_ = w.WriteField("signature", signature)
	w.Close()

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)
	req, err := http.NewRequest("POST", uploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var res CloudinaryUploadResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	if res.Error.Message != "" {
		return "", fmt.Errorf("error: %s", res.Error.Message)
	}

	return res.PublicID, nil
}

func deleteFromCloudinary(client *http.Client, cloudName, apiKey, apiSecret string, publicIDs []string) error {
	deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/image/upload", cloudName)

	u, err := url.Parse(deleteURL)
	if err != nil {
		return err
	}
	q := u.Query()
	for _, pid := range publicIDs {
		q.Add("public_ids[]", pid)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	auth := apiKey + ":" + apiSecret
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encodedAuth)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func GenerateCloudinarySlideshow(images []string, cloudName, apiKey, apiSecret string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	tag := fmt.Sprintf("deck_%d_%d", time.Now().Unix(), rand.Intn(10000))
	
	type uploadResult struct {
		index    int
		publicID string
		err      error
	}
	ch := make(chan uploadResult, len(images))
	publicIDs := make([]string, len(images))

	for i, imgURL := range images {
		go func(index int, urlStr string) {
			publicID := fmt.Sprintf("%s_%03d", tag, index)
			pid, err := uploadToCloudinary(client, cloudName, apiKey, apiSecret, urlStr, publicID, tag)
			ch <- uploadResult{index: index, publicID: pid, err: err}
		}(i, imgURL)
	}

	var uploadErrors []error
	for range images {
		res := <-ch
		if res.err != nil {
			uploadErrors = append(uploadErrors, res.err)
		} else {
			publicIDs[res.index] = res.publicID
		}
	}

	if len(uploadErrors) > 0 {
		var toDelete []string
		for _, pid := range publicIDs {
			if pid != "" {
				toDelete = append(toDelete, pid)
			}
		}
		if len(toDelete) > 0 {
			_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, toDelete)
		}
		return nil, fmt.Errorf("cloudinary upload failed: %v", uploadErrors[0])
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/w_500,h_500,c_fill,du_1.5/", cloudName))
	for i := 1; i < len(publicIDs); i++ {
		buf.WriteString(fmt.Sprintf("fl_splice:transition_(name_slideright;du_0.5),l_%s/w_500,h_500,c_fill,du_1.5/fl_layer_apply/", publicIDs[i]))
	}
	buf.WriteString(fmt.Sprintf("%s.mp4", publicIDs[0]))
	spliceURL := buf.String()

	fmt.Printf("[Cloudinary] Splicing URL on Go server: %s\n", spliceURL)

	resp, err := client.Get(spliceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spliced video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spliced video fetch failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	videoBuffer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read video response: %w", err)
	}

	go func() {
		cleanupClient := &http.Client{Timeout: 30 * time.Second}
		_ = deleteFromCloudinary(cleanupClient, cloudName, apiKey, apiSecret, publicIDs)
	}()

	return videoBuffer, nil
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

	// 1. Try Cloudinary slideshow first
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	if cloudName == "" {
		cloudName = "dwgct8qng"
	}
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	if apiKey == "" {
		apiKey = "551174662862282"
	}
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	if apiSecret == "" {
		apiSecret = "ez0LwjGnhNiBSE5esFpbBnwjRGg"
	}

	if cloudName != "" && apiKey != "" && apiSecret != "" {
		fmt.Printf("[Cloudinary] Splicing %d images on Go image server...\n", len(req.Images))
		vidData, err := GenerateCloudinarySlideshow(req.Images, cloudName, apiKey, apiSecret)
		if err == nil {
			fmt.Printf("[Cloudinary] Successfully generated slideshow! Size: %d bytes\n", len(vidData))
			c.Data(200, "video/mp4", vidData)
			return
		}
		fmt.Printf("[Cloudinary] Failed generating slideshow: %v. Falling back to local FFmpeg...\n", err)
	}

	// 2. Local FFmpeg fallback
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

	for _, job := range jobs {
		cmd := exec.Command("ffmpeg", job.args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("[Cards] FFmpeg error on job (isTrans=%t, idx=%d): %v\nOutput: %s\n", job.isTransition, job.index, err, string(output))
			c.JSON(500, gin.H{"error": "Failed to render showcase animation"})
			return
		}
	}

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
