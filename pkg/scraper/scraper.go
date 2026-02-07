package scraper

import (
        "encoding/json"
        "fmt"
        "net/http"
        "net/url"
        "os"
        "regexp"
        "strings"
        "sync"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/go-rod/rod"
        "github.com/go-rod/rod/lib/launcher"
        "github.com/gocolly/colly/v2"
)

// =============================================================================
// GLOBAL BROWSER (SHARED ACROSS ALL REQUESTS - SAVES RAM)
// =============================================================================

var (
        browser      *rod.Browser
        browserOnce  sync.Once
        pageLimiter  = make(chan struct{}, 2) // Max 2 concurrent pages
        browserMutex sync.Mutex
)

func InitBrowser() {
        browserOnce.Do(func() {
                fmt.Println("🚀 Starting shared browser instance...")

                // Check for system-installed Chromium first
                chromiumPath := os.Getenv("CHROMIUM_PATH")
                if chromiumPath == "" {
                        chromiumPath = os.Getenv("CHROME_PATH")
                }
                if chromiumPath == "" {
                        chromiumPath = "/usr/bin/chromium-browser"
                }

                l := launcher.New().
                        Bin(chromiumPath).  // Use system Chromium
                        Headless(true).
                        NoSandbox(true).    // Required for Docker
                        Devtools(false).
                        Set("disable-gpu"). // Better for headless
                        Set("disable-dev-shm-usage"). // Prevent shared memory issues
                        Set("disable-setuid-sandbox").
                        Set("no-first-run").
                        Set("no-default-browser-check")

                controlURL, err := l.Launch()
                if err != nil {
                        panic(fmt.Sprintf("Failed to launch browser: %v", err))
                }

                browser = rod.New().
                        ControlURL(controlURL).
                        MustConnect()

                fmt.Println("✅ Browser ready!")
        })
}

func withPage(fn func(*rod.Page) error) error {
        // Ensure browser is initialized
        InitBrowser()

        // Limit concurrent pages (prevents RAM overflow)
        pageLimiter <- struct{}{}
        defer func() { <-pageLimiter }()

        page := browser.MustPage()
        defer page.MustClose()

        return fn(page)
}

// =============================================================================
// VS BATTLES SCRAPER (WITH ROD - PUPPETEER-LIKE)
// =============================================================================

type VSBCharacter struct {
        Name string `json:"name"`
        URL  string `json:"url"`
}

type VSBSearchResponse struct {
        Characters []VSBCharacter `json:"characters"`
}

type VSBDetailResponse struct {
        Name          string            `json:"name"`
        ImageURL      string            `json:"imageURL"`
        ImageWidth    int               `json:"imageWidth"`
        ImageHeight   int               `json:"imageHeight"`
        Summary       string            `json:"summary"`
        Tier          string            `json:"tier"`
        AttackPotency string            `json:"attackPotency"`
        Speed         string            `json:"speed"`
        Durability    string            `json:"durability"`
        Stamina       string            `json:"stamina"`
        Range         string            `json:"range"`
        Stats         map[string]string `json:"stats"`
}

// SearchVSBattles searches VS Battles Wiki using headless Chrome
func SearchVSBattles(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        var searchLinks []VSBCharacter

        err := withPage(func(page *rod.Page) error {
                searchURL := fmt.Sprintf("https://vsbattles.fandom.com/wiki/Special:Search?query=%s", url.QueryEscape(query))

                page.Timeout(60 * time.Second).MustNavigate(searchURL).MustWaitLoad()
                time.Sleep(2 * time.Second)

                htmlContent := page.MustHTML()
                searchLinks = parseVSBSearchResults(htmlContent)

                return nil
        })

        if err != nil {
                c.JSON(500, gin.H{"error": fmt.Sprintf("Browser automation failed: %s", err.Error())})
                return
        }

        if len(searchLinks) == 0 {
                c.JSON(404, gin.H{"error": "Character not found"})
                return
        }

        c.JSON(200, VSBSearchResponse{Characters: searchLinks})
}

// parseVSBSearchResults extracts character links from search HTML
func parseVSBSearchResults(html string) []VSBCharacter {
        var results []VSBCharacter

        re := regexp.MustCompile(`<a[^>]+href="(https://vsbattles\.fandom\.com/wiki/[^"]+)"[^>]*>([^<]+)</a>`)
        matches := re.FindAllStringSubmatch(html, -1)

        seen := make(map[string]bool)

        for _, match := range matches {
                if len(match) < 3 {
                        continue
                }

                url := match[1]
                name := strings.TrimSpace(match[2])

                if strings.Contains(url, "Special:") || strings.Contains(url, "Category:") {
                        continue
                }

                if seen[url] {
                        continue
                }
                seen[url] = true

                name = strings.ReplaceAll(name, "_", " ")

                results = append(results, VSBCharacter{
                        Name: name,
                        URL:  url,
                })
        }

        return results
}

// GetVSBattlesDetail scrapes detailed character info using headless Chrome
func GetVSBattlesDetail(c *gin.Context) {
        pageURL := c.Query("url")
        if pageURL == "" {
                c.JSON(400, gin.H{"error": "URL required"})
                return
        }

        var detail VSBDetailResponse
        detail.Stats = make(map[string]string)

        err := withPage(func(page *rod.Page) error {
                page.Timeout(60 * time.Second).MustNavigate(pageURL).MustWaitLoad()
                time.Sleep(2 * time.Second)

                // Try to get high-quality image with retries
                maxRetries := 10
                minImageSize := 100

                for i := 0; i < maxRetries; i++ {
                        img, err := page.Element("img.pi-image-thumbnail")
                        if err == nil {
                                dataSrc, _ := img.Attribute("data-src")
                                src, _ := img.Attribute("src")
                                widthStr, _ := img.Attribute("data-image-width")
                                heightStr, _ := img.Attribute("data-image-height")

                                if widthStr != nil {
                                        fmt.Sscanf(*widthStr, "%d", &detail.ImageWidth)
                                }
                                if heightStr != nil {
                                        fmt.Sscanf(*heightStr, "%d", &detail.ImageHeight)
                                }

                                rawURL := ""
                                if dataSrc != nil {
                                        rawURL = *dataSrc
                                } else if src != nil {
                                        rawURL = *src
                                }

                                if rawURL != "" && !strings.HasPrefix(rawURL, "data:") &&
                                        detail.ImageWidth >= minImageSize && detail.ImageHeight >= minImageSize {

                                        // Clean image URL
                                        if strings.Contains(rawURL, "/revision/") {
                                                parts := strings.Split(rawURL, "/revision/")
                                                detail.ImageURL = parts[0]
                                        } else {
                                                detail.ImageURL = rawURL
                                        }
                                        break
                                }
                        }

                        time.Sleep(2 * time.Second)
                }

                // Get HTML for stats extraction
                htmlContent := page.MustHTML()
                extractVSBStats(&detail, htmlContent)

                return nil
        })

        if err != nil {
                c.JSON(500, gin.H{"error": fmt.Sprintf("Browser automation failed: %s", err.Error())})
                return
        }

        // Apply "Peak Logic" - extract highest values
        detail.AttackPotency = extractPeakValue(detail.AttackPotency)
        detail.Speed = extractPeakValue(detail.Speed)
        detail.Durability = extractPeakValue(detail.Durability)
        detail.Stamina = extractPeakValue(detail.Stamina)
        detail.Range = extractPeakValue(detail.Range)
        detail.Tier = extractHighestTier(detail.Tier)

        c.JSON(200, detail)
}

// extractVSBStats extracts character stats from HTML content
func extractVSBStats(detail *VSBDetailResponse, html string) {
        nameRe := regexp.MustCompile(`<h1[^>]*class="page-header__title"[^>]*>([^<]+)</h1>`)
        if matches := nameRe.FindStringSubmatch(html); len(matches) > 1 {
                detail.Name = strings.TrimSpace(matches[1])
        }

        summaryRe := regexp.MustCompile(`(?s)<aside[^>]*>.*?<p>(.*?)</p>`)
        if matches := summaryRe.FindStringSubmatch(html); len(matches) > 1 {
                detail.Summary = stripHTML(matches[1])
        }

        tierRe := regexp.MustCompile(`(?s)<h3[^>]*>\s*<span[^>]*>Tier[^<]*</span>\s*</h3>\s*<div[^>]*>(.*?)</div>`)
        if matches := tierRe.FindStringSubmatch(html); len(matches) > 1 {
                detail.Tier = stripHTML(matches[1])
        }

        statRegex := regexp.MustCompile(`(?s)<h3[^>]*>\s*<span[^>]*>([^<]+)</span>\s*</h3>\s*<div[^>]*>(.*?)</div>`)
        statMatches := statRegex.FindAllStringSubmatch(html, -1)

        for _, match := range statMatches {
                if len(match) < 3 {
                        continue
                }

                key := strings.TrimSpace(match[1])
                value := stripHTML(match[2])
                value = strings.TrimSpace(value)

                switch key {
                case "Attack Potency":
                        detail.AttackPotency = value
                case "Speed":
                        detail.Speed = value
                case "Lifting Strength":
                        detail.Stats["Lifting Strength"] = value
                case "Striking Strength":
                        detail.Stats["Striking Strength"] = value
                case "Durability":
                        detail.Durability = value
                case "Stamina":
                        detail.Stamina = value
                case "Range":
                        detail.Range = value
                case "Intelligence":
                        detail.Stats["Intelligence"] = value
                default:
                        detail.Stats[key] = value
                }
        }
}

// =============================================================================
// PINTEREST SCRAPER (COLLY FIRST - BROWSER FALLBACK)
// =============================================================================

type PinterestResponse struct {
        Images []string `json:"images"`
}

// SearchPinterest searches Pinterest for images
func SearchPinterest(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        maxResults := 10
        if maxStr := c.Query("maxResults"); maxStr != "" {
                fmt.Sscanf(maxStr, "%d", &maxResults)
                if maxResults > 50 {
                        maxResults = 50
                }
        }

        // Try Colly first
        images := tryPinterestColly(query, maxResults)

        // Fallback to browser
        if len(images) == 0 {
                images = tryPinterestBrowser(query, maxResults)
        }

        c.JSON(200, PinterestResponse{Images: images})
}

// tryPinterestColly uses Colly for Pinterest (fast)
func tryPinterestColly(query string, maxResults int) []string {
        collector := colly.NewCollector(
                colly.UserAgent("Mozilla/5.0"),
        )

        var images []string

        collector.OnHTML("img[src*='pinimg.com']", func(e *colly.HTMLElement) {
                src := e.Attr("src")

                if strings.Contains(src, "75x75") || strings.Contains(src, "avatar") {
                        return
                }

                highRes := strings.ReplaceAll(src, "236x", "736x")
                highRes = strings.ReplaceAll(highRes, "474x", "736x")

                if len(images) < maxResults {
                        images = append(images, highRes)
                }
        })

        searchURL := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)
        collector.Visit(searchURL)

        return deduplicateStrings(images)
}

// tryPinterestBrowser uses headless Chrome for Pinterest (fallback)
func tryPinterestBrowser(query string, maxResults int) []string {
        var images []string

        err := withPage(func(page *rod.Page) error {
                searchURL := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)

                page.Timeout(60 * time.Second).MustNavigate(searchURL).MustWaitLoad()
                time.Sleep(3 * time.Second)

                htmlContent := page.MustHTML()

                re := regexp.MustCompile(`https://i\.pinimg\.com/[^"'\s]+`)
                matches := re.FindAllString(htmlContent, -1)

                seen := make(map[string]bool)
                for _, match := range matches {
                        if seen[match] || strings.Contains(match, "75x75") {
                                continue
                        }
                        seen[match] = true

                        highRes := strings.ReplaceAll(match, "236x", "736x")
                        highRes = strings.ReplaceAll(highRes, "474x", "736x")

                        images = append(images, highRes)

                        if len(images) >= maxResults {
                                break
                        }
                }

                return nil
        })

        if err != nil {
                return []string{}
        }

        return images
}

// =============================================================================
// RULE34 SCRAPER (API FIRST - NO BROWSER NEEDED)
// =============================================================================

type Rule34Response struct {
        Images []string `json:"images"`
}

type Rule34Post struct {
        FileURL string `json:"file_url"`
}

// SearchRule34 searches Rule34 using API first, then browser fallback
func SearchRule34(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        count := 10
        if countStr := c.Query("count"); countStr != "" {
                fmt.Sscanf(countStr, "%d", &count)
                if count > 50 {
                        count = 50
                }
        }

        // Try API first (fast, no browser needed)
        images := tryRule34API(query, count)

        // Fallback to scraping
        if len(images) == 0 {
                images = tryRule34WebScrape(query, count)
        }

        c.JSON(200, Rule34Response{Images: images})
}

// tryRule34API attempts to use Rule34 JSON API
func tryRule34API(searchTerm string, count int) []string {
        tag := strings.TrimSpace(searchTerm)
        tag = strings.ReplaceAll(tag, " ", "_")

        apiURL := fmt.Sprintf("https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=200&tags=%s",
                url.QueryEscape(tag))

        client := &http.Client{Timeout: 10 * time.Second}
        req, _ := http.NewRequest("GET", apiURL, nil)
        req.Header.Set("User-Agent", "Mozilla/5.0")

        resp, err := client.Do(req)
        if err != nil {
                return []string{}
        }
        defer resp.Body.Close()

        var posts []Rule34Post
        json.NewDecoder(resp.Body).Decode(&posts)

        var images []string
        for _, post := range posts {
                if post.FileURL != "" {
                        imageURL := post.FileURL
                        if strings.HasPrefix(imageURL, "//") {
                                imageURL = "https:" + imageURL
                        }
                        images = append(images, imageURL)

                        if len(images) >= count {
                                break
                        }
                }
        }

        return images
}

// tryRule34WebScrape uses Colly for Rule34 scraping
func tryRule34WebScrape(searchTerm string, count int) []string {
        tag := strings.ReplaceAll(searchTerm, " ", "_")
        scrapeURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=list&tags=%s", url.QueryEscape(tag))

        collector := colly.NewCollector(
                colly.UserAgent("Mozilla/5.0"),
        )

        var postIDs []string
        collector.OnHTML(".thumb a", func(e *colly.HTMLElement) {
                if len(postIDs) >= count {
                        return
                }

                href := e.Attr("href")
                re := regexp.MustCompile(`id=(\d+)`)
                if matches := re.FindStringSubmatch(href); len(matches) > 1 {
                        postIDs = append(postIDs, matches[1])
                }
        })

        collector.Visit(scrapeURL)

        var images []string
        for _, id := range postIDs {
                imageURL := getRule34Image(id)
                if imageURL != "" {
                        images = append(images, imageURL)
                }
        }

        return images
}

// getRule34Image fetches full image URL for a post
func getRule34Image(postID string) string {
        postURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=view&id=%s", postID)

        collector := colly.NewCollector()
        var imageURL string

        collector.OnHTML("#image", func(e *colly.HTMLElement) {
                imageURL = e.Attr("src")
        })

        collector.Visit(postURL)

        if imageURL != "" && strings.HasPrefix(imageURL, "//") {
                imageURL = "https:" + imageURL
        }

        return imageURL
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func stripHTML(s string) string {
        re := regexp.MustCompile(`<[^>]+>`)
        return re.ReplaceAllString(s, "")
}

func cleanVSBText(s string) string {
        s = strings.TrimSpace(s)
        re := regexp.MustCompile(`\[.*?\]`)
        s = re.ReplaceAllString(s, "")
        re = regexp.MustCompile(`\(.*?\)`)
        s = re.ReplaceAllString(s, "")
        return strings.TrimSpace(s)
}

func extractPeakValue(text string) string {
        if text == "" {
                return "N/A"
        }

        parts := strings.Split(text, "|")
        if len(parts) > 0 {
                peak := strings.TrimSpace(parts[len(parts)-1])
                return cleanVSBText(peak)
        }

        return cleanVSBText(text)
}

func extractHighestTier(text string) string {
        re := regexp.MustCompile(`\b([0-9]+)-([A-Z])\b`)
        matches := re.FindAllString(text, -1)

        if len(matches) > 0 {
                return matches[len(matches)-1]
        }

        return "Unknown"
}

func deduplicateStrings(input []string) []string {
        seen := make(map[string]bool)
        result := []string{}

        for _, str := range input {
                if !seen[str] {
                        seen[str] = true
                        result = append(result, str)
                }
        }

        return result
}