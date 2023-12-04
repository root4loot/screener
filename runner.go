package screener

import (
	"net/url"
	"strings"
	"sync"

	"github.com/chromedp/chromedp"
	"github.com/root4loot/goscope"
	"github.com/root4loot/goutils/domainutil"
	"github.com/root4loot/goutils/log"
)

type Runner struct {
	Options *Options
	visited map[string]bool
	mutex   sync.Mutex
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
	WaitTime                int            // Wait time before capturing (seconds)
	Headless                bool           // Run in headless mode
	// Resolvers               []string       // List of resolvers to use
	FollowRedirects bool // Follow redirects
	Silence         bool // Silence output
	Verbose         bool // Verbose logging
}

type Result struct {
	Target     string
	RequestURL string
	FinalURL   string
	Image      []byte
	Error      error
}

func init() {
	log.Init("screener")
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	log.Debug("Getting default options...")

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
		WaitTime:                1,
		FollowRedirects:         true,
		Headless:                true,
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	log.Debug("Creating new runner...")

	options := DefaultOptions()
	newScope := goscope.NewScope()
	options.Scope = newScope

	return &Runner{
		Options: options,
		visited: make(map[string]bool),
	}
}

// NewRunnerWithOptions returns a new runner with the specified options
func NewRunnerWithOptions(options Options) *Runner {
	SetLogLevel(&options)
	log.Debug("Creating new runner with options...")

	// If no scope is specified, create a new one
	if options.Scope == nil {
		newScope := goscope.NewScope()
		options.Scope = newScope
	}

	return &Runner{
		Options: &options,
		visited: make(map[string]bool),
	}
}

// Single captures a single target and returns the result
func (r *Runner) Single(target string) (results []Result) {

	// Normalize target first without locking
	normalizedTarget, err := normalize(target)
	if err != nil {
		log.Warnf("Could not normalize target: %v", err)
		return
	}
	// Ensure target has a trailing slash
	normalizedTarget, _ = domainutil.EnsureTrailingSlash(normalizedTarget)

	r.mutex.Lock()
	r.Options.Scope.AddTargetToScope(target) // Add target to scope
	r.mutex.Unlock()

	// Continue with the rest of the function
	if !r.isVisited(normalizedTarget) {
		if !r.Options.Scope.IsTargetExcluded(normalizedTarget) {
			if !hasScheme(normalizedTarget) {
				log.Debug("Target missing scheme. Trying http://" + normalizedTarget)
				result := r.runWorker("http://" + normalizedTarget)
				result.Target = target
				if strings.HasPrefix(result.FinalURL, "https://") {
					log.Debug(target, "redirected to ", result.FinalURL)
				} else if strings.HasPrefix(result.FinalURL, "http://") {
					results = append(results, result)
					log.Debug(target, "did not redirect. Trying https://"+normalizedTarget)
					result := r.runWorker("https://" + normalizedTarget)
					result.Target = target
					if strings.HasPrefix(result.FinalURL, "https://") {
						return append(results, result)
					}
				}
			} else {
				result := r.runWorker(normalizedTarget)
				result.Target = target
				results = append(results, result)
				r.addVisited(normalizedTarget)
			}
		}
	}
	return
}

// Multiple captures multiple targets and returns the results
func (r *Runner) Multiple(targets []string) (results []Result) {
	log.Debug("Running multiple...")

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			res := r.Single(t)
			results = append(results, res...)
		}(target)
	}
	wg.Wait()

	return results
}

// MultipleStream captures multiple targets and streams the results using channels
func (r *Runner) MultipleStream(resultsChan chan<- Result, targets ...string) {
	log.Debug("Running multiple stream...")
	defer close(resultsChan)

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			results := r.Single(t)
			for _, result := range results {
				resultsChan <- result
			}
		}(target)
	}
	wg.Wait()
}

func (r *Runner) runWorker(url string) Result {
	if !r.isVisited(url) {
		r.addVisited(url)
		return r.worker(url)
	}
	return Result{}
}

// getCustomFlags returns custom chromedp.ExecAllocatorOptions based on the Runner's Options.
func (r *Runner) GetCustomFlags() []chromedp.ExecAllocatorOption {
	// log.Debug("Getting custom flags...")

	var customFlags []chromedp.ExecAllocatorOption

	// Headless mode
	if r.Options.Headless {
		customFlags = append(customFlags, chromedp.Flag("headless", true))
	}

	// Add custom flags based on the Runner's Options.
	if r.Options.IgnoreCertificateErrors {
		customFlags = append(customFlags, chromedp.Flag("ignore-certificate-errors", true))
	}

	// Disable HTTP2
	if r.Options.DisableHTTP2 {
		customFlags = append(customFlags, chromedp.Flag("disable-http2", true))
	}

	return customFlags
}

// normalize ensures that the target has a scheme and a trailing slash.
func normalize(target string) (string, error) {
	target = strings.TrimSpace(target) // Trim whitespace

	// Add temporary scheme if missing
	if !hasScheme(target) {
		target = "http://" + target
	}

	// Parse the target
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}

	// Ensure the URL ends with a trailing slash
	if !strings.HasSuffix(target, u.Path) {
		u.Path += "/"
	}

	// Set scheme to https if port is 443
	if u.Port() != "" {
		if u.Port() == "443" {
			u.Scheme = "https"
			u.Host = strings.Split(u.Host, ":")[0] // Remove port from host
		} else if u.Port() == "80" {
			u.Scheme = "http"
			u.Host = strings.Split(u.Host, ":")[0] // Remove port from host
		}
	}

	target = strings.TrimPrefix(u.String(), "x://") // Remove temporary scheme

	return target, nil
}

// hasScheme checks if the target has a scheme
func hasScheme(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")
}

func (r *Runner) addVisited(str string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.visited[str] = true
}

func (r *Runner) isVisited(str string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.visited[str]
}

// SetLogLevel initiates the logger and sets the log level based on the options
func SetLogLevel(options *Options) {
	if options.Silence {
		log.SetLevel(log.FatalLevel)
	} else if options.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
