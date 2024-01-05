package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	screener "github.com/root4loot/screener"
)

func (c *CLI) banner() {
	fmt.Println("\nnpmjack", screener.Version, "by", author)
}

func (c *CLI) usage() {
	w := tabwriter.NewWriter(os.Stdout, 2, 0, 3, ' ', 0)

	fmt.Fprintf(w, "Usage:\t%s [options] (-u <target> | -i <targets.txt>)\n", os.Args[0])

	fmt.Fprintf(w, "\nINPUT:\n")
	fmt.Fprintf(w, "\t%s,  %s\t\t\t\t       %s\n", "-t", "--target", "single target")
	fmt.Fprintf(w, "\t%s,  %s\t\t\t         %s\n", "-l", "--list", " input file containing list of targets (one per line)")

	fmt.Fprintf(w, "\nCONFIGURATIONS:\n")
	fmt.Fprintf(w, "\t%s,   %s\t%s\t(Default: %d)\n", "-c", "--concurrency", "number of concurrent requests", screener.DefaultOptions().Concurrency)
	fmt.Fprintf(w, "\t%s,   %s\t%s\t(Default: %d seconds)\n", "-to", "--timeout", "timeout for screenshot capture", screener.DefaultOptions().Timeout)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %s)\n", "-ua", "--user-agent", "set user agent", "Chrome Headless")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-su", "--save-unique", "save unique screenshots only", screener.DefaultOptions().SaveUnique)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-dh", "--disable-http2", "disable HTTP2", screener.DefaultOptions().DisableHTTP2)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-fr", "--follow-redirects", "follow redirects", screener.DefaultOptions().FollowRedirects)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d)\n", "-cw", "--capture-width", "screenshot pixel width", screener.DefaultOptions().CaptureWidth)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d)\n", "-ch", "--capture-height", "screenshot pixel height", screener.DefaultOptions().CaptureHeight)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d)\n", "-cf", "--capture-full", "capture full page", screener.DefaultOptions().CaptureHeight)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-mw", "--max-wait", "max wait time before capturing (seconds)", screener.DefaultOptions().MaxWait)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-fw", "--fixed-wait", "fixed wait time before capturing (seconds)", screener.DefaultOptions().FixedWait)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-dc", "--delay-between-capture", "delay between capture (seconds)", screener.DefaultOptions().DelayBetweenCapture)
	fmt.Fprintf(w, "\t%s, %s\t%s\t(Default: %v)\n", "-ice", "--ignore-cert-err", "ignore certificate errors", screener.DefaultOptions().IgnoreCertificateErrors)
	fmt.Fprintf(w, "\t%s, %s\t%s\t(Default: %v)\n", "-isc", "--ignore-status-codes", "ignore HTTP status codes  (comma separated)", screener.DefaultOptions().IgnoreStatusCodes)
	fmt.Fprintf(w, "\t%s,   %s\t%s\t(Default: %v)\n", "-s", "--silence", "silence output", screener.DefaultOptions().Silence)

	fmt.Fprintf(w, "\nOUTPUT:\n")
	fmt.Fprintf(w, "\t%s,   %s\t\t\t\t %s\t\t\t\t\t    (Default: %s)\n", "-o", "--outfolder", "save images to given folder", screener.DefaultOptions().SaveScreenshotsPath)
	fmt.Fprintf(w, "\t%s,  %s\t\t\t\t %s\t\t\t\t\t    (Default: %v)\n", "-wu", "--without-url", "without URL in image", !screener.DefaultOptions().URLInImage)
	fmt.Fprintf(w, "\t%s,   %s\t\t\t\t %s\n", "-s", "--silence", "silence output")
	fmt.Fprintf(w, "\t%s,   %s\t\t\t\t %s\n", "-v", "--verbose", "verbose output")
	fmt.Fprintf(w, "\t%s    %s\t\t\t\t %s\n", "  ", "--version", "display version")

	w.Flush()
	fmt.Println("")
}

// parseAndSetOptions parses the command line options and sets the options
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
	flag.BoolVar(&c.Options.DisableHTTP2, "disable-http2", screener.DefaultOptions().DisableHTTP2, "")
	flag.BoolVar(&c.Options.DisableHTTP2, "dh", screener.DefaultOptions().DisableHTTP2, "")
	flag.BoolVar(&c.Options.FollowRedirects, "follow-redirects", screener.DefaultOptions().FollowRedirects, "")
	flag.BoolVar(&c.Options.FollowRedirects, "fr", screener.DefaultOptions().FollowRedirects, "")
	flag.BoolVar(&c.Options.IgnoreCertificateErrors, "ignore-cert-err", screener.DefaultOptions().IgnoreCertificateErrors, "")
	flag.BoolVar(&c.Options.IgnoreCertificateErrors, "ice", screener.DefaultOptions().IgnoreCertificateErrors, "")
	flag.IntVar(&c.Options.MaxWait, "max-wait", screener.DefaultOptions().MaxWait, "")
	flag.IntVar(&c.Options.MaxWait, "mw", screener.DefaultOptions().MaxWait, "")
	flag.IntVar(&c.Options.MaxWait, "fixed-wait", screener.DefaultOptions().FixedWait, "")
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
	flag.BoolVar(&c.Options.URLInImage, "without-url", !screener.DefaultOptions().URLInImage, "")
	flag.BoolVar(&c.Options.URLInImage, "wu", !screener.DefaultOptions().URLInImage, "")
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
