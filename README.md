<p align="center">
<img src="./assets/logo.png" alt="screener logo" width="300"/>
</p>

screener is a command-line interface (CLI) and Golang library for capturing screenshots of web pages. Built on top of [Rod](https://github.com/go-rod/rod).

## Features
- **Stream URLs**: Input URLs via standard input (STDIN) for real-time processing.
- **Max Page Load Wait**: Define a maximum wait time for web page loading before capturing screenshots.
- **Redirect Handling**: Customize redirect behavior to follow or ignore URL changes.
- **Unique Screenshots**: Save only unique screenshots to avoid duplicates.
- **Concurrency**: Support for concurrent requests for faster processing.
- **Certificate Error Handling**: Option to ignore SSL certificate errors for testing environments.
- **HTTP/2 Control**: Disable HTTP/2 for compatibility with various server configurations.
- **Custom User-Agent**: Set a custom user-agent for requests to simulate different browsers or devices.
- **URL in Image**: Choose to include the URL directly in the captured image for context and reference.

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
   -t,   --target                  single target
   -l,   --list                    input file containing list of targets (one per line)

CONFIGURATIONS:
   -c,  --concurrency              number of concurrent requests                  (Default: 10)
   -t,  --timeout                  timeout for screenshot capture                 (Default: 15 seconds)
   -ua,  --user-agent              set user agent                                 (Default: Chrome Headless)
   -su,  --save-unique             save unique screenshots only                   (Default: false)
   -dh,  --disable-http2           disable HTTP2                                  (Default: true)
   -fr,  --follow-redirects        follow redirects                               (Default: true)
   -cw,  --capture-width           screenshot pixel width                         (Default: 1366)
   -ch,  --capture-height          screenshot pixel height                        (Default: 768)
   -cf,  --capture-full            capture full page                              (Default: 768)
   -wp,  --wait-page               wait for page to fully load before capturing   (Default: true)
   -wt,  --wait-time               wait time before capturing (seconds)           (Default: 30)
   -dc,  --delay-between-capture   delay between capture (seconds)                (Default: 0)
   -ice, --ignore-cert-err         ignore certificate errors                      (Default: true)
   -isc, --ignore-status-codes     ignore HTTP status codes  (comma separated)    (Default: [])
   -s,   --silence                 silence output                                 (Default: false)

OUTPUT:
   -o,   --outfolder               save images to given folder                    (Default: ./screenshots)
   -wu,  --without-url             without URL in image                           (Default: false)
   -s,   --silence                 silence output
   -v,   --verbose                 verbose output
         --version                 display version
```

## Example

### Screenshotting a Single Target
Capture a screenshot from a single URL. If the URL scheme (http/https) is not specified, then it will handle both:

```sh
✗ screener -t "example.com"
# Captures both http://example.com/ and https://example.com/
[screener] (RES) Screenshot http://example.com/ saved to  ./screenshots                         
[screener] (RES) Screenshot https://example.com/ saved to ./screenshots

✗ screener -t "google.com"
# Captures https://www.google.com only due to redirect
[screener] (RES) Screenshot https://www.google.com saved to ./screenshots 
```

### Screenshotting URLs from a File
Capture screenshots from multiple URLs listed in a file but wait for pages to load first and only save unique images.

```sh
✗ screener -l urls.txt --save-unique 
[screener] (RES) Screenshot http://example.com/ saved to ./screenshots                         
[screener] (RES) Screenshot https://example.com/ saved to ./screenshots                         
[screener] (RES) Screenshot https://github.com/ saved to ./screenshots                         
[screener] (RES) Screenshot https://consent.yahoo.com saved to ./screenshots                   
[screener] (RES) Screenshot https://www.google.com saved to ./screenshots                      
[screener] (RES) Screenshot https://www.facebook.com saved to ./screenshots                    
[screener] (RES) Screenshot https://www.hackerone.com saved to ./screenshots                   
[screener] (RES) Screenshot https://www.bugcrowd.com saved to ./screenshots 
```

## Piping URLs from SDIN
Stream URLs to Screener, capturing screenshots as they are received:

```sh
✗ cat urls.txt | screener                        
[screener] (RES) Screenshot http://example.com/ saved to ./screenshots
[screener] (RES) Screenshot https://example.com/ saved to ./screenshots
[screener] (RES) Screenshot https://www.hackerone.com saved to ./screenshots
[screener] (RES) Screenshot https://www.bugcrowd.com saved to ./screenshots
[screener] (RES) Screenshot https://www.google.com saved to ./screenshots
[screener] (RES) Screenshot https://www.facebook.com saved to ./screenshots
[screener] (RES) Screenshot https://consent.yahoo.com saved to ./screenshots
[screener] (RES) Screenshot https://github.com/ saved to ./screenshots
```

## Example Screenshot

<p align="center">
<img src="./assets/https_example.com.png" alt="screenshot example"/>
</p>

**Note:** You can remove the URL from the image by using the `-wu` or `--without-url` flag when running the tool.


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
	// Create runner with default options
	runner := screener.NewRunner()
	runner.Options.SaveScreenshots = true

	// Capture a single URL
	result := runner.Run("https://example.com", "https://hackerone.com")

	// Process the result
	for _, result := range result {
		fmt.Println(result.TargetURL, result.LandingURL, result.Error, len(result.Image))
	}
}

```

For more, see [examples](https://github.com/root4loot/screener/tree/master/examples)

## License

See [LICENSE](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)