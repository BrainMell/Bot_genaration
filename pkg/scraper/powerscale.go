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

func ScrapePowerscale(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "Character name required"})
		return
	}

	fmt.Printf("[Powerscale] Searching for: %s\n", query)

	var result *VSBatleDetail

	err := WithPage(func(page *rod.Page) error {

		// ── Step 1: Navigate to search page ───────────────────────────────────
		// Original: page.goto(searchUrl, { waitUntil: "domcontentloaded" })
		searchURL := fmt.Sprintf(
			"https://vsbattles.fandom.com/wiki/Special:Search?query=%s&ns0=1",
			url.QueryEscape(query),
		)
		fmt.Printf("[Powerscale] Navigating to search: %s\n", searchURL)
		page.MustNavigate(searchURL).MustWaitLoad()

		// ── Step 2: Extract search result links ───────────────────────────────
		// Original: $$eval(".unified-search__result a", links => links.map(a => a.href)
		//           .filter(h => h.includes("/wiki/") && !h.includes("Special:") && !h.includes("Category:"))
		linkEls, err := page.Elements(".unified-search__result__link")
		if err != nil || len(linkEls) == 0 {
			linkEls, _ = page.Elements(".unified-search__result a")
		}

		var pageURLs []string
		seen := map[string]bool{}
		for _, el := range linkEls {
			href, err := el.Attribute("href")
			if err != nil || href == nil {
				continue
			}
			u := *href
			if strings.HasPrefix(u, "/") {
				u = "https://vsbattles.fandom.com" + u
			}
			if !strings.Contains(u, "/wiki/") { continue }
			if strings.Contains(u, "Special:") { continue }
			if strings.Contains(u, "Category:") { continue }
			if strings.Contains(u, "Talk:") { continue }
			if strings.Contains(u, "User:") { continue }
			if strings.Contains(u, "File:") { continue }
			if seen[u] { continue }
			seen[u] = true
			pageURLs = append(pageURLs, u)
			fmt.Printf("[Powerscale] Found link: %s\n", u)
			if len(pageURLs) >= 5 { break }
		}

		if len(pageURLs) == 0 {
			direct := "https://vsbattles.fandom.com/wiki/" + url.PathEscape(strings.ReplaceAll(query, " ", "_"))
			fmt.Printf("[Powerscale] No search results, trying direct: %s\n", direct)
			pageURLs = []string{direct}
		}

		// ── Step 3: Visit each result until valid data found ──────────────────
		// Original: for (const url of searchResults) { page.goto, page.evaluate() }
		for _, pageURL := range pageURLs {
			fmt.Printf("[Powerscale] Trying: %s\n", pageURL)

			err := page.Navigate(pageURL)
			if err != nil {
				fmt.Printf("[Powerscale] navigate error: %v\n", err)
				continue
			}
			page.MustWaitLoad()

			// Original: page.evaluate(() => window.scrollTo(0, document.body.scrollHeight / 2))
			// + await new Promise(resolve => setTimeout(resolve, 2000))
			page.MustEval(`() => window.scrollTo(0, document.body.scrollHeight / 2)`)
			time.Sleep(2 * time.Second)

			// ── Image ─────────────────────────────────────────────────────────
			// Original: img = content.querySelector('img.pi-image-thumbnail')
			// rawUrl = img.dataset.src || img.src
			// out.imageUrl = rawUrl.substring(0, revisionIndex)
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

			// Fallback: any wikia image in article content
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

			// ── Summary ───────────────────────────────────────────────────────
			// Original: const firstP = content.querySelector("p")
			// out.summary = firstP ? firstP.innerText : ""
			summary := ""
			firstP, err := page.Element("#mw-content-text p")
			if err == nil && firstP != nil {
				text, _ := firstP.Text()
				summary = strings.TrimSpace(text)
				if len(summary) > 400 {
					summary = summary[:400] + "..."
				}
			}

			// ── Stats ─────────────────────────────────────────────────────────
			// Original: statFields.forEach(field => { content.innerText.match(field + ":\s*(.+)") })
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

			// ── Name ──────────────────────────────────────────────────────────
			name := ""
			h1, err := page.Element("h1.page-header__title")
			if err != nil || h1 == nil {
				h1, err = page.Element("#firstHeading")
			}
			if err == nil && h1 != nil {
				name, _ = h1.Text()
				name = strings.TrimSpace(name)
			}

			// Original check: hasStats || summary.length > 0
			hasStats := len(stats) > 0
			hasSummary := len(summary) > 0

			fmt.Printf("[Powerscale] name=%q stats=%d summary=%d\n", name, len(stats), len(summary))

			if name != "" && (hasStats || hasSummary) {
				result = &VSBatleDetail{
					Name:     name,
					ImageURL: imageURL,
					Summary:  summary,
					Stats:    stats,
					PageURL:  pageURL,
				}
				fmt.Printf("[Powerscale] ✅ Success: %s\n", name)
				return nil
			}

			fmt.Printf("[Powerscale] ❌ Skipping, trying next...\n")
		}

		return nil
	})

	if err != nil {
		fmt.Printf("[Powerscale] Rod error: %v\n", err)
		c.JSON(500, gin.H{"error": "browser error: " + err.Error()})
		return
	}

	if result == nil {
		c.JSON(404, gin.H{"error": "no valid character data found"})
		return
	}

	c.JSON(200, result)
}

// vsbPeakClean replicates PeakLogic.clean() from original powerscale.js:
//   text.split('|').pop().trim()
//   .replace(/\([^)]+\)/g, "")
//   .replace(/\[[^\]]+\]/g, "")
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