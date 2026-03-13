package scraper

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var (
	browser     *rod.Browser
	browserOnce sync.Once
	pagePool    chan *rod.Page
)

// InitBrowser initializes the global Rod browser instance.
func InitBrowser() {
	browserOnce.Do(func() {
		fmt.Println("🚀 Initializing Rod Browser Engine...")

		// FORCED: Use the Chromium binary installed in the Docker container
		// Alpine Chromium path: /usr/bin/chromium-browser
		chromePath := os.Getenv("CHROME_PATH")
		if chromePath == "" {
			chromePath = "/usr/bin/chromium-browser"
		}

		fmt.Printf("📂 Using System Chromium: %s\n", chromePath)

		// Create a launcher that STRICTLY uses the existing binary
		l := launcher.New().
			Bin(chromePath).
			Headless(true).
			NoSandbox(true).
			Devtools(false)

		// Launch the browser
		u, err := l.Launch()
		if err != nil {
			fmt.Printf("❌ Failed to launch system chromium: %v\n", err)
			// Fallback: try default launcher logic but this likely won't work on HF
			return
		}

		browser = rod.New().
			ControlURL(u).
			MustConnect()

		// Limit to 2 concurrent tabs to save RAM on HF Spaces
		pagePool = make(chan *rod.Page, 2)

		fmt.Println("✅ Rod Browser Engine Ready!")
	})
}

// WithPage safely acquires a page from the browser.
func WithPage(action func(*rod.Page) error) error {
	if browser == nil {
		InitBrowser()
	}
	
	if browser == nil {
		return fmt.Errorf("browser engine not initialized")
	}

	// Acquire slot (concurrency control)
	pagePool <- nil
	defer func() { <-pagePool }()

	// Create new page
	page, err := browser.MustIncognito().Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return fmt.Errorf("failed to create page: %v", err)
	}
	defer page.MustClose()

	return action(page)
}

func CloseBrowser() {
	if browser != nil {
		browser.MustClose()
	}
}
