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

	"github.com/chromedp/chromedp"
)

var seenHashes = make(map[string]struct{}) // map of hashes to check for uniqueness

func (r *Runner) worker(url string) Result {
	Log.Debugln("Running worker on ", url)

	shouldSave := true // Flag to decide if the screenshot should be saved

	// Initialize the result.
	result := Result{URL: url}

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
	cctx, cancel := chromedp.NewContext(allocator)
	defer cancel()

	tasks := []chromedp.Action{
		chromedp.EmulateViewport(int64(r.Options.CaptureWidth), int64(r.Options.CaptureHeight)),
	}

	if r.Options.FollowRedirects {
		tasks = append(tasks, chromedp.Navigate(url))
	}

	// Conditionally wait for the page to fully load based on the WaitForPageBody option.
	if r.Options.WaitForPageBody {
		tasks = append(tasks, chromedp.WaitVisible("html body", chromedp.ByQuery))
	}

	tasks = append(tasks, chromedp.CaptureScreenshot(&result.Image))

	err := chromedp.Run(cctx, chromedp.Tasks(tasks))
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			Log.Warnf("Timeout exceeded for %s", url)
		} else {
			result.Error = err
		}
		return result
	}

	// If SaveUnique is enabled, check for uniqueness
	if r.Options.SaveUnique {
		unique, err := checkHashUnique(result.Image)
		if err != nil {
			Log.Warnf("Could not perform uniqueness check: %v", err)
		} else if !unique {
			Log.Infof("Duplicate screenshot found for %s. Skipping save.", url)
			shouldSave = false
		}
	}

	if r.Options.SaveScreenshots && shouldSave {
		if r.Options.SaveScreenshotsPath == "" {
			r.Options.SaveScreenshotsPath = DefaultOptions().SaveScreenshotsPath
		}
		_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
		if err != nil {
			Log.Warnf("Could not save screenshot for %s: %v", url, err)
		} else {
			Log.Infoln("Screenshot", url, "saved to", r.Options.SaveScreenshotsPath)
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
	u, err := url.Parse(result.URL)
	if err != nil {
		return "", err
	}

	var path string
	if u.Path != "" {
		path = "_" + u.Path
	}

	// Create a filename that includes the scheme, host, and port.
	fileName := filepath.Join(folderPath, u.Scheme+"_"+u.Host+path+".png")

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
