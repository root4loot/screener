package screener

import (
	"bytes"
	_ "embed"
	"net/url"
	"testing"
)

//go:embed assets/screenshot_without_text.png
var referenceImageData []byte

func TestCaptureScreenshot(t *testing.T) {
	options := NewOptions()
	options.CaptureWidth = 1080
	options.CaptureHeight = 720

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

	if !bytes.Equal(referenceImageData, result.Image) {
		t.Fatal("Captured image does not match reference image")
	}
}
