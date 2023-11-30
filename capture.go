package screener

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/root4loot/goutils/log"
)

var seenHashes = make(map[string]struct{}) // map of hashes to check for uniqueness

func (r *Runner) worker(rawURL string) Result {
	log.Debug("Running worker on ", rawURL)

	var redirected bool // Flag to indicate if the URL was redirected
	var finalURL string

	shouldSave := true // Flag to decide if the screenshot should be saved

	// Initialize the result.
	result := Result{RequestURL: rawURL}

	// Create a master context for the whole operation.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()

	// Create custom chromedp options by appending the custom flags to the default options.
	opts := append(chromedp.DefaultExecAllocatorOptions[:], r.GetCustomFlags()...)

	// Set custom user-agent if provided in the options.
	if r.Options.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(r.Options.UserAgent))
	}

	// Create an ExecAllocator with the custom options.
	allocator, _ := chromedp.NewExecAllocator(ctx, opts...)

	// Create a context with the custom allocator.
	cctx, cancelContext := chromedp.NewContext(allocator)
	defer cancelContext()

	// Add tasks to emulate viewport and navigate to the URL.
	tasks := chromedp.Tasks{
		chromedp.EmulateViewport(int64(r.Options.CaptureWidth), int64(r.Options.CaptureHeight)),
		chromedp.Navigate(rawURL),
	}

	// Listen to network events
	chromedp.ListenTarget(cctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *page.EventFrameNavigated:
			// Check if it's the main frame (ParentID is empty for the main frame).
			if ev.Frame.ParentID == "" {
				finalURL = ev.Frame.URL
				log.Debugf("Main frame navigated to %s", finalURL)

				// Check if the frame's URL is different from the initial URL for redirect handling.
				if finalURL != rawURL {
					log.Debugf("Redirect detected from %s to %s", rawURL, finalURL)
					redirected = true
					result.FinalURL = finalURL
					if !r.Options.FollowRedirects {
						// If FollowRedirects is false, cancel the context to stop loading.
						log.Infof("Cancelling context due to redirect")
						cancelContext()
					}

					// update the rawURL to the final URL host
					u, err := url.Parse(finalURL)
					if err != nil {
						log.Warnf("Could not parse final URL: %v", err)
					} else {
						// remove the port if it's the default port for the scheme.
						if u.Scheme == "http" && u.Port() == "80" || u.Scheme == "https" && u.Port() == "443" {
							u.Host = strings.Split(u.Host, ":")[0]
						}
						rawURL = u.Scheme + "://" + u.Host + "/"
					}
				}
			}
		}
	})

	// WaitForPageLoad, when enabled, ensures that the screenshot is taken
	// only after all network activity has completed, providing a fully loaded page.
	if r.Options.WaitForPageLoad {
		// Listen for network events to track the status of network requests.
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			if _, ok := ev.(*network.EventLoadingFinished); ok {
				// A network event indicates that the page is fully loaded.
				shouldSave = true
			}
		})
	}

	// Wait for the specified time before capturing the screenshot
	if r.Options.WaitTime > 0 {
		time.Sleep(time.Duration(r.Options.WaitTime) * time.Second)
	}

	// Before taking a screenshot, check if there was a redirect and FollowRedirects is false
	if redirected && !r.Options.FollowRedirects {
		log.Debugf("Redirect occurred and FollowRedirects is false. Skipping screenshot for %s", rawURL)
		return Result{RequestURL: rawURL, FinalURL: finalURL, Error: fmt.Errorf("redirect occurred but FollowRedirects is false")}
	}

	tasks = append(tasks, chromedp.CaptureScreenshot(&result.Image))

	// Run the tasks in the context.
	err := chromedp.Run(cctx, tasks)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Warnf("Timeout exceeded for %s", rawURL)
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
			log.Infof("Duplicate screenshot found for %s. Skipping save.", rawURL)
			shouldSave = false
		}
	}

	if r.Options.SaveScreenshots && shouldSave {
		if r.Options.SaveScreenshotsPath == "" {
			r.Options.SaveScreenshotsPath = DefaultOptions().SaveScreenshotsPath
		}
		_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
		if err != nil {
			log.Warnf("Could not save screenshot for %s: %v", rawURL, err)
		} else {
			log.Info("Screenshot", rawURL, "saved to", r.Options.SaveScreenshotsPath)
		}
	}

	return result
}

func (result Result) WriteToFolder(folderPath string) (filename string, err error) {
	// Check if the screenshot data is empty.
	if len(result.Image) == 0 {
		return "", nil // Skip saving if data is empty.
	}

	// Create a folder for screenshots if it doesn't exist.
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	// Parse the URL to extract the scheme, host, and port.
	u, err := url.Parse(result.RequestURL)
	if err != nil {
		return "", err
	}

	// remove the port if it's the default port for the scheme.
	if u.Scheme == "http" || u.Scheme == "https" && u.Port() == "80" || u.Port() == "443" {
		u.Host = strings.Split(u.Host, ":")[0]
	}

	filename = u.Scheme + "_" + u.Host + u.Path

	// Process the path to remove a trailing slash and prepend with an underscore
	filename = strings.TrimSuffix(filename, "/")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, ":", "-")

	// Create a filename that includes the scheme, host, and port.
	fileName := filepath.Join(folderPath, filename+".png")

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
