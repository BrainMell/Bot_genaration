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

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("[Powerscale] Searching for: %s\n", query)

	// ── Step 1: Fetch search page via Jina ────────────────────────────────────
	// Original: page.goto Special:Search?query=X then $$eval(".unified-search__result a")
	searchURL := fmt.Sprintf(
		"https://r.jina.ai/https://vsbattles.fandom.com/wiki/Special:Search?query=%s&ns0=1",
		url.QueryEscape(query),
	)

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "search request failed: " + err.Error()})
		return
	}
	searchBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	searchText := string(searchBody)

	fmt.Printf("[Powerscale] Search page fetched (%d bytes)\n", len(searchText))

	// ── Step 2: Extract search result links ───────────────────────────────────
	// Original CSS: .unified-search__result a
	// Jina renders as markdown links — extract and rank by query relevance
	pageURLs := extractSearchResultLinks(searchText, query)

	if len(pageURLs) == 0 {
		fmt.Printf("[Powerscale] No links found, trying direct URL fallback\n")
		directURL := "https://vsbattles.fandom.com/wiki/" + url.PathEscape(strings.ReplaceAll(query, " ", "_"))
		pageURLs = []string{directURL}
	}

	fmt.Printf("[Powerscale] Candidates: %v\n", pageURLs)

	// ── Step 3: Fetch each page via Jina and extract data ─────────────────────
	// Original: page.goto URL, page.evaluate() → image + summary + stats
	for _, pageURL := range pageURLs {
		fmt.Printf("[Powerscale] Trying: %s\n", pageURL)

		jinaPageURL := "https://r.jina.ai/" + pageURL
		r, err := client.Get(jinaPageURL)
		if err != nil {
			fmt.Printf("[Powerscale] fetch error: %v\n", err)
			continue
		}
		raw, _ := io.ReadAll(r.Body)
		r.Body.Close()
		text := string(raw)

		detail := parseVSBPage(text, pageURL)

		// Original check: hasStats || summary.length > 0
		hasStats := len(detail.Stats) > 0
		hasSummary := len(detail.Summary) > 0

		if detail.Name != "" && (hasStats || hasSummary) {
			// Original fallback: no wiki image → try SuperHero API
			if detail.ImageURL == "" {
				detail.ImageURL = fetchSuperheroImage(client, query)
			}
			fmt.Printf("[Powerscale] Success: %s (stats=%d summary=%d)\n",
				detail.Name, len(detail.Stats), len(detail.Summary))
			c.JSON(200, detail)
			return
		}
		fmt.Printf("[Powerscale] Skipping name=%q stats=%d summary=%d\n",
			detail.Name, len(detail.Stats), len(detail.Summary))
	}

	c.JSON(404, gin.H{"error": "no valid character data found"})
}

// extractSearchResultLinks replicates $$eval(".unified-search__result a", ...)
// Original filter: url.includes("/wiki/") && !url.includes("Special:") && !url.includes("Category:")
// We rank by query match so exact title matches come first
func extractSearchResultLinks(text, query string) []string {
	queryLower := strings.ToLower(strings.TrimSpace(query))

	skipPatterns := []string{
		"Special:", "Category:", "Talk:", "User:", "File:",
		"Help:", "Thread:", "Board:", "Forum:", "Blog:", "Message_Wall",
	}

	isSkipped := func(u string) bool {
		for _, s := range skipPatterns {
			if strings.Contains(u, s) {
				return true
			}
		}
		return false
	}

	seen := map[string]bool{}
	var exact []string
	var strong []string
	var weak []string

	addLink := func(rawURL, title string) {
		// Strip query/fragment
		if idx := strings.Index(rawURL, "?"); idx != -1 {
			rawURL = rawURL[:idx]
		}
		if idx := strings.Index(rawURL, "#"); idx != -1 {
			rawURL = rawURL[:idx]
		}
		if isSkipped(rawURL) || seen[rawURL] {
			return
		}
		seen[rawURL] = true

		titleNorm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(title), "_", " "))

		if titleNorm == queryLower {
			exact = append(exact, rawURL)
		} else if strings.HasPrefix(titleNorm, queryLower) || strings.Contains(titleNorm, queryLower) {
			strong = append(strong, rawURL)
		} else {
			weak = append(weak, rawURL)
		}
	}

	// Parse markdown links: [Title](URL)
	reMdLink := regexp.MustCompile(`\[([^\]]+)\]\((https://vsbattles\.fandom\.com/wiki/[^)]+)\)`)
	for _, m := range reMdLink.FindAllStringSubmatch(text, -1) {
		addLink(m[2], m[1])
	}

	// Parse bare URLs
	reRawLink := regexp.MustCompile(`https://vsbattles\.fandom\.com/wiki/([^\s"'\])<]+)`)
	for _, u := range reRawLink.FindAllString(text, -1) {
		parts := strings.Split(u, "/wiki/")
		title := ""
		if len(parts) > 1 {
			title = strings.ReplaceAll(parts[1], "_", " ")
		}
		addLink(u, title)
	}

	result := append(exact, append(strong, weak...)...)
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

// parseVSBPage replicates the full page.evaluate() block from scrapeVSBPage()
// Fields: image (pi-image-thumbnail → data-src → strip /revision/),
//         summary (first <p> in #mw-content-text),
//         stats (innerText regex for each field)
func parseVSBPage(text, pageURL string) *VSBatleDetail {
	detail := &VSBatleDetail{
		Stats:   make(map[string]string),
		PageURL: pageURL,
	}

	lines := strings.Split(text, "\n")

	// Name from first H1 (# Title)
	reH1 := regexp.MustCompile(`^#\s+(.+)`)
	for _, line := range lines {
		if m := reH1.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
			name := strings.TrimSpace(m[1])
			name = regexp.MustCompile(`[*_~` + "`" + `]`).ReplaceAllString(name, "")
			detail.Name = name
			break
		}
	}

	// Summary — first paragraph equivalent (original: content.querySelector("p").innerText)
	reClean := regexp.MustCompile(`\[.*?\]|\(.*?\)`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 50 {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") ||
			strings.HasPrefix(line, "!") || strings.HasPrefix(line, "http") ||
			strings.HasPrefix(line, ">") || strings.HasPrefix(line, "-") ||
			strings.HasPrefix(line, "*") {
			continue
		}
		candidate := reClean.ReplaceAllString(line, "")
		candidate = strings.TrimSpace(candidate)
		if len(candidate) > 50 {
			if len(candidate) > 400 {
				candidate = candidate[:400] + "..."
			}
			detail.Summary = candidate
			break
		}
	}

	// Stats — original statFields forEach with content.innerText.match(field + ":\s*(.+)")
	// Fields from original powerscale.js + olderindex.js combined
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
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			val := peakLogicClean(m[1])
			if val != "" && val != "N/A" && len(val) < 300 {
				detail.Stats[field] = val
			}
		}
	}

	// Image — original: img.pi-image-thumbnail → data-src || src → strip /revision/
	// Prefer vsbattles-specific images first
	reVSBImg := regexp.MustCompile(`https://static\.wikia\.nocookie\.net/vsbattles/images/[^\s"'\])<]+\.(?:jpg|jpeg|png|gif|webp)`)
	reAnyImg := regexp.MustCompile(`https://static\.wikia\.nocookie\.net/[^\s"'\])<]+\.(?:jpg|jpeg|png|gif|webp)`)

	findImage := func(re *regexp.Regexp) string {
		for _, img := range re.FindAllString(text, -1) {
			lower := strings.ToLower(img)
			if strings.Contains(lower, "wikia-visualization") ||
				strings.Contains(lower, "wiki-wordmark") ||
				strings.Contains(lower, "site-logo") ||
				strings.Contains(lower, "favicon") ||
				strings.Contains(lower, "avatar") {
				continue
			}
			// Strip /revision/ suffix — original: rawUrl.substring(0, revisionIndex)
			if idx := strings.Index(img, "/revision/"); idx != -1 {
				img = img[:idx]
			}
			return img
		}
		return ""
	}

	detail.ImageURL = findImage(reVSBImg)
	if detail.ImageURL == "" {
		detail.ImageURL = findImage(reAnyImg)
	}

	return detail
}

// peakLogicClean replicates PeakLogic.clean() from original powerscale.js:
//   text.split('|').pop().trim()
//   .replace(/\([^)]+\)/g, "")
//   .replace(/\[[^\]]+\]/g, "")
func peakLogicClean(text string) string {
	if strings.Contains(text, "|") {
		parts := strings.Split(text, "|")
		text = parts[len(parts)-1]
	}
	text = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(text, "")
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "N/A"
	}
	return text
}

// fetchSuperheroImage replicates the SuperHero API fallback from olderindex.js:
//   axios.get(`https://superheroapi.com/api/.../search/${character}`)
//   → results[0].image.url
func fetchSuperheroImage(client *http.Client, character string) string {
	apiURL := fmt.Sprintf(
		"https://superheroapi.com/api/6e934a5989d474065c897c8fcc68df21/search/%s",
		url.PathEscape(character),
	)
	r, err := client.Get(apiURL)
	if err != nil {
		return ""
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)

	reImg := regexp.MustCompile(`"url"\s*:\s*"(https://[^"]+)"`)
	if m := reImg.FindSubmatch(body); len(m) > 1 {
		fmt.Printf("[Powerscale] SuperHero API image: %s\n", string(m[1]))
		return string(m[1])
	}
	return ""
}