package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/logrusorgru/aurora"
	flag "github.com/spf13/pflag"
)

const (
	LOG_SILENT = iota
	LOG_VERBOSE
)

type Logger struct {
	logLevel int
}

func (l *Logger) Log(msg string) {
	if l.logLevel == LOG_VERBOSE {
		fmt.Println(au.Cyan(msg))
	}
}

func (l *Logger) LogF(msg string, a ...interface{}) {
	l.Log(fmt.Sprintf(msg, a...))
}

func (l *Logger) Error(msg string, err error) {
	if l.logLevel == LOG_VERBOSE {
		fmt.Fprintln(os.Stderr, au.Red(msg))
		if err != nil {
			fmt.Fprintln(os.Stderr, au.Red("[!] Error: "), au.Red(err))
		}
	}
}

func (l *Logger) ErrorF(msg string, err error, a ...interface{}) {
	l.Error(fmt.Sprintf(msg, a...), err)
}

var output *Logger
var au aurora.Aurora

func main() {
	urlArg := flag.StringP("url", "u", "", "The url to get the javascript sources from")
	methodArg := flag.StringP("method", "X", "GET", "The request method. e.g. GET or POST")
	outputFileArg := flag.StringP("output", "o", "", "Output file to save the results to")
	inputFileArg := flag.String("input", "", "Input file with urls")
	resolveArg := flag.Bool("resolve", false, "Output only existing files")
	completeArg := flag.Bool("complete", false, "Complete the url. e.g. append the domain to the path")
	verboseArg := flag.BoolP("verbose", "v", false, "Display info of what is going on")
	noColorsArg := flag.BoolP("nocolors", "nc", false, "Enable or disable colors")
	HeaderArg := flag.StringArrayP("header", "H", nil, "Any HTTP headers(-H \"Authorization:Bearer token\")")
	insecureArg := flag.BoolP("insecure", "k", false, "Check the SSL security checks. Use when the certificate is expired or invalid")
	timeoutArg := flag.Int("timeout", 10, "Max timeout for the requests")
	flag.Parse()

	au = aurora.NewAurora(!*noColorsArg)

	var urls []string
	var allSources []string

	output = &Logger{logLevel: LOG_SILENT}
	if *verboseArg {
		output.logLevel = LOG_VERBOSE
	}

	if stat, err := os.Stdin.Stat(); err != nil {
		output.Error("[!] Couldnt read Stdin", err)
	} else if (stat.Mode() & os.ModeCharDevice) == 0 {
		if lines, err := readLines(os.Stdin); err != nil {
			output.Error("[!] Couldnt read Stdin", err)
		} else {
			urls = append(urls, lines...)
		}
		if len(urls) > 0 {
			output.Log("[+] Received urls from Stdin")
		}
	}

	if *inputFileArg != "" {
		f, err := os.Open(*inputFileArg)
		if err != nil {
			output.Error("[!] Couldn't open input file", err)
			return
		}
		defer f.Close()

		if lines, err := readLines(f); err != nil {
			output.Error("[!] Couldn't read from input file", err)
		} else {
			output.LogF("[+] Set url file to %s", *inputFileArg)
			urls = append(urls, lines...)
		}
	}

	if *urlArg != "" {
		output.LogF("[+] Set url to %s", *urlArg)
		urls = append(urls, *urlArg)
	}

	if len(urls) == 0 {
		output.Error("[!] No urls supplied", nil)
		os.Exit(3)
	}

	if *resolveArg && !*completeArg {
		output.Error("[!] Resolve can only be used in combination with -complete", nil)
		os.Exit(3)
	}

	for _, e := range urls {
		var completedSuccessfully = true
		output.Log("[+] Getting sources from " + e)
		sources, err := getScriptSrc(e, *methodArg, *HeaderArg, *insecureArg, *timeoutArg)
		if err != nil {
			output.ErrorF("[!] Couldn't get sources from %s", err, e)
			continue
		}

		if *completeArg {
			output.Log("[+] Completing URLs")
			if completedSources, err := completeUrls(sources, e); err != nil {
				output.Error("[!] Couldn't complete URLs", err)
				completedSuccessfully = false
			} else {
				sources = completedSources
			}
		}

		if *resolveArg && *completeArg {
			if completedSuccessfully {
				output.Log("[+] Resolving files")
				if resolvedSources, err := resolveUrls(sources); err != nil {
					output.Error("[!] Couldn't resolve URLs", err)
				} else {
					sources = resolvedSources
				}
			} else {
				output.Error("[!] Couldn't resolve URLs", nil)
			}
		}

		for _, i := range sources {
			fmt.Println(i)
		}

		if *outputFileArg != "" {
			allSources = append(allSources, sources...)
		}

	}

	// Save to file
	if *outputFileArg != "" {
		output.LogF("[+] Saving output to %s", *outputFileArg)
		if err := saveToFile(allSources, *outputFileArg); err != nil {
			output.ErrorF("[!] Couldn't save to output file %s", err, *outputFileArg)
		}
	}

}

func saveToFile(sources []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range sources {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func getScriptSrc(url string, method string, headers []string, insecure bool, timeout int) ([]string, error) {
	// Request the HTML page.
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return []string{}, err
	}

	for _, d := range headers {
		values := strings.Split(d, ":")
		if len(values) == 2 {
			output.LogF("[+] New Header: %s: %s", values[0], values[1])
			req.Header.Set(values[0], values[1])
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

	res, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		output.ErrorF("[!] %s returned an %d instead of %d", nil, url, res.StatusCode, http.StatusOK)
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

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func resolveUrls(s []string) ([]string, error) {
	var resolved []string
	for _, url := range s {
		if resp, err := http.Get(url); err != nil {
			return nil, err
		} else if resp.StatusCode == 200 || resp.StatusCode == 304 {
			resolved = append(resolved, url)
		}
	}
	return s, nil
}

func completeUrls(s []string, mainUrl string) ([]string, error) {
	u, err := url.Parse(mainUrl)
	if err != nil {
		return nil, err
	}

	for i := range s {
		if strings.HasPrefix(s[i], "//") {
			s[i] = u.Scheme + ":" + s[i]
		} else if strings.HasPrefix(s[i], "/") && string(s[i][1]) != "/" {
			s[i] = u.Scheme + "://" + u.Host + s[i]
		} else if !strings.HasPrefix(s[i], "http://") && !strings.HasPrefix(s[i], "https://") {
			s[i] = u.Scheme + "://" + u.Host + u.Path + "/" + s[i]
		}
	}
	return s, nil
}
