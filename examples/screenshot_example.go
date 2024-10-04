package main

import (
	"fmt"
	"net/url"

	"github.com/root4loot/goutils/urlutil"
	"github.com/root4loot/screener/pkg/screener"
)

func main() {
	options := screener.NewOptions()
	options.CaptureWidth = 1024
	options.CaptureHeight = 768
	// more options ...

	s := screener.NewScreenerWithOptions(options)

	urls := []string{
		"https://example.com",
		"https://example.org",
		"https://scanme.sh",
	}

	var results []screener.Result

	for _, u := range urls {
		parsedURL, err := url.Parse(u)
		if err != nil {
			fmt.Printf("Error parsing URL %s: %v\n", u, err)
			continue
		}

		// Capture screenshot
		result, err := s.CaptureScreenshot(parsedURL)
		if err != nil {
			fmt.Printf("Error capturing screenshot for %s: %v\n", u, err)
			continue
		}

		// Check for duplicates
		if result.IsSimilarToAny(results, 96) { // 96% similarity
			fmt.Printf("Screenshot for %s is a duplicate, skipping\n", u)
			continue
		}

		// Add text to image
		origin, err := urlutil.GetOrigin(result.TargetURL)
		if err != nil {
			fmt.Printf("Error getting origin for %s: %v", result.TargetURL, err)
			return
		}

		result.Image, err = result.Image.AddTextToImage(origin)
		if err != nil {
			fmt.Printf("Error adding text to image for %s: %v\n", u, err)
			continue
		}

		// Save image to file
		filename, err := result.SaveImageToFolder("./screenshots")
		if err != nil {
			fmt.Printf("Error saving screenshot for %s: %v\n", u, err)
			continue
		}

		fmt.Printf("Screenshot saved to %s\n", filename)
		results = append(results, *result)
	}
}
