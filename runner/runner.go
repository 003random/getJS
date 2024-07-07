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

	"github.com/003random/getJS/extractor"
)

// ExtractionPoints defines the default HTML tags and their attributes from which JavaScript sources are extracted.
var ExtractionPoints = map[string][]string{
	"script": {"src", "data-src"},
}

// New creates a new runner with the provided options.
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

// Run starts processing the inputs and extracts JavaScript sources into the runner's Results channel.
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

// ProcessURLs will fetch the HTTP response for all URLs in the provided reader
// and stream the extracted sources to the runner's Results channel.
func (r *runner) ProcessURLs(data io.Reader) {
	var (
		next = Read(data)
		wg   = sync.WaitGroup{}

		throttle = make(chan struct{}, r.Options.Threads)
	)

	for i := 0; i < r.Options.Threads; i++ {
		throttle <- struct{}{}
	}

	for {
		u, err := next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Println(fmt.Errorf("[error] parsing url %v: %w", u, err))
			continue
		}

		wg.Add(1)
		go func(u *url.URL) {
			defer func() {
				throttle <- struct{}{}
				wg.Done()
			}()

			resp, err := extractor.FetchResponse(u.String(), r.Options.Request.Method, r.Options.Request.Headers)
			if err != nil {
				log.Println(fmt.Errorf("[error] fetching response for url %s: %w", u.String(), err))
				return
			}
			defer resp.Body.Close()

			sources, err := extractor.ExtractSources(resp.Body)
			if err != nil {
				log.Println(fmt.Errorf("[error] extracting sources from response for url %s: %w", u.String(), err))
				return
			}

			filtered, err := extractor.Filter(sources, r.filters(u)...)
			if err != nil {
				log.Println(fmt.Errorf("[error] filtering sources for url %s: %w", u.String(), err))
				return
			}

			for source := range filtered {
				r.Results <- source
			}
		}(u)

		<-throttle
	}

	wg.Wait()
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

func (r *runner) ProcessResponse(data io.Reader) {
	sources, err := extractor.ExtractSources(data)
	if err != nil {
		log.Println(fmt.Errorf("[error] extracting sources from response file: %w", err))
	}

	filtered, err := extractor.Filter(sources, r.filters(nil)...)
	if err != nil {
		log.Println(fmt.Errorf("[error] filtering sources from response file: %w", err))
		return
	}

	for source := range filtered {
		r.Results <- source
	}
}

func (r *runner) filters(base *url.URL) (options []func([]url.URL) []url.URL) {
	if r.Options.Complete && base != nil {
		options = append(options, extractor.WithComplete(base))
	}

	if r.Options.Resolve {
		options = append(options, extractor.WithResolve())
	}

	return
}
