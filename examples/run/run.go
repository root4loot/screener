package main

import (
	"fmt"

	"github.com/root4loot/screener"
)

func main() {
	// Create runner with default options
	runner := screener.NewRunner()
	runner.Options.SaveScreenshots = true

	// Capture a single URL
	result := runner.Run("https://example.com", "https://hackerone.com")

	// Process the result
	for _, result := range result {
		fmt.Println(result.TargetURL, result.LandingURL, result.Error, len(result.Image))
	}
}
