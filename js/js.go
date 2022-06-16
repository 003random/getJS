package js

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/003random/getJS/logger"
	"github.com/PuerkitoBio/goquery"
)

func NewSourcer(logger *logger.Logger, method string, headers []string, insecure bool, timeout int) *Sourcer {
	hdrs := http.Header{}
	for _, d := range headers {
		values := strings.Split(d, ":")
		if len(values) == 2 {
			hdrs.Add(values[0], values[1])
		}
	}

	tr := &http.Transport{
		ResponseHeaderTimeout: time.Duration(time.Duration(timeout) * time.Second),
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
	}

	client := &http.Client{
		Timeout:   time.Duration(time.Duration(timeout) * time.Second),
		Transport: tr,
	}

	return &Sourcer{
		logger:   logger,
		client:   client,
		method:   method,
		headers:  hdrs,
		insecure: insecure,
		timeout:  timeout,
	}
}

type Sourcer struct {
	logger   *logger.Logger
	client   *http.Client
	method   string
	headers  http.Header
	insecure bool
	timeout  int
}

func (s *Sourcer) GetScriptSrc(ur string) ([]string, error) {
	// Request the HTML page.
	req, err := http.NewRequest(s.method, ur, nil)
	if err != nil {
		return []string{}, err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		s.logger.Error(fmt.Sprintf("%s returned %d instead of %d", ur, res.StatusCode, http.StatusOK), nil)
		return nil, nil
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var sources []string

	// Find the script tags, and get the src
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		dsrc, _ := s.Attr("data-src")
		if src != "" {
			sources = append(sources, src)
		}
		if dsrc != "" {
			sources = append(sources, dsrc)
		}
	})

	return sources, nil
}

func (s *Sourcer) Client() *http.Client {
	return s.client
}
