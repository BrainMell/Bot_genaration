package scraper

import (
        "encoding/json"
        "fmt"
        "io"
        "math/rand"
        "net/http"
        "net/url"
        "os"
        "regexp"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "golang.org/x/net/html"
)

// =============================================================================
// CONFIGURATION
// =============================================================================

const (
        klipyBaseURL = "https://api.klipy.com/v2"
)

var (
        klipyAPIKey string
        httpClient  *http.Client
        userAgents  = []string{
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
                "Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
                "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
        }
        searchSources = []string{"reddit", "twitter", "pinterest", "tumblr"}
)

func init() {
        klipyAPIKey = os.Getenv("KLIPY_API_KEY")
        httpClient = &http.Client{Timeout: 30 * time.Second}
        rand.Seed(time.Now().UnixNano())
}

// =============================================================================
// PINTEREST REPLACEMENT - DuckDuckGo Image Search for .j img command
// Strategy: Rotate search sources (reddit, twitter, pinterest, tumblr)
// First half: memes, Second half: reactions
// =============================================================================

type PinterestResponse struct {
        Images []string `json:"images"`
        Count  int      `json:"count"`
        Note   string   `json:"note,omitempty"`
}

func SearchPinterest(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        maxResults := 10
        if max := c.Query("maxResults"); max != "" {
                fmt.Sscanf(max, "%d", &maxResults)
        }

        images := []string{}

        // STRATEGY: Get variety by rotating sources
        // First half: memes from different sources
        // Second half: reactions from different sources
        halfResults := maxResults / 2

        // Part 1: Search for character memes across sources
        for i := 0; i < 2 && len(images) < halfResults; i++ {
                source := searchSources[rand.Intn(len(searchSources))]
                searchQuery := fmt.Sprintf("%s memes %s", query, source)
                
                urls := searchDuckDuckGoImages(searchQuery, 5)
                images = append(images, urls...)
        }

        // Part 2: Search for reactions across sources
        for i := 0; i < 2 && len(images) < maxResults; i++ {
                source := searchSources[rand.Intn(len(searchSources))]
                searchQuery := fmt.Sprintf("%s reactions %s", query, source)
                
                urls := searchDuckDuckGoImages(searchQuery, 5)
                images = append(images, urls...)
        }

        // Deduplicate
        images = deduplicateStrings(images)
        
        // Limit to maxResults
        if len(images) > maxResults {
                images = images[:maxResults]
        }

        c.JSON(200, PinterestResponse{
                Images: images,
                Count:  len(images),
                Note:   "DuckDuckGo image search - rotating sources (reddit, twitter, pinterest, tumblr)",
        })
}

func searchDuckDuckGoImages(query string, limit int) []string {
        // Use DuckDuckGo to find image URLs
        searchURL := "https://html.duckduckgo.com/html/"
        
        formData := url.Values{}
        formData.Set("q", query)
        formData.Set("iax", "images")
        formData.Set("ia", "images")

        req, err := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
        if err != nil {
                return []string{}
        }

        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                return []string{}
        }
        defer resp.Body.Close()

        // Parse HTML for image links
        urls := extractImageURLsFromHTML(resp.Body, limit)
        return urls
}

func extractImageURLsFromHTML(body io.Reader, limit int) []string {
        tokenizer := html.NewTokenizer(body)
        var urls []string
        seen := make(map[string]bool)

        for len(urls) < limit {
                tt := tokenizer.Next()
                if tt == html.ErrorToken {
                        break
                }

                token := tokenizer.Token()
                if token.Type == html.StartTagToken {
                        if token.Data == "img" {
                                for _, attr := range token.Attr {
                                        if attr.Key == "src" && !seen[attr.Val] {
                                                // Filter out tiny/tracking images
                                                if !strings.Contains(attr.Val, "data:image") &&
                                                   !strings.Contains(attr.Val, "1x1") &&
                                                   len(attr.Val) > 20 {
                                                        seen[attr.Val] = true
                                                        urls = append(urls, attr.Val)
                                                        if len(urls) >= limit {
                                                                return urls
                                                        }
                                                }
                                        }
                                }
                        } else if token.Data == "a" {
                                // Also check for image links in anchor tags
                                for _, attr := range token.Attr {
                                        if attr.Key == "href" {
                                                cleaned := cleanDDGLink(attr.Val)
                                                if isImageURL(cleaned) && !seen[cleaned] {
                                                        seen[cleaned] = true
                                                        urls = append(urls, cleaned)
                                                        if len(urls) >= limit {
                                                                return urls
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        return urls
}

func isImageURL(url string) bool {
        lower := strings.ToLower(url)
        return strings.HasSuffix(lower, ".jpg") ||
               strings.HasSuffix(lower, ".jpeg") ||
               strings.HasSuffix(lower, ".png") ||
               strings.HasSuffix(lower, ".gif") ||
               strings.HasSuffix(lower, ".webp")
}

// =============================================================================
// KLIPY GIF/STICKER SEARCH - For .j sticker command
// =============================================================================

type KlipyResponse struct {
        Results []struct {
                ID    string `json:"id"`
                Title string `json:"title"`
                Files struct {
                        Gif struct {
                                URL string `json:"url"`
                        } `json:"gif"`
                        Mp4 struct {
                                URL string `json:"url"`
                        } `json:"mp4"`
                        TinyGif struct {
                                URL string `json:"url"`
                        } `json:"tinygif"`
                } `json:"files"`
                Media []struct {
                        Gif struct {
                                URL string `json:"url"`
                        } `json:"gif"`
                } `json:"media"`
        } `json:"results"`
}

type StickerResponse struct {
        Stickers []string `json:"stickers"`
        Count    int      `json:"count"`
        Note     string   `json:"note,omitempty"`
}

// NEW: Separate endpoint for sticker search using Klipy
func SearchStickers(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        maxResults := 10
        if max := c.Query("maxResults"); max != "" {
                fmt.Sscanf(max, "%d", &maxResults)
        }

        if klipyAPIKey == "" {
                c.JSON(500, gin.H{"error": "KLIPY_API_KEY not set"})
                return
        }

        stickers := searchKlipy(query, maxResults)

        c.JSON(200, StickerResponse{
                Stickers: stickers,
                Count:    len(stickers),
                Note:     "Klipy GIF API - 100% free forever",
        })
}

func searchKlipy(query string, limit int) []string {
        if limit <= 0 {
                limit = 10
        }

        // Klipy search endpoint
        apiURL := fmt.Sprintf("%s/search?key=%s&q=%s&limit=%d&media_filter=gif,tinygif",
                klipyBaseURL, klipyAPIKey, url.QueryEscape(query), limit)

        req, err := http.NewRequest("GET", apiURL, nil)
        if err != nil {
                return []string{}
        }

        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                return []string{}
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
                return []string{}
        }

        var klipyResp KlipyResponse
        body, _ := io.ReadAll(resp.Body)
        if err := json.Unmarshal(body, &klipyResp); err != nil {
                return []string{}
        }

        gifs := []string{}
        for _, result := range klipyResp.Results {
                gifURL := ""
                if result.Files.Gif.URL != "" {
                        gifURL = result.Files.Gif.URL
                } else if len(result.Media) > 0 && result.Media[0].Gif.URL != "" {
                        gifURL = result.Media[0].Gif.URL
                } else if result.Files.TinyGif.URL != "" {
                        gifURL = result.Files.TinyGif.URL
                }

                if gifURL != "" {
                        gifs = append(gifs, gifURL)
                }
        }

        return gifs
}

// =============================================================================
// VS BATTLES - Wikipedia Image + Basic Text Scraping
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
        WikiImageURL  string            `json:"wikiImageURL"`
        Summary       string            `json:"summary"`
        Tier          string            `json:"tier"`
        AttackPotency string            `json:"attackPotency"`
        Speed         string            `json:"speed"`
        Durability    string            `json:"durability"`
        Stamina       string            `json:"stamina"`
        Range         string            `json:"range"`
        Stats         map[string]string `json:"stats"`
}

func SearchVSBattles(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        // Use DuckDuckGo to find VS Battles pages
        searchURL := "https://html.duckduckgo.com/html/"
        formData := url.Values{}
        formData.Set("q", query+" vsbattles")

        req, _ := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                c.JSON(500, gin.H{"error": "Search failed"})
                return
        }
        defer resp.Body.Close()

        characters := parseVSBSearchResults(resp.Body)

        if len(characters) == 0 {
                c.JSON(404, gin.H{"error": "No characters found"})
                return
        }

        c.JSON(200, VSBSearchResponse{Characters: characters})
}

func parseVSBSearchResults(body io.Reader) []VSBCharacter {
        tokenizer := html.NewTokenizer(body)
        var characters []VSBCharacter
        seen := make(map[string]bool)

        for {
                tt := tokenizer.Next()
                if tt == html.ErrorToken {
                        break
                }

                token := tokenizer.Token()
                if token.Type == html.StartTagToken && token.Data == "a" {
                        for _, attr := range token.Attr {
                                if attr.Key == "href" {
                                        link := cleanDDGLink(attr.Val)
                                        if strings.Contains(link, "vsbattles.fandom.com/wiki/") &&
                                           !strings.Contains(link, "Special:") &&
                                           !strings.Contains(link, "Category:") &&
                                           !seen[link] {
                                                seen[link] = true
                                                
                                                parts := strings.Split(link, "/wiki/")
                                                name := "Unknown"
                                                if len(parts) > 1 {
                                                        name = strings.ReplaceAll(parts[1], "_", " ")
                                                }

                                                characters = append(characters, VSBCharacter{
                                                        Name: name,
                                                        URL:  link,
                                                })

                                                if len(characters) >= 10 {
                                                        return characters
                                                }
                                        }
                                }
                        }
                }
        }

        return characters
}

func GetVSBattlesDetail(c *gin.Context) {
        pageURL := c.Query("url")
        if pageURL == "" {
                c.JSON(400, gin.H{"error": "URL required"})
                return
        }

        // Extract character name from VSB URL
        parts := strings.Split(pageURL, "/wiki/")
        characterName := "Unknown"
        if len(parts) > 1 {
                characterName = strings.ReplaceAll(parts[1], "_", " ")
                // Remove any (Verse) or other suffixes for Wikipedia search
                characterName = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(characterName, "")
                characterName = strings.TrimSpace(characterName)
        }

        detail := VSBDetailResponse{
                Name:  characterName,
                Stats: make(map[string]string),
        }

        // STEP 1: Get image from Wikipedia (NOT from VSB)
        wikiImage := getWikipediaImage(characterName)
        if wikiImage != "" {
                detail.WikiImageURL = wikiImage
        }

        // STEP 2: Scrape VSB page for text stats ONLY
        req, _ := http.NewRequest("GET", pageURL, nil)
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                c.JSON(500, gin.H{"error": "Failed to fetch VSB page"})
                return
        }
        defer resp.Body.Close()

        body, _ := io.ReadAll(resp.Body)
        htmlContent := string(body)

        // Extract stats using basic text scraping
        detail = extractVSBStatsFromText(detail, htmlContent)

        c.JSON(200, detail)
}

func getWikipediaImage(characterName string) string {
        // Wikipedia API to get character image
        apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&titles=%s&prop=pageimages&format=json&pithumbsize=500",
                url.QueryEscape(characterName))

        resp, err := httpClient.Get(apiURL)
        if err != nil {
                return ""
        }
        defer resp.Body.Close()

        var result struct {
                Query struct {
                        Pages map[string]struct {
                                Thumbnail struct {
                                        Source string `json:"source"`
                                } `json:"thumbnail"`
                        } `json:"pages"`
                } `json:"query"`
        }

        if json.NewDecoder(resp.Body).Decode(&result) == nil {
                for _, page := range result.Query.Pages {
                        if page.Thumbnail.Source != "" {
                                return page.Thumbnail.Source
                        }
                }
        }

        return ""
}

func extractVSBStatsFromText(detail VSBDetailResponse, htmlContent string) VSBDetailResponse {
        // Strip HTML for text-only extraction
        text := stripHTML(htmlContent)

        // Extract stats using regex
        tierRe := regexp.MustCompile(`(?i)Tier[:\s]*([0-9A-Z\-]+)`)
        if matches := tierRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Tier = strings.TrimSpace(matches[1])
        }

        apRe := regexp.MustCompile(`(?i)Attack Potency[:\s]*([^\.]+)`)
        if matches := apRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.AttackPotency = strings.TrimSpace(matches[1])
        }

        speedRe := regexp.MustCompile(`(?i)Speed[:\s]*([^\.]+)`)
        if matches := speedRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Speed = strings.TrimSpace(matches[1])
        }

        durRe := regexp.MustCompile(`(?i)Durability[:\s]*([^\.]+)`)
        if matches := durRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Durability = strings.TrimSpace(matches[1])
        }

        stamRe := regexp.MustCompile(`(?i)Stamina[:\s]*([^\.]+)`)
        if matches := stamRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Stamina = strings.TrimSpace(matches[1])
        }

        rangeRe := regexp.MustCompile(`(?i)Range[:\s]*([^\.]+)`)
        if matches := rangeRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Range = strings.TrimSpace(matches[1])
        }

        summaryRe := regexp.MustCompile(`(?s)([A-Z][^\.]+\.[^\.]+\.)`)
        if matches := summaryRe.FindStringSubmatch(text); len(matches) > 1 {
                detail.Summary = strings.TrimSpace(matches[1])
        }

        return detail
}

// =============================================================================
// RULE34 - Direct API
// =============================================================================

type Rule34Response struct {
        Images []string `json:"images"`
        Count  int      `json:"count"`
}

type Rule34Post struct {
        FileURL string `json:"file_url"`
}

func SearchRule34(c *gin.Context) {
        query := c.Query("query")
        if query == "" {
                c.JSON(400, gin.H{"error": "Query required"})
                return
        }

        maxResults := 10
        if max := c.Query("maxResults"); max != "" {
                fmt.Sscanf(max, "%d", &maxResults)
        }

        tag := url.QueryEscape(query)
        apiURL := fmt.Sprintf("https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=%d&tags=%s",
                maxResults*2, tag)

        req, _ := http.NewRequest("GET", apiURL, nil)
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                c.JSON(200, Rule34Response{Images: []string{}, Count: 0})
                return
        }
        defer resp.Body.Close()

        var posts []Rule34Post
        json.NewDecoder(resp.Body).Decode(&posts)

        images := []string{}
        for _, post := range posts {
                if post.FileURL == "" {
                        continue
                }

                imgURL := post.FileURL
                if len(imgURL) > 1 && imgURL[0:2] == "//" {
                        imgURL = "https:" + imgURL
                }

                if strings.HasSuffix(imgURL, ".jpg") || strings.HasSuffix(imgURL, ".jpeg") ||
                   strings.HasSuffix(imgURL, ".png") || strings.HasSuffix(imgURL, ".gif") {
                        images = append(images, imgURL)
                }

                if len(images) >= maxResults {
                        break
                }
        }

        c.JSON(200, Rule34Response{
                Images: images,
                Count:  len(images),
        })
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func getRandomUA() string {
        return userAgents[rand.Intn(len(userAgents))]
}

func cleanDDGLink(link string) string {
        if strings.Contains(link, "uddg=") {
                parts := strings.Split(link, "uddg=")
                if len(parts) > 1 {
                        actual, _ := url.QueryUnescape(parts[1])
                        return actual
                }
        }
        return link
}

func stripHTML(s string) string {
        re := regexp.MustCompile(`<[^>]+>`)
        cleaned := re.ReplaceAllString(s, " ")
        cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
        return strings.TrimSpace(cleaned)
}

func deduplicateStrings(input []string) []string {
        seen := make(map[string]bool)
        result := []string{}
        for _, str := range input {
                if !seen[str] && str != "" {
                        seen[str] = true
                        result = append(result, str)
                }
        }
        return result
}
