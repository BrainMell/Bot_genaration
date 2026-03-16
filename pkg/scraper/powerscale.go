package scraper

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
)

type VSBatleDetail struct {
	Name     string            `json:"name"`
	ImageURL string            `json:"imageUrl"`
	Summary  string            `json:"summary"`
	Stats    map[string]string `json:"stats"`
	PageURL  string            `json:"pageUrl"`
}

type VSBSearchResult struct {
	Characters []VSBCharacterOption `json:"characters"`
}

type VSBCharacterOption struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
}

// ── Endpoint 1: Search → return character list ────────────────────────────────
// GET /api/scrape/powerscale?query=goku
// Returns list of found characters for user to pick from
func ScrapePowerscale(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Character name required"})
		return
	}

	fmt.Printf("[Powerscale] Searching for: %s\n", query)

	var characters []VSBCharacterOption

	err := WithPage(func(page *rod.Page) error {
		searchURL := fmt.Sprintf(
			"https://vsbattles.fandom.com/wiki/Special:Search?scope=internal&navigationSearch=true&query=%s",
			url.QueryEscape(query),
		)
		fmt.Printf("[Powerscale] Navigating to search: %s\n", searchURL)
		page.MustNavigate(searchURL).MustWaitLoad()
		time.Sleep(3 * time.Second)

		// Grab all search result links
		linkEls, _ := page.Elements(".unified-search__result a")
		if len(linkEls) == 0 {
			linkEls, _ = page.Elements("article.unified-search__result a")
		}
		if len(linkEls) == 0 {
			linkEls, _ = page.Elements("li.unified-search__result a")
		}
		if len(linkEls) == 0 {
			linkEls, _ = page.Elements(".unified-search__result__link")
		}
		fmt.Printf("[Powerscale] Found %d raw link elements\n", len(linkEls))

		seen := map[string]bool{}
		id := 1
		for _, el := range linkEls {
			href, err := el.Attribute("href")
			if err != nil || href == nil {
				continue
			}
			u := *href
			if strings.HasPrefix(u, "/") {
				u = "https://vsbattles.fandom.com" + u
			}
			// Filter junk pages
			if !strings.Contains(u, "/wiki/") { continue }
			if strings.Contains(u, "Special:") { continue }
			if strings.Contains(u, "Category:") { continue }
			if strings.Contains(u, "Talk:") { continue }
			if strings.Contains(u, "User:") { continue }
			if strings.Contains(u, "File:") { continue }
			if seen[u] { continue }
			seen[u] = true

			// Extract readable name from URL
			// e.g. https://...wiki/Son_Goku_(DBS_Anime) → "Son Goku (DBS Anime)"
			parts := strings.Split(u, "/wiki/")
			if len(parts) < 2 { continue }
			rawName := parts[1]
			// Remove query/fragment
			if idx := strings.Index(rawName, "?"); idx != -1 {
				rawName = rawName[:idx]
			}
			// Decode URL encoding
			decoded, err := url.PathUnescape(rawName)
			if err != nil { decoded = rawName }
			// Replace underscores with spaces
			name := strings.ReplaceAll(decoded, "_", " ")

			fmt.Printf("[Powerscale] Found link: %s\n", u)
			characters = append(characters, VSBCharacterOption{
				ID:   id,
				Name: name,
				URL:  u,
			})
			id++
			if id > 10 { break }
		}
		return nil
	})

	if err != nil {
		c.JSON(500, gin.H{"error": "browser error: " + err.Error()})
		return
	}

	if len(characters) == 0 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("no results found for '%s'", query)})
		return
	}

	c.JSON(200, VSBSearchResult{Characters: characters})
}

// ── Endpoint 2: Scrape a specific character page by URL ───────────────────────
// GET /api/scrape/powerscale/fetch?url=https://vsbattles.fandom.com/wiki/Son_Goku_(DBS_Anime)
// Called after user picks a character from the list
func ScrapePowerscalePage(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(400, gin.H{"error": "url required"})
		return
	}

	// Safety check — only allow vsbattles URLs
	if !strings.Contains(pageURL, "vsbattles.fandom.com/wiki/") {
		c.JSON(400, gin.H{"error": "invalid url"})
		return
	}

	fmt.Printf("[Powerscale] Scraping page: %s\n", pageURL)

	var result *VSBatleDetail

	err := WithPage(func(page *rod.Page) error {
		err := page.Navigate(pageURL)
		if err != nil {
			return err
		}
		page.MustWaitLoad()

		// Scroll and wait for lazy images
		page.MustEval(`() => window.scrollTo(0, document.body.scrollHeight / 2)`)
		time.Sleep(2 * time.Second)

		// ── Image ─────────────────────────────────────────────────────────────
		// img.pi-image-thumbnail → data-src || src → strip /revision/
		imageURL := ""
		imgEl, err := page.Element("img.pi-image-thumbnail")
		if err == nil && imgEl != nil {
			dataSrc, _ := imgEl.Attribute("data-src")
			src, _ := imgEl.Attribute("src")
			rawURL := ""
			if dataSrc != nil && *dataSrc != "" && !strings.HasPrefix(*dataSrc, "data:") {
				rawURL = *dataSrc
			} else if src != nil && *src != "" && !strings.HasPrefix(*src, "data:") {
				rawURL = *src
			}
			if rawURL != "" {
				if idx := strings.Index(rawURL, "/revision/"); idx != -1 {
					rawURL = rawURL[:idx]
				}
				imageURL = rawURL
				fmt.Printf("[Powerscale] Image: %s\n", imageURL)
			}
		}
		// Fallback: any wikia image in article
		if imageURL == "" {
			allImgs, _ := page.Elements("#mw-content-text img")
			for _, img := range allImgs {
				src, _ := img.Attribute("src")
				if src == nil { continue }
				s := *src
				if !strings.Contains(s, "static.wikia.nocookie.net") { continue }
				sl := strings.ToLower(s)
				if strings.Contains(sl, "wikia-visualization") ||
					strings.Contains(sl, "wiki-wordmark") ||
					strings.Contains(sl, "site-logo") { continue }
				if idx := strings.Index(s, "/revision/"); idx != -1 {
					s = s[:idx]
				}
				imageURL = s
				break
			}
		}

		// ── Summary ───────────────────────────────────────────────────────────
		// first <p> in #mw-content-text
		summary := ""
		firstP, err := page.Element("#mw-content-text p")
		if err == nil && firstP != nil {
			text, _ := firstP.Text()
			summary = strings.TrimSpace(text)
			if len(summary) > 400 {
				summary = summary[:400] + "..."
			}
		}

		// ── Stats ─────────────────────────────────────────────────────────────
		// content.innerText regex per field
		pageText := ""
		contentEl, err := page.Element("#mw-content-text")
		if err == nil && contentEl != nil {
			pageText, _ = contentEl.Text()
		}

		stats := map[string]string{}
		statFields := []string{
			"Tier",
			"Attack Potency",
			"Speed",
			"Durability",
			"Stamina",
			"Range",
			"Striking Strength",
			"Lifting Strength",
			"Intelligence",
			"Standard Equipment",
		}
		for _, field := range statFields {
			re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(field) + `\s*:\s*(.+)`)
			if m := re.FindStringSubmatch(pageText); len(m) > 1 {
				val := vsbPeakClean(m[1])
				if val != "" && val != "N/A" && len(val) < 300 {
					stats[field] = val
				}
			}
		}

		// ── Name ──────────────────────────────────────────────────────────────
		name := ""
		h1, err := page.Element("h1.page-header__title")
		if err != nil || h1 == nil {
			h1, err = page.Element("#firstHeading")
		}
		if err == nil && h1 != nil {
			name, _ = h1.Text()
			name = strings.TrimSpace(name)
		}

		fmt.Printf("[Powerscale] name=%q stats=%d summary=%d\n", name, len(stats), len(summary))

		if name != "" && (len(stats) > 0 || len(summary) > 0) {
			result = &VSBatleDetail{
				Name:     name,
				ImageURL: imageURL,
				Summary:  summary,
				Stats:    stats,
				PageURL:  pageURL,
			}
			fmt.Printf("[Powerscale] ✅ Success: %s\n", name)
		}

		return nil
	})

	if err != nil {
		c.JSON(500, gin.H{"error": "browser error: " + err.Error()})
		return
	}

	if result == nil {
		c.JSON(404, gin.H{"error": "no valid character data found"})
		return
	}

	c.JSON(200, result)
}

// vsbPeakClean replicates PeakLogic.clean() from original powerscale.js
func vsbPeakClean(text string) string {
	if strings.Contains(text, "|") {
		parts := strings.Split(text, "|")
		text = parts[len(parts)-1]
	}
	text = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(text, "")
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}
