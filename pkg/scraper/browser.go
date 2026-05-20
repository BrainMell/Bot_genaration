package scraper

import (
	"fmt"
	"os"

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

// connectBrowserless connects Rod to Browserless using the managed launcher.
// This hits /json/version first (HTTP) to resolve the actual CDP WebSocket URL,
// which is required — directly passing a wss:// URL causes a bad handshake.
func connectBrowserless(token string) {
	// Use HTTP URL — launcher.MustNewManaged resolves it to the real WS endpoint
	serviceURL := fmt.Sprintf("wss://production-sfo.browserless.io?token=%s", token)

	fmt.Println("[BROWSER] Connecting to Browserless via managed launcher...")

	l := launcher.MustNewManaged(serviceURL)
	l.Headless(true)

	Browser = rod.New().Client(l.MustClient()).MustConnect()
	fmt.Println("[BROWSER] ✅ Connected to Browserless.")
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

// NewPage returns a fresh page from the shared browser.
// Always call page.MustClose() when done to avoid burning Browserless units.
func NewPage() *rod.Page {
	return Browser.MustPage("")
}
