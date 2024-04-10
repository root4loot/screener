package screener

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/golang/freetype/truetype"
	"github.com/root4loot/goutils/log"
	"golang.org/x/image/font"
)

func Init() {
	log.Init("screener")
	log.SetLevel(log.InfoLevel)
}

var seenHashes = make(map[string]struct{}) // map of hashes to check for uniqueness

func (r *Runner) worker(TargetURL string) Result {
	log.Info("Preparing screenshot:", TargetURL)
	result := Result{TargetURL: TargetURL}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()

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
			log.Fatalf("Error setting viewport: %v", err)
		}
	}

	if err := page.Context(ctx).Navigate(TargetURL); err != nil {
		log.Warnf("Error navigating to %s: %v", TargetURL, err)
		result.Error = err
		return result
	}

	// Handle redirects
	if !r.Options.FollowRedirects && page.MustInfo().URL != TargetURL {
		log.Warn("Redirect detected, but FollowRedirects is disabled")
		return result
	}

	// Wait for the page to load with a timeout
	err := page.Context(ctx).WaitLoad()
	if err != nil {
		log.Warnf("%s timed out after %v: %v", time.Duration(r.Options.Timeout)*time.Second, TargetURL, err)
	}

	// Additional fixed wait time after page load event
	if r.Options.FixedWait > 0 {
		time.Sleep(time.Duration(r.Options.FixedWait) * time.Second)
	}

	// Update final URL and return result
	result.LandingURL = page.MustInfo().URL

	// Take and process screenshot
	if err := processScreenshot(page, &result, r); err != nil {
		log.Warnf("Error processing screenshot for %s: %v", TargetURL, err)
		result.Error = err
		return result
	}

	return result
}

// processScreenshot handles taking, saving, and uniqueness checking of screenshots.
func processScreenshot(page *rod.Page, result *Result, r *Runner) error {
	var err error

	shouldSave := true
	screenshot, err := page.Screenshot(r.Options.CaptureFull, nil)
	if err != nil {
		return err
	}
	result.Image = screenshot

	// Add text to image if required
	if !r.Options.URLInImage {
		result.Image, err = r.addURLtoImage(result.Image, result.LandingURL)
		if err != nil {
			log.Warnf("Error adding text to image for %s: %v", result.LandingURL, err)
			result.Error = err
		}
	}

	// Check for screenshot uniqueness if required
	if r.Options.SaveUnique {
		unique, err := checkHashUnique(result.Image)
		if err != nil {
			log.Warnf("Could not perform uniqueness check: %v", err)
		} else if !unique {
			log.Infof("Duplicate screenshot found for %s. Skipping save.", result.LandingURL)
			shouldSave = false
		}
	}

	// Save screenshot if required
	if r.Options.SaveScreenshots && shouldSave {
		_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
		if err != nil {
			return err
		}
		log.Resultf("Successful screenshot: %s", result.LandingURL)
	}
	return nil
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

	parsedTargetURL, err := url.Parse(result.TargetURL)
	if err != nil {
		return "", err
	}

	parsedRedirectURL, err := url.Parse(result.LandingURL)
	if err != nil {
		return "", err
	}

	parsedWriteURL := parsedTargetURL

	// Remove path from the URL unless specified in target.
	if parsedTargetURL.Path == "" {
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

// newLauncher creates a new browser launcher with the specified options.
func newLauncher(options Options) *launcher.Launcher {
	// Find the browser path
	path, _ := launcher.LookPath()

	l := launcher.New().
		Headless(true).
		Bin(path).
		NoSandbox(true)

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

// addURLtoImage adds text to the bottom of the image with padding and border.
func (r *Runner) addURLtoImage(imgBytes []byte, rawURL string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get the printable URL.
	printURL, err := getPrintableURL(rawURL, 159)
	if err != nil {
		return nil, err
	}

	// Constants for drawing the text box.
	const padding = 20
	const borderSize = 1

	// Calculate new image height with text box.
	w := img.Bounds().Dx()
	h := img.Bounds().Dy() + padding*2 + borderSize
	dc := gg.NewContext(w, h)

	// Draw the original image.
	dc.DrawImage(img, 0, 0)

	// Draw the border line and background for text.
	yLine := float64(img.Bounds().Dy())
	dc.SetColor(color.Black)
	dc.DrawLine(0, yLine, float64(w), yLine)
	dc.SetLineWidth(float64(borderSize))
	dc.Stroke()
	dc.SetColor(color.White)
	dc.DrawRectangle(0, yLine, float64(w), float64(padding*2))
	dc.Fill()

	// Set up text properties and draw text.
	dc.SetColor(color.Black) // Use black for the text color.

	// Load and set the custom font.
	fontFace := loadFont()
	dc.SetFontFace(fontFace)

	// Draw the string.
	dc.DrawStringAnchored(printURL, float64(w)/2, yLine+float64(padding), 0.2, 0.3)

	// Encode the context to a new image.
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// getPrintableURL returns a shortened URL if the length exceeds a certain limit.
func getPrintableURL(rawURL string, maxLength int) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	if len(parsedURL.String()) > maxLength {
		return parsedURL.Scheme + "://" + parsedURL.Host, nil
	}
	return parsedURL.String(), nil
}

// Embed the font file.
//
//go:embed assets/Roboto-Medium.ttf
var fontBytes embed.FS

func loadFont() font.Face {
	// Load the font
	fontData, err := fontBytes.ReadFile("assets/Roboto-Medium.ttf")
	if err != nil {
		log.Fatalf("Failed to read embedded font: %v", err)
	}

	// Parse the font
	ttFont, err := truetype.Parse(fontData)
	if err != nil {
		log.Fatalf("Failed to parse embedded font: %v", err)
	}

	// Return the font face to be used in your drawing context
	return truetype.NewFace(ttFont, &truetype.Options{
		Size: 14,
	})
}
