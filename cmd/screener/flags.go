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

	fmt.Fprintf(w, "Usage:\t%s [options] (-u <target> | -i <targets.txt>)\n\n", os.Args[0])

	fmt.Fprintf(w, "\nINPUT:\n")
	fmt.Fprintf(w, "\t%s,   %s\t\t\t\t %s\n", "-t", "--target", "single target")
	fmt.Fprintf(w, "\t%s,   %s\t\t\t   %s\n", "-i", "--infile", " file containing targets (one per line)")

	fmt.Fprintf(w, "\nCONFIGURATIONS:\n")
	fmt.Fprintf(w, "\t%s,   %s\t%s\t(Default: %d)\n", "-c", "--concurrency", "number of concurrent requests", screener.DefaultOptions().Concurrency)
	fmt.Fprintf(w, "\t%s,   %s\t%s\t(Default: %d seconds)\n", "-t", "--timeout", "timeout for screenshot capture", screener.DefaultOptions().Timeout)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %s)\n", "-ua", "--user-agent", "set user agent", "Chrome Headless")
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-su", "--save-unique", "save unique screenshots only", screener.DefaultOptions().SaveUnique)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-dh", "--disable-http2", "disable HTTP2", screener.DefaultOptions().DisableHTTP2)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-fr", "--follow-redirects", "follow redirects", screener.DefaultOptions().FollowRedirects)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d)\n", "-cw", "--capture-width", "screenshot pixel width", screener.DefaultOptions().CaptureWidth)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %d)\n", "-ch", "--capture-height", "screenshot pixel height", screener.DefaultOptions().CaptureHeight)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-wp", "--wait-page", "wait for page to fully load before capturing", screener.DefaultOptions().WaitForPageLoad)
	fmt.Fprintf(w, "\t%s,  %s\t%s\t(Default: %v)\n", "-wt", "--wait-time", "wait time before capturing (seconds)", screener.DefaultOptions().WaitTime)
	fmt.Fprintf(w, "\t%s, %s\t%s\t(Default: %v)\n", "-ice", "--ignore-cert-err", "ignore certificate errors", screener.DefaultOptions().IgnoreCertificateErrors)

	fmt.Fprintf(w, "\nOUTPUT:\n")
	fmt.Fprintf(w, "\t%s,  %s\t\t\t  %s\t  (Default: %s)\n", "-o", "--outfolder", "save images to given folder", screener.DefaultOptions().SaveScreenshotsPath)
	fmt.Fprintf(w, "\t%s,  %s\t\t\t  %s\n", "-s", "--silence", "silence output")
	fmt.Fprintf(w, "\t%s,  %s\t\t\t  %s\n", "-v", "--verbose", "verbose output")
	fmt.Fprintf(w, "\t%s   %s\t\t\t  %s\n", "  ", "--version", "display version")

	w.Flush()
	fmt.Println("")
}

// parseAndSetOptions parses the command line options and sets the options
func (c *CLI) parseFlags() {
	// TARGET
	flag.StringVar(&c.TargetURL, "target", "", "")
	flag.StringVar(&c.TargetURL, "t", "", "")
	flag.StringVar(&c.Infile, "i", "", "")
	flag.StringVar(&c.Infile, "infile", "", "")

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
	flag.BoolVar(&c.Options.WaitForPageLoad, "wait-page", screener.DefaultOptions().WaitForPageLoad, "")
	flag.BoolVar(&c.Options.WaitForPageLoad, "wp", screener.DefaultOptions().WaitForPageLoad, "")
	flag.IntVar(&c.Options.WaitTime, "wait-time", screener.DefaultOptions().WaitTime, "")
	flag.IntVar(&c.Options.WaitTime, "wt", screener.DefaultOptions().WaitTime, "")
	flag.IntVar(&c.Options.Timeout, "timeout", screener.DefaultOptions().Timeout, "")
	flag.IntVar(&c.Options.Timeout, "to", screener.DefaultOptions().Timeout, "")
	flag.StringVar(&c.Options.UserAgent, "user-agent", screener.DefaultOptions().UserAgent, "")
	flag.StringVar(&c.Options.UserAgent, "ua", screener.DefaultOptions().UserAgent, "")
	flag.StringVar(&c.Outfolder, "outfolder", screener.DefaultOptions().SaveScreenshotsPath, "")
	flag.StringVar(&c.Outfolder, "o", screener.DefaultOptions().SaveScreenshotsPath, "")
	flag.BoolVar(&c.Options.SaveUnique, "save-unique", screener.DefaultOptions().SaveUnique, "")
	flag.BoolVar(&c.Options.SaveUnique, "su", screener.DefaultOptions().SaveUnique, "")

	// OUTPUT
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
