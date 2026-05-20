package scraper

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Browser is the shared remote/local browser instance used by all scrapers.
var Browser *rod.Browser

// InitBrowser connects to Browserless if BROWSERLESS_TOKEN is set,
// otherwise falls back to a local Chromium launch (useful for local dev).
func InitBrowser() {
	token := strings.TrimSpace(os.Getenv("BROWSERLESS_TOKEN"))
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

// connectBrowserless connects Rod to Browserless v2 via raw WebSocket (CDP).
// We use direct ControlURL because Rod's launcher.MustNewManaged is NOT 
// compatible with Browserless v2's connection protocol.
func connectBrowserless(token string) {
	// For Browserless v2, the path /chromium is the standard CDP endpoint.
	wsURL := fmt.Sprintf("wss://production-sfo.browserless.io/chromium?token=%s", token)

	fmt.Printf("[BROWSER] Connecting to Browserless: wss://production-sfo.browserless.io/chromium?token=%s...\n", token[:4])

	// Initialize the browser with the WebSocket URL
	Browser = rod.New().ControlURL(wsURL)
	
	// Attempt to connect and capture potential handshake errors
	err := Browser.Connect()
	if err != nil {
		fmt.Printf("[BROWSER] ❌ Connection error: %v\n", err)
		fmt.Println("[BROWSER] Troubleshooting: Ensure BROWSERLESS_TOKEN is correct and not expired.")
		panic(err)
	}
	
	fmt.Println("[BROWSER] ✅ Successfully connected to Browserless v2.")
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
