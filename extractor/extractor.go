package extractor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

// ExtractionPoints defines the default HTML tags and their attributes from which JavaScript sources are extracted.
var ExtractionPoints = map[string][]string{
	"script": {"src", "data-src"},
}

// FetchResponse fetches the HTTP response for the given URL.
func FetchResponse(u string, method string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header = headers

	return http.DefaultClient.Do(req)
}

// ExtractSources extracts all JavaScript sources found in the provided HTTP response reader.
// The optional extractionPoints can be used to overwrite the default extraction points map
// with a set of HTML tag names, together with a list of what attributes to extract from.
func ExtractSources(input io.Reader, extractionPoints ...map[string][]string) (<-chan url.URL, error) {
	doc, err := goquery.NewDocumentFromReader(input)
	if err != nil {
		return nil, err
	}

	var (
		urls   = make(chan url.URL)
		points = ExtractionPoints
	)

	if len(extractionPoints) > 0 {
		points = extractionPoints[0]
	}

	go func() {
		defer close(urls)
		for tag, attributes := range points {
			doc.Find(tag).Each(func(i int, s *goquery.Selection) {
				for _, a := range attributes {
					if value, exists := s.Attr(a); exists {
						u, err := url.Parse(value)
						if err != nil {
							log.Println(fmt.Errorf("invalid attribute value %s cannot be parsed to a URL: %w", value, err))
							continue
						}

						urls <- *u
					}
				}
			})
		}
	}()

	return urls, nil
}

// Filter applies options to filter URLs from the input channel.
func Filter(input <-chan url.URL, options ...func([]url.URL) []url.URL) (<-chan url.URL, error) {
	output := make(chan url.URL)
	go func() {
		defer close(output)
		var urls []url.URL
		for u := range input {
			urls = append(urls, u)
		}

		for _, option := range options {
			urls = option(urls)
		}

		for _, u := range urls {
			output <- u
		}
	}()
	return output, nil
}

// WithComplete is an option to complete relative URLs.
func WithComplete(base *url.URL) func([]url.URL) []url.URL {
	return func(urls []url.URL) []url.URL {
		var result []url.URL
		for _, u := range urls {
			result = append(result, complete(u, base))
		}
		return result
	}
}

// WithResolve is an option to filter URLs that resolve successfully.
func WithResolve() func([]url.URL) []url.URL {
	return func(urls []url.URL) []url.URL {
		var result []url.URL
		for _, u := range urls {
			if resolve(u) {
				result = append(result, u)
			}
		}
		return result
	}
}

// complete completes relative URLs by adding the base URL.
func complete(source url.URL, base *url.URL) url.URL {
	if source.IsAbs() {
		return source
	}
	return *base.ResolveReference(&source)
}

// resolve checks if the provided URL resolves successfully.
func resolve(source url.URL) bool {
	resp, err := http.Get(source.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
	return err == nil && (resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices)
}
