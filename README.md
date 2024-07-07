<h2 align="center">JavaScript Extraction CLI & Package</h2>
<p align="center">
  <a href="https://pkg.go.dev/github.com/003random/getJS">
    <img src="https://pkg.go.dev/badge/github.com/003random/getJS">
  </a>
  <a href="https://github.com/003random/getJS/releases">
    <img src="https://img.shields.io/github/release/003random/getJS.svg">
  </a>
</p>


[getJS](https://github.com/003random/getJS) is a versatile tool designed to extract JavaScript sources from web pages. It offers both a command-line interface (CLI) for straightforward URL processing and a package interface for more customized integrations.

## Table of Contents

- [Installation](#installation)
- [CLI Usage](#cli-usage)
  - [Options](#options)
  - [Examples](#examples)
- [Package Usage](#package-usage)
  - [Importing the Extractor](#importing-the-extractor)
  - [Example](#example)
- [Version Information](#version-information)
- [Contributing](#contributing)
- [License](#license)

## Installation

To install `getJS`, use the following command:

`go get github.com/003random/getJS`

## CLI Usage

### Options

`getJS` provides several command-line options to customize its behavior:

- `-url string`: The URL from which JavaScript sources should be extracted.
- `-input string`: Optional URLs input files. Each URL should be on a new line in plain text format. Can be used multiple times.
- `-output string`: Optional output file where results are written to. Can be used multiple times.
- `-complete`: Complete/Autofill relative URLs by adding the current origin.
- `-resolve`: Resolve the JavaScript files. Can only be used in combination with `--complete`.
- `-threads int`: The number of processing threads to spawn (default: 2).
- `-verbose`: Print verbose runtime information and errors.
- `-method string`: The request method used to fetch remote contents (default: "GET").
- `-header string`: Optional request headers to add to the requests. Can be used multiple times.
- `-timeout duration`: The request timeout while fetching remote contents (default: 5s).

### Examples

#### Extracting JavaScript from a Single URL

`getJS -url https://destroy.ai`

or 

`curl https://destroy.ai | getJS`

#### Using Custom Request Options

`getJS -url "http://example.com" -header "User-Agent: foo bar" -method POST --timeout=15s`

#### Processing Multiple URLs from a File

`getJS -input foo.txt -input bar.txt`

#### Saving Results to an Output File

`getJS -url "http://example.com" -output results.txt`

## Package Usage

### Importing the Extractor

To use `getJS` as a package, you need to import the `extractor` package and utilize its functions directly.

### Example

```Go
package main

import (
    "fmt"
    "log"
    "net/http"
    "net/url"

    "github.com/003random/getJS/extractor"
)

func main() {
    baseURL, err := url.Parse("https://google.com")
    if (err != nil) {
        log.Fatalf("Error parsing base URL: %v", err)
    }

    resp, err := extractor.FetchResponse(baseURL.String(), "GET", http.Header{})
    if (err != nil) {
        log.Fatalf("Error fetching response: %v", err)
    }
    defer resp.Body.Close()

    // Custom extraction points (optional).
    extractionPoints := map[string][]string{
        "script": {"src", "data-src"},
        "a": {"href"},
    }

    sources, err := extractor.ExtractSources(resp.Body, extractionPoints)
    if (err != nil) {
        log.Fatalf("Error extracting sources: %v", err)
    }

    // Filtering and extending extracted sources.
    filtered, err := extractor.Filter(sources, extractor.WithComplete(baseURL), extractor.WithResolve())
    if (err != nil) {
        log.Fatalf("Error filtering sources: %v", err)
    }

    for source := range filtered {
        fmt.Println(source.String())
    }
}
```

## Version Information

This is the v2 version of `getJS`. The original version can be found under the tag [v1](https://github.com/003random/getJS/tree/v1).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any bugs, feature requests, or improvements.

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/003random/getJS/blob/main/LICENSE) file for details.
