package main

import (
	"net/url"
	"os"
	"testing"

	"github.com/root4loot/screener/pkg/screener"
)

func TestWorker(t *testing.T) {
	cli := NewCLI()
	cli.Screener = screener.NewScreenerWithOptions(screener.NewOptions())

	testURL := "https://example.com"
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse test URL: %v", err)
	}

	result, err := cli.Screener.CaptureScreenshot(parsedURL)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	if result.TargetURL != testURL+"/" {
		t.Errorf("Expected TargetURL to be %s, got %s", testURL, result.TargetURL)
	}

	testURL = "https://example.com/robots.txt"
	parsedURL, err = url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse test URL: %v", err)
	}

	result, err = cli.Screener.CaptureScreenshot(parsedURL)
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	if result.TargetURL != testURL {
		t.Errorf("Expected TargetURL to be %s, got %s", testURL, result.TargetURL)
	}
}

func TestParseFlags(t *testing.T) {
	cli := NewCLI()
	args := []string{"-t", "https://example.com", "-c", "5", "-o", "./output"}
	os.Args = append([]string{"cmd"}, args...)
	cli.parseFlags()

	if cli.TargetURL != "https://example.com" {
		t.Errorf("Expected TargetURL to be 'https://example.com', got %s", cli.TargetURL)
	}

	if cli.Concurrency != 5 {
		t.Errorf("Expected Concurrency to be 5, got %d", cli.Concurrency)
	}

	if cli.SaveScreenshotFolder != "./output" {
		t.Errorf("Expected SaveScreenshotFolder to be './output', got %s", cli.SaveScreenshotFolder)
	}
}

// TestWorkerWithInvalidURL tests handling of an invalid URL.
func TestWorkerWithInvalidURL(t *testing.T) {
	cli := NewCLI()
	cli.Screener = screener.NewScreenerWithOptions(screener.NewOptions())

	invalidURL := "://invalid-url"
	cli.worker(invalidURL)

	if len(results) != 0 {
		t.Errorf("Expected no results for invalid URL, got %d", len(results))
	}
}
