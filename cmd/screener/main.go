package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/root4loot/goutils/fileutil"
	"github.com/root4loot/goutils/log"
	"github.com/root4loot/goutils/urlutil"
	"github.com/root4loot/screener/pkg/screener"
)

const (
	author  = "@danielantonsen"
	version = "0.1.0"
	usage   = `
Usage:
  screener [options] (-t <target> | -l <targets.txt>)

INPUT:
  -t, --target                   target input (domain, IP, URL)
  -l, --list                     input file with list of targets (one per line)

CONFIGURATIONS:
  -c,   --concurrency            number of concurrent operations                         (Default: 10)
  -ad,  --avoid-duplicates       prevent saving duplicate outputs                        (Default: false)
  -dt,  --duplicate-threshold    threshold for similarity percentage (0-100)             (Default: 96)
                                 consider outputs as duplicates when similarity score is 
                                 greater than or equal to this value; outputs will not be 
                                 saved when --avoid-duplicates is enabled.
  -to,  --timeout                operation timeout                                       (Default: 15 seconds)
  -ua,  --user-agent             specify user agent                                      (Default: Chrome UA)
  -uh,  --use-http2              enable HTTP2                                            (Default: false)
  -nr,  --ignore-redirects       disable following redirects                             (Default: false)
  -cw,  --capture-width          output width                                            (Default: 1366)
  -ch,  --capture-height         output height                                           (Default: 768)
  -cf,  --capture-full           capture entire content                                  (Default: false)
  -dc,  --delay-capture          delay before operation (seconds)                        (Default: 2)
  -dbc, --delay-between-capture  delay between operations (seconds)                      (Default: 0)
  -rce, --respect-cert-err       respect certificate errors                              (Default: false)
  -isc, --ignore-status-codes    ignore specific status codes (comma separated)          (Default: false)

OUTPUT:
  -o,   --outfolder              save outputs to specified folder                        (Default: ./outputs)
  -nt,  --no-text                do not add text to output images                        (Default: false)
  -v,   --debug                  enable debug mode
        --version                display version
`
)

type cli struct {
	*screener.Screener
	TargetURL            string
	Concurrency          int
	Infile               string
	SaveScreenshotFolder string
	NoImprint            bool
	AvoidDuplicates      bool
	DuplicateThreshold   int
	Debug                bool
}
type cliOptions struct {
	TargetURL            string
	Concurrency          int
	Infile               string
	SaveScreenshotFolder string
	NoImprint            bool
	AvoidDuplicates      bool
	DuplicateThreshold   int
	IgnoreStatusCodes    []int
}

func NewCLIOptions() *cliOptions {
	return &cliOptions{
		Concurrency:          10,
		SaveScreenshotFolder: "./screenshots",
		NoImprint:            false,
		AvoidDuplicates:      false,
		DuplicateThreshold:   96,
		IgnoreStatusCodes:    []int{},
	}
}

func init() {
	log.Init("screener")
}

func main() {
	cli := NewCLI()
	cli.parseFlags()
	targetChannel := make(chan string)
	done := make(chan struct{})

	go processTarget(cli.worker, cli.Concurrency, targetChannel, done)

	processTargets(cli, targetChannel)
	close(targetChannel)
	<-done
}

func processTargets(cli *cli, targetChannel chan<- string) {
	if cli.hasStdin() {
		processStdinTargets(targetChannel)
	}

	if cli.hasInfile() {
		processFileTargets(cli.Infile, targetChannel)
	}

	if cli.hasTarget() {
		processDirectTargets(cli.TargetURL, targetChannel)
	}
}

func processStdinTargets(targetChannel chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		for _, target := range strings.Fields(scanner.Text()) {
			targetChannel <- target
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
		close(targetChannel)
		os.Exit(1)
	}
}

func processFileTargets(infile string, targetChannel chan<- string) {
	fileTargets, err := fileutil.ReadFile(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		close(targetChannel)
		os.Exit(1)
	}
	for _, target := range fileTargets {
		targetChannel <- target
	}
}

func processDirectTargets(targetURL string, targetChannel chan<- string) {
	if strings.Contains(targetURL, ",") {
		targets := strings.Split(targetURL, ",")
		for _, target := range targets {
			targetChannel <- target
		}
	} else {
		targetChannel <- targetURL
	}
}

func NewCLI() *cli {
	cli := &cli{Screener: screener.NewScreenerWithOptions(screener.NewOptions())}
	return cli
}

func processTarget(worker func(string) error, concurrency int, targetChannel <-chan string, done chan struct{}) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for target := range targetChannel {
		log.Debug("Processing CLI target:", target)
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()

			if err := worker(t); err != nil {
				log.Errorf("Failed to process target %s: %v", t, err)
			}
		}(target)
	}

	wg.Wait()
	close(done)
}

func (cli *cli) parseFlags() {
	var help, ver, debug bool
	var ignoreStatusCodes string

	// Initialize default options
	options := NewCLIOptions()
	captureOptions := screener.NewOptions()

	// TARGET
	flag.StringVar(&cli.TargetURL, "target", "", "")
	flag.StringVar(&cli.TargetURL, "t", "", "")
	flag.StringVar(&cli.Infile, "l", "", "")
	flag.StringVar(&cli.Infile, "list", "", "")

	// CONFIGURATIONS
	flag.IntVar(&cli.Concurrency, "concurrency", options.Concurrency, "")
	flag.IntVar(&cli.Concurrency, "c", options.Concurrency, "")
	flag.StringVar(&cli.SaveScreenshotFolder, "outfolder", options.SaveScreenshotFolder, "")
	flag.StringVar(&cli.SaveScreenshotFolder, "o", options.SaveScreenshotFolder, "")
	flag.BoolVar(&cli.AvoidDuplicates, "avoid-duplicates", options.AvoidDuplicates, "")
	flag.BoolVar(&cli.AvoidDuplicates, "ad", options.AvoidDuplicates, "")
	flag.IntVar(&cli.DuplicateThreshold, "duplicate-threshold", options.DuplicateThreshold, "")
	flag.IntVar(&cli.DuplicateThreshold, "dt", options.DuplicateThreshold, "")
	flag.StringVar(&ignoreStatusCodes, "ignore-status-codes", "", "")
	flag.StringVar(&ignoreStatusCodes, "isc", "", "")

	flag.IntVar(&cli.CaptureOptions.CaptureHeight, "capture-height", captureOptions.CaptureHeight, "")
	flag.IntVar(&cli.CaptureOptions.CaptureHeight, "ch", captureOptions.CaptureHeight, "")
	flag.IntVar(&cli.CaptureOptions.CaptureWidth, "capture-width", captureOptions.CaptureWidth, "")
	flag.IntVar(&cli.CaptureOptions.CaptureWidth, "cw", captureOptions.CaptureWidth, "")
	flag.BoolVar(&cli.CaptureOptions.UseHTTP2, "use-http2", captureOptions.UseHTTP2, "")
	flag.BoolVar(&cli.CaptureOptions.UseHTTP2, "uh", captureOptions.UseHTTP2, "")
	flag.BoolVar(&cli.CaptureOptions.IgnoreRedirects, "ignore-redirects", captureOptions.IgnoreRedirects, "")
	flag.BoolVar(&cli.CaptureOptions.IgnoreRedirects, "ir", captureOptions.IgnoreRedirects, "")
	flag.BoolVar(&cli.CaptureOptions.RespectCertificateErrors, "respect-cert-err", captureOptions.RespectCertificateErrors, "")
	flag.BoolVar(&cli.CaptureOptions.RespectCertificateErrors, "rce", captureOptions.RespectCertificateErrors, "")
	flag.IntVar(&cli.CaptureOptions.DelayBeforeCapture, "delay-capture", captureOptions.DelayBeforeCapture, "")
	flag.IntVar(&cli.CaptureOptions.DelayBeforeCapture, "dc", captureOptions.DelayBeforeCapture, "")
	flag.IntVar(&cli.CaptureOptions.Timeout, "timeout", captureOptions.Timeout, "")
	flag.IntVar(&cli.CaptureOptions.Timeout, "to", captureOptions.Timeout, "")
	flag.StringVar(&cli.CaptureOptions.UserAgent, "user-agent", captureOptions.UserAgent, "")
	flag.StringVar(&cli.CaptureOptions.UserAgent, "ua", captureOptions.UserAgent, "")
	flag.IntVar(&cli.CaptureOptions.DelayBetweenCapture, "delay-between-capture", captureOptions.DelayBetweenCapture, "")
	flag.IntVar(&cli.CaptureOptions.DelayBetweenCapture, "dbc", captureOptions.DelayBetweenCapture, "")
	flag.BoolVar(&cli.CaptureOptions.CaptureFull, "capture-full", captureOptions.CaptureFull, "")
	flag.BoolVar(&cli.CaptureOptions.CaptureFull, "cf", captureOptions.CaptureFull, "")

	// OUTPUT
	flag.BoolVar(&cli.NoImprint, "no-text", false, "")
	flag.BoolVar(&cli.NoImprint, "nt", false, "")
	flag.BoolVar(&debug, "debug", false, "")
	flag.BoolVar(&debug, "v", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.BoolVar(&help, "h", false, "")
	flag.BoolVar(&ver, "version", false, "")

	flag.Usage = func() {
		fmt.Print(usage)
	}

	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if help {
		fmt.Print(usage)
		os.Exit(0)
	}

	if ver {
		fmt.Println("screener ", version)
		os.Exit(0)
	}

	if !cli.hasStdin() && !cli.hasInfile() && !cli.hasTarget() && !help {
		log.Error("No target specified")
		fmt.Print(usage)
		os.Exit(0)
	}

	if ignoreStatusCodes != "" {
		statusCodes := strings.Split(ignoreStatusCodes, ",")

		for _, code := range statusCodes {
			statusCode, err := strconv.Atoi(code)
			if err != nil {
				log.Errorf("Invalid status code: %s", code)
				os.Exit(1)
			}
			cli.CaptureOptions.IgnoreStatusCodes = append(cli.CaptureOptions.IgnoreStatusCodes, statusCode)
		}
	}
}

var results []screener.Result

// Modify the worker function to return an error.
func (cli *cli) worker(target string) error {
	var err error
	var result *screener.Result

	target, err = urlutil.RemoveDefaultPort(target)
	if err != nil {
		return fmt.Errorf("error removing default port for %s: %w", target, err)
	}

	// Handle cases where URL does not have a scheme (http or https)
	if !urlutil.HasScheme(target) {
		parsedURL, err := url.Parse("https://" + target)
		if err != nil {
			return fmt.Errorf("error parsing target %s: %w", target, err)
		}

		httpsResult, err := cli.Screener.CaptureScreenshot(parsedURL)
		if err != nil {
			log.Debugf("HTTPS failed for %s. Trying HTTP", target)
			parsedURL.Scheme = "http"
			httpResult, err := cli.Screener.CaptureScreenshot(parsedURL)
			if err == nil && !strings.HasPrefix(httpResult.LandingURL, "https://") {
				result = httpResult
			} else {
				result = httpsResult
			}
		} else {
			result = httpsResult
		}
	} else {
		parsedURL, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("error parsing target %s: %w", target, err)
		}
		result, err = cli.Screener.CaptureScreenshot(parsedURL)
		if err != nil {
			return fmt.Errorf("error capturing screenshot for %s: %w", target, err)
		}
	}

	if cli.AvoidDuplicates && result.IsSimilarToAny(results, cli.DuplicateThreshold) {
		return nil
	}

	if result != nil {
		results = append(results, *result)
	}

	// Add imprint if needed
	if !cli.NoImprint {
		origin, err := urlutil.GetOrigin(result.TargetURL)
		if err != nil {
			return fmt.Errorf("error getting origin for %s: %w", result.TargetURL, err)
		}

		result.Image, err = result.Image.AddTextToImage(origin)
		if err != nil {
			return fmt.Errorf("error adding text to image for %s: %w", origin, err)
		}
	}

	fn, err := result.SaveImageToFolder(cli.SaveScreenshotFolder)
	if err != nil {
		return fmt.Errorf("error saving screenshot: %w", err)
	}
	log.Infof("Screenshot saved to %s", fn)

	return nil
}

func (cli *cli) hasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	mode := stat.Mode()

	isPipedFromChrDev := (mode & os.ModeCharDevice) == 0
	isPipedFromFIFO := (mode & os.ModeNamedPipe) != 0

	return isPipedFromChrDev || isPipedFromFIFO
}

func (cli *cli) hasTarget() bool {
	return cli.TargetURL != ""
}

func (cli *cli) hasInfile() bool {
	return cli.Infile != ""
}
