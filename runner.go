package screener

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/root4loot/goutils/log"
	"github.com/root4loot/goutils/urlutil"
	"github.com/root4loot/scope"
)

type Runner struct {
	Options *Options
	visited map[string]bool
	mutex   sync.Mutex
}

type Options struct {
	Concurrency              int          // number of concurrent requests
	CaptureHeight            int          // height of the capture
	CaptureWidth             int          // width of the capture
	Timeout                  int          // Timeout for each capture (seconds)
	RespectCertificateErrors bool         // Respect certificate errors
	UseHTTP2                 bool         // Use HTTP2
	SaveScreenshots          bool         // Save screenshot to file
	SaveScreenshotsPath      string       // Path to save screenshots
	SaveUnique               bool         // Save unique screenshots only
	Scope                    *scope.Scope // Scope to use
	UserAgent                string       // User agent to use
	MaxWait                  int          // Max wait time in seconds before taking screenshot, regardless of page load completion
	FixedWait                int          // Fixed wait time in seconds before taking screenshot, regardless of page load completion
	IgnoreStatusCodes        []int64      // List of status codes to ignore
	DelayBetweenCapture      int          // Delay in seconds between captures for multiple targets
	IgnoreRedirects          bool         // Do not follow redirects
	CaptureFull              bool         // Whether to take a full page screenshot
	ImprintURL               bool         // Whether to include the URL in the image
	Silence                  bool         // Silence output
	Verbose                  bool         // Verbose logging
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
	return &Options{
		Concurrency:              10,
		Timeout:                  15,
		CaptureWidth:             1366,
		CaptureHeight:            768,
		RespectCertificateErrors: false,
		SaveUnique:               false,
		UseHTTP2:                 false,
		SaveScreenshots:          false,
		SaveScreenshotsPath:      "./screenshots",
		CaptureFull:              false,
		MaxWait:                  30,
		FixedWait:                2,
		DelayBetweenCapture:      0,
		IgnoreRedirects:          false,
		ImprintURL:               true,
		IgnoreStatusCodes:        []int64{},
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	options := DefaultOptions()
	newScope := scope.NewScope()
	options.Scope = newScope

	return &Runner{
		Options: options,
		visited: make(map[string]bool),
	}
}

// NewRunnerWithOptions returns a new runner with the specified options
func NewRunnerWithOptions(options Options) *Runner {
	SetLogLevel(&options)

	if options.Scope == nil {
		newScope := scope.NewScope()
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
		return []Result{r.capture(targets[0])}
	}

	r.runWithConcurrency(func(t string) {
		results = append(results, r.capture(t))
	}, targets...)

	return results
}

// RunAsync captures multiple targets asynchronously and streams the results using channels.
func (r *Runner) RunAsync(resultsChan chan<- Result, targets ...string) {
	defer close(resultsChan)

	r.runWithConcurrency(func(t string) {
		resultsChan <- r.capture(t)
	}, targets...)
}

func (r *Runner) runWithConcurrency(worker func(string), targets ...string) {
	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup
	for _, target := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(t string) {
			defer func() { <-sem }()
			defer wg.Done()
			worker(t)
		}(target)
	}
	wg.Wait()
}

func (r *Runner) capture(target string) Result {
	if r.Options.DelayBetweenCapture > 0 {
		time.Sleep(time.Duration(r.Options.DelayBetweenCapture) * time.Second)
	}

	normalizedTarget, err := normalize(target)
	if err != nil {
		log.Warnf("Could not normalize target: %v", err)
		return Result{Error: err}
	}

	normalizedTarget = urlutil.EnsureTrailingSlash(normalizedTarget)

	r.mutex.Lock()
	r.Options.Scope.AddInclude(target)
	r.mutex.Unlock()

	if r.isVisited(normalizedTarget) || !r.Options.Scope.IsInScope(normalizedTarget) {
		log.Debugf("Target skipped (already visited or not in scope): %s", normalizedTarget)
		return Result{}
	}

	return r.processTarget(target, normalizedTarget)
}

func (r *Runner) processTarget(target, normalizedTarget string) (result Result) {
	if !urlutil.HasScheme(normalizedTarget) {
		resultWithHTTPS := r.tryScheme("https://", target, normalizedTarget)

		// If HTTPS fails
		if resultWithHTTPS.Error != nil {
			log.Infof("HTTPS failed for %s. Trying HTTP", target)
			resultWithHTTP := r.tryScheme("http://", target, normalizedTarget)
			if resultWithHTTP.Error == nil || !strings.HasPrefix(resultWithHTTP.LandingURL, "https://") {
				return resultWithHTTP
			}
		}

		return resultWithHTTPS
	}

	return r.captureTarget(normalizedTarget)
}

func (r *Runner) tryScheme(scheme, target, normalizedTarget string) (result Result) {
	result = r.captureTarget(scheme + normalizedTarget)
	result.Target = target
	return result
}

func (r *Runner) GetCustomFlags() []chromedp.ExecAllocatorOption {
	var customFlags []chromedp.ExecAllocatorOption

	if !r.Options.RespectCertificateErrors {
		customFlags = append(customFlags, chromedp.Flag("ignore-certificate-errors", true))
	}

	if !r.Options.UseHTTP2 {
		customFlags = append(customFlags, chromedp.Flag("disable-http2", true))
	}

	return customFlags
}

func normalize(target string) (string, error) {
	target = strings.TrimSpace(target)

	if !urlutil.HasScheme(target) {
		target = "x://" + target
	}

	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(target, u.Path) {
		u.Path += "/"
	}

	if u.Port() != "" {
		if u.Port() == "443" {
			u.Scheme = "https"
			u.Host = strings.Split(u.Host, ":")[0]
		} else if u.Port() == "80" {
			u.Scheme = "http"
			u.Host = strings.Split(u.Host, ":")[0]
		}
	}

	target = strings.TrimPrefix(u.String(), "x://")

	return target, nil
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

func SetLogLevel(options *Options) {
	if options.Silence {
		log.SetLevel(log.FatalLevel)
	} else if options.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
