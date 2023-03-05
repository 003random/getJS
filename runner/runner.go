package runner

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var (
	ExtractionPoints = map[string][]string{
		"script": {"src", "data-src"},
	}
)

func New(options *Options) *runner {
	http.DefaultClient.Transport = &http.Transport{
		TLSHandshakeTimeout: options.Request.Timeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: options.Request.InsecureSkipVerify,
		},
	}
	http.DefaultClient.Timeout = options.Request.Timeout

	return &runner{
		Options: *options,
		Results: make(chan url.URL),
	}
}

func (r *runner) Run() error {
	if !r.Options.Verbose {
		log.SetOutput(io.Discard)
	}

	go func() {
		for _, input := range r.Options.Inputs {
			switch input.Type {
			case InputURL:
				r.ProcessURLs(input.Data)
			case InputResponse:
				r.ProcessResponse(input.Data)
			}

			if input, ok := input.Data.(io.Closer); ok {
				input.Close()
			}
		}

		close(r.Results)
	}()

	r.listen()

	return nil
}

func (r *runner) listen() {
	for s := range r.Results {
		for _, output := range r.Options.Outputs {
			_, err := output.Write([]byte(fmt.Sprintf("%s\n", s.String())))
			if err != nil {
				log.Println(fmt.Errorf("[error] writing result %s to output: %v", s.String(), err))
			}
		}
	}

	for _, output := range r.Options.Outputs {
		if o, ok := output.(io.Closer); ok {
			o.Close()
		}
	}
}

func (r *runner) ProcessURLs(data io.Reader) {
	next := Read(data)

	wg := sync.WaitGroup{}

	throttle := make(chan struct{}, r.Options.Threads)
	for i := 0; i < r.Options.Threads; i++ {
		throttle <- struct{}{}
	}

	for {
		u, err := next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Println(fmt.Errorf("[error] parsing url %s: %v", u.String(), err))
			continue
		}

		wg.Add(1)
		go func() {
			defer func() {
				throttle <- struct{}{}
				wg.Done()
			}()

			req, err := http.NewRequest(r.Options.Request.Method, u.String(), nil)
			if err != nil {
				log.Println(fmt.Errorf("[error] creating request for url %s: %v", u.String(), err))
				return
			}

			req.Header = r.Options.Request.Headers

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println(fmt.Errorf("[error] requesting url %s: %v", u.String(), err))
				return
			}
			defer resp.Body.Close()

			urls, err := Sources(resp.Body)
			if err != nil {
				log.Println(fmt.Errorf("[error] extracting sources for url %s: %v", u.String(), err))
				return
			}

			for source := range urls {
				if r.Options.Complete {
					source = Complete(source, u)
				}

				if r.Options.Resolve && r.Options.Complete {
					if !Resolves(source) {
						log.Printf("[error] source %s did not resolve\n", source.String())
						continue
					}
				}

				r.Results <- source
			}
		}()

		<-throttle
	}

	wg.Wait()
}

func (r *runner) ProcessResponse(data io.Reader) {
	urls, err := Sources(data)
	if err != nil {
		log.Println(fmt.Errorf("[error] extracting sources from response file: %v", err))
	}

	for sources := range urls {
		r.Results <- sources
	}
}

// Sources ...
func Sources(input io.Reader) (<-chan url.URL, error) {
	doc, err := goquery.NewDocumentFromReader(input)
	if err != nil {
		return nil, err
	}

	urls := make(chan url.URL)

	go func() {
		for tag, attributes := range ExtractionPoints {
			doc.Find(tag).Each(func(i int, s *goquery.Selection) {
				for _, a := range attributes {
					if value, exists := s.Attr(a); exists {
						u, err := url.Parse(value)
						if err != nil {
							log.Println(fmt.Errorf("invalid attribute value %s can not be parsed to an url: %v", value, err))
						}

						urls <- *u
					}
				}
			})
		}

		close(urls)
	}()

	return urls, nil
}

// Complete ...
func Complete(source url.URL, base *url.URL) url.URL {
	if source.IsAbs() {
		return source
	}

	return *base.ResolveReference(&source)
}

// Resolve ...
func Resolves(source url.URL) bool {
	resp, err := http.Get(source.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)

	// A source is valid if there is no error reading the response body, and
	// the status code is within the 200 range.
	return err == nil && (resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices)
}

// Read is a wrapper around the bufio.Scanner Text() method.
// Upon reading from the input, the line is automatically parsed to a *url.URL.
// An io.EOF error is returned when there are no more lines.
func Read(input io.Reader) func() (*url.URL, error) {
	scanner := bufio.NewScanner(input)
	return func() (*url.URL, error) {
		if !scanner.Scan() {
			return nil, io.EOF
		}

		return url.Parse(scanner.Text())
	}
}
