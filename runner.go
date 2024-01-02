package screener

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/root4loot/goscope"
	"github.com/root4loot/goutils/log"
	"github.com/root4loot/goutils/urlutil"
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
	WaitTime                int            // Max wait time in seconds before taking screenshot, regardless of page load completion
	IgnoreStatusCodes       []int64        // List of status codes to ignore
	DelayBetweenCapture     int            // Delay in seconds between captures for multiple targets
	FollowRedirects         bool           // Follow redirects
	CaptureFull             bool           // Whether to take a full page screenshot
	URLInImage              bool           // Whether to include the URL in the image
	Silence                 bool           // Silence output
	Verbose                 bool           // Verbose logging
}

type Result struct {
	Target     string
	TargetURL  string
	LandingURL string
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
		CaptureWidth:            1366,
		CaptureHeight:           768,
		IgnoreCertificateErrors: true,
		SaveUnique:              false,
		DisableHTTP2:            true,
		SaveScreenshots:         false,
		SaveScreenshotsPath:     "./screenshots",
		CaptureFull:             false,
		WaitTime:                30,
		DelayBetweenCapture:     0,
		FollowRedirects:         true,
		URLInImage:              true,
		IgnoreStatusCodes:       []int64{},
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

// Run captures one or more targets and returns the results. It handles both single and multiple targets.
func (r *Runner) Run(targets ...string) (results []Result) {
	if len(targets) == 1 {
		// Handle single target
		return []Result{r.capture(targets[0])}
	}

	// Handle multiple targets
	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			results = append(results, r.capture(t))
		}(target)
	}
	wg.Wait()

	return results
}

// RunAsync captures multiple targets asynchronously and streams the results using channels.
func (r *Runner) RunAsync(resultsChan chan<- Result, targets ...string) {
	log.Debug("Running async capture...")
	defer close(resultsChan)

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			resultsChan <- r.capture(t)
		}(target)
	}
	wg.Wait()
}

// runWorker runs a worker on the given URL.
func (r *Runner) runWorker(url string) Result {
	if !r.isVisited(url) {
		r.addVisited(url)
		return r.worker(url)
	}
	return Result{}
}

// capture encapsulates the logic to capture a single target.
func (r *Runner) capture(target string) Result {

	// Add delay between captures if specified
	if r.Options.DelayBetweenCapture > 0 {
		time.Sleep(time.Duration(r.Options.DelayBetweenCapture) * time.Second)
	}

	log.Debugf("Capturing target: %s", target)

	// Normalize target.
	normalizedTarget, err := normalize(target)
	if err != nil {
		log.Warnf("Could not normalize target: %v", err)
		return Result{Error: err}
	}

	// Ensure target has a trailing slash.
	normalizedTarget, _ = urlutil.EnsureTrailingSlash(normalizedTarget)

	// Add target to scope.
	r.mutex.Lock()
	r.Options.Scope.AddTargetToScope(target)
	r.mutex.Unlock()

	// Skip if already visited or excluded.
	if r.isVisited(normalizedTarget) || r.Options.Scope.IsTargetExcluded(normalizedTarget) {
		log.Debugf("Target skipped (already visited or excluded): %s", normalizedTarget)
		return Result{}
	}

	// Process the target based on its scheme.
	return r.processTarget(target, normalizedTarget)
}

// processTarget processes the given target based on its scheme.
func (r *Runner) processTarget(target, normalizedTarget string) (result Result) {
	if !hasScheme(normalizedTarget) {
		// Try with http scheme.
		resultWithHTTP := r.tryScheme("http://", target, normalizedTarget)

		// If HTTP fails or redirects to HTTPS and FollowRedirects is true, then try HTTPS.
		if resultWithHTTP.Error != nil || (strings.HasPrefix(resultWithHTTP.LandingURL, "https://") && r.Options.FollowRedirects) {
			log.Debugf("HTTP failed or redirected to HTTPS for %s: Trying HTTPS", target)
			resultWithHTTPS := r.tryScheme("https://", target, normalizedTarget)
			if resultWithHTTPS.Error == nil {
				return resultWithHTTPS
			}
		}

		// Return the HTTP result if HTTPS is not attempted or fails.
		return resultWithHTTP
	}

	// Directly run worker if scheme is present.
	return r.runWorker(normalizedTarget)
}

// tryScheme tries to capture the target with the given scheme.
func (r *Runner) tryScheme(scheme, target, normalizedTarget string) (result Result) {
	result = r.runWorker(scheme + normalizedTarget)
	result.Target = target
	if strings.HasPrefix(result.LandingURL, scheme) {
		log.Debug(target, "found ", scheme, ", returning ", result.LandingURL)
	}
	return result
}

// getCustomFlags returns custom chromedp.ExecAllocatorOptions based on the Runner's Options.
func (r *Runner) GetCustomFlags() []chromedp.ExecAllocatorOption {
	// log.Debug("Getting custom flags...")

	var customFlags []chromedp.ExecAllocatorOption

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
