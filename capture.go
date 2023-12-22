package screener

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/root4loot/goutils/log"
)

func Init() {
	log.Init("screener")
	log.SetLevel(log.InfoLevel)
}

var seenHashes = make(map[string]struct{}) // map of hashes to check for uniqueness

func (r *Runner) worker(requestURL string) Result {
	log.Info("Running worker on", requestURL)
	result := Result{RequestURL: requestURL}

	// Launch browser with configured options
	l := newLauncher(*r.Options)
	browserURL := l.MustLaunch()
	browser := rod.New().ControlURL(browserURL).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	// Set the viewport if CaptureWidth and CaptureHeight are specified
	if r.Options.CaptureWidth != 0 && r.Options.CaptureHeight != 0 {
		viewport := &proto.EmulationSetDeviceMetricsOverride{
			Width:             r.Options.CaptureWidth,
			Height:            r.Options.CaptureHeight,
			DeviceScaleFactor: 1,
			Mobile:            false,
		}

		err := page.SetViewport(viewport)
		if err != nil {
			log.Warnf("Error setting viewport: %v", err)
			// Handle the error as needed
		}
	}

	if err := page.Navigate(requestURL); err != nil {
		log.Warnf("Error navigating to %s: %v", requestURL, err)
		result.Error = err
		return result
	}

	// Handle redirects
	if !r.Options.FollowRedirects && page.MustInfo().URL != requestURL {
		log.Warn("Redirect detected, but FollowRedirects is disabled")
		return result
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.WaitTime)*time.Second)
	defer cancel()

	// Wait for the page to load with a timeout
	if r.Options.WaitForPageLoad {
		err := page.Context(ctx).WaitLoad()
		if err != nil {
			log.Warnf("Wait for page load timed out after %v: %v", time.Duration(r.Options.WaitTime)*time.Second, err)
			log.Warn("Proceeding to take a screenshot anyway.")
		}
	}

	// Take and process screenshot
	if err := processScreenshot(page, &result, r); err != nil {
		log.Warnf("Error processing screenshot for %s: %v", requestURL, err)
		result.Error = err
		return result
	}

	// Update final URL and return result
	result.FinalURL = page.MustInfo().URL

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
	if (parsedWriteURL.Scheme == "http" || parsedWriteURL.Scheme == "https") && parsedWriteURL.Port() == "80" || parsedWriteURL.Port() == "443" {
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

// processScreenshot handles taking, saving, and uniqueness checking of screenshots.
func processScreenshot(page *rod.Page, result *Result, r *Runner) error {
	shouldSave := true
	screenshot, err := page.Screenshot(false, nil)
	if err != nil {
		return err
	}
	result.Image = screenshot

	// Check for screenshot uniqueness if required
	if r.Options.SaveUnique {
		unique, err := checkHashUnique(result.Image)
		if err != nil {
			log.Warnf("Could not perform uniqueness check: %v", err)
		} else if !unique {
			log.Infof("Duplicate screenshot found for %s. Skipping save.", result.RequestURL)
			shouldSave = false
		}
	}

	// Save screenshot if required
	if r.Options.SaveScreenshots && shouldSave {
		_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
		if err != nil {
			return err
		}
		log.Resultf("Screenshot for %s saved to %s", result.RequestURL, r.Options.SaveScreenshotsPath)
	}
	return nil
}

// newLauncher creates a new browser launcher with the specified options.
func newLauncher(options Options) *launcher.Launcher {
	l := launcher.New().
		Headless(true) // Set to true to ensure headless mode

	if options.UserAgent != "" {
		l.Set("user-agent", options.UserAgent)
	}

	if options.IgnoreCertificateErrors {
		l.Set("ignore-certificate-errors", "true")
	}

	if options.DisableHTTP2 {
		l.Set("disable-http2", "true")
	}

	return l
}

// checkHashUnique checks if the hash of the screenshot data is unique.
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
