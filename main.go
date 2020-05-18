package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/logrusorgru/aurora"
)

type logger interface {
	Log(msg string)
	Error(msg string, err error)
}

type silent struct{}

func (s silent) Log(msg string) {
}

func (s silent) Error(msg string, err error) {
}

type verbose struct {
}

func (v verbose) Log(msg string) {
	fmt.Println(au.Cyan(msg))
}

func Log(l logger, msg string) {
	l.Log(msg)
}

func (v verbose) Error(msg string, err error) {
	fmt.Fprintln(os.Stderr, au.Red(msg))
	if err != nil {
		fmt.Fprintln(os.Stderr, au.Red("[!] Error: "), au.Red(err))
	}
}

func Error(l logger, msg string, err error) {
	l.Error(msg, err)
}

var resolveArg = flag.Bool("resolve", false, "Output only existing files")
var completeArg = flag.Bool("complete", false, "Complete the url. e.g. append the domain to the path")
var saveArg = flag.Bool("save", false, "Save scripts to files")
var output logger
var au aurora.Aurora

func main() {
	urlArg := flag.String("url", "", "The url to get the javascript sources from")
	outputFileArg := flag.String("output", "", "Output file to save the results to")
	inputFileArg := flag.String("input", "", "Input file with urls")
	verboseArg := flag.Bool("verbose", false, "Display info of what is going on")
	noColorsArg := flag.Bool("nocolors", false, "Enable or disable colors")
	flag.Parse()

	au = aurora.NewAurora(!*noColorsArg)

	var urls []string

	output = silent{}

	if *verboseArg {
		output = verbose{}
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		output.Error("[!] Couldnt read Stdin", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			output.Error("[!] Couldnt read Stdin", err)
		}
		if len(urls) > 0 {
			output.Log("[+] Received urls from Stdin")
		}
	}

	if *inputFileArg != "" {
		lines, err := readLines(*inputFileArg)
		if err != nil {
			output.Error("[!] Couldn't read from input file", err)
		}
		output.Log("[+] Set url file to " + *inputFileArg)
		urls = append(urls, lines...)
	}

	if *urlArg != "" {
		output.Log("[+] Set url to " + *urlArg)
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

	allSources := processURLs(urls)

	// Save to file or print
	if *outputFileArg != "" {
		output.Log("[+] Saving output to " + *outputFileArg)
		err := saveToFile(allSources, *outputFileArg)
		if err != nil {
			output.Error("[!] Couldn't save to output file "+*outputFileArg, err)
		}
	} else {
		for _, i := range allSources {
			fmt.Println(i)
		}
	}
}

func processURL(url string, result chan []string, wg *sync.WaitGroup) {
	defer wg.Done()

	output.Log("[+] Getting sources from " + url)
	sources, err := getScriptSrc(url)
	if err != nil {
		output.Error("[!] Couldn't get sources from "+url, err)
	}

	var completedSuccessfully bool
	if *completeArg || *saveArg {
		completedSuccessfully, sources = complete(sources, url)
	}

	if (*resolveArg && *completeArg) || *saveArg {
		if completedSuccessfully {
			output.Log("[+] Resolving files")
			sources = resolveUrls(sources)
		} else {
			output.Error("[!] Couldn't resolve URLs", nil)
		}
	} else if *resolveArg {
		output.Error("[!] Resolve can only be used in combination with -complete", nil)
	}

	result <- sources
}

func processURLs(urls []string) []string {
	var allSources []string
	sources := make(chan []string)

	done := make(chan int)
	go func() {
		for source := range sources {
			allSources = append(allSources, source...)
		}
		done <- 1
	}()

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go processURL(url, sources, &wg)
	}

	wg.Wait()
	close(sources)
	<-done
	return allSources
}

func complete(sources []string, url string) (bool, []string) {
	output.Log("[+] Completing URLs")
	sources, err := completeUrls(sources, url)
	if err != nil {
		output.Error("[!] Couldn't complete URLs", err)
		return false, sources
	}

	return true, sources
}

// ToDO: Use channel instead of slide, and use io.Writer instead of file path
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

func getScriptSrc(url string) ([]string, error) {
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s returned an %d instead of an 200 OK", url, res.StatusCode)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var sources []string
	attributes := []string{"src", "data-src"}

	// Find the script tags, and get the src
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		for _, attr := range attributes {
			value, _ := s.Attr(attr)
			if value != "" {
				sources = append(sources, value)
			}
		}
	})

	return sources, nil
}

// ToDo: Use io.Writer instead of a file path
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func saveJS(link string, body io.ReadCloser) {
	u, err := url.Parse(link)
	if err != nil {
		output.Error("[!] Couldn't parse URL", err)
		return
	}

	elements := strings.Split(u.Path, "/")
	filename := elements[len(elements)-1]

	out, err := os.Create(filename)
	if err != nil {
		output.Error("[!] Couldn't create the file", err)
		return
	}

	_, err = io.Copy(out, body)
	if err != nil {
		output.Error("[!] Problem saving to file", err)
	}

	out.Close()
}

func resolveUrls(s []string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		resp, err := http.Get(s[i])
		if err != nil {
			output.Error("[!] Couldn't resolve URL", err)
			s = append(s[:i], s[i+1:]...)
		}
		if resp.StatusCode != 200 && resp.StatusCode != 304 {
			s = append(s[:i], s[i+1:]...)
		}

		if resp != nil {
			saveJS(s[i], resp.Body)
			resp.Body.Close()
		}
	}
	return s
}

func completeUrls(s []string, mainUrl string) ([]string, error) {
	u, err := url.Parse(mainUrl)
	if err != nil {
		return s, err
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
