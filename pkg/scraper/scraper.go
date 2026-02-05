package scraper

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
    "github.com/gin-gonic/gin"
)

// --- Pinterest Scraper ---

type PinterestResponse struct {
    Images []string `json:"images"`
}

func SearchPinterest(c *gin.Context) {
    query := c.Query("query")
    if query == "" {
        c.JSON(400, gin.H{"error": "Query required"})
        return
    }

    collector := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
    )

    var images []string

    collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
        src := e.Attr("src")
        if strings.Contains(src, "pinimg.com") && !strings.Contains(src, "75x75") {
            // Upgrade quality
            highRes := strings.Replace(src, "236x", "originals", 1)
            images = append(images, highRes)
        }
    })

    url := "https://www.pinterest.com/search/pins/?q=" + url.QueryEscape(query)
    collector.Visit(url)

    // Dedupe
    unique := make(map[string]bool)
    var result []string
    for _, img := range images {
        if !unique[img] {
            unique[img] = true
            result = append(result, img)
        }
    }

    c.JSON(200, PinterestResponse{Images: result})
}

// --- VS Battles Scraper ---

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
}

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

func GetVSBattlesDetail(c *gin.Context) {
    pageURL := c.Query("url")
    if pageURL == "" {
        c.JSON(400, gin.H{"error": "URL required"})
        return
    }

    collector := colly.NewCollector()
    var detail VSBDetailResponse

    collector.OnHTML("h1.page-header__title", func(e *colly.HTMLElement) {
        detail.Name = strings.TrimSpace(e.Text)
    })

    collector.OnHTML("aside.portable-infobox figure.pi-item.pi-image img", func(e *colly.HTMLElement) {
        if detail.ImageURL == "" {
            detail.ImageURL = e.Attr("src")
        }
    })

    collector.OnHTML("div.mw-parser-output p", func(e *colly.HTMLElement) {
        text := e.Text
        if strings.Contains(text, "Tier:") {
            detail.Tier = cleanText(strings.Split(text, ":")[1])
        }
        if strings.Contains(text, "Attack Potency:") {
            detail.AttackPotency = cleanText(strings.Split(text, ":")[1])
        }
        if strings.Contains(text, "Speed:") {
            detail.Speed = cleanText(strings.Split(text, ":")[1])
        }
        if strings.Contains(text, "Durability:") {
            detail.Durability = cleanText(strings.Split(text, ":")[1])
        }
        if strings.Contains(text, "Stamina:") {
            detail.Stamina = cleanText(strings.Split(text, ":")[1])
        }
        if strings.Contains(text, "Range:") {
            detail.Range = cleanText(strings.Split(text, ":")[1])
        }
        if detail.Summary == "" && len(text) > 50 && !strings.Contains(text, ":") {
            detail.Summary = cleanText(text)
        }
    })

    collector.Visit(pageURL)
    c.JSON(200, detail)
}

// --- Rule34 Scraper (Go Implementation) ---

type Rule34Response struct {
    Images []string `json:"images"`
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

    // 1. Try API First
    apiURL := "https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=20&tags=" + url.QueryEscape(query)
    
    resp, err := http.Get(apiURL)
    if err == nil {
        defer resp.Body.Close()
        var posts []Rule34Post
        if err := json.NewDecoder(resp.Body).Decode(&posts); err == nil && len(posts) > 0 {
            var images []string
            for _, p := range posts {
                if p.FileURL != "" {
                    images = append(images, p.FileURL)
                }
            }
            c.JSON(200, Rule34Response{Images: images})
            return
        }
    }

    // 2. Fallback to Web Scraping (Colly) if API fails or returns empty
    collector := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
    )

    var images []string

    // On Rule34 list page, thumbnails wrap links to posts
    // We can grab thumbnails directly or follow links. 
    // Faster: Extract Post ID from thumb link, construct image URL guessing logic, or visit post.
    // For speed, we'll visit the first 5 posts in parallel.
    
    detailCollector := collector.Clone()
    detailCollector.OnHTML("#image", func(e *colly.HTMLElement) {
        images = append(images, e.Attr("src"))
    })
    detailCollector.OnHTML("video source", func(e *colly.HTMLElement) {
        images = append(images, e.Attr("src"))
    })

    collector.OnHTML(".thumb a", func(e *colly.HTMLElement) {
        link := e.Attr("href")
        if strings.Contains(link, "id=") {
            detailCollector.Visit(e.Request.AbsoluteURL(link))
        }
    })

    scrapeURL := "https://rule34.xxx/index.php?page=post&s=list&tags=" + url.QueryEscape(query)
    collector.Visit(scrapeURL)

    c.JSON(200, Rule34Response{Images: images})
}

func cleanText(s string) string {
    s = strings.TrimSpace(s)
    re := regexp.MustCompile(`\[.*?\]`)
    s = re.ReplaceAllString(s, "")
    return s
}