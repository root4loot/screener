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
	results := runner.Single("https://hackerone.com")
	for _, result := range results {
		fmt.Println(result.RequestURL, result.FinalURL, result.Error, len(result.Image))
	}
}
