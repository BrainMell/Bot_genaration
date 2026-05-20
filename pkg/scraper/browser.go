package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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

// connectBrowserless connects Rod to Browserless v2 by first fetching
// /json/version to get the real CDP WebSocket debugger URL, then connecting
// Rod to that URL. This is the correct approach per the Browserless docs for
// Go libraries — direct wss:// connections fail due to WebSocket handshake
// incompatibilities between Rod's nhooyr.io/websocket and Browserless v2.
func connectBrowserless(token string) {
	fmt.Println("[BROWSER] Fetching Browserless CDP endpoint via /json/version...")

	// Step 1: GET /json/version to resolve the actual debugger WebSocket URL.
	// Browserless mocks the Chrome DevTools /json/version payload and returns
	// a webSocketDebuggerUrl that Rod can connect to via ControlURL.
	versionURL := fmt.Sprintf("https://production-sfo.browserless.io/json/version?token=%s", token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(versionURL)
	if err != nil {
		panic(fmt.Sprintf("[BROWSER] ❌ Failed to reach Browserless /json/version: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("[BROWSER] ❌ /json/version returned HTTP %d — check your BROWSERLESS_TOKEN", resp.StatusCode))
	}

	var versionData struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionData); err != nil {
		panic(fmt.Sprintf("[BROWSER] ❌ Failed to parse /json/version JSON: %v", err))
	}

	if versionData.WebSocketDebuggerURL == "" {
		panic("[BROWSER] ❌ webSocketDebuggerUrl missing from /json/version response")
	}

	fmt.Printf("[BROWSER] Got debugger URL: %s\n", versionData.WebSocketDebuggerURL[:min(len(versionData.WebSocketDebuggerURL), 60)])

	// Step 2: Connect Rod to the actual CDP debugger WebSocket URL.
	// Browserless strips the token from the webSocketDebuggerUrl it returns,
	// so we must re-append it or the connection gets a 401 Unauthorized.
	debuggerURL := versionData.WebSocketDebuggerURL
	if !strings.Contains(debuggerURL, "token=") {
		sep := "?"
		if strings.Contains(debuggerURL, "?") {
			sep = "&"
		}
		debuggerURL = debuggerURL + sep + "token=" + token
	}
	fmt.Printf("[BROWSER] Connecting to debugger URL...\n")
	Browser = rod.New().ControlURL(debuggerURL).MustConnect()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
