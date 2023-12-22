<p align="center">
<img src="./assets/logo.png" alt="screener logo" width="300"/>
</p>

screener is a command-line interface (CLI) and Golang library for capturing screenshots of web pages. Built on top of [Rod](https://github.com/go-rod/rod).

## Features

- **Stream URLs for Screenshotting**: Accepts URLs via standard input (STDIN) and processes them in real-time.
- **Max Wait for Page Load**: Waits for a specified maximum time for the web page to load before capturing the screenshot, ensuring the capture of the most relevant content.
- **Redirect Handling**: Customizable option for following or ignoring redirects, providing control over how URL changes are managed.
- **Unique Screenshots**: Offers an option to save only unique screenshots, avoiding duplication.
- **Concurrency Support**: Facilitates fast processing with concurrent requests.
- **Certificate Error Handling**: Includes an option to ignore SSL certificate errors, useful for testing environments.
- **HTTP/2 Control**: Allows disabling HTTP/2, offering compatibility with different server configurations.
- **Custom User-Agent**: Enables setting a custom user-agent for requests, allowing simulation of different browsers or devices.

## Installation

### Go
```
go install github.com/root4loot/screener/cmd/screener@latest
```

### Docker

```
git clone https://github.com/root4loot/screener.git && cd screener
docker build -t screener .
docker run -it screener -h
```

## Usage

```
Usage: screener [options] (-u <target> | -i <targets.txt>)

INPUT:
   -t,      --target             single target
   -i,      --infile             file containing targets (one per line)

CONFIGURATIONS:
   -c,   --concurrency           number of concurrent requests                  (Default: 10)
   -t,   --timeout               timeout for screenshot capture                 (Default: 15 seconds)
   -ua,  --user-agent            set user agent                                 (Default: Chrome Headless)
   -su,  --save-unique           save unique screenshots only                   (Default: false)
   -dh,  --disable-http2         disable HTTP2                                  (Default: true)
   -fr,  --follow-redirects      follow redirects                               (Default: true)
   -cw,  --capture-width         screenshot pixel width                         (Default: 1920)
   -ch,  --capture-height        screenshot pixel height                        (Default: 1080)
   -wp,  --wait-page             wait for page to fully load before capturing   (Default: true)
   -wt,  --wait-time             wait time before capturing (seconds)           (Default: 30)
   -ice, --ignore-cert-err       ignore certificate errors                      (Default: true)
   -isc, --ignore-status-codes   ignore HTTP status codes  (comma separated)    (Default: [])
   -s,   --silence               silence output                                 (Default: false)

OUTPUT:
   -o,     --outfolder           save images to given folder     (Default: ./screenshots)
   -s,     --silence             silence output
   -v,     --verbose             verbose output
           --version             display version
```

## Example

### Screenshotting a Single Target
Capture a screenshot from a single URL. If the URL scheme (http/https) is not specified, then it will handle both:

```sh
✗ screener -t "example.com"
# Captures both http://example.com/ and https://example.com/
[screener] (INFO) Screenshot http://example.com/ saved to ./screenshots                                                                                                                                                                                                                                                                                                  
[screener] (INFO) Screenshot https://example.com/ saved to ./screenshots

✗ screener -t "google.com"
# Captures https://www.google.com (redirects from http to https)
[screener] (INFO) Screenshot https://www.google.com saved to ./screenshots 
```

### Screenshotting URLs from a File
Capture screenshots from multiple URLs listed in a file but wait for pages to load first and only save unique images.

```sh
✗ screener -i urls.txt --wait-page --save-unique 
[screener] (INFO) Screenshot http://example.com/ saved to ./screenshots                                                                                                                                                                                                                                                                                                  
[screener] (INFO) Screenshot https://example.com/ saved to ./screenshots                                                                                                                                                                                                                                                                                                 
[screener] (INFO) Screenshot https://github.com/ saved to ./screenshots                                                                                                                                                                                                                                                                                                  
[screener] (INFO) Screenshot https://consent.yahoo.com saved to ./screenshots                                                                                                                                                                                                                                                                                            
[screener] (INFO) Screenshot https://www.google.com saved to ./screenshots                                                                                                                                                                                                                                                                                               
[screener] (INFO) Screenshot https://www.facebook.com saved to ./screenshots                                                                                                                                                                                                                                                                                             
[screener] (INFO) Screenshot https://www.hackerone.com saved to ./screenshots                                                                                                                                                                                                                                                                                            
[screener] (INFO) Screenshot https://www.bugcrowd.com saved to ./screenshots 
```

## Piping URLs from SDIN
Stream URLs to Screener, capturing screenshots as they are received:

```sh
✗ cat urls.txt | screener                        
[screener] (INFO) Screenshot http://example.com/ saved to ./screenshots
[screener] (INFO) Screenshot https://example.com/ saved to ./screenshots
[screener] (INFO) Screenshot https://www.hackerone.com saved to ./screenshots
[screener] (INFO) Screenshot https://www.bugcrowd.com saved to ./screenshots
[screener] (INFO) Screenshot https://www.google.com saved to ./screenshots
[screener] (INFO) Screenshot https://www.facebook.com saved to ./screenshots
[screener] (INFO) Screenshot https://consent.yahoo.com saved to ./screenshots
[screener] (INFO) Screenshot https://github.com/ saved to ./screenshots
```


## Library Example 📦

```
go get github.com/root4loot/screener
```

```go
package main

import (
	"fmt"

	"github.com/root4loot/screener"
)

func main() {

	// List of URLs to capture
	urls := []string{
		"https://example.com",
		"https://hackerone.com",
		"https://bugcrowd.com",
		"https://google.com",
		"https://facebook.com",
		"https://yahoo.com",
		"https://tesla.com",
		"https://github.com",
	}

	// Set options
	options := screener.Options{
		Concurrency:             10,
		Timeout:                 10,
		SaveScreenshots:         true,
		SaveScreenshotsPath:     "customfolder",
		WaitForPageLoad:         true,
		WaitTime:                1,
		FollowRedirects:         true,
		DisableHTTP2:            true,
		IgnoreCertificateErrors: true,
		Verbose:                 false,
		Silence:                 true,
		CaptureWidth:            1920,
		CaptureHeight:           1080,
	}

	// Create a screener runner with options
	runner := screener.NewRunnerWithOptions(options)

	// Create a channel to receive results
	results := make(chan screener.Result)

	// Start capturing URLs using multiple goroutines
	go runner.MultipleStream(results, urls...)

	// Process the results as they come in
	for result := range results {
		fmt.Println(result.RequestURL, result.FinalURL, result.Error, len(result.Image))
	}
}

```

For more, see [examples](https://github.com/root4loot/screener/tree/master/examples)

## License

See [LICENSE](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)