<p align="center">
<img src="./assets/logo.png" alt="screener logo" width="300"/>
</p>

screener is a command-line interface (CLI) and Golang library for capturing screenshots of web pages. Built on top of [chromedp](https://github.com/chromedp/chromedp), it utilizes a headless browser for screenshot rendering.

## Features

- **Stream URLs for Screenshotting**: Accepts URLs via standard input (STDIN) and processes them in real-time.
- **Wait for Page to Load**: Allows waiting for the complete rendering of the web page body for more accurate screenshots. This avoids capturing loading icons.
- **Redirect Handling**: Customizable option for following or ignoring redirects.
- **Unique Screenshots**: Provides the option to save only unique screenshots.
- **Concurrency Support**: Enables fast processing through concurrent requests.

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
Usage: screener [options] (-u <url> | -i <urls.txt>)

INPUT:
   -u,   --url                single URL
   -i,   --infile             file containing URLs (one per line)

CONFIGURATIONS:
   -c,   --concurrency        number of concurrent requests                  (Default: 10)
   -t,   --timeout            timeout for screenshot capture                 (Default: 15 seconds)
   -ua,  --user-agent         set user agent                                 (Default: Chrome Headless)
   -su,  --save-unique        save unique screenshots only                   (Default: false)
   -dh,  --disable-http2      disable HTTP2                                  (Default: true)
   -fr,  --follow-redirects   follow redirects                               (Default: true)
   -cw,  --capture-width      screenshot pixel width                         (Default: 1920)
   -ch,  --capture-height     screenshot pixel height                        (Default: 1080)
   -wp,  --wait-page          wait for page to fully load before capturing   (Default: true)
   -wt,  --wait-time          wait time before capturing (seconds)           (Default: 1)
   -ice, --ignore-cert-err    ignore certificate errors                      (Default: true)

OUTPUT:
   -o,  --outfolder           save images to given folder     (Default: ./screenshots)
   -s,  --silence             silence output
   -v,  --verbose             verbose output
        --version             display version
```


## Example Usage

### Screenshotting URLs in file
In this example, Screener reads URLs from urls.txt and waits for the web pages to fully load before capturing screenshots.

```sh
✗ screener -i urls.txt --wait-page
[screener] (INFO) Screenshot http://example.com saved to ./screenshots
[screener] (INFO) Screenshot https://example.com saved to ./screenshots
[screener] (INFO) Screenshot http://hackerone.com saved to ./screenshots
[screener] (INFO) Screenshot https://hackerone.com saved to ./screenshots
[screener] (INFO) Screenshot http://google.com saved to ./screenshots 
[screener] (INFO) Screenshot http://bugcrowd.com saved to ./screenshots
[screener] (INFO) Screenshot https://bugcrowd.com saved to ./screenshots
[screener] (INFO) Screenshot https://google.com saved to ./screenshots
[screener] (INFO) Screenshot https://facebook.com saved to ./screenshots 
[screener] (INFO) Screenshot http://facebook.com saved to ./screenshots
[screener] (INFO) Screenshot http://yahoo.com saved to ./screenshots   
[screener] (INFO) Screenshot https://yahoo.com saved to ./screenshots
[screener] (INFO) Screenshot http://github.com saved to ./screenshots
[screener] (INFO) Screenshot https://github.com saved to ./screenshots
```

## Piping URLs from SDIN
Pipe URLs from a file to screener using standard input (STDIN).

```sh
✗ cat urls.txt | screener                        
[screener] (INFO) Screenshot http://example.com saved to ./screenshots
[screener] (INFO) Screenshot https://example.com saved to ./screenshots
[screener] (INFO) Screenshot http://hackerone.com saved to ./screenshots
[screener] (INFO) Screenshot https://hackerone.com saved to ./screenshots
[screener] (INFO) Screenshot http://google.com saved to ./screenshots 
[screener] (INFO) Screenshot http://bugcrowd.com saved to ./screenshots
[screener] (INFO) Screenshot https://bugcrowd.com saved to ./screenshots
[screener] (INFO) Screenshot https://google.com saved to ./screenshots
[screener] (INFO) Screenshot https://facebook.com saved to ./screenshots 
[screener] (INFO) Screenshot http://facebook.com saved to ./screenshots
[screener] (INFO) Screenshot http://yahoo.com saved to ./screenshots   
[screener] (INFO) Screenshot https://yahoo.com saved to ./screenshots
[screener] (INFO) Screenshot http://github.com saved to ./screenshots
[screener] (INFO) Screenshot https://github.com saved to ./screenshots
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
		Timeout:                 15,
		SaveScreenshots:         true,
		WaitForPageBody:         false,
		FollowRedirects:         true,
		DisableHTTP2:            true,
		IgnoreCertificateErrors: true,
		Verbose:                 false,
		Silence:                 true,
	}

	// Create a screener runner with options
	runner := screener.NewRunnerWithOptions(options)

	// Create a channel to receive results
	results := make(chan screener.Result)

	// Start capturing URLs using multiple goroutines
	go runner.MultipleStream(results, urls...)

	// Process the results as they come in
	for result := range results {
		fmt.Println(result.URL, result.Error, len(result.Image))
	}
}

```

For more, see [examples](https://github.com/root4loot/screener/tree/master/examples)

## License

See [LICENSE](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)