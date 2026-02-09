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
	klipyBaseURL = "https://api.klipy.com/api/v1"
)

var (
	klipyAPIKey string
	httpClient  *http.Client
	userAgents  = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
)

func init() {
	klipyAPIKey = os.Getenv("KLIPY_API_KEY")
	httpClient = &http.Client{Timeout: 20 * time.Second}
	rand.Seed(time.Now().UnixNano())
}

func SetBrowser(b interface{}) {
	// No-op for memory optimization (rod removed)
}

// =============================================================================
// IMAGE SEARCH - DuckDuckGo Scraper (High Reliability)
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

	// Diversified searches
	queries := []string{
		fmt.Sprintf("%s hd wallpaper", query),
		fmt.Sprintf("%s aesthetic photo", query),
		fmt.Sprintf("%s high resolution", query),
	}

	images := []string{}
	for _, q := range queries {
		if len(images) >= maxResults {
			break
		}
		found := scrapeImagesFromDDG(q, 15)
		images = append(images, found...)
		images = deduplicateStrings(images)
	}

	if len(images) > maxResults {
		images = images[:maxResults]
	}

	c.JSON(200, PinterestResponse{
		Images: images,
		Count:  len(images),
		Note:   "DuckDuckGo Image Scraper V2",
	})
}

func scrapeImagesFromDDG(query string, limit int) []string {
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

	return parseDDGHTML(resp.Body, limit)
}

func parseDDGHTML(body io.Reader, limit int) []string {
	tokenizer := html.NewTokenizer(body)
	var urls []string
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
					link := attr.Val
					actual := ""

					if strings.Contains(link, "uddg=") {
						parts := strings.Split(link, "uddg=")
						if len(parts) > 1 {
							decoded, err := url.QueryUnescape(parts[1])
							if err == nil {
								actual = strings.Split(decoded, "&")[0]
							}
						}
					} else if strings.HasPrefix(link, "http") {
						actual = link
					}

					if actual != "" && isImageURL(actual) && !seen[actual] && !strings.Contains(actual, "duckduckgo.com") {
						seen[actual] = true
						urls = append(urls, actual)
						if len(urls) >= limit {
							return urls
						}
					}
				}
			}
		}
		if token.Type == html.StartTagToken && token.Data == "img" {
			for _, attr := range token.Attr {
				if attr.Key == "src" {
					val := attr.Val
					if !seen[val] && isImageURL(val) && !strings.Contains(val, "duckduckgo.com") && len(val) > 20 {
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
	return urls
}

func isImageURL(url string) bool {
	l := strings.ToLower(url)
	return strings.HasSuffix(l, ".jpg") || strings.HasSuffix(l, ".jpeg") ||
		strings.HasSuffix(l, ".png") || strings.HasSuffix(l, ".gif") ||
		strings.HasSuffix(l, ".webp") || strings.Contains(l, ".jpg?") ||
		strings.Contains(l, ".png?")
}

// =============================================================================
// STICKERS - Klipy v1 API
// =============================================================================

type KlipyResponse struct {
	Result bool `json:"result"`
	Data   struct {
		Data []struct {
			File struct {
				HD struct {
					GIF struct {
						URL string `json:"url"`
					} `json:"gif"`
				} `json:"hd"`
			} `json:"file"`
		} `json:"data"`
	} `json:"data"`
}

func SearchStickers(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	key := os.Getenv("KLIPY_API_KEY")
	if key == "" {
		c.JSON(500, gin.H{"error": "KLIPY_API_KEY missing"})
		return
	}

	apiURL := fmt.Sprintf("%s/%s/stickers/search?q=%s&per_page=10&customer_id=goten_bot", klipyBaseURL, key, url.QueryEscape(query))

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "API request failed"})
		return
	}
	defer resp.Body.Close()

	var kr KlipyResponse
	if err := json.NewDecoder(resp.Body).Decode(&kr); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse API response"})
		return
	}

	stickers := []string{}
	for _, r := range kr.Data.Data {
		if r.File.HD.GIF.URL != "" {
			stickers = append(stickers, r.File.HD.GIF.URL)
		}
	}

	c.JSON(200, gin.H{"stickers": stickers, "count": len(stickers)})
}

// =============================================================================
// VS BATTLES - Robust Low-RAM Extraction
// =============================================================================

func SearchVSBattles(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	searchURL := "https://html.duckduckgo.com/html/"
	formData := url.Values{}
	formData.Set("q", query+" vs battles wiki")

	req, _ := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", getRandomUA())

	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}
	defer resp.Body.Close()

	characters := []gin.H{}
	tokenizer := html.NewTokenizer(resp.Body)
	seen := make(map[string]bool)

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "a" {
			for _, attr := range token.Attr {
				if attr.Key == "href" && (strings.Contains(attr.Val, "vsbattles.fandom.com/wiki/") || strings.Contains(attr.Val, "uddg=")) {
					link := attr.Val
					actual := ""

					if strings.Contains(link, "uddg=") {
						parts := strings.Split(link, "uddg=")
						if len(parts) > 1 {
							decoded, _ := url.QueryUnescape(parts[1])
							actual = strings.Split(decoded, "&")[0]
						}
					} else if strings.Contains(link, "vsbattles.fandom.com/wiki/") {
						actual = link
					}

					if actual != "" && strings.Contains(actual, "vsbattles.fandom.com/wiki/") && !seen[actual] && !strings.Contains(actual, "Special:") && !strings.Contains(actual, "Category:") {
						seen[actual] = true
						name := actual[strings.LastIndex(actual, "/")+1:]
						name = strings.ReplaceAll(name, "_", " ")
						characters = append(characters, gin.H{"name": name, "url": actual})
					}
				}
			}
		}
		if len(characters) >= 5 {
			break
		}
	}

	if len(characters) == 0 {
		c.JSON(404, gin.H{"error": "No characters found"})
		return
	}

	c.JSON(200, gin.H{"characters": characters})
}

func GetVSBattlesDetail(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(400, gin.H{"error": "URL required"})
		return
	}

	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch page"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	htmlContent := string(body)

	detail := gin.H{
		"tier":          "Unknown",
		"attackPotency": "N/A",
		"speed":         "N/A",
		"durability":    "N/A",
		"stamina":       "N/A",
		"range":         "N/A",
		"summary":       "No summary available.",
		"imageUrl":      "",
	}

	// 1. Precise Image Extraction
	imgRe := regexp.MustCompile(`class="pi-image-thumbnail"[^>]*src="([^"]+)"`)
	if m := imgRe.FindStringSubmatch(htmlContent); len(m) > 1 {
		img := m[1]
		if strings.Contains(img, "/revision/") {
			img = strings.Split(img, "/revision/")[0]
		}
		detail["imageUrl"] = img
	}

	// 2. Clean text for stat extraction
	text := stripHTML(htmlContent)

	stats := map[string]string{
		"tier":          "Tier",
		"attackPotency": "Attack Potency",
		"speed":         "Speed",
		"durability":    "Durability",
		"stamina":       "Stamina",
		"range":         "Range",
	}

	for key, label := range stats {
		re := regexp.MustCompile(`(?i)` + label + `\s*:\s*([^(\n|]+)`)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			val := strings.TrimSpace(m[1])
			if len(val) > 1 {
				detail[key] = val
			}
		}
	}

	// 3. Summary Extraction
	paragraphs := strings.Split(text, ". ")
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if len(p) > 100 && !strings.Contains(p, "Tier:") && !strings.Contains(p, "Attack Potency:") {
			detail["summary"] = p + "."
			break
		}
	}

	nameParts := strings.Split(pageURL, "/wiki/")
	if len(nameParts) > 1 {
		detail["name"] = strings.ReplaceAll(nameParts[1], "_", " ")
	}

	c.JSON(200, detail)
}

func getWikipediaImage(name string) string {
	name = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(name, "")
	apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&titles=%s&prop=pageimages&format=json&pithumbsize=500&redirects=1", url.QueryEscape(strings.TrimSpace(name)))
	resp, _ := http.Get(apiURL)
	if resp != nil {
		defer resp.Body.Close()
		var result struct {
			Query struct {
				Pages map[string]struct {
					Thumbnail struct{ Source string `json:"source"` } `json:"thumbnail"`
				} `json:"pages"`
			} `json:"query"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		for _, p := range result.Query.Pages {
			if p.Thumbnail.Source != "" {
				return p.Thumbnail.Source
			}
		}
	}
	return ""
}

// =============================================================================
// RULE34 - API + WEB FALLBACK
// =============================================================================

func SearchRule34(c *gin.Context) {
	query := c.Query("query")
	images := []string{}

	apiURL := fmt.Sprintf("https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=10&tags=%s", url.QueryEscape(query))
	resp, err := httpClient.Get(apiURL)
	if err == nil {
		defer resp.Body.Close()
		var posts []struct {
			FileURL string `json:"file_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&posts); err == nil {
			for _, p := range posts {
				if p.FileURL != "" {
					img := p.FileURL
					if strings.HasPrefix(img, "//") {
						img = "https:" + img
					}
					images = append(images, img)
				}
			}
		}
	}

	if len(images) == 0 {
		images = scrapeRule34Web(query, 10)
	}

	c.JSON(200, gin.H{"images": images, "count": len(images)})
}

func scrapeRule34Web(query string, limit int) []string {
	tag := strings.ReplaceAll(query, " ", "_")
	searchURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=list&tags=%s", url.QueryEscape(tag))

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := httpClient.Do(req)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()

	var postIDs []string
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "span" {
			for _, attr := range token.Attr {
				if attr.Key == "id" && strings.HasPrefix(attr.Val, "s") {
					id := attr.Val[1:]
					postIDs = append(postIDs, id)
				}
			}
		}
		if len(postIDs) >= limit {
			break
		}
	}

	images := []string{}
	for _, id := range postIDs {
		imgURL := getRule34ImageDirect(id)
		if imgURL != "" {
			images = append(images, imgURL)
		}
	}
	return images
}

func getRule34ImageDirect(postID string) string {
	postURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=view&id=%s", postID)
	req, _ := http.NewRequest("GET", postURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if (token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken) && token.Data == "img" {
			for _, attr := range token.Attr {
				if attr.Key == "id" && attr.Val == "image" {
					for _, a := range token.Attr {
						if a.Key == "src" {
							src := a.Val
							if strings.HasPrefix(src, "//") {
								src = "https:" + src
							}
							return src
						}
					}
				}
			}
		}
	}
	return ""
}

// =============================================================================
// UTILS
// =============================================================================

func getRandomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func stripHTML(s string) string {
	reStyle := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	s = reStyle.ReplaceAllString(s, "")
	reScript := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	s = reScript.ReplaceAllString(s, "")

	reTags := regexp.MustCompile(`<[^>]+>`)
	cleaned := reTags.ReplaceAllString(s, " ")
	return regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
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