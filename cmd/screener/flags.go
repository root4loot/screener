package main

import (
	"flag"
	"fmt"
	"os"
	"text/template"

	screener "github.com/root4loot/screener"
)

type usageData struct {
	AppName                    string
	DefaultConcurrency         int
	DefaultTimeout             int
	DefaultUserAgent           string
	DefaultSaveUnique          bool
	DefaultUseHTTP2            bool
	DefaultIgnoreRedirects     bool
	DefaultCaptureWidth        int
	DefaultCaptureHeight       int
	DefaultCaptureFull         bool
	DefaultFixedWait           int
	DefaultDelayBetweenCapture int
	DefaultRespectCertErr      bool
	DefaultIgnoreStatusCodes   bool
	DefaultSilence             bool
	DefaultOutFolder           string
	DefaultNoURL               bool
}

func (c *CLI) usage() {
	options := screener.DefaultOptions()
	data := usageData{
		AppName:                    os.Args[0],
		DefaultConcurrency:         options.Concurrency,
		DefaultTimeout:             options.Timeout,
		DefaultUserAgent:           "Chrome Headless",
		DefaultSaveUnique:          options.SaveUnique,
		DefaultUseHTTP2:            options.UseHTTP2,
		DefaultIgnoreRedirects:     options.IgnoreRedirects,
		DefaultCaptureWidth:        options.CaptureWidth,
		DefaultCaptureHeight:       options.CaptureHeight,
		DefaultCaptureFull:         options.CaptureFull,
		DefaultFixedWait:           options.FixedWait,
		DefaultDelayBetweenCapture: options.DelayBetweenCapture,
		DefaultRespectCertErr:      options.RespectCertificateErrors,
		DefaultIgnoreStatusCodes:   len(options.IgnoreStatusCodes) > 0,
		DefaultSilence:             options.Silence,
		DefaultOutFolder:           options.SaveScreenshotsPath,
		DefaultNoURL:               !options.ImprintURL,
	}

	tmpl, err := template.New("usage").Parse(usageTemplate)
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

const usageTemplate = `
Usage:
  {{.AppName}} [options] (-u <target> | -l <targets.txt>)

INPUT:
  -t, --target                   single target
  -l, --list                     input file containing list of targets (one per line)

CONFIGURATIONS:
  -c,   --concurrency            number of concurrent requests               (Default: {{.DefaultConcurrency}})
  -to,  --timeout                timeout for screenshot capture              (Default: {{.DefaultTimeout}} seconds)
  -ua,  --user-agent             set user agent                              (Default: {{.DefaultUserAgent}})
  -su,  --save-unique            save unique screenshots only                (Default: {{.DefaultSaveUnique}})
  -uh,  --use-http2              use HTTP2                                   (Default: {{.DefaultUseHTTP2}})
  -nr,  --ignore-redirects       do not follow redirects                     (Default: {{.DefaultIgnoreRedirects}})
  -cw,  --capture-width          screenshot pixel width                      (Default: {{.DefaultCaptureWidth}})
  -ch,  --capture-height         screenshot pixel height                     (Default: {{.DefaultCaptureHeight}})
  -cf,  --capture-full           capture full page                           (Default: {{.DefaultCaptureFull}})
  -fw,  --fixed-wait             fixed wait time before capturing (seconds)  (Default: {{.DefaultFixedWait}})
  -dc,  --delay-between-capture  delay between capture (seconds)             (Default: {{.DefaultDelayBetweenCapture}})
  -rce, --respect-cert-err       ignore certificate errors                   (Default: {{.DefaultRespectCertErr}})
  -isc, --ignore-status-codes    ignore HTTP status codes (comma separated)  (Default: {{.DefaultIgnoreStatusCodes}})
  -s,   --silence                silence output                              (Default: {{.DefaultSilence}})

OUTPUT:
  -o,   --outfolder              save images to given folder                 (Default: {{.DefaultOutFolder}})
  -nu,  --no-url                 do not imprint URL in image                 (Default: {{.DefaultNoURL}})
  -s,   --silence                silence output
  -v,   --verbose                verbose output
        --version                display version
`

func (c *CLI) parseFlags() {
	// TARGET
	flag.StringVar(&c.TargetURL, "target", "", "")
	flag.StringVar(&c.TargetURL, "t", "", "")
	flag.StringVar(&c.Infile, "l", "", "")
	flag.StringVar(&c.Infile, "list", "", "")

	// CONFIGURATIONS
	flag.IntVar(&c.Options.Concurrency, "concurrency", screener.DefaultOptions().Concurrency, "")
	flag.IntVar(&c.Options.Concurrency, "c", screener.DefaultOptions().Concurrency, "")
	flag.IntVar(&c.Options.CaptureHeight, "capture-height", screener.DefaultOptions().CaptureHeight, "")
	flag.IntVar(&c.Options.CaptureHeight, "ch", screener.DefaultOptions().CaptureHeight, "")
	flag.IntVar(&c.Options.CaptureWidth, "capture-width", screener.DefaultOptions().CaptureWidth, "")
	flag.IntVar(&c.Options.CaptureWidth, "cw", screener.DefaultOptions().CaptureWidth, "")
	flag.BoolVar(&c.Options.UseHTTP2, "use-http2", screener.DefaultOptions().UseHTTP2, "")
	flag.BoolVar(&c.Options.UseHTTP2, "uh", screener.DefaultOptions().UseHTTP2, "")
	flag.BoolVar(&c.Options.IgnoreRedirects, "ignore-redirects", screener.DefaultOptions().IgnoreRedirects, "")
	flag.BoolVar(&c.Options.IgnoreRedirects, "ir", screener.DefaultOptions().IgnoreRedirects, "")
	flag.BoolVar(&c.Options.RespectCertificateErrors, "respect-cert-err", screener.DefaultOptions().RespectCertificateErrors, "")
	flag.BoolVar(&c.Options.RespectCertificateErrors, "rce", screener.DefaultOptions().RespectCertificateErrors, "")
	flag.IntVar(&c.Options.FixedWait, "fixed-wait", screener.DefaultOptions().FixedWait, "")
	flag.IntVar(&c.Options.FixedWait, "fw", screener.DefaultOptions().FixedWait, "")
	flag.IntVar(&c.Options.Timeout, "timeout", screener.DefaultOptions().Timeout, "")
	flag.IntVar(&c.Options.Timeout, "to", screener.DefaultOptions().Timeout, "")
	flag.StringVar(&c.Options.UserAgent, "user-agent", screener.DefaultOptions().UserAgent, "")
	flag.StringVar(&c.Options.UserAgent, "ua", screener.DefaultOptions().UserAgent, "")
	flag.StringVar(&c.Options.SaveScreenshotsPath, "outfolder", screener.DefaultOptions().SaveScreenshotsPath, "")
	flag.StringVar(&c.Options.SaveScreenshotsPath, "o", screener.DefaultOptions().SaveScreenshotsPath, "")
	flag.BoolVar(&c.Options.SaveUnique, "save-unique", screener.DefaultOptions().SaveUnique, "")
	flag.BoolVar(&c.Options.SaveUnique, "su", screener.DefaultOptions().SaveUnique, "")
	flag.StringVar(&c.IgnoreStatusCodes, "ignore-status-codes", "", "")
	flag.StringVar(&c.IgnoreStatusCodes, "isc", "", "")
	flag.IntVar(&c.Options.DelayBetweenCapture, "delay-between-capture", screener.DefaultOptions().DelayBetweenCapture, "")
	flag.IntVar(&c.Options.DelayBetweenCapture, "dc", screener.DefaultOptions().DelayBetweenCapture, "")
	flag.BoolVar(&c.Options.CaptureFull, "capture-full", screener.DefaultOptions().CaptureFull, "")
	flag.BoolVar(&c.Options.CaptureFull, "cf", screener.DefaultOptions().CaptureFull, "")

	// OUTPUT
	flag.BoolVar(&c.Options.ImprintURL, "no-url", screener.DefaultOptions().ImprintURL, "")
	flag.BoolVar(&c.Options.ImprintURL, "nu", screener.DefaultOptions().ImprintURL, "")
	flag.BoolVar(&c.Options.Silence, "s", false, "")
	flag.BoolVar(&c.Options.Silence, "silence", false, "")
	flag.BoolVar(&c.Options.Verbose, "v", false, "")
	flag.BoolVar(&c.Options.Verbose, "verbose", false, "")
	flag.BoolVar(&c.Help, "help", false, "")
	flag.BoolVar(&c.Help, "h", false, "")
	flag.BoolVar(&c.Version, "version", false, "")

	flag.Usage = func() {
		c.banner()
		c.usage()
	}
	flag.Parse()
}

func (c *CLI) banner() {
	fmt.Println("\nscreener", screener.Version, "by", author, "\n")
}
