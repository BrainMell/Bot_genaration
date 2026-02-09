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

	"github.com/PuerkitoBio/goquery"
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

func SetBrowser(b interface{}) {}

// =============================================================================
// IMAGE SEARCH - DuckDuckGo
// =============================================================================

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

	searchURL := "https://html.duckduckgo.com/html/"
	formData := url.Values{}
	formData.Set("q", query+" hd wallpaper")

	req, _ := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", getRandomUA())

	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed"})
		return
	}
	defer resp.Body.Close()

	images := parseDDGHTML(resp.Body, maxResults)
	c.JSON(200, gin.H{"images": images, "count": len(images)})
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
							decoded, _ := url.QueryUnescape(parts[1])
							actual = strings.Split(decoded, "&")[0]
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
	}
	return urls
}

func isImageURL(url string) bool {
	l := strings.ToLower(url)
	return strings.HasSuffix(l, ".jpg") || strings.HasSuffix(l, ".jpeg") ||
		strings.HasSuffix(l, ".png") || strings.HasSuffix(l, ".gif") ||
		strings.HasSuffix(l, ".webp")
}

// =============================================================================
// STICKERS - Klipy
// =============================================================================

func SearchStickers(c *gin.Context) {
	query := c.Query("query")
	key := os.Getenv("KLIPY_API_KEY")
	if key == "" {
		c.JSON(500, gin.H{"error": "KLIPY_API_KEY environment variable is not set"})
		return
	}

	apiURL := fmt.Sprintf("https://api.klipy.com/api/v1/%s/stickers/search?q=%s&per_page=10&customer_id=goten_bot", key, url.QueryEscape(query))
	
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Klipy API request failed"})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Data []struct {
				File struct {
					HD struct {
						GIF struct { URL string `json:"url"` } `json:"gif"`
					} `json:"hd"`
				} `json:"file"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode Klipy response"})
		return
	}

	stickers := []string{}
	for _, item := range result.Data.Data {
		if item.File.HD.GIF.URL != "" {
			stickers = append(stickers, item.File.HD.GIF.URL)
		}
	}

	c.JSON(200, gin.H{"stickers": stickers, "count": len(stickers)})
}

// =============================================================================
// VS BATTLES - Heavy Duty goquery Parser
// =============================================================================

func SearchVSBattles(c *gin.Context) {
	query := c.Query("query")
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse search results"})
		return
	}

	characters := []gin.H{}
	doc.Find("a.result__a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, "vsbattles.fandom.com/wiki/") && len(characters) < 5 {
			// Clean URL
			actual := href
			if strings.Contains(href, "uddg=") {
				parts := strings.Split(href, "uddg=")
				decoded, _ := url.QueryUnescape(parts[1])
				actual = strings.Split(decoded, "&")[0]
			}
			
			if !strings.Contains(actual, "Special:") && !strings.Contains(actual, "Category:") {
				name := actual[strings.LastIndex(actual, "/")+1:]
				name = strings.ReplaceAll(name, "_", " ")
				characters = append(characters, gin.H{"name": name, "url": actual})
			}
		}
	})

	c.JSON(200, gin.H{"characters": characters})
}

func GetVSBattlesDetail(c *gin.Context) {
	pageURL := c.Query("url")
	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := httpClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch page"})
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse character page"})
		return
	}

	detail := gin.H{
		"tier":          "Unknown",
		"attackPotency": "N/A",
		"speed":         "N/A",
		"durability":    "N/A",
		"stamina":       "N/A",
		"range":         "N/A",
		"summary":       "",
		"imageUrl":      "",
	}

	// 1. Precise Data Extraction from Portable Infobox
	doc.Find(".pi-item").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find(".pi-data-label").Text())
		value := strings.TrimSpace(s.Find(".pi-data-value").Text())
		
		// Clean value (remove citations like [1])
		value = regexp.MustCompile(`\[.*?\]`).ReplaceAllString(value, "")
		value = strings.Split(value, "(")[0] // Take first part before extra info
		value = strings.TrimSpace(value)

		switch {
		case strings.Contains(label, "Tier"): detail["tier"] = value
		case strings.Contains(label, "Attack Potency"): detail["attackPotency"] = value
		case strings.Contains(label, "Speed"): detail["speed"] = value
		case strings.Contains(label, "Durability"): detail["durability"] = value
		case strings.Contains(label, "Stamina"): detail["stamina"] = value
		case strings.Contains(label, "Range"): detail["range"] = value
		}
	})

	// 2. Image
	img, _ := doc.Find("img.pi-image-thumbnail").Attr("src")
	if img != "" {
		detail["imageUrl"] = strings.Split(img, "/revision/")[0]
	}

	// 3. Summary (First paragraph after infobox)
	doc.Find(".mw-parser-output > p").EachWithBreak(func(i int, s *goquery.Selection) bool {
		txt := strings.TrimSpace(s.Text())
		if len(txt) > 50 {
			detail["summary"] = strings.Split(txt, ".")[0] + "."
			return false
		}
		return true
	})

	c.JSON(200, detail)
}

// =============================================================================
// RULE34 - API + WEB
// =============================================================================

func SearchRule34(c *gin.Context) {
	query := c.Query("query")
	apiURL := fmt.Sprintf("https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=10&tags=%s", url.QueryEscape(query))
	resp, err := httpClient.Get(apiURL)
	images := []string{}
	
	if err == nil {
		defer resp.Body.Close()
		var posts []struct { FileURL string `json:"file_url"` }
		if err := json.NewDecoder(resp.Body).Decode(&posts); err == nil {
			for _, p := range posts {
				if p.FileURL != "" {
					img := p.FileURL
					if strings.HasPrefix(img, "//") { img = "https:" + img }
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
	if err != nil { return []string{} }
	defer resp.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	images := []string{}
	doc.Find("span.thumb").Each(func(i int, s *goquery.Selection) {
		if len(images) < limit {
			id, _ := s.Attr("id")
			if strings.HasPrefix(id, "s") {
				img := getRule34ImageDirect(id[1:])
				if img != "" { images = append(images, img) }
			}
		}
	})
	return images
}

func getRule34ImageDirect(id string) string {
	postURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=view&id=%s", id)
	req, _ := http.NewRequest("GET", postURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := httpClient.Do(req)
	if err != nil { return "" }
	defer resp.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	src, _ := doc.Find("#image").Attr("src")
	if src != "" && strings.HasPrefix(src, "//") { src = "https:" + src }
	return src
}

func getRandomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
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
