package screener

import (
	"bytes"
	"context"
	"embed"
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
	"github.com/glaslos/ssdeep"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/golang/freetype/truetype"
	"github.com/root4loot/goutils/log"
	"github.com/root4loot/goutils/urlutil"
	"golang.org/x/image/font"
)

func Init() {
	log.Init("screener")
	log.SetLevel(log.InfoLevel)
}

var fuzzyHashes = make(map[string]map[string]bool) // Map of fuzzy hashes for duplicate detection

func (r *Runner) captureTarget(TargetURL string) Result {
	log.Debugf("Running capture on %s", TargetURL)
	result := Result{TargetURL: TargetURL}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	defer cancel()

	l := newLauncher(*r.Options)
	browserURL := l.MustLaunch()
	browser := rod.New().ControlURL(browserURL).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

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

	if r.Options.IgnoreRedirects && page.MustInfo().URL != TargetURL {
		log.Warn("Not following redirects as --ignore-redirects flag is set")
		return result
	}

	err := page.Context(ctx).WaitLoad()
	if err != nil {
		log.Warnf("%s timed out after %v: %v", time.Duration(r.Options.Timeout)*time.Second, TargetURL, err)
	}

	if r.Options.FixedWait > 0 {
		time.Sleep(time.Duration(r.Options.FixedWait) * time.Second)
	}

	result.LandingURL = page.MustInfo().URL

	screenshot, err := page.Screenshot(r.Options.CaptureFull, nil)
	if err != nil {
		log.Warnf("Error capturing screenshot for %s: %v", TargetURL, err)
		result.Error = err
		return result
	}
	result.Image = screenshot

	origin, err := urlutil.GetOrigin(result.TargetURL)
	if err != nil {
		log.Warnf("Error getting origin for %s: %v", TargetURL, err)
		result.Error = err
		return result
	}

	if r.Options.ImprintURL {
		result.Image, err = r.addURLtoImage(result.Image, origin)
		if err != nil {
			log.Warnf("Error adding text to image for %s: %v", origin, err)
			result.Error = err
		}
	}

	if r.Options.SaveScreenshots {
		if r.Options.SaveUnique && isDuplicate(result.TargetURL, result.Image) {
			return result
		} else {
			_, err := result.WriteToFolder(r.Options.SaveScreenshotsPath)
			if err != nil {
				log.Warnf("Error saving screenshot for %s: %v", TargetURL, err)
				result.Error = err
			}
		}
	}

	return result
}

func (result Result) WriteToFolder(writeFolderPath string) (filename string, err error) {
	if len(result.Image) == 0 {
		return "", nil
	}

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

	if parsedTargetURL.Path == "" {
		parsedWriteURL.Path = ""
	}

	if parsedRedirectURL.Scheme != "" {
		parsedWriteURL.Scheme = parsedRedirectURL.Scheme
	}

	if (parsedWriteURL.Scheme == "http" || parsedWriteURL.Scheme == "https") && parsedWriteURL.Port() == "80" || parsedWriteURL.Port() == "443" {
		parsedWriteURL.Host = strings.Split(parsedWriteURL.Host, ":")[0]
	}

	filename = parsedWriteURL.Scheme + "_" + parsedWriteURL.Host + parsedWriteURL.Path
	filename = strings.TrimSuffix(filename, "/")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename = filepath.Join(writeFolderPath, filename+".png")

	file, err := os.Create(strings.ToLower(filename))
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(result.Image)
	if err == nil {
		log.Resultf("Saved screenshot to %s", filename)
	} else {
		return "", err
	}

	return filename, nil
}

func newLauncher(options Options) *launcher.Launcher {
	path, _ := launcher.LookPath()

	l := launcher.New().
		Headless(true).
		Bin(path).
		NoSandbox(true)

	if options.UserAgent != "" {
		l.Set("user-agent", options.UserAgent)
	}

	if !options.RespectCertificateErrors {
		l.Set("ignore-certificate-errors", "true")
	}

	if !options.UseHTTP2 {
		l.Set("disable-http2", "true")
	}

	return l
}

func isDuplicate(rawURL string, image []byte) bool {

	u, err := url.Parse(rawURL)
	if err != nil {
		log.Warnf("Error getting hostname for %s: %v", u, err)
		return false
	}

	hash, _ := ssdeep.FuzzyBytes(image)

	if fuzzyHashes[u.Host] == nil {
		fuzzyHashes[u.Host] = make(map[string]bool)
	}

	for existingHash := range fuzzyHashes[u.Host] {
		score, _ := ssdeep.Distance(existingHash, hash)

		if score < 96 {
			log.Info("Skipping duplicate screenshot for", rawURL)
			return false
		}
	}

	fuzzyHashes[u.Host][hash] = true
	return true
}

func (r *Runner) addURLtoImage(imgBytes []byte, rawURL string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	printURL, err := getPrintableURL(rawURL, 159)
	if err != nil {
		return nil, err
	}

	const padding = 20
	const borderSize = 1

	w := img.Bounds().Dx()
	h := img.Bounds().Dy() + padding*2 + borderSize
	dc := gg.NewContext(w, h)

	dc.DrawImage(img, 0, 0)

	yLine := float64(img.Bounds().Dy())
	dc.SetColor(color.Black)
	dc.DrawLine(0, yLine, float64(w), yLine)
	dc.SetLineWidth(float64(borderSize))
	dc.Stroke()
	dc.SetColor(color.White)
	dc.DrawRectangle(0, yLine, float64(w), float64(padding*2))
	dc.Fill()
	dc.SetColor(color.Black)
	dc.SetFontFace(loadFont())
	dc.DrawStringAnchored(printURL, float64(w)/2, yLine+float64(padding), 0.2, 0.3)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

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

//go:embed assets/Roboto-Medium.ttf
var fontBytes embed.FS

func loadFont() font.Face {
	fontData, err := fontBytes.ReadFile("assets/Roboto-Medium.ttf")
	if err != nil {
		log.Fatalf("Failed to read embedded font: %v", err)
	}

	ttFont, err := truetype.Parse(fontData)
	if err != nil {
		log.Fatalf("Failed to parse embedded font: %v", err)
	}

	return truetype.NewFace(ttFont, &truetype.Options{
		Size: 14,
	})
}
