package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/root4loot/goutils/log"
	screener "github.com/root4loot/screener"
)

const author = "@danielantonsen"

type CLI struct {
	*screener.Runner
	TargetURL         string
	Infile            string
	Version           bool
	Help              bool
	IgnoreStatusCodes string
}

func init() {
	log.Init("screener")
}

func main() {
	cli := &CLI{screener.NewRunner(), "", "", false, false, ""}
	cli.parseFlags()
	cli.checkForExits()
	cli.SetCLIOpts()

	runner := cli.Runner
	runner.Options = cli.Options
	runner.Options.SaveScreenshots = true
	screener.SetLogLevel((runner.Options))

	if cli.hasStdin() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			target := strings.TrimSpace(scanner.Text())
			processResults(runner, target)
		}
	} else if cli.hasInfile() {
		targets, err := cli.readFileLines()
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
		processResults(runner, targets...)
	} else if cli.hasTarget() {
		runner.Run(cli.TargetURL)
	}
}

// SetCLIOpts sets the CLI options
func (cli *CLI) SetCLIOpts() {
	if cli.IgnoreStatusCodes != "" {
		codes := strings.Split(cli.IgnoreStatusCodes, ",")
		for _, code := range codes {
			intVal, err := strconv.ParseInt(code, 10, 64)
			if err != nil {
				log.Fatalf("Invalid status code %s: %v", code, err)
			}
			cli.Options.IgnoreStatusCodes = append(cli.Options.IgnoreStatusCodes, intVal)
		}
	}
}

// processResults processes the results as they come in
func processResults(runner *screener.Runner, targets ...string) {
	// Create a channel to receive results
	results := make(chan screener.Result)

	// Start capturing URLs using multiple goroutines
	go runner.RunAsync(results, targets...)

	// Need to do something with the results
	for result := range results {
		_ = result
	}
}

// checkForExits checks for the presence of the -h|--help and -v|--version flags
func (c *CLI) checkForExits() {
	if c.Help {
		c.banner()
		c.usage()
		os.Exit(0)
	}
	if c.Version {
		fmt.Println("screener ", screener.Version)
		os.Exit(0)
	}

	if !c.hasStdin() && !c.hasInfile() && !c.hasTarget() {
		fmt.Println("")
		fmt.Printf("%s\n\n", "Missing target")
		c.usage()
	}
}

// hasStdin determines if the user has piped input
func (c *CLI) hasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	mode := stat.Mode()

	isPipedFromChrDev := (mode & os.ModeCharDevice) == 0
	isPipedFromFIFO := (mode & os.ModeNamedPipe) != 0

	return isPipedFromChrDev || isPipedFromFIFO
}

// hasTarget determines if the user has provided a target
func (c *CLI) hasTarget() bool {
	return c.TargetURL != ""
}

// hasInfile determines if the user has provided an input file
func (c *CLI) hasInfile() bool {
	return c.Infile != ""
}

// readFileLines reads a file line by line
func (c *CLI) readFileLines() (lines []string, err error) {
	file, err := os.Open(c.Infile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	return
}
