package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/root4loot/goutils/log"
	"github.com/root4loot/screener"
)

const usage = `
Usage:
  screener [options] (-t <target> | -l <targets.txt>)

INPUT:
  -t, --target                   screenshot target (domain, IP, URL)
  -l, --list                     input file containing list of targets (one per line)

CONFIGURATIONS:
  -c,   --concurrency            number of concurrent requests               (Default: 10)
  -to,  --timeout                timeout for screenshot capture              (Default: 15 seconds)
  -ua,  --user-agent             set user agent                              (Default: Chrome Headless)
  -su,  --save-unique            save unique screenshots only                (Default: false)
  -uh,  --use-http2              use HTTP2                                   (Default: false)
  -nr,  --ignore-redirects       do not follow redirects                     (Default: false)
  -cw,  --capture-width          screenshot pixel width                      (Default: 1366)
  -ch,  --capture-height         screenshot pixel height                     (Default: 768)
  -cf,  --capture-full           capture full page                           (Default: false)
  -fw,  --fixed-wait             fixed wait time before capturing (seconds)  (Default: 0)
  -dc,  --delay-between-capture  delay between capture (seconds)             (Default: 0)
  -rce, --respect-cert-err       ignore certificate errors                   (Default: false)
  -isc, --ignore-status-codes    ignore HTTP status codes (comma separated)  (Default: false)
  -s,   --silence                silence output                              (Default: false)

OUTPUT:
  -o,   --outfolder              save images to given folder                 (Default: screenshots)
  -nu,  --no-url                 do not imprint URL in image                 (Default: false)
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

	flag.Parse()

	if c.Help {
		fmt.Print(usage)
		os.Exit(0)
	}

	if c.Version {
		fmt.Println("screener ", screener.Version)
		os.Exit(0)
	}

	if !c.hasStdin() && !c.hasInfile() && !c.hasTarget() && !c.Help {
		log.Error("Missing target")
	}
}
