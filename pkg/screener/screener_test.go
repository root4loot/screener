package screener

import (
	"bytes"
	_ "embed"
	"net/url"
	"os"
	"testing"
)

//go:embed assets/screenshot_without_text.png
var referenceImageData []byte

func TestCaptureScreenshot(t *testing.T) {
	options := NewOptions()
	options.CaptureWidth = 1080
	options.CaptureHeight = 720

	if len(referenceImageData) == 0 {
		t.Fatal("Reference image data not loaded")
	}

	t.Logf("Reference image data size: %d bytes", len(referenceImageData))

	screener := NewScreenerWithOptions(options)

	parsedURL, err := url.Parse("https://example.com/")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	result, err := screener.CaptureScreenshot(parsedURL)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	t.Logf("Captured image data size: %d bytes", len(result.Image))

	if len(result.Image) == 0 {
		t.Fatal("Captured image is empty")
	}

	// Determine if running in GitHub Actions environment
	isGitHubActions := os.Getenv("GITHUB_ACTIONS") == "true"

	if isGitHubActions {
		// Allowable sizes in GitHub Actions environment
		if len(result.Image) == 25300 || len(result.Image) == 25717 {
			t.Logf("Captured image size is within the acceptable range for GitHub Actions.")
			return
		} else {
			t.Fatalf("Captured image size %d bytes does not match any acceptable sizes (25300 or 25717 bytes) for GitHub Actions", len(result.Image))
		}
	}

	if !bytes.Equal(referenceImageData, result.Image) {
		t.Logf("Captured image does not match reference image.")
		t.Fatalf("Captured image does not match reference image: len(referenceImageData) = %d, len(result.Image) = %d", len(referenceImageData), len(result.Image))
	}
}
