<p align="center">
<img src="./assets/logo.png" alt="screener logo" width="300"/>
</p>

screener is a command-line interface (CLI) and Golang library for capturing screenshots of web pages. Built on top of [Rod](https://github.com/go-rod/rod).

## Features

- **Stream URLs**: Input URLs via standard input (STDIN) for real-time processing.
- **Fixed Page Load Wait**: Define a maximum wait time for web page loading before capturing screenshots.
- **Redirect Handling**: Customize redirect behavior to follow or ignore URL changes.
- **Unique screenshots**: Option to avoid saving duplicate screenshots, useful for large-scale scanning.
- **Concurrency**: Support for concurrent requests for faster processing.
- **Certificate Error Handling**: Option to ignore SSL certificate errors for testing environments.
- **HTTP/2 Control**: Disable HTTP/2 for compatibility with various server configurations.
- **Custom User-Agent**: Set a custom user-agent for requests to simulate different browsers or devices.
- **Imprint URL in Image**: Choose to include the URL directly in the captured image for context and reference.

## Installation

### Go

```
go install github.com/root4loot/screener/cmd/screener@latest
```

### Docker

```
git clone https://github.com/root4loot/screener.git && cd screener
docker build -t screener .
docker run -it -v "$(pwd)/screenshots:/app/screenshots" screener -t example.com
```

## Usage

```
Usage: screener [options] (-t <target> | -i <targets.txt>)

INPUT:
   -t,  --target                   single target
   -l,  --list                     input file containing list of targets (one per line)

CONFIGURATIONS:
   -c,   --concurrency             number of concurrent requests                 (Default: 10)
   -to,   --timeout                timeout for screenshot capture                (Default: 15 seconds)
   -ua,  --user-agent              set user agent                                (Default: Chrome Headless)
   -su,  --save-unique             save unique screenshots only                  (Default: false)
   -dh,  --disable-http2           disable HTTP2                                 (Default: true)
   -fr,  --follow-redirects        follow redirects                              (Default: true)
   -cw,  --capture-width           screenshot pixel width                        (Default: 1366)
   -ch,  --capture-height          screenshot pixel height                       (Default: 768)
   -cf,  --capture-full            capture full page                             (Default: 768)
   -fw,  --fixed-wait              fixed wait time before capturing (seconds)    (Default: 2)
   -dc,  --delay-between-capture   delay between capture (seconds)               (Default: 0)
   -ice, --ignore-cert-err         ignore certificate errors                     (Default: true)
   -isc, --ignore-status-codes     ignore HTTP status codes  (comma separated)   (Default: [])
   -s,   --silence                 silence output                                (Default: false)

OUTPUT:
   -o,   --outfolder               save images to given folder                   (Default: ./screenshots)
   -wu,  --without-url             without URL in image                          (Default: false)
   -s,   --silence                 silence output
   -v,   --verbose                 verbose output
         --version                 display version
```

## Example

### Screenshot Single Target

Capture a single target. If the scheme (http/https) is not specified, then it will default to https and fallback to http if the former fails.

```sh
screener -t "example.com"
[screener] (INF) Preparing screenshot: https://example.com
[screener] (RES) Successful screenshot: https://example.com/
```

### Screenshot Multiple Targets

Capture multiple targets.

```sh
$ cat targets.txt
google.com
bugcrowd.com
hackerone.com/sitemap.xml
http://example.com
https://scanme.sh
```

Note that targets can be IP, domain, or full URL.

```sh
$ screener -l targets.txt
[screener] (INF) Preparing screenshot: https://scanme.sh/
[screener] (INF) Preparing screenshot: https://google.com
[screener] (INF) Preparing screenshot: http://example.com/
[screener] (INF) Preparing screenshot: https://bugcrowd.com
[screener] (INF) Preparing screenshot: https://hackerone.com/sitemap.xml
[screener] (INF) Preparing screenshot: http://172.64.151.42
[screener] (RES) Successful screenshot: http://example.com/
[screener] (RES) Successful screenshot: https://hackerone.com/sitemap.xml
[screener] (RES) Successful screenshot: http://172.64.151.42/
[screener] (RES) Successful screenshot: https://scanme.sh/
[screener] (RES) Successful screenshot: https://www.google.com/
[screener] (RES) Successful screenshot: https://www.bugcrowd.com
```

You may also "stream" targets to screener, capturing screenshots as they are received:

```sh
✗ cat targets.txt | screener
[screener] (INF) Preparing screenshot: https://google.com
[screener] (RES) Successful screenshot: https://www.google.com/
[screener] (INF) Preparing screenshot: https://bugcrowd.com
[screener] (RES) Successful screenshot: https://www.bugcrowd.com/
[screener] (INF) Preparing screenshot: http://172.64.151.42
[screener] (RES) Successful screenshot: http://172.64.151.42/
[screener] (INF) Preparing screenshot: https://hackerone.com/sitemap.xml
[screener] (RES) Successful screenshot: https://hackerone.com/sitemap.xml
[screener] (INF) Preparing screenshot: http://example.com/
[screener] (RES) Successful screenshot: http://example.com/
[screener] (INF) Preparing screenshot: https://scanme.sh/
[screener] (RES) Successful screenshot: https://scanme.sh/
```

## Example Screenshot

<p align="center">
<img src="./assets/https_example.com.png" alt="screenshot example"/>
</p>

**Note:** You can remove the URL from the image by using the `-wu` or `--without-url` flag when running the tool.

## Tips

- Use `-wu` or `--without-url` flag to remove the URL from the image.
- Use `-su` or `--save-unique` flag to save only unique screenshots

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
