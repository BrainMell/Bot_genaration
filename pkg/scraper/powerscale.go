package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	client := &http.Client{Timeout: 20 * time.Second}

	fmt.Printf("[Powerscale] Searching for: %s\n", query)

	// Step 1: Search via Jina proxy
	searchURL := fmt.Sprintf(
		"https://r.jina.ai/https://vsbattles.fandom.com/wiki/Special:Search?query=%s",
		url.QueryEscape(query),
	)
	resp, err := client.Get(searchURL)
	if err != nil {
		c.JSON(500, gin.H{"error": "search request failed: " + err.Error()})
		return
	}
	searchBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Extract wiki page URLs from search results
	reLinks := regexp.MustCompile(`https://vsbattles\.fandom\.com/wiki/([^)\s"'\]]+)`)
	allMatches := reLinks.FindAllString(string(searchBody), -1)

	// Deduplicate and filter out Special/Category/Talk pages
	seen := map[string]bool{}
	var pageURLs []string
	for _, u := range allMatches {
		if seen[u] { continue }
		if strings.Contains(u, "Special:") { continue }
		if strings.Contains(u, "Category:") { continue }
		if strings.Contains(u, "Talk:") { continue }
		if strings.Contains(u, "User:") { continue }
		if strings.Contains(u, "File:") { continue }
		seen[u] = true
		pageURLs = append(pageURLs, u)
		if len(pageURLs) >= 3 { break }
	}

	if len(pageURLs) == 0 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("no results found for '%s'", query)})
		return
	}

	// Step 2: Fetch each page via Jina and extract stats
	for _, pageURL := range pageURLs {
		fmt.Printf("[Powerscale] Trying: %s\n", pageURL)

		jinaURL := "https://r.jina.ai/" + pageURL
		r, err := client.Get(jinaURL)
		if err != nil {
			fmt.Printf("[Powerscale] fetch error: %v\n", err)
			continue
		}
		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		text := string(raw)

		detail := parseVSBPage(text, pageURL)

		if detail.Name != "" && len(detail.Stats) > 0 {
			fmt.Printf("[Powerscale] Success: %s\n", detail.Name)
			c.JSON(200, detail)
			return
		}
		fmt.Printf("[Powerscale] Skipping (insufficient data)\n")
	}

	c.JSON(404, gin.H{"error": "no valid character data found"})
}

func parseVSBPage(text, pageURL string) *VSBatleDetail {
	detail := &VSBatleDetail{
		Stats:   make(map[string]string),
		PageURL: pageURL,
	}

	lines := strings.Split(text, "\n")

	// Extract name from first H1 heading
	reH1 := regexp.MustCompile(`^#\s+(.+)`)
	for _, line := range lines {
		if m := reH1.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
			detail.Name = strings.TrimSpace(m[1])
			break
		}
	}

	// Extract summary — first paragraph with >50 chars
	reClean := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 50 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "|") {
			detail.Summary = reClean.ReplaceAllString(line, "")
			detail.Summary = strings.TrimSpace(detail.Summary)
			if len(detail.Summary) > 50 {
				if len(detail.Summary) > 400 {
					detail.Summary = detail.Summary[:400] + "..."
				}
				break
			}
		}
	}

	// Extract stats via regex
	statFields := map[string]string{
		"Tier":             "Tier",
		"Attack Potency":   "Attack Potency",
		"Speed":            "Speed",
		"Durability":       "Durability",
		"Stamina":          "Stamina",
		"Range":            "Range",
		"Lifting Strength": "Lifting Strength",
		"Intelligence":     "Intelligence",
	}

	for field, key := range statFields {
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(field) + `\s*[:\|]\s*(.+)`)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			val := cleanVSBText(m[1])
			if val != "" && len(val) < 200 {
				detail.Stats[key] = val
			}
		}
	}

	// Extract image URL
	reImg := regexp.MustCompile(`https://static\.wikia\.nocookie\.net/[^\s"')]+\.(?:jpg|jpeg|png|gif|webp)`)
	if imgs := reImg.FindAllString(text, -1); len(imgs) > 0 {
		for _, img := range imgs {
			if !strings.Contains(img, "Wikia-Visualization") &&
				!strings.Contains(img, "Wiki-wordmark") &&
				!strings.Contains(img, "Site-logo") {
				// Strip revision params
				if idx := strings.Index(img, "/revision/"); idx != -1 {
					img = img[:idx]
				}
				detail.ImageURL = img
				break
			}
		}
	}

	return detail
}

func cleanVSBText(text string) string {
	// Take last segment after '|' (peak logic)
	if strings.Contains(text, "|") {
		parts := strings.Split(text, "|")
		text = parts[len(parts)-1]
	}
	// Remove refs and parens
	text = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(text, "")
	// Trim at newline
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}