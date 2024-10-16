package screener

import (
	_ "embed"
	"net/url"
	"testing"
)

func TestCaptureScreenshot(t *testing.T) {
	screener := NewScreener()

	urlsToTest := []string{
		"https://example.com",
		"https://google.com/robots.txt",
	}

	for _, urlStr := range urlsToTest {
		t.Run(urlStr, func(t *testing.T) {
			parsedURL, err := url.Parse(urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL %s: %v", urlStr, err)
			}

			result, err := screener.CaptureScreenshot(parsedURL)
			if err != nil {
				t.Fatalf("Failed to capture screenshot for %s: %v", urlStr, err)
			}
			if result == nil {
				t.Fatalf("Result is nil for %s", urlStr)
			}

			if len(result.Image) == 0 {
				t.Fatalf("Captured image is empty for %s", urlStr)
			}

			if result.StatusCode != 200 {
				t.Fatalf("Expected status code 200 for %s, got %d", urlStr, result.StatusCode)
			}

			t.Logf("Successfully captured screenshot for %s with status code %d", urlStr, result.StatusCode)
		})
	}
}
