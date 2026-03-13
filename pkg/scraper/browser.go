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

		// Rod's Set(name, value) requires NO "=" in the name.
		// Boolean flags use Append("--flag-name").
		l := launcher.New().
			Bin(chromePath).
			Headless(true).
			NoSandbox(true).
			Devtools(false).
			// DNS-over-HTTPS: routes DNS through Cloudflare HTTPS (port 443)
			// bypassing HF's broken UDP DNS resolver
			Set("dns-over-https-mode", "secure").
			Set("dns-over-https-templates", "https://cloudflare-dns.com/dns-query").
			// Container stability
			Append("--disable-dev-shm-usage").
			Append("--disable-gpu").
			Append("--no-first-run").
			Append("--no-default-browser-check").
			Append("--disable-extensions").
			Append("--disable-sync").
			Append("--mute-audio")

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