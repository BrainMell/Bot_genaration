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
                "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1",
        }
        searchSources = []string{"reddit", "twitter", "pinterest", "tumblr", "deviantart", "artstation"}
)

func init() {
        klipyAPIKey = os.Getenv("KLIPY_API_KEY")
        httpClient = &http.Client{Timeout: 30 * time.Second}
        rand.Seed(time.Now().UnixNano())
}

// =============================================================================
// PINTEREST REPLACEMENT - DuckDuckGo Image Search for .j img command
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

        // Increase variety and retry count
        searchTerms := []string{
                fmt.Sprintf("%s memes", query),
                fmt.Sprintf("%s reaction", query),
                fmt.Sprintf("%s wallpaper", query),
                fmt.Sprintf("%s site:%s", query, searchSources[rand.Intn(len(searchSources))]),
        }

        for _, term := range searchTerms {
                if len(images) >= maxResults {
                        break
                }
                urls := searchDuckDuckGoImages(term, 15)
                images = append(images, urls...)
                // Deduplicate during accumulation
                images = deduplicateStrings(images)
        }

        // Limit to maxResults
        if len(images) > maxResults {
                images = images[:maxResults]
        }

        c.JSON(200, PinterestResponse{
                Images: images,
                Count:  len(images),
                Note:   "DuckDuckGo image search - robust multi-term mode",
        })
}

func searchDuckDuckGoImages(query string, limit int) []string {
        // Method 1: Try HTML POST (more results)
        urls := tryDDGPost(query, limit)
        if len(urls) > 0 {
                return urls
        }

        // Method 2: Try standard HTML GET (fallback)
        return tryDDGGet(query, limit)
}

func tryDDGPost(query string, limit int) []string {
        searchURL := "https://html.duckduckgo.com/html/"
        formData := url.Values{}
        formData.Set("q", query)
        
        req, _ := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                return []string{}
        }
        defer resp.Body.Close()

        return extractImageURLsFromHTML(resp.Body, limit)
}

func tryDDGGet(query string, limit int) []string {
        searchURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)
        req, _ := http.NewRequest("GET", searchURL, nil)
        req.Header.Set("User-Agent", getRandomUA())

        resp, err := httpClient.Do(req)
        if err != nil {
                return []string{}
        }
        defer resp.Body.Close()

        return extractImageURLsFromHTML(resp.Body, limit)
}

func extractImageURLsFromHTML(body io.Reader, limit int) []string {
        tokenizer := html.NewTokenizer(body)
        var urls []string
        seen := make(map[string]bool)

        for {
                tt := tokenizer.Next()
                if tt == html.ErrorToken {
                        break
                }

                token := tokenizer.Token()
                if token.Type == html.StartTagToken {
                        // In DDG HTML, thumbnails are in <img> tags
                        if token.Data == "img" {
                                for _, attr := range token.Attr {
                                        if attr.Key == "src" {
                                                val := attr.Val
                                                // Clean potential proxy/encoded URLs
                                                if strings.Contains(val, "uddg=") {
                                                        parts := strings.Split(val, "uddg=")
                                                        if len(parts) > 1 {
                                                                decoded, _ := url.QueryUnescape(parts[1])
                                                                val = decoded
                                                        }
                                                }
                                                
                                                if !seen[val] && isImageURL(val) && !strings.Contains(val, "duckduckgo.com") {
                                                        // Filter out tracking pixels or very small images
                                                        if !strings.Contains(val, "1x1") && len(val) > 20 {
                                                                seen[val] = true
                                                                urls = append(urls, val)
                                                                if len(urls) >= limit {
                                                                        return urls
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                        // High res links are sometimes in <a> tags
                        if token.Data == "a" {
                                for _, attr := range token.Attr {
                                        if attr.Key == "href" {
                                                val := attr.Val
                                                if strings.Contains(val, "uddg=") {
                                                        parts := strings.Split(val, "uddg=")
                                                        if len(parts) > 1 {
                                                                decoded, _ := url.QueryUnescape(parts[1])
                                                                if isImageURL(decoded) && !seen[decoded] {
                                                                        seen[decoded] = true
                                                                        urls = append(urls, decoded)
                                                                        if len(urls) >= limit {
                                                                                return urls
                                                                        }
                                                                }
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
        // Check for common image extensions
        extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
        for _, ext := range extensions {
                if strings.HasSuffix(lower, ext) || strings.Contains(lower, ext+"?") {
                        return true
                }
        }
        return false
}

// =============================================================================
// KLIPY GIF/STICKER SEARCH
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
                klipyAPIKey = os.Getenv("KLIPY_API_KEY")
        }

        if klipyAPIKey == "" {
                c.JSON(500, gin.H{"error": "KLIPY_API_KEY not set in environment"})
                return
        }

        stickers := searchKlipy(query, maxResults)

        c.JSON(200, StickerResponse{
                Stickers: stickers,
                Count:    len(stickers),
                Note:     "Klipy GIF API - stable search",
        })
}

func searchKlipy(query string, limit int) []string {
        if limit <= 0 {
                limit = 10
        }

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
// VS BATTLES
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

        searchURL := "https://html.duckduckgo.com/html/"
        formData := url.Values{}
        formData.Set("q", query+" vsbattles wiki")

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
                                                        name = strings.ReplaceAll(name, "%28", "(")
                                                        name = strings.ReplaceAll(name, "%29", ")")
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

        parts := strings.Split(pageURL, "/wiki/")
        characterName := "Unknown"
        if len(parts) > 1 {
                characterName = strings.ReplaceAll(parts[1], "_", " ")
                characterName = strings.ReplaceAll(characterName, "%28", "(")
                characterName = strings.ReplaceAll(characterName, "%29", ")")
                
                wikiSearchName := regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(characterName, "")
                wikiSearchName = strings.TrimSpace(wikiSearchName)
                
                detail := VSBDetailResponse{
                        Name:  characterName,
                        Stats: make(map[string]string),
                }

                wikiImage := getWikipediaImage(wikiSearchName)
                if wikiImage != "" {
                        detail.WikiImageURL = wikiImage
                }

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

                detail = extractVSBStatsFromText(detail, htmlContent)
                c.JSON(200, detail)
                return
        }

        c.JSON(400, gin.H{"error": "Invalid URL"})
}

func getWikipediaImage(characterName string) string {
        apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&titles=%s&prop=pageimages&format=json&pithumbsize=500&redirects=1",
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
        text := stripHTML(htmlContent)

        tierRe := regexp.MustCompile(`(?i)Tier[:\s]*([0-9A-Z\-\s]+)`)
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

        // Improved Summary extraction: Try to find the first real paragraph
        paragraphs := strings.Split(text, ". ")
        summary := ""
        for _, p := range paragraphs {
                trimmed := strings.TrimSpace(p)
                if len(trimmed) > 60 && !strings.Contains(trimmed, "Tier:") && !strings.Contains(trimmed, "Summary") {
                        summary = trimmed + "."
                        break
                }
        }
        detail.Summary = summary

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
        reStyles := regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</\1>`)
        s = reStyles.ReplaceAllString(s, "")

        reTags := regexp.MustCompile(`<[^>]+>`)
        cleaned := reTags.ReplaceAllString(s, " ")

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