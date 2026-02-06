package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gin-gonic/gin"
)

// =============================================================================
// PINTEREST SCRAPER
// =============================================================================

type PinterestResponse struct {
	Images []string `json:"images"`
}

// SearchPinterest scrapes Pinterest for images based on a query
// Note: Pinterest is JavaScript-heavy, so static scraping may have limitations
// For full functionality, consider using chromedp for browser automation
func SearchPinterest(c *gin.Context) {
	query := c.Query("query")
	count := 10 // default count
	
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	// Parse count parameter
	if countStr := c.Query("count"); countStr != "" {
		fmt.Sscanf(countStr, "%d", &count)
		if count > 30 {
			count = 30 // Cap at 30
		}
	}

	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.AllowedDomains("www.pinterest.com", "pinterest.com"),
	)

	var images []string
	var mu sync.Mutex

	// Look for images with pinimg.com in src
	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		
		// Filter for Pinterest CDN images
		if strings.Contains(src, "pinimg.com") && !strings.Contains(src, "75x75") {
			mu.Lock()
			defer mu.Unlock()
			
			// Upgrade to higher quality
			// Replace 236x or 474x with 736x for better resolution
			highRes := src
			highRes = strings.Replace(highRes, "236x", "736x", 1)
			highRes = strings.Replace(highRes, "474x", "736x", 1)
			
			images = append(images, highRes)
		}
	})

	// Also check for data-test-id="pinWrapper" pattern (newer Pinterest)
	collector.OnHTML("div[data-test-id=\"pinWrapper\"] img", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		
		if src != "" && strings.Contains(src, "pinimg.com") {
			mu.Lock()
			defer mu.Unlock()
			
			highRes := src
			highRes = strings.Replace(highRes, "236x", "736x", 1)
			highRes = strings.Replace(highRes, "474x", "736x", 1)
			
			images = append(images, highRes)
		}
	})

	searchURL := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)
	collector.Visit(searchURL)

	// Deduplicate
	uniqueImages := deduplicateStrings(images)
	
	// Limit to requested count
	if len(uniqueImages) > count {
		uniqueImages = uniqueImages[:count]
	}

	c.JSON(200, PinterestResponse{Images: uniqueImages})
}

// =============================================================================
// RULE34 SCRAPER (Enhanced with API + Scraping fallback)
// =============================================================================

type Rule34Response struct {
	Images []string `json:"images"`
}

type Rule34Post struct {
	FileURL string `json:"file_url"`
}

// SearchRule34 searches Rule34 using API first, then falls back to scraping
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

	// Try API first
	images := tryRule34API(query, count)
	
	// Fallback to web scraping if API fails
	if len(images) == 0 {
		fmt.Println("⚠️ API returned nothing, trying web scraping...")
		images = tryRule34WebScrape(query, count)
	}

	c.JSON(200, Rule34Response{Images: images})
}

// tryRule34API attempts to fetch images from Rule34 API
func tryRule34API(searchTerm string, count int) []string {
	tag := strings.TrimSpace(searchTerm)
	tag = strings.ReplaceAll(tag, " ", "_")

	endpoints := []string{
		fmt.Sprintf("https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=200&tags=%s", url.QueryEscape(tag)),
		fmt.Sprintf("https://rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=200&tags=%s", url.QueryEscape(tag)),
	}

	for _, endpoint := range endpoints {
		fmt.Println("🔍 Trying API:", endpoint)
		
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://rule34.xxx/")
		req.Header.Set("Cookie", "filter_ai=1")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		var posts []Rule34Post
		if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
			// Try parsing as object with "post" field
			var wrapper map[string]interface{}
			resp.Body.Close()
			
			resp, _ = client.Do(req)
			if resp != nil {
				defer resp.Body.Close()
				if json.NewDecoder(resp.Body).Decode(&wrapper) == nil {
					if postData, ok := wrapper["post"].([]interface{}); ok {
						for _, p := range postData {
							if postMap, ok := p.(map[string]interface{}); ok {
								if fileURL, ok := postMap["file_url"].(string); ok {
									posts = append(posts, Rule34Post{FileURL: fileURL})
								}
							}
						}
					}
				}
			}
		}

		if len(posts) == 0 {
			fmt.Println("⚠️ API returned 0 posts")
			continue
		}

		var images []string
		for _, post := range posts {
			if post.FileURL != "" {
				imageURL := post.FileURL
				
				// Normalize URL
				if strings.HasPrefix(imageURL, "//") {
					imageURL = "https:" + imageURL
				}
				
				images = append(images, imageURL)
				
				if len(images) >= count {
					break
				}
			}
		}

		if len(images) > 0 {
			return images
		}
	}

	return []string{}
}

// tryRule34WebScrape scrapes Rule34 website when API fails
func tryRule34WebScrape(searchTerm string, count int) []string {
	tag := strings.TrimSpace(searchTerm)
	tag = strings.ReplaceAll(tag, " ", "_")
	
	scrapeURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=list&tags=%s", url.QueryEscape(tag))
	
	fmt.Println("🌐 Scraping webpage:", scrapeURL)

	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	var postIDs []string
	var mu sync.Mutex

	// Extract post IDs from thumbnail links
	collector.OnHTML(".thumb a", func(e *colly.HTMLElement) {
		mu.Lock()
		defer mu.Unlock()
		
		if len(postIDs) >= count {
			return
		}

		href := e.Attr("href")
		if strings.Contains(href, "id=") {
			re := regexp.MustCompile(`id=(\d+)`)
			matches := re.FindStringSubmatch(href)
			if len(matches) > 1 {
				postIDs = append(postIDs, matches[1])
			}
		}
	})

	collector.Visit(scrapeURL)

	// Fetch full images from post pages concurrently
	var images []string
	var wg sync.WaitGroup
	var imagesMu sync.Mutex
	
	// Limit concurrent requests
	semaphore := make(chan struct{}, 5)

	for _, postID := range postIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			imageURL := getFullRule34Image(id)
			if imageURL != "" {
				imagesMu.Lock()
				images = append(images, imageURL)
				imagesMu.Unlock()
			}
		}(postID)
	}

	wg.Wait()

	return images
}

// getFullRule34Image fetches the full image URL from a post page
func getFullRule34Image(postID string) string {
	postURL := fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=view&id=%s", postID)
	
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0"),
	)

	var imageURL string

	// Method 1: Direct image
	collector.OnHTML("#image", func(e *colly.HTMLElement) {
		if imageURL == "" {
			imageURL = e.Attr("src")
		}
	})

	// Method 2: Video source
	collector.OnHTML("video source", func(e *colly.HTMLElement) {
		if imageURL == "" {
			imageURL = e.Attr("src")
		}
	})

	// Method 3: Meta tag
	collector.OnHTML("meta[property=\"og:image\"]", func(e *colly.HTMLElement) {
		if imageURL == "" {
			imageURL = e.Attr("content")
		}
	})

	collector.Visit(postURL)

	// Normalize URL
	if imageURL != "" && strings.HasPrefix(imageURL, "//") {
		imageURL = "https:" + imageURL
	}

	return imageURL
}

// =============================================================================
// VS BATTLES SCRAPER (PowerScale)
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
	Summary       string            `json:"summary"`
	Tier          string            `json:"tier"`
	AttackPotency string            `json:"attackPotency"`
	Speed         string            `json:"speed"`
	Durability    string            `json:"durability"`
	Stamina       string            `json:"stamina"`
	Range         string            `json:"range"`
	Stats         map[string]string `json:"stats"`
}

// SearchVSBattles searches VS Battles Wiki using their API
func SearchVSBattles(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	apiURL := "https://vsbattles.fandom.com/api.php?action=opensearch&format=json&limit=10&search=" + url.QueryEscape(query)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to contact VSB API"})
		return
	}
	defer resp.Body.Close()

	var raw []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse VSB API"})
		return
	}

	if len(raw) < 4 {
		c.JSON(200, VSBSearchResponse{Characters: []VSBCharacter{}})
		return
	}

	names := raw[1].([]interface{})
	urls := raw[3].([]interface{})
	
	var chars []VSBCharacter
	for i := range names {
		chars = append(chars, VSBCharacter{
			Name: names[i].(string),
			URL:  urls[i].(string),
		})
	}

	c.JSON(200, VSBSearchResponse{Characters: chars})
}

// GetVSBattlesDetail scrapes detailed information from a VS Battles character page
func GetVSBattlesDetail(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(400, gin.H{"error": "URL required"})
		return
	}

	collector := colly.NewCollector()
	var detail VSBDetailResponse
	detail.Stats = make(map[string]string)

	// Extract character name
	collector.OnHTML("h1.page-header__title", func(e *colly.HTMLElement) {
		detail.Name = strings.TrimSpace(e.Text)
	})

	// Extract character image
	collector.OnHTML("aside.portable-infobox figure.pi-item.pi-image img", func(e *colly.HTMLElement) {
		if detail.ImageURL == "" {
			imgSrc := e.Attr("data-src")
			if imgSrc == "" {
				imgSrc = e.Attr("src")
			}
			
			// Remove /revision/ suffix to get original image
			if strings.Contains(imgSrc, "/revision/") {
				parts := strings.Split(imgSrc, "/revision/")
				imgSrc = parts[0]
			}
			
			detail.ImageURL = imgSrc
		}
	})

	// Extract summary from first paragraph
	var summaryFound bool
	collector.OnHTML("div.mw-parser-output p", func(e *colly.HTMLElement) {
		if !summaryFound {
			text := strings.TrimSpace(e.Text)
			if len(text) > 50 && !strings.Contains(text, ":") {
				detail.Summary = text
				summaryFound = true
			}
		}
	})

	// Extract stats using regex patterns
	var pageText string
	collector.OnHTML("div.mw-parser-output", func(e *colly.HTMLElement) {
		pageText = e.Text
	})

	collector.OnScraped(func(r *colly.Response) {
		// Extract Tier
		tierRe := regexp.MustCompile(`(?i)Tier\s*:\s*(.+?)(?:\n|$)`)
		if matches := tierRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.Tier = cleanVSBText(matches[1])
			detail.Stats["Tier"] = detail.Tier
		}

		// Extract Attack Potency
		apRe := regexp.MustCompile(`(?i)Attack Potency\s*:\s*(.+?)(?:\n|$)`)
		if matches := apRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.AttackPotency = cleanVSBText(matches[1])
			detail.Stats["Attack Potency"] = detail.AttackPotency
		}

		// Extract Speed
		speedRe := regexp.MustCompile(`(?i)Speed\s*:\s*(.+?)(?:\n|$)`)
		if matches := speedRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.Speed = cleanVSBText(matches[1])
			detail.Stats["Speed"] = detail.Speed
		}

		// Extract Durability
		durRe := regexp.MustCompile(`(?i)Durability\s*:\s*(.+?)(?:\n|$)`)
		if matches := durRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.Durability = cleanVSBText(matches[1])
			detail.Stats["Durability"] = detail.Durability
		}

		// Extract Stamina
		stamRe := regexp.MustCompile(`(?i)Stamina\s*:\s*(.+?)(?:\n|$)`)
		if matches := stamRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.Stamina = cleanVSBText(matches[1])
			detail.Stats["Stamina"] = detail.Stamina
		}

		// Extract Range
		rangeRe := regexp.MustCompile(`(?i)Range\s*:\s*(.+?)(?:\n|$)`)
		if matches := rangeRe.FindStringSubmatch(pageText); len(matches) > 1 {
			detail.Range = cleanVSBText(matches[1])
			detail.Stats["Range"] = detail.Range
		}

		// Apply "Peak Logic" - extract highest tier/value
		if detail.AttackPotency != "" {
			detail.AttackPotency = extractPeakValue(detail.AttackPotency)
		}
		if detail.Speed != "" {
			detail.Speed = extractPeakValue(detail.Speed)
		}
		if detail.Tier != "" {
			detail.Tier = extractHighestTier(detail.Tier)
		}
	})

	collector.Visit(pageURL)
	c.JSON(200, detail)
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// deduplicateStrings removes duplicate strings from a slice
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

// cleanVSBText cleans up text extracted from VS Battles
func cleanVSBText(s string) string {
	s = strings.TrimSpace(s)
	
	// Remove reference markers like [1], [2], etc.
	re := regexp.MustCompile(`\[.*?\]`)
	s = re.ReplaceAllString(s, "")
	
	// Remove parenthetical notes
	re = regexp.MustCompile(`\(.*?\)`)
	s = re.ReplaceAllString(s, "")
	
	return strings.TrimSpace(s)
}

// extractPeakValue extracts the highest/peak value from a stat string
// Example: "Wall level | Building level | City level" -> "City level"
func extractPeakValue(text string) string {
	if text == "" {
		return "N/A"
	}
	
	// Split by | and take the last value (usually the peak)
	parts := strings.Split(text, "|")
	if len(parts) > 0 {
		peak := strings.TrimSpace(parts[len(parts)-1])
		return cleanVSBText(peak)
	}
	
	return cleanVSBText(text)
}

// extractHighestTier extracts the highest tier from a tier string
// Example: "9-B, 9-A, 8-C" -> "8-C"
func extractHighestTier(text string) string {
	// Find all tier patterns like "9-B", "8-C", "3-A", etc.
	re := regexp.MustCompile(`\b([0-9]+)-([A-Z])\b`)
	matches := re.FindAllString(text, -1)
	
	if len(matches) > 0 {
		// Return the last match (typically the highest)
		return matches[len(matches)-1]
	}
	
	return "Unknown"
}
