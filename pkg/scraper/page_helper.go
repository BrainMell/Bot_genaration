package scraper

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

// WithPage opens a fresh browser tab, runs fn, then always closes the tab.
// Use this in your scrapers to avoid leaking Browserless units:
//
//	err := WithPage(func(page *rod.Page) error {
//	    page.MustNavigate("https://example.com")
//	    // ... scrape ...
//	    return nil
//	})
func WithPage(fn func(page *rod.Page) error) error {
	page := Browser.MustPage("")
	defer func() {
		if err := page.Close(); err != nil {
			fmt.Printf("[BROWSER] Warning: failed to close page: %v\n", err)
		}
	}()
	return fn(page)
}

// WithPageTimeout is like WithPage but with a custom timeout.
// If fn doesn't finish within d, the page is closed and an error is returned.
func WithPageTimeout(d time.Duration, fn func(page *rod.Page) error) error {
	page := Browser.MustPage("").Timeout(d)
	defer func() {
		if err := page.Close(); err != nil {
			fmt.Printf("[BROWSER] Warning: failed to close page: %v\n", err)
		}
	}()
	return fn(page)
}
