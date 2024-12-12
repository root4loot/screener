package main

import (
	"bufio"
	"context"
	"errors"
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
	usage   = `USAGE:
  screener [options] (-t <target> | -l <targets.txt>)

INPUT:
  -t, --target                   target input (domain, IP, URL)
  -l, --list                     input file with list of targets (one per line)

CONFIGURATIONS:
  -c,   --concurrency            number of concurrent operations                         (Default: 10)
  -ad,  --avoid-duplicates       prevent saving duplicate outputs                        (Default: false)
  -dt,  --duplicate-threshold    threshold for similarity percentage (0-100)             (Default: 96)
                                 Applicable only when --avoid-duplicates is enabled. Outputs
                                 with a similarity score greater than or equal to this value
                                 will be considered duplicates and will not be saved.
  -to,  --timeout                screenshot timeout                                      (Default: 15 seconds)
  -ua,  --user-agent             specify user agent                                      (Default: Chrome UA)
  -uh,  --use-http2              use HTTP2                                               (Default: false)
  -nr,  --ignore-redirects       do not follow redirects                                 (Default: false)
  -cw,  --capture-width          output width                                            (Default: 1366)
  -ch,  --capture-height         output height                                           (Default: 768)
  -cf,  --capture-full           capture entire content                                  (Default: false)
  -dc,  --delay-capture          delay before operation (seconds)                        (Default: 2)
  -dbc, --delay-between-capture  delay between operations (seconds)                      (Default: 0)
  -rce, --respect-cert-err       respect certificate errors                              (Default: false)
  -isc, --ignore-status-codes    ignore specific status codes (comma separated)          (Default: false)

OUTPUT:
  -o,   --outfolder              save outputs to specified folder                        (Default: ./screenshots)
  -nt,  --no-text                do not add text to output images                        (Default: false)
        --debug                  enable debug mode
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

func NewCLI() *cli {
	cli := &cli{Screener: screener.NewScreenerWithOptions(screener.NewOptions())}
	return cli
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

func processTarget(worker func(string) error, concurrency int, targetChannel <-chan string, done chan struct{}) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for target := range targetChannel {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			if err := worker(t); err != nil {
				log.Errorf("Error processing target %s: %v", t, err)
			}
		}(target)
	}

	wg.Wait()
	close(done)
}

func (cli *cli) parseFlags() {
	var help, ver, debug bool
	var ignoreStatusCodes string

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

func (cli *cli) worker(target string) error {
	var err error
	var result *screener.Result

	target, err = urlutil.RemoveDefaultPort(target)
	if err != nil {
		log.Errorf("Error processing target %s: %v", target, err)
		return nil
	}

	urlStr := target
	if !urlutil.HasScheme(target) {
		log.Debugf("No scheme specified for %s: trying HTTPS", target)
		urlStr = "https://" + target
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Errorf("Invalid URL %s: %v", urlStr, err)
		return nil
	}

	result, err = cli.Screener.CaptureScreenshot(parsedURL)
	if err != nil {
		if shouldRetryWithHTTP(err) {
			log.Debugf("HTTPS failed for %s: %s. Trying HTTP.", target, unwrapError(err))
			parsedURL.Scheme = "http"
			result, err = cli.Screener.CaptureScreenshot(parsedURL)
		}
	}

	if err != nil {
		handleCaptureError(target, err)
		return nil
	}

	if result == nil {
		log.Warnf("Screenshot capture failed for %s: no valid result", target)
		return nil
	}

	if result.StatusCode != 200 {
		log.Warnf("Could not capture %s: received status code %d", target, result.StatusCode)
		return nil
	}

	if cli.AvoidDuplicates && result.IsSimilarToAny(results, cli.DuplicateThreshold) {
		return nil
	}

	results = append(results, *result)

	if !cli.NoImprint {
		origin, err := urlutil.GetOrigin(result.TargetURL)
		if err != nil {
			log.Errorf("Error processing result URL %s: %v", result.TargetURL, err)
			return nil
		}

		result.Image, err = result.Image.AddTextToImage(origin)
		if err != nil {
			log.Errorf("Error adding text to image for %s: %v", origin, err)
			return nil
		}
	}

	fn, err := result.SaveImageToFolder(cli.SaveScreenshotFolder)
	if err != nil {
		log.Errorf("Error saving screenshot for %s: %v", target, err)
		return nil
	}

	log.Resultf("Screenshot saved to %s", fn)
	return nil
}

func shouldRetryWithHTTP(err error) bool {
	if isDNSError(err) || isTimeoutError(err) {
		return false
	}
	return true
}

func isDNSError(err error) bool {
	if err == nil {
		return false
	}

	errMessage := getFullErrorMessage(err)
	return strings.Contains(errMessage, "net::ERR_NAME_NOT_RESOLVED") ||
		strings.Contains(errMessage, "no such host")
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errMessage := getFullErrorMessage(err)
	return strings.Contains(errMessage, "context deadline exceeded") ||
		strings.Contains(errMessage, "timeout")
}

func getFullErrorMessage(err error) string {
	var sb strings.Builder
	for err != nil {
		sb.WriteString(err.Error())
		err = errors.Unwrap(err)
		if err != nil {
			sb.WriteString(" | ")
		}
	}
	return sb.String()
}

func unwrapError(err error) string {
	rootErr := err
	for {
		unwrappedErr := errors.Unwrap(rootErr)
		if unwrappedErr == nil {
			break
		}
		rootErr = unwrappedErr
	}
	return rootErr.Error()
}

func handleCaptureError(target string, err error) {
	switch {
	case isDNSError(err):
		log.Warnf("DNS lookup failed for %s", target)
	case isTimeoutError(err):
		log.Debugf("Timeout occurred while capturing screenshot for %s", target)
	default:
		log.Errorf("Error capturing screenshot for %s: %s", target, unwrapError(err))
	}
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
