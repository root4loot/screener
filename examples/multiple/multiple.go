package main

import (
	"fmt"

	"github.com/root4loot/screener"
)

func main() {
	// List of URLs to capture
	urls := []string{
		"https://example.com",
		"https://hackerone.com",
		"https://bugcrowd.com",
		"https://google.com",
		"https://facebook.com",
		"https://yahoo.com",
		"https://tesla.com",
		"https://github.com",
	}

	// Set options
	options := screener.Options{
		Concurrency:             10,
		Timeout:                 10,
		SaveScreenshots:         true,
		SaveScreenshotsPath:     "custom",
		WaitForPageBody:         false,
		FollowRedirects:         true,
		DisableHTTP2:            true,
		IgnoreCertificateErrors: true,
		Verbose:                 false,
		Silence:                 true,
	}

	// Create a screener runner with options
	runner := screener.NewRunnerWithOptions(options)

	// Capture screenshots of multiple URLs
	results := runner.Multiple(urls)

	// Process the results
	for _, result := range results {
		fmt.Println(result.URL, result.Error, len(result.Image))
	}
}
