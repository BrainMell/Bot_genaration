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

		chromePath := os.Getenv("CHROME_PATH")
		if chromePath == "" {
			chromePath = "/usr/bin/chromium-browser"
		}

		fmt.Printf("📂 Using System Chromium: %s\n", chromePath)

		l := launcher.New().
			Bin(chromePath).
			Headless(true).
			NoSandbox(true).
			Devtools(false).
			// --- DNS-over-HTTPS: bypasses HF broken system DNS resolver ---
			// Routes all DNS through Cloudflare DoH (port 443, always open)
			Append("--dns-over-https-mode=secure").
			Append("--dns-over-https-templates=https://cloudflare-dns.com/dns-query").
			// --- Container stability flags ---
			Append("--disable-dev-shm-usage").   // Prevents crashes in low /dev/shm Docker envs
			Append("--disable-gpu").              // No GPU in container
			Append("--no-first-run").             // Skip first-run setup
			Append("--no-default-browser-check"). // Skip browser check dialog
			Append("--disable-extensions").       // No extensions needed
			Append("--disable-background-networking"). // Reduce unnecessary network calls
			Append("--disable-sync").             // No Google sync
			Append("--metrics-recording-only").   // Disable reporting
			Append("--mute-audio")                // No audio needed

		u, err := l.Launch()
		if err != nil {
			fmt.Printf("❌ Failed to launch system chromium: %v\n", err)
			return
		}

		browser = rod.New().
			ControlURL(u).
			MustConnect()

		// 3 concurrent tabs — balanced for HF Spaces RAM
		pagePool = make(chan *rod.Page, 3)

		fmt.Println("✅ Rod Browser Engine Ready!")
	})
}

// WithPage safely acquires a page from the pool and runs an action.
// It recovers from panics so a single bad request can't crash the service.
func WithPage(action func(*rod.Page) error) (err error) {
	if browser == nil {
		InitBrowser()
	}

	if browser == nil {
		return fmt.Errorf("browser engine not initialized")
	}

	// Acquire concurrency slot
	pagePool <- nil
	defer func() { <-pagePool }()

	// Recover from rod panics (e.g. MustNavigate on DNS failure)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("browser panic: %v", r)
		}
	}()

	page, createErr := browser.MustIncognito().Page(proto.TargetCreateTarget{URL: "about:blank"})
	if createErr != nil {
		return fmt.Errorf("failed to create page: %v", createErr)
	}
	defer page.MustClose()

	return action(page)
}

func CloseBrowser() {
	if browser != nil {
		browser.MustClose()
	}
}