package cards

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
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

func uploadToCloudinary(client *http.Client, cloudName, apiKey, apiSecret, fileURL, publicID, tag, resourceType string) (string, error) {
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

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload", cloudName, resourceType)
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

func deleteFromCloudinary(client *http.Client, cloudName, apiKey, apiSecret, resourceType string, publicIDs []string) error {
	deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/%s/upload", cloudName, resourceType)

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
	
	// Step 1: Upload images in parallel, skipping failed ones
	type uploadResult struct {
		index    int
		publicID string
		err      error
	}
	imgCh := make(chan uploadResult, len(images))

	for i, urlStr := range images {
		go func(index int, uStr string) {
			publicID := fmt.Sprintf("%s_img_%d", tag, index)
			pid, err := uploadToCloudinary(client, cloudName, apiKey, apiSecret, uStr, publicID, tag, "image")
			imgCh <- uploadResult{index: index, publicID: pid, err: err}
		}(i, urlStr)
	}

	type successfulUpload struct {
		index    int
		publicID string
	}
	var successfulImages []successfulUpload

	for range images {
		res := <-imgCh
		if res.err == nil {
			successfulImages = append(successfulImages, successfulUpload{index: res.index, publicID: res.publicID})
		} else {
			fmt.Printf("[Cloudinary] Skipped failed card image %d upload: %v\n", res.index, res.err)
		}
	}

	if len(successfulImages) == 0 {
		return nil, fmt.Errorf("all image uploads failed")
	}

	// Sort successfulImages to preserve original slideshow order
	sort.Slice(successfulImages, func(i, j int) bool {
		return successfulImages[i].index < successfulImages[j].index
	})

	// Step 2: Upload MP4 conversion URLs in parallel
	vidCh := make(chan uploadResult, len(successfulImages))
	for i, sImg := range successfulImages {
		go func(index int, pid string) {
			mp4URL := fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/w_500,h_500,c_fill,du_1.5/%s.mp4", cloudName, pid)
			videoPublicID := fmt.Sprintf("%s_vid_%d", tag, index)
			vPID, err := uploadToCloudinary(client, cloudName, apiKey, apiSecret, mp4URL, videoPublicID, tag, "video")
			vidCh <- uploadResult{index: index, publicID: vPID, err: err}
		}(i, sImg.publicID)
	}

	vidResults := make([]string, len(successfulImages))
	for range successfulImages {
		res := <-vidCh
		if res.err == nil {
			vidResults[res.index] = res.publicID
		} else {
			fmt.Printf("[Cloudinary] Failed to upload video conversion for index %d: %v\n", res.index, res.err)
		}
	}

	var vidPublicIDs []string
	for _, pid := range vidResults {
		if pid != "" {
			vidPublicIDs = append(vidPublicIDs, pid)
		}
	}

	// Collect successful image IDs for cleanup
	var imgPublicIDs []string
	for _, sImg := range successfulImages {
		imgPublicIDs = append(imgPublicIDs, sImg.publicID)
	}

	if len(vidPublicIDs) == 0 {
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		return nil, fmt.Errorf("all video conversion uploads failed")
	}

	// Step 3: Splice in the video namespace
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/w_500,h_500,c_fill/", cloudName))
	for i := 1; i < len(vidPublicIDs); i++ {
		buf.WriteString(fmt.Sprintf("fl_splice:transition_(name_slideright;du_0.5),l_video:%s/fl_layer_apply/", vidPublicIDs[i]))
	}
	buf.WriteString(fmt.Sprintf("%s.mp4", vidPublicIDs[0]))
	spliceURL := buf.String()

	fmt.Printf("[Cloudinary] Generated Video Splicing URL on Go server: %s\n", spliceURL)

	// Step 4: Download the final video/mp4 buffer
	resp, err := client.Get(spliceURL)
	if err != nil {
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
		return nil, fmt.Errorf("failed to fetch spliced video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spliced video fetch failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	videoBuffer, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
		return nil, fmt.Errorf("failed to read video response: %w", err)
	}

	// Step 5: Clean up everything in background
	go func() {
		cleanupClient := &http.Client{Timeout: 30 * time.Second}
		_ = deleteFromCloudinary(cleanupClient, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		_ = deleteFromCloudinary(cleanupClient, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
	}()

	return videoBuffer, nil
}

type CardInput struct {
	Path string
}

func GenerateCardGridImage(inputs []CardInput) ([]byte, error) {
	cols := 5
	n := len(inputs)
	if n == 0 {
		return nil, fmt.Errorf("no input cards")
	}
	if n < 5 {
		cols = n
	}

	rows := (n + cols - 1) / cols
	cardW := 200
	cardH := 300
	spacing := 4

	canvasW := cols*cardW + (cols-1)*spacing
	canvasH := rows*cardH + (rows-1)*spacing

	dc := gg.NewContext(canvasW, canvasH)
	
	// Set a deep, elegant dark neutral background
	dc.SetHexColor("#121212")
	dc.Clear()

	for i, input := range inputs {
		col := i % cols
		row := i / cols
		x := col * (cardW + spacing)
		y := row * (cardH + spacing)

		// Decode the downloaded image
		file, err := os.Open(input.Path)
		if err != nil {
			fmt.Printf("[Cards] Failed to open card path %s: %v\n", input.Path, err)
			continue
		}
		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			fmt.Printf("[Cards] Failed to decode card image %s: %v\n", input.Path, err)
			continue
		}

		// Resize to portrait 2:3 aspect ratio
		resized := imaging.Resize(img, cardW, cardH, imaging.Lanczos)

		// Draw onto canvas edge to edge
		dc.DrawImage(resized, x, y)
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, dc.Image())
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
		fmt.Printf("[Cloudinary] Failed generating slideshow: %v. Falling back to Grid Image Generator...\n", err)
	}

	// 2. Fallback to lightweight Go-based Card Grid Image Generator
	tempDir, err := os.MkdirTemp("", "cardgrid_*")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.RemoveAll(tempDir)

	client := &http.Client{Timeout: 15 * time.Second}
	var localInputs []CardInput

	// Download images in parallel
	type downloadResult struct {
		index    int
		filePath string
		err      error
	}
	dlCh := make(chan downloadResult, len(req.Images))

	for i, urlStr := range req.Images {
		go func(index int, downloadUrl string) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", index))
			err := downloadFile(client, downloadUrl, filePath)
			dlCh <- downloadResult{
				index:    index,
				filePath: filePath,
				err:      err,
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
				Path: res.filePath,
			})
		} else {
			fmt.Printf("[Cards] Failed to download card for grid: %v\n", res.err)
		}
	}

	if len(localInputs) == 0 {
		c.JSON(500, gin.H{"error": "All image downloads failed"})
		return
	}

	fmt.Printf("[Cards] Rendering %d cards in fallback grid layout...\n", len(localInputs))
	gridData, err := GenerateCardGridImage(localInputs)
	if err != nil {
		fmt.Printf("[Cards] Grid image generation failed: %v\n", err)
		c.JSON(500, gin.H{"error": "Grid fallback generation failed"})
		return
	}

	c.Data(200, "image/png", gridData)
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
