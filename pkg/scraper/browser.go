package scraper

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Browser is the shared remote/local browser instance used by all scrapers.
var Browser *rod.Browser

// InitBrowser connects to Browserless if BROWSERLESS_TOKEN is set,
// otherwise falls back to a local Chromium launch (useful for local dev).
func InitBrowser() {
	token := os.Getenv("BROWSERLESS_TOKEN")

	if token != "" {
		connectBrowserless(token)
	} else {
		connectLocal()
	}
}

// CloseBrowser gracefully disconnects from the browser.
func CloseBrowser() {
	if Browser != nil {
		Browser.MustClose()
	}
}

// connectBrowserless connects Rod to a remote Browserless.io Chrome instance.
// This uses zero local RAM — all browser work runs on Browserless servers.
func connectBrowserless(token string) {
	wsURL := fmt.Sprintf("wss://production-sfo.browserless.io/chromium?token=%s", token)

	fmt.Println("[BROWSER] Connecting to Browserless remote Chrome...")

	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		Browser, err = tryConnect(wsURL)
		if err == nil {
			fmt.Println("[BROWSER] ✅ Connected to Browserless.")
			return
		}
		fmt.Printf("[BROWSER] Attempt %d failed: %v — retrying...\n", attempt, err)
		time.Sleep(2 * time.Second)
	}

	// If Browserless is down, panic loudly so we know immediately.
	panic(fmt.Sprintf("[BROWSER] ❌ Could not connect to Browserless after 3 attempts: %v", err))
}

// connectLocal launches a local headless Chromium (fallback for local dev).
func connectLocal() {
	fmt.Println("[BROWSER] BROWSERLESS_TOKEN not set — launching local Chromium (dev mode).")

	u := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("disable-dev-shm-usage").
		MustLaunch()

	Browser = rod.New().ControlURL(u).MustConnect()
	fmt.Println("[BROWSER] ✅ Local Chromium launched.")
}

// tryConnect attempts a single WebSocket connection to the given URL.
func tryConnect(wsURL string) (*rod.Browser, error) {
	b := rod.New().ControlURL(wsURL)
	if err := b.Connect(); err != nil {
		return nil, err
	}
	return b, nil
}

// NewPage returns a fresh page from the shared browser.
// Each call opens a new tab. Always call page.MustClose() when done
// to avoid burning Browserless units unnecessarily.
func NewPage() *rod.Page {
	return Browser.MustPage("")
}
