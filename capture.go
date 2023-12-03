package screener

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/root4loot/goutils/log"
)

func Init() {
	log.Init("screener")
	log.SetLevel(log.InfoLevel)
}

var seenHashes = make(map[string]struct{}) // map of hashes to check for uniqueness

func (r *Runner) worker(requestURL string) Result {
	log.Debug("Running worker on ", requestURL)
	var err error
	var finalURL string
	var htmlBody string
	shouldSave := true // Flag to decide if the screenshot should be saved

	// Initialize the result.
	result := Result{RequestURL: requestURL}

	masterContext, chromeContext, cancelMasterContext, cancelChromeContext := r.initializeChromeDPContext()
	defer cancelMasterContext()
	defer cancelChromeContext()

	// Add tasks
	tasks := chromedp.Tasks{
		chromedp.EmulateViewport(int64(r.Options.CaptureWidth), int64(r.Options.CaptureHeight)),
		chromedp.Navigate(requestURL),
		chromedp.OuterHTML("html", &htmlBody),
		chromedp.CaptureScreenshot(&result.Image),
	}

	// All chromedp.ListenTarget calls
	chromedp.ListenTarget(chromeContext, func(ev interface{}) {
		switch ev := ev.(type) {
		case *page.EventFrameNavigated:
			// Handling EventFrameNavigated
			if ev.Frame.ParentID == "" {
				finalURL = ev.Frame.URL
				log.Debugf("Main frame navigated to %s", finalURL)
				result.FinalURL = finalURL

				if !r.Options.FollowRedirects { // If FollowRedirects is disabled, cancel the context
					log.Infof("Cancelling context due to redirect")
					cancelChromeContext()
				} else if finalURL == "chrome-error://chromewebdata/" { // Check for chrome-error://chromewebdata/ and extract the meta refresh URL
					log.Debugf("chrome-error://chromewebdata/ detected for %s", requestURL)

					htmlBody, _ := fetchURLBody(requestURL)
					finalURL = extractMetaRefreshURL(htmlBody, requestURL)

					if finalURL != "" && r.Options.FollowRedirects {
						log.Debugf("Meta refresh detected for %s. Redirecting to %s", requestURL, finalURL)
						result.FinalURL = finalURL

						// runnning the worker again with the new finalURL
						_ = r.worker(finalURL)
						cancelChromeContext()
					}
				}
			}
		case *network.EventLoadingFinished:
			// WaitForPageLoad, when enabled, ensures that the screenshot is taken
			// only after all network activity has completed, providing a fully loaded page.
			if r.Options.WaitForPageLoad {
				shouldSave = true
			}
		case *network.EventResponseReceived: // TODO: add option for this
			if ev.Response.Status == 400 || ev.Response.Status == 404 {
				log.Debugf("Ignoring HTTP status %d", ev.Response.Status)
				cancelChromeContext()
			}
		}
	})

	// Wait for the specified time before capturing the screenshot
	if r.Options.WaitTime > 0 {
		time.Sleep(time.Duration(r.Options.WaitTime) * time.Second)
	}

	// Run the tasks in the context.
	err = chromedp.Run(chromeContext, tasks)
	if err != nil {
		if masterContext.Err() == context.DeadlineExceeded {
			log.Warnf("Timeout exceeded for %s", requestURL)
		} else {
			result.Error = err
		}
		return result
	}

	// If SaveUnique is enabled, check for uniqueness
	if r.Options.SaveUnique {
		unique, err := checkHashUnique(result.Image)
		if err != nil {
			log.Warnf("Could not perform uniqueness check: %v", err)
		} else if !unique {
			log.Infof("Duplicate screenshot found for %s. Skipping save.", requestURL)
			shouldSave = false
		}
	}

	if r.Options.SaveScreenshots && shouldSave {
		if r.Options.SaveScreenshotsPath == "" {
			r.Options.SaveScreenshotsPath = DefaultOptions().SaveScreenshotsPath // TODO: on runner init
		}
		_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
		if err != nil {
			log.Warnf("Could not save screenshot for %s: %v", requestURL, err)
		} else {
			log.Info("Screenshot", requestURL, "saved to", r.Options.SaveScreenshotsPath)
		}
	}

	return result
}

func (result Result) WriteToFolder(writeFolderPath string) (filename string, err error) {
	// Check if the screenshot data is empty.
	if len(result.Image) == 0 {
		return "", nil // Skip saving if data is empty.
	}

	// Create a folder for screenshots if it doesn't exist.
	err = os.MkdirAll(writeFolderPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	parsedRequestURL, err := url.Parse(result.RequestURL)
	if err != nil {
		return "", err
	}

	parsedRedirectURL, err := url.Parse(result.FinalURL)
	if err != nil {
		return "", err
	}

	parsedWriteURL := parsedRequestURL

	// Remove path from the URL unless specified in target.
	if parsedRequestURL.Path == "" {
		parsedWriteURL.Path = ""
	}

	// Set URL scheme to final URL scheme.
	if parsedRedirectURL.Scheme != "" {
		parsedWriteURL.Scheme = parsedRedirectURL.Scheme
	}

	// remove the port if it's the default port for the scheme.
	if parsedWriteURL.Scheme == "http" || parsedWriteURL.Scheme == "https" && parsedWriteURL.Port() == "80" || parsedWriteURL.Port() == "443" {
		parsedWriteURL.Host = strings.Split(parsedWriteURL.Host, ":")[0]
	}

	filename = parsedWriteURL.Scheme + "_" + parsedWriteURL.Host + parsedWriteURL.Path

	// Process the path to remove a trailing slash and prepend with an underscore
	filename = strings.TrimSuffix(filename, "/")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, ":", "-")

	// Create a filename that includes the scheme, host, and port.
	fileName := filepath.Join(writeFolderPath, filename+".png")

	// Open the file for writing. Ensure the filename is in lower case.
	file, err := os.Create(strings.ToLower(fileName))
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write the screenshot data to the file.
	_, err = file.Write(result.Image)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func (r *Runner) initializeChromeDPContext() (context.Context, context.Context, context.CancelFunc, context.CancelFunc) {
	// Create a master context for the whole operation.
	masterContext, cancelMasterContext := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)

	// Create custom chromedp options by appending the custom flags to the default options.
	opts := append(chromedp.DefaultExecAllocatorOptions[:], r.GetCustomFlags()...)

	// Set custom user-agent if provided in the options.
	if r.Options.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(r.Options.UserAgent))
	}

	// Create an ExecAllocator with the custom options.
	allocator, _ := chromedp.NewExecAllocator(masterContext, opts...)

	// Create a context with the custom allocator.
	chromeContext, cancelChromeContext := chromedp.NewContext(allocator)

	return masterContext, chromeContext, cancelMasterContext, cancelChromeContext
}

// extractMetaRefreshURL parses the HTML content and extracts the meta refresh URL, if present.
// It resolves relative URLs against the provided baseURL.
func extractMetaRefreshURL(html, baseURL string) string {
	// Regex to match the content of URL= in meta refresh tag
	pattern := `(?i)<meta\s+http-equiv="refresh"\s+content="\d+;\s*url=([^"]+)"`
	re := regexp.MustCompile(pattern)

	// Find the match
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}

	// Extract the URL from the regex match
	metaURL := matches[1]

	// Parse the base URL
	baseParsedURL, err := url.Parse(baseURL)
	if err != nil {
		// Handle the error as needed
		return ""
	}

	// Parse the URL found in the meta tag
	parsedMetaURL, err := url.Parse(metaURL)
	if err != nil {
		// Handle the error as needed
		return ""
	}

	// return relative URL against base URL
	return baseParsedURL.ResolveReference(parsedMetaURL).String()
}

func checkHashUnique(imageData []byte) (bool, error) {
	hasher := sha256.New()
	_, err := hasher.Write(imageData)
	if err != nil {
		return false, err
	}
	hashStr := hex.EncodeToString(hasher.Sum(nil))

	_, exists := seenHashes[hashStr]
	if exists {
		return false, nil
	}

	seenHashes[hashStr] = struct{}{}
	return true, nil
}

// fetchURLBody makes an HTTP GET request to the specified URL and returns the response body as a string.
func fetchURLBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
