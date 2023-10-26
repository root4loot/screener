package screener

import (
	"strings"
	"sync"

	"github.com/chromedp/chromedp"
	"github.com/root4loot/goscope"
	"github.com/root4loot/relog"
)

var Log = relog.NewLogger("screener")

type Runner struct {
	Options *Options
}

// Options contains options for the runner
type Options struct {
	Concurrency             int            // number of concurrent requests
	CaptureHeight           int            // height of the capture
	CaptureWidth            int            // width of the capture
	Timeout                 int            // Timeout for each capture (seconds)
	IgnoreCertificateErrors bool           // Ignore certificate errors
	DisableHTTP2            bool           // Disable HTTP2
	SaveScreenshots         bool           // Save screenshot to file
	SaveScreenshotsPath     string         // Path to save screenshots
	SaveUnique              bool           // Save unique screenshots only
	Scope                   *goscope.Scope // Scope to use
	UserAgent               string         // User agent to use
	WaitForPageLoad         bool           // Wait for page to load before capturing
	// Resolvers               []string       // List of resolvers to use
	FollowRedirects bool // Follow redirects
	Silence         bool // Silence output
	Verbose         bool // Verbose logging
}

type Result struct {
	URL      string
	Image    []byte
	Resolver string
	Error    error
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	Log.Debugln("Getting default options...")

	return &Options{
		Concurrency:             10,
		Timeout:                 15,
		CaptureWidth:            1920,
		CaptureHeight:           1080,
		IgnoreCertificateErrors: true,
		SaveUnique:              false,
		DisableHTTP2:            true,
		SaveScreenshots:         false,
		SaveScreenshotsPath:     "./screenshots",
		WaitForPageLoad:         true,
		FollowRedirects:         true,
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	Log.Debugln("Creating new runner...")

	options := DefaultOptions()
	newScope := goscope.NewScope()
	options.Scope = newScope

	return &Runner{
		Options: options,
	}
}

// NewRunnerWithOptions returns a new runner with the specified options
func NewRunnerWithOptions(options Options) *Runner {
	SetLogLevel(&options)

	Log.Debugln("Creating new runner with options...")

	// If no scope is specified, create a new one
	if options.Scope == nil {
		newScope := goscope.NewScope()
		options.Scope = newScope
	}

	return &Runner{
		Options: &options,
	}
}

// Single captures a single target and returns the result
func (r *Runner) Single(target string) (result Result) {
	Log.Debugln("Running single...")
	r.Options.SaveScreenshots = true
	urls := r.initializeTargets(target)
	if r.Options.Scope.InScope(urls[0]) {
		return r.worker(urls[0])
	}
	return
}

// Multiple captures multiple targets and returns the results
func (r *Runner) Multiple(targets []string) (results []Result) {
	Log.Debugln("Running multiple...")

	urls := r.initializeTargets(targets...)
	resultsChan := make(chan Result)

	inScopeCount := 0 // Counter for in-scope URLs

	for _, url := range urls {
		if r.Options.Scope.InScope(url) {
			inScopeCount++ // Increment counter for in-scope URLs
			go func(u string) {
				result := r.worker(u)
				resultsChan <- result
			}(url)
		}
	}

	for i := 0; i < inScopeCount; i++ { // Range over in-scope URLs only
		result := <-resultsChan
		results = append(results, result)
	}
	close(resultsChan)

	return results
}

// MultipleStream captures multiple targets and streams the results using channels
func (r *Runner) MultipleStream(results chan<- Result, targets ...string) {
	Log.Debugln("Running multiple stream...")

	defer close(results)
	urls := r.initializeTargets(removeWhitespaces(targets...)...)

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, url := range urls {
		if r.Options.Scope.InScope(url) {
			sem <- struct{}{}
			wg.Add(1)
			go func(u string) {
				defer func() { <-sem }()
				defer wg.Done()
				results <- r.worker(u)
			}(url)
		}
	}
	wg.Wait()
}

// getCustomFlags returns custom chromedp.ExecAllocatorOptions based on the Runner's Options.
func (r *Runner) GetCustomFlags() []chromedp.ExecAllocatorOption {
	// Log.Debugln("Getting custom flags...")

	var customFlags []chromedp.ExecAllocatorOption

	// Headless mode
	customFlags = append(customFlags, chromedp.Flag("headless", true))

	// Add custom flags based on the Runner's Options.
	if r.Options.IgnoreCertificateErrors {
		customFlags = append(customFlags, chromedp.Flag("ignore-certificate-errors", true))
	}

	if r.Options.DisableHTTP2 {
		customFlags = append(customFlags, chromedp.Flag("disable-http2", true))
	}

	return customFlags
}

// makeURLs returns a list of URLs with http:// and https:// prefixes.
func makeURLs(targets ...string) (urls []string) {
	Log.Debugln("Making URLs...")

	for _, target := range targets {
		// Remove http:// or https:// if they are there
		cleanTarget := strings.TrimPrefix(strings.TrimPrefix(target, "http://"), "https://")

		// Always append http and https versions
		urls = append(urls, "http://"+cleanTarget)
		urls = append(urls, "https://"+cleanTarget)
	}
	return
}

// initializeTargets sets the scope and returns a list of URLs
func (r *Runner) initializeTargets(targets ...string) (urls []string) {
	Log.Debugln("Initializing targets...")
	urls = makeURLs(targets...)
	err := r.Options.Scope.AddInclude(urls...)
	if err != nil {
		Log.Warningln(err)
	}

	return
}

func SetLogLevel(options *Options) {
	Log.Debugln("Setting logger level...")

	if options.Verbose {
		Log.SetLevel(relog.DebugLevel)
	} else if options.Silence {
		Log.SetLevel(relog.FatalLevel)
	} else {
		Log.SetLevel(relog.InfoLevel)
	}
}

// removeWhitespaces removes all leading and trailing whitespaces
func removeWhitespaces(input ...string) []string {
	var output []string
	for _, str := range input {
		cleanedStr := strings.TrimSpace(str)
		output = append(output, cleanedStr)
	}
	return output
}
