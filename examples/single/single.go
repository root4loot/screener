package main

import (
	"fmt"

	"github.com/root4loot/screener"
)

func main() {
	// Create runner with default options
	runner := screener.NewRunner()

	// Capture a single URL
	result := runner.Single("https://example.com")
	fmt.Println(result.URL, result.Error, len(result.Image))
}
