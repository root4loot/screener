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
		Timeout:                 15,
		SaveScreenshots:         true,
		WaitForPageLoad:         true,
		WaitTime:                1,
		FollowRedirects:         true,
		DisableHTTP2:            true,
		IgnoreCertificateErrors: true,
		Verbose:                 false,
		Silence:                 true,
		CaptureWidth:            1366,
		CaptureHeight:           768,
	}

	// Create a screener runner with options
	runner := screener.NewRunnerWithOptions(options)

	// Create a channel to receive results
	results := make(chan screener.Result)

	// Start capturing URLs using multiple goroutines
	go runner.RunAsync(results, urls...)

	// Process the results as they come in
	for result := range results {
		fmt.Println(result.RequestURL, result.FinalURL, result.Error, len(result.Image))
	}
}
