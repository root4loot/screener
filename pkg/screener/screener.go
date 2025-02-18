package screener

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/glaslos/ssdeep"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/golang/freetype/truetype"
	"github.com/root4loot/goutils/log"
	"github.com/root4loot/goutils/sliceutil"
	"github.com/root4loot/goutils/urlutil"
	"golang.org/x/image/font"
)

type Screener struct {
	Debug          bool
	CaptureOptions captureOptions
	visited        map[string]bool
	mutex          sync.Mutex
}

// Result contains the result of a screenshot capture.
type Result struct {
	TargetURL  string
	LandingURL string
	Image      Image
	StatusCode int
	Error      error
}

type Image []byte

// captureOptions contains the options for capturing screenshots.
type captureOptions struct {
	CaptureHeight            int    // Height of the capture
	CaptureWidth             int    // Width of the capture
	Timeout                  int    // Timeout for each capture (seconds)
	RespectCertificateErrors bool   // Respect certificate errors
	UseHTTP2                 bool   // Use HTTP2
	UserAgent                string // User agent
	DelayBeforeCapture       int    // Delay before capture (seconds)
	DelayBetweenCapture      int    // Delay between captures (seconds)
	IgnoreRedirects          bool   // Do not follow redirects
	IgnoreStatusCodes        []int  // Status codes to ignore
	CaptureFull              bool   // Take a full-page screenshot
}

// NewOptions returns an CaptureOptions struct initialized with default values.
func NewOptions() captureOptions {
	return captureOptions{
		CaptureHeight:            768,
		CaptureWidth:             1366,
		Timeout:                  15,
		RespectCertificateErrors: false,
		UseHTTP2:                 false,
		DelayBeforeCapture:       2,
		DelayBetweenCapture:      0,
		IgnoreRedirects:          false,
		CaptureFull:              false,
		UserAgent:                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	}
}

// NewScreener creates a Screener with default options.
func NewScreener() *Screener {
	return &Screener{
		CaptureOptions: NewOptions(),
		visited:        make(map[string]bool),
	}
}

// NewScreenerWithOptions creates a Screener with the provided options.
func NewScreenerWithOptions(options captureOptions) *Screener {
	return &Screener{
		CaptureOptions: options,
		visited:        make(map[string]bool),
	}
}

// SetDebug enables or disables debug mode.
func (s *Screener) SetDebug(debug bool) {
	s.Debug = debug
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func Init() {
	log.Init("screener")
	log.SetLevel(log.InfoLevel)
}

// CaptureScreenshot takes a screenshot of the provided URL and returns the result.
func (s *Screener) CaptureScreenshot(parsedURL *url.URL) (*Result, error) {
	var result = &Result{}

	captureURL := parsedURL.String()
	result.TargetURL = captureURL

	if s.CaptureOptions.DelayBetweenCapture > 0 {
		time.Sleep(time.Duration(s.CaptureOptions.DelayBetweenCapture) * time.Second)
	}

	if !strings.HasSuffix(parsedURL.Path, "/") && !urlutil.HasFileExtension(parsedURL.Path) {
		parsedURL.Path += "/"
		captureURL = parsedURL.String()
	}

	if s.isVisited(captureURL) {
		log.Warnf("Skipping %s as it has already been visited", captureURL)
		return nil, nil
	} else {
		log.Debugf("Attempting capture on %s", captureURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.CaptureOptions.Timeout)*time.Second)
	defer cancel()

	path, _ := launcher.LookPath()

	l := launcher.New().
		Headless(true).
		Bin(path).
		NoSandbox(true)

	if s.CaptureOptions.UserAgent != "" {
		l.Set("user-agent", s.CaptureOptions.UserAgent)
	}

	if !s.CaptureOptions.RespectCertificateErrors {
		l.Set("ignore-certificate-errors", "true")
	}

	if !s.CaptureOptions.UseHTTP2 {
		l.Set("disable-http2", "true")
	}

	browserURL := l.MustLaunch()
	browser := rod.New().ControlURL(browserURL).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	if s.CaptureOptions.CaptureWidth != 0 && s.CaptureOptions.CaptureHeight != 0 {
		viewport := &proto.EmulationSetDeviceMetricsOverride{
			Width:             s.CaptureOptions.CaptureWidth,
			Height:            s.CaptureOptions.CaptureHeight,
			DeviceScaleFactor: 1,
			Mobile:            false,
		}

		err := page.SetViewport(viewport)
		if err != nil {
			log.Fatalf("Error setting viewport: %v", err)
		}
	}

	var e proto.NetworkResponseReceived
	wait := page.WaitEvent(&e)

	if err := page.Context(ctx).Navigate(captureURL); err != nil {
		return nil, fmt.Errorf("error navigating to %s: %w", captureURL, err)
	}

	wait()

	if s.CaptureOptions.IgnoreRedirects && page.MustInfo().URL != captureURL {
		log.Warn("Not following redirects as --ignore-redirects flag is set")
		return nil, nil
	}

	if err := page.Context(ctx).WaitLoad(); err != nil {
		return nil, fmt.Errorf("%s timed out after %v: %w", time.Duration(s.CaptureOptions.Timeout)*time.Second, captureURL, err)
	}

	if s.CaptureOptions.DelayBeforeCapture > 0 {
		time.Sleep(time.Duration(s.CaptureOptions.DelayBeforeCapture) * time.Second)
	}

	result.LandingURL = page.MustInfo().URL

	var err error
	result.Image, err = page.Screenshot(s.CaptureOptions.CaptureFull, nil)
	if err != nil {
		return nil, fmt.Errorf("error capturing screenshot for %s: %w", captureURL, err)
	}

	if sliceutil.Contains(s.CaptureOptions.IgnoreStatusCodes, e.Response.Status) {
		log.Warnf("Ignoring %s as it returned status code %d", captureURL, e.Response.Status)
		return nil, nil
	}

	result.StatusCode = e.Response.Status
	s.addVisited(captureURL)

	return result, nil
}

// SaveImageToFolder saves the image to the provided path.
func (result Result) SaveImageToFolder(localFilePath string) (filename string, err error) {
	if len(result.Image) == 0 {
		return "", err
	}

	err = os.MkdirAll(localFilePath, os.ModePerm)
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
	filename = filepath.Join(localFilePath, filename+".png")

	file, err := os.Create(strings.ToLower(filename))
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(result.Image)
	if err != nil {
		return "", err
	}

	return filename, nil
}

// IsSimilarToAny checks if the image is a duplicate of any of the images in the results slice
func (result Result) IsSimilarToAny(results []Result, similarityThreshold int) bool {
	if similarityThreshold < 1 || similarityThreshold > 100 {
		log.Errorf("Invalid similarity threshold: %d. Must be between 1 and 100", similarityThreshold)
		os.Exit(1)
	}

	hash1, _ := ssdeep.FuzzyBytes(result.Image)

	for _, r := range results {
		hash2, _ := ssdeep.FuzzyBytes(r.Image)
		score, _ := ssdeep.Distance(hash1, hash2)

		if score >= similarityThreshold {
			log.Debugf("%s is similar to %s with a score of %d. Skipping.. ", result.TargetURL, r.TargetURL, score)
			return true
		}
	}
	return false
}

// AddTextToImage adds text to bottom of the image
func (imgB Image) AddTextToImage(rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	host := parsedURL.Host
	if strings.Contains(host, ":") {
		hostWithoutPort, port, _ := strings.Cut(host, ":")
		if (parsedURL.Scheme == "http" && port == "80") || (parsedURL.Scheme == "https" && port == "443") {
			host = hostWithoutPort
		}
	}

	printURL := parsedURL.Scheme + "://" + host

	img, err := png.Decode(bytes.NewReader(imgB))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
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

func (s *Screener) addVisited(str string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.visited[str] = true
}

func (s *Screener) isVisited(str string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.visited[str]
}
