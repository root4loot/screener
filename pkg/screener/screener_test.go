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

	if !bytes.Equal(referenceImageData, result.Image) {
		t.Logf("Captured image does not match reference image.")
		t.Fatalf("Captured image does not match reference image: len(referenceImageData) = %d, len(result.Image) = %d", len(referenceImageData), len(result.Image))
	}
}
