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
	result := runner.Single("https://hackerone.com")
	fmt.Println(result.RequestURL, result.FinalURL, result.Error, len(result.Image))
}
