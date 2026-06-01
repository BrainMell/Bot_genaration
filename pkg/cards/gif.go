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
	"strings"
	"time"

	"image-service/pkg/utils"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
)

type CardImageInput struct {
	URL      string `json:"url" binding:"required"`
	Animated bool   `json:"animated"`
}

type GifRequest struct {
	Images []CardImageInput `json:"images" binding:"required"`
	Title  string           `json:"title"`
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

type MultiResponse struct {
	URL       string `json:"url"`
	SecureURL string `json:"secure_url"`
	Error     struct {
		Message string `json:"message"`
	} `json:"error"`
}

func GenerateCloudinarySlideshow(images []CardImageInput, cloudName, apiKey, apiSecret string) ([]byte, error) {
	client := &http.Client{Timeout: 90 * time.Second}
	tag := fmt.Sprintf("deck_%d_%d", time.Now().Unix(), rand.Intn(10000))

	type segmentResult struct {
		index    int
		imgPid   string
		vidPid   string
		err      error
	}

	segmentCh := make(chan segmentResult, len(images))

	for i, imgInput := range images {
		go func(index int, input CardImageInput) {
			imgPid := fmt.Sprintf("%s_img_%03d", tag, index)
			vidPid := fmt.Sprintf("%s_vid_%03d", tag, index)

			// Upload original image to image namespace
			imageTag := tag
			if !input.Animated && !isAnimated(input.URL) {
				imageTag = fmt.Sprintf("%s_static_%03d", tag, index)
			}

			_, err := uploadToCloudinary(client, cloudName, apiKey, apiSecret, input.URL, imgPid, imageTag, "image")
			if err != nil {
				segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed image upload: %w", err)}
				return
			}

			var mp4URL string
			if input.Animated || isAnimated(input.URL) {
				// Animated GIF conversion URL
				mp4URL = fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/c_pad,w_500,h_500,b_rgb:121212,du_1.5/%s.mp4", cloudName, imgPid)
			} else {
				// Static image multi URL
				multiURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/multi", cloudName)
				data := url.Values{}
				data.Set("tag", imageTag)
				data.Set("format", "mp4")
				data.Set("transformation", "c_pad,w_500,h_500,b_rgb:121212/dl_1500")

				req, err := http.NewRequest("POST", multiURL, strings.NewReader(data.Encode()))
				if err != nil {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed multi request init: %w", err)}
					return
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth(apiKey, apiSecret)

				resp, err := client.Do(req)
				if err != nil {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed multi request: %w", err)}
					return
				}
				defer resp.Body.Close()

				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed read multi response: %w", err)}
					return
				}

				if resp.StatusCode != 200 {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("multi request status %d: %s", resp.StatusCode, string(respBody))}
					return
				}

				var multiRes MultiResponse
				if err := json.Unmarshal(respBody, &multiRes); err != nil {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed parse multi response: %w", err)}
					return
				}

				if multiRes.Error.Message != "" {
					segmentCh <- segmentResult{index: index, err: fmt.Errorf("multi api error: %s", multiRes.Error.Message)}
					return
				}

				mp4URL = multiRes.SecureURL
			}

			// Download the 1.5s video segment
			videoResp, err := client.Get(mp4URL)
			if err != nil {
				segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed download video segment: %w", err)}
				return
			}
			defer videoResp.Body.Close()

			if videoResp.StatusCode != 200 {
				videoBody, _ := io.ReadAll(videoResp.Body)
				segmentCh <- segmentResult{index: index, err: fmt.Errorf("download video segment status %d: %s", videoResp.StatusCode, string(videoBody))}
				return
			}

			videoBuffer, err := io.ReadAll(videoResp.Body)
			if err != nil {
				segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed reading download buffer: %w", err)}
				return
			}

			// Base64 encode the video buffer to upload to Cloudinary directly in-memory
			encoded := base64.StdEncoding.EncodeToString(videoBuffer)
			dataURI := "data:video/mp4;base64," + encoded

			// Upload MP4 to video namespace
			_, err = uploadToCloudinary(client, cloudName, apiKey, apiSecret, dataURI, vidPid, tag, "video")
			if err != nil {
				segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed video upload to cloudinary: %w", err)}
				return
			}

			segmentCh <- segmentResult{
				index:  index,
				imgPid: imgPid,
				vidPid: vidPid,
				err:    nil,
			}
		}(i, imgInput)
	}

	results := make([]segmentResult, len(images))
	for range images {
		res := <-segmentCh
		results[res.index] = res
	}

	var imgPublicIDs []string
	var vidPublicIDs []string
	var firstErr error

	for _, res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
		}
		if res.imgPid != "" {
			imgPublicIDs = append(imgPublicIDs, res.imgPid)
		}
		if res.vidPid != "" {
			vidPublicIDs = append(vidPublicIDs, res.vidPid)
		}
	}

	if firstErr != nil {
		if len(imgPublicIDs) > 0 {
			_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		}
		if len(vidPublicIDs) > 0 {
			_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
		}
		return nil, fmt.Errorf("failed to process segments: %w", firstErr)
	}

	// Splicing
	var orderedVidPids []string
	for _, res := range results {
		orderedVidPids = append(orderedVidPids, res.vidPid)
	}

	var spliceURLBuilder strings.Builder
	spliceURLBuilder.WriteString(fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/c_pad,w_500,h_500,b_rgb:121212/", cloudName))
	for i := 1; i < len(orderedVidPids); i++ {
		spliceURLBuilder.WriteString(fmt.Sprintf("fl_splice:transition_(name_slideright;du_0.5),l_video:%s/c_pad,w_500,h_500,b_rgb:121212/fl_layer_apply/", orderedVidPids[i]))
	}
	spliceURLBuilder.WriteString(fmt.Sprintf("%s.mp4", orderedVidPids[0]))
	spliceURL := spliceURLBuilder.String()

	var videoBuffer []byte
	var downloadErr error

	for attempt := 0; attempt < 5; attempt++ {
		fmt.Printf("[Cloudinary] Fetching final video (attempt %d/5)... URL: %s\n", attempt+1, spliceURL)
		videoResp, err := client.Get(spliceURL)
		if err != nil {
			downloadErr = err
			time.Sleep(3 * time.Second)
			continue
		}
		defer videoResp.Body.Close()

		if videoResp.StatusCode == 200 {
			videoBuffer, err = io.ReadAll(videoResp.Body)
			if err == nil {
				downloadErr = nil
				break
			}
			downloadErr = err
		} else {
			bodyBytes, _ := io.ReadAll(videoResp.Body)
			downloadErr = fmt.Errorf("status %d: %s", videoResp.StatusCode, string(bodyBytes))
		}
		time.Sleep(3 * time.Second)
	}

	if downloadErr != nil || len(videoBuffer) == 0 {
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "image", imgPublicIDs)
		_ = deleteFromCloudinary(client, cloudName, apiKey, apiSecret, "video", vidPublicIDs)
		if downloadErr != nil {
			return nil, fmt.Errorf("failed to download spliced video after retries: %w", downloadErr)
		}
		return nil, fmt.Errorf("failed to download spliced video: empty buffer")
	}

	// Clean up resources in the background
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

	for i, urlInput := range req.Images {
		go func(index int, downloadUrl string) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", index))
			err := downloadFile(client, downloadUrl, filePath)
			dlCh <- downloadResult{
				index:    index,
				filePath: filePath,
				err:      err,
			}
		}(i, urlInput.URL)
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
