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
        "image/jpeg"
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

        _ "golang.org/x/image/webp"

        "image-service/pkg/utils"

        "github.com/disintegration/imaging"
        "github.com/fogleman/gg"
        "github.com/gin-gonic/gin"
)

type CardImageInput struct {
        URL      string `json:"url" binding:"required"`
        Animated bool   `json:"animated"`
        Name     string `json:"name"`
        Tier     string `json:"tier"`
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

func downloadFileToMemory(client *http.Client, urlStr string) ([]byte, string, error) {
        req, err := http.NewRequest("GET", urlStr, nil)
        if err != nil {
                return nil, "", err
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
        req.Header.Set("Referer", "https://www.pinterest.com/")
        req.Header.Set("Accept", "image/webp,image/png,image/jpeg,image/gif")
        req.Header.Set("Accept-Language", "en-US,en;q=0.9")

        resp, err := client.Do(req)
        if err != nil {
                return nil, "", err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                return nil, "", fmt.Errorf("bad status: %s", resp.Status)
        }

        contentType := resp.Header.Get("Content-Type")
        if contentType == "" {
                contentType = "image/png"
        }

        data, err := io.ReadAll(resp.Body)
        if err != nil {
                return nil, "", err
        }

        return data, contentType, nil
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

                        // Download the image to memory first
                        imgBytes, contentType, err := downloadFileToMemory(client, input.URL)
                        if err != nil {
                                segmentCh <- segmentResult{index: index, err: fmt.Errorf("failed image download to memory: %w", err)}
                                return
                        }

                        // Base64 encode the image to upload directly to Cloudinary
                        encodedImg := base64.StdEncoding.EncodeToString(imgBytes)
                        imgDataURI := fmt.Sprintf("data:%s;base64,%s", contentType, encodedImg)

                        // Upload original image to image namespace
                        imageTag := tag
                        if !input.Animated && !isAnimated(input.URL) {
                                imageTag = fmt.Sprintf("%s_static_%03d", tag, index)
                        }

                        _, err = uploadToCloudinary(client, cloudName, apiKey, apiSecret, imgDataURI, imgPid, imageTag, "image")
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
        Name string
        Tier string
}

// ──────────────────────────────────────────────────────────────────────────
//  OPTIMIZED GRID RENDERER (500MB / 0.1 CPU friendly)
// ──────────────────────────────────────────────────────────────────────────
//  Designed for Render free tier. Key optimizations vs old grid:
//    1. Smaller cards (160×240 vs 240×360) = 44% less memory per card
//    2. JPEG output (quality 85) instead of PNG = 5-10x smaller file
//    3. NearestNeighbor resize instead of Lanczos = 10x faster
//    4. Card name + tier badge overlays for better UX
//    5. Placeholder cards for failed downloads (no gaps in grid)
//    6. Sequential downloads (not parallel) to avoid connection exhaustion
//
//  Memory budget for 12 cards:
//    Canvas: 560×980 × 4 bytes = ~2.2MB
//    Per card decoded: 160×240 × 4 = 154KB → freed after compositing
//    Peak: ~3MB (vs ~12MB old grid)
//  CPU: 12 × NearestNeighbor resize ≈ 0.5s on 0.1 CPU (vs 5s+ with Lanczos)

func GenerateCardGridImage(inputs []CardInput, title string) ([]byte, error) {
        n := len(inputs)
        if n == 0 {
                return nil, fmt.Errorf("no input cards")
        }

        // 3 columns, max 12 cards (4 rows)
        maxCards := 12
        if n > maxCards {
                n = maxCards
        }
        cols := 3
        if n < 3 {
                cols = n
        }
        rows := (n + cols - 1) / cols

        // Smaller card dimensions for memory efficiency
        cardW := 160
        cardH := 240
        spacing := 12
        padX := 20
        padY := 20
        headerH := 0
        if title != "" {
                headerH = 60
        }

        canvasW := cols*cardW + (cols-1)*spacing + padX*2
        canvasH := rows*cardH + (rows-1)*spacing + padY*2 + headerH

        dc := gg.NewContext(canvasW, canvasH)

        // Premium dark background gradient
        grad := gg.NewLinearGradient(0, 0, 0, float64(canvasH))
        grad.AddColorStop(0, utils.ParseHexColor("#0f1015"))
        grad.AddColorStop(1, utils.ParseHexColor("#060709"))
        dc.SetFillStyle(grad)
        dc.DrawRectangle(0, 0, float64(canvasW), float64(canvasH))
        dc.Fill()

        // Title header
        if title != "" {
                fontBoldPath := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")
                if err := dc.LoadFontFace(fontBoldPath, 20); err == nil {
                        dc.SetHexColor("#000000")
                        dc.DrawStringAnchored(title, float64(canvasW)/2+1, 32+1, 0.5, 0.5)
                        dc.SetHexColor("#e5c158")
                        dc.DrawStringAnchored(title, float64(canvasW)/2, 32, 0.5, 0.5)
                }
                dc.SetHexColor("#1e2330")
                dc.SetLineWidth(1.5)
                dc.DrawLine(20, 55, float64(canvasW-20), 55)
                dc.Stroke()
        }

        // Tier color map for badges
        tierColors := map[string]string{
                "1": "#8a8a9a", "2": "#4a9eff", "3": "#a855f7",
                "4": "#f59e0b", "5": "#ef4444", "6": "#e5c158",
                "S": "#ff00ff", "E": "#00ff88",
        }
        tierLabels := map[string]string{
                "1": "T1", "2": "T2", "3": "T3", "4": "T4",
                "5": "T5", "6": "T6", "S": "TS", "E": "EV",
        }

        // Load fonts for card names (smaller size for smaller cards)
        fontMediumPath := utils.GetAssetPath("rpgasset", "ui", "Inter-Medium.ttf")
        fontBoldPath := utils.GetAssetPath("rpgasset", "ui", "Inter-Bold.ttf")

        // Render cards
        for i := 0; i < n; i++ {
                input := inputs[i]
                col := i % cols
                row := i / cols
                x := float64(padX + col*(cardW+spacing))
                y := float64(padY + headerH + row*(cardH+spacing))

                // Try to load and draw the card image
                imgDrawn := false
                if input.Path != "" {
                        file, err := os.Open(input.Path)
                        if err == nil {
                                img, _, err := image.Decode(file)
                                file.Close()
                                if err == nil {
                                        // NearestNeighbor is 10x faster than Lanczos on 0.1 CPU
                                        // Fill crops to 2:3 aspect ratio without stretching
                                        resized := imaging.Fill(img, cardW, cardH, imaging.Center, imaging.NearestNeighbor)

                                        dc.Push()
                                        dc.DrawRoundedRectangle(x, y, float64(cardW), float64(cardH), 8)
                                        dc.Clip()
                                        dc.DrawImage(resized, int(x), int(y))
                                        dc.Pop()
                                        imgDrawn = true
                                }
                        }
                }

                // Placeholder for failed downloads — no gaps in grid
                if !imgDrawn {
                        dc.Push()
                        dc.SetHexColor("#1a1d28")
                        dc.DrawRoundedRectangle(x, y, float64(cardW), float64(cardH), 8)
                        dc.Fill()
                        dc.Pop()
                        if err := dc.LoadFontFace(fontMediumPath, 14); err == nil {
                                dc.SetHexColor("#4a4f5d")
                                dc.DrawStringAnchored("?", x+float64(cardW)/2, y+float64(cardH)/2, 0.5, 0.5)
                        }
                }

                // Card border — colored by tier
                tierColor := tierColors[input.Tier]
                if tierColor == "" {
                        tierColor = "#2a2f3d"
                }
                dc.DrawRoundedRectangle(x, y, float64(cardW), float64(cardH), 8)
                dc.SetHexColor(tierColor)
                dc.SetLineWidth(2)
                dc.Stroke()

                // Tier badge (top-left corner)
                tierLabel := tierLabels[input.Tier]
                if tierLabel != "" {
                        badgeW := 28.0
                        badgeH := 16.0
                        badgeX := x + 4
                        badgeY := y + 4
                        dc.DrawRoundedRectangle(badgeX, badgeY, badgeW, badgeH, 4)
                        dc.SetHexColor(tierColor)
                        dc.Fill()
                        if err := dc.LoadFontFace(fontBoldPath, 10); err == nil {
                                dc.SetHexColor("#ffffff")
                                dc.DrawStringAnchored(tierLabel, badgeX+badgeW/2, badgeY+badgeH/2, 0.5, 0.5)
                        }
                }

                // Card name overlay (bottom strip)
                if input.Name != "" && input.Name != "Unknown" {
                        nameY := y + float64(cardH) - 28
                        // Semi-transparent black strip
                        dc.Push()
                        dc.DrawRoundedRectangle(x, nameY, float64(cardW), 28, 0)
                        dc.SetRGBA(0, 0, 0, 0.7)
                        dc.Fill()
                        dc.Pop()
                        // Draw name (truncate if too long)
                        if err := dc.LoadFontFace(fontMediumPath, 11); err == nil {
                                dc.SetHexColor("#ffffff")
                                name := input.Name
                                // Truncate to fit — MeasureString returns (w, h)
                                maxW := float64(cardW - 8)
                                w, _ := dc.MeasureString(name)
                                for w > maxW && len(name) > 3 {
                                        name = name[:len(name)-1]
                                        w, _ = dc.MeasureString(name)
                                }
                                if len(name) < len(input.Name) {
                                        name = name[:len(name)-1] + "…"
                                }
                                dc.DrawStringAnchored(name, x+float64(cardW)/2, nameY+14, 0.5, 0.5)
                        }
                }
        }

        // Encode as JPEG (quality 85) — 5-10x smaller than PNG, fine for photos
        var buf bytes.Buffer
        err := jpeg.Encode(&buf, dc.Image(), &jpeg.Options{Quality: 85})
        if err != nil {
                // Fallback to PNG if JPEG fails
                pngBuf := new(bytes.Buffer)
                png.Encode(pngBuf, dc.Image())
                return pngBuf.Bytes(), nil
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

        // Max 12 cards (3×4 grid) — keeps memory under 3MB on 500MB servers
        maxImages := 12
        if len(req.Images) > maxImages {
                req.Images = req.Images[:maxImages]
        }

        // 💡 SKIP CLOUDINARY ENTIRELY on low-memory servers.
        // The Cloudinary path spawns N parallel goroutines, each doing 2
        // downloads + 2 base64 encodings + 2 uploads. On 500MB/0.1CPU this
        // always fails (connection exhaustion + timeouts), wasting 30-60s
        // before falling back. Go straight to the optimized grid renderer.
        // Cloudinary can be re-enabled by setting ENABLE_CLOUDINARY=true.
        if os.Getenv("ENABLE_CLOUDINARY") == "true" {
                cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
                if cloudName == "" {
                        cloudName = "dwgct8qng"
                }
                apiKey := os.Getenv("CLOUDINARY_API_KEY")
                if apiKey == "" {
                        apiKey = "551174662862282"
                }
                // No hardcoded fallback for the API secret - it must come from the
                // environment. A literal here leaks the credential into the public
                // repo and into every compiled binary.
                apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

                if cloudName != "" && apiKey != "" && apiSecret != "" {
                        fmt.Printf("[Cloudinary] Splicing %d images...\n", len(req.Images))
                        vidData, err := GenerateCloudinarySlideshow(req.Images, cloudName, apiKey, apiSecret)
                        if err == nil {
                                fmt.Printf("[Cloudinary] Success! Size: %d bytes\n", len(vidData))
                                c.Data(200, "video/mp4", vidData)
                                return
                        }
                        fmt.Printf("[Cloudinary] Failed: %v. Falling back to grid...\n", err)
                }
        }

        // ── Optimized grid renderer (500MB / 0.1 CPU friendly) ──
        tempDir, err := os.MkdirTemp("", "cardgrid_*")
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to create temp directory"})
                return
        }
        defer os.RemoveAll(tempDir)

        // SEQUENTIAL downloads — parallel downloads exhaust the connection
        // pool on 0.1 CPU servers, causing most downloads to timeout.
        // Sequential with a short timeout per card is faster overall
        // because no retries are needed.
        client := &http.Client{Timeout: 30 * time.Second}
        localInputs := make([]CardInput, len(req.Images))

        for i, urlInput := range req.Images {
                filePath := filepath.Join(tempDir, fmt.Sprintf("card_%d", i))
                err := downloadFile(client, urlInput.URL, filePath)
                if err != nil {
                        fmt.Printf("[Cards] Download failed [%d]: %v\n", i, err)
                        // Don't skip — use empty path so placeholder renders
                        localInputs[i] = CardInput{
                                Path: "",
                                Name: urlInput.Name,
                                Tier: urlInput.Tier,
                        }
                        continue
                }
                localInputs[i] = CardInput{
                        Path: filePath,
                        Name: urlInput.Name,
                        Tier: urlInput.Tier,
                }
        }

        fmt.Printf("[Cards] Rendering %d cards in optimized grid...\n", len(localInputs))
        gridData, err := GenerateCardGridImage(localInputs, req.Title)
        if err != nil {
                fmt.Printf("[Cards] Grid generation failed: %v\n", err)
                c.JSON(500, gin.H{"error": "Grid generation failed"})
                return
        }

        // Return as JPEG (the grid renderer outputs JPEG for smaller size)
        c.Data(200, "image/jpeg", gridData)
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
