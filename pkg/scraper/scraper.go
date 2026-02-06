package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
)

// =============================================================================
// VS BATTLES SCRAPER (WITH CHROMEDP - LIKE PUPPETEER)
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

// SearchVSBattles searches VS Battles Wiki using headless Chrome (like Puppeteer)
func SearchVSBattles(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	// Create Chrome context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	searchURL := fmt.Sprintf("https://vsbattles.fandom.com/wiki/Special:Search?query=%s", url.QueryEscape(query))

	var searchLinks []VSBCharacter
	var htmlContent string

	// Run Chrome automation
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second), // Wait for JS to load
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Browser automation failed: %s", err.Error())})
		return
	}

	// Parse search results
	searchLinks = parseVSBSearchResults(htmlContent)

	if len(searchLinks) == 0 {
		c.JSON(404, gin.H{"error": "Character not found"})
		return
	}

	c.JSON(200, VSBSearchResponse{Characters: searchLinks})
}

// parseVSBSearchResults extracts character links from search HTML
func parseVSBSearchResults(html string) []VSBCharacter {
	var results []VSBCharacter

	// Match search result links
	re := regexp.MustCompile(`<a[^>]+href="(https://vsbattles\.fandom\.com/wiki/[^"]+)"[^>]*>([^<]+)</a>`)
	matches := re.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		url := match[1]
		name := strings.TrimSpace(match[2])

		// Filter out special pages
		if strings.Contains(url, "Special:") || strings.Contains(url, "Category:") {
			continue
		}

		// Deduplicate
		if seen[url] {
			continue
		}
		seen[url] = true

		// Clean up name
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

	// Create Chrome context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var detail VSBDetailResponse
	detail.Stats = make(map[string]string)

	var htmlContent string
	var imageURL string
	var imageWidth, imageHeight int64

	// Run Chrome automation (EXACTLY like Puppeteer logic)
	err := chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second), // Wait for lazy-loaded images

		// Try to get high-quality image with retries (like the JS version)
		chromedp.ActionFunc(func(ctx context.Context) error {
			maxRetries := 10
			minImageSize := 100

			for i := 0; i < maxRetries; i++ {
				// Try to get image attributes
				var dataSrc, src string

				chromedp.AttributeValue(`img.pi-image-thumbnail`, "data-src", &dataSrc, nil).Do(ctx)
				chromedp.AttributeValue(`img.pi-image-thumbnail`, "src", &src, nil).Do(ctx)
				chromedp.AttributeValue(`img.pi-image-thumbnail`, "data-image-width", &imageWidth, nil).Do(ctx)
				chromedp.AttributeValue(`img.pi-image-thumbnail`, "data-image-height", &imageHeight, nil).Do(ctx)

				// Use data-src if available, fallback to src
				rawURL := dataSrc
				if rawURL == "" {
					rawURL = src
				}

				// Check if we have a valid image
				if rawURL != "" && !strings.HasPrefix(rawURL, "data:") &&
					imageWidth >= int64(minImageSize) && imageHeight >= int64(minImageSize) {
					imageURL = rawURL
					break
				}

				// Wait and retry
				time.Sleep(2 * time.Second)
			}

			return nil
		}),

		// Get full HTML content
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Browser automation failed: %s", err.Error())})
		return
	}

	// Clean image URL (remove /revision/ suffix)
	if imageURL != "" && strings.Contains(imageURL, "/revision/") {
		parts := strings.Split(imageURL, "/revision/")
		imageURL = parts[0]
	}

	detail.ImageURL = imageURL
	detail.ImageWidth = int(imageWidth)
	detail.ImageHeight = int(imageHeight)

	// Extract stats from HTML
	extractVSBStats(&detail, htmlContent)

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
	// Extract character name
	nameRe := regexp.MustCompile(`<h1[^>]*class="page-header__title"[^>]*>([^<]+)</h1>`)
	if matches := nameRe.FindStringSubmatch(html); len(matches) > 1 {
		detail.Name = strings.TrimSpace(matches[1])
	}

	// Extract summary from first paragraph
	summaryRe := regexp.MustCompile(`<div[^>]*class="mw-parser-output"[^>]*>.*?<p>([^<]{50,}?)</p>`)
	if matches := summaryRe.FindStringSubmatch(html); len(matches) > 1 {
		text := stripHTML(matches[1])
		if !strings.Contains(text, ":") && len(text) > 50 {
			detail.Summary = text
		}
	}

	// Extract stats using regex (like the JS version)
	statPatterns := map[string]*string{
		"Tier":           &detail.Tier,
		"Attack Potency": &detail.AttackPotency,
		"Speed":          &detail.Speed,
		"Durability":     &detail.Durability,
		"Stamina":        &detail.Stamina,
		"Range":          &detail.Range,
	}

	for statName, statPtr := range statPatterns {
		pattern := fmt.Sprintf(`(?i)%s\s*:\s*<[^>]+>(.*?)(?:<br|</div|</p|$)`, regexp.QuoteMeta(statName))
		re := regexp.MustCompile(pattern)

		if matches := re.FindStringSubmatch(html); len(matches) > 1 {
			value := stripHTML(matches[1])
			value = cleanVSBText(value)
			*statPtr = value
			detail.Stats[statName] = value
		}
	}
}

// =============================================================================
// PINTEREST SCRAPER (API-based - faster than browser)
// =============================================================================

type PinterestResponse struct {
	Images []string `json:"images"`
}

// SearchPinterest uses Pinterest's unofficial API (faster than browser scraping)
func SearchPinterest(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query required"})
		return
	}

	maxResults := 10
	if countStr := c.Query("count"); countStr != "" {
		fmt.Sscanf(countStr, "%d", &maxResults)
		if maxResults > 50 {
			maxResults = 50
		}
	}

	// Try static scraping first (faster)
	images := tryPinterestStatic(query, maxResults)

	// Fallback to browser if needed
	if len(images) == 0 {
		images = tryPinterestBrowser(query, maxResults)
	}

	c.JSON(200, PinterestResponse{Images: images})
}

// tryPinterestStatic attempts static scraping (fast)
func tryPinterestStatic(query string, maxResults int) []string {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	var images []string
	var mu sync.Mutex

	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if strings.Contains(src, "pinimg.com") && !strings.Contains(src, "75x75") {
			mu.Lock()
			defer mu.Unlock()

			// Upgrade quality
			highRes := strings.ReplaceAll(src, "236x", "736x")
			highRes = strings.ReplaceAll(highRes, "474x", "736x")

			if len(images) < maxResults {
				images = append(images, highRes)
			}
		}
	})

	searchURL := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)
	collector.Visit(searchURL)

	return deduplicateStrings(images)
}

// tryPinterestBrowser uses headless Chrome for Pinterest (fallback)
func tryPinterestBrowser(query string, maxResults int) []string {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	searchURL := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)

	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(3*time.Second), // Let Pinterest load
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return []string{}
	}

	// Extract image URLs
	re := regexp.MustCompile(`https://i\.pinimg\.com/[^"'\s]+`)
	matches := re.FindAllString(htmlContent, -1)

	var images []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if seen[match] || strings.Contains(match, "75x75") {
			continue
		}
		seen[match] = true

		// Upgrade quality
		highRes := strings.ReplaceAll(match, "236x", "736x")
		highRes = strings.ReplaceAll(highRes, "474x", "736x")

		images = append(images, highRes)

		if len(images) >= maxResults {
			break
		}
	}

	return images
}

// =============================================================================
// RULE34 SCRAPER (API + Browser Fallback)
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

	// Try API first
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

	// Fetch full images
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

// stripHTML removes HTML tags from text
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(s, "")
}

// cleanVSBText cleans up VS Battles text
func cleanVSBText(s string) string {
	s = strings.TrimSpace(s)

	// Remove references
	re := regexp.MustCompile(`\[.*?\]`)
	s = re.ReplaceAllString(s, "")

	// Remove parentheses
	re = regexp.MustCompile(`\(.*?\)`)
	s = re.ReplaceAllString(s, "")

	return strings.TrimSpace(s)
}

// extractPeakValue gets the highest/last value from a stat
func extractPeakValue(text string) string {
	if text == "" {
		return "N/A"
	}

	// Split by | and take last (peak)
	parts := strings.Split(text, "|")
	if len(parts) > 0 {
		peak := strings.TrimSpace(parts[len(parts)-1])
		return cleanVSBText(peak)
	}

	return cleanVSBText(text)
}

// extractHighestTier extracts the highest tier
func extractHighestTier(text string) string {
	re := regexp.MustCompile(`\b([0-9]+)-([A-Z])\b`)
	matches := re.FindAllString(text, -1)

	if len(matches) > 0 {
		return matches[len(matches)-1]
	}

	return "Unknown"
}

// deduplicateStrings removes duplicates
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