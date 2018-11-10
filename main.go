package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	urlArg := flag.String("url", "", "The url to get the javascript sources from")
	outputFile := flag.String("output", "", "Output file to save the results to")
	inputFile := flag.String("input", "", "Input file with urls")
	resolve := flag.Bool("resolve", false, "Output only existing files")
	complete := flag.Bool("complete", false, "Complete the url. e.g. append the domain to the path")
	plain := flag.Bool("plain", false, "Output only the results")
	silent := flag.Bool("silent", false, "Dont output anything")
	flag.Parse()

	var urls []string
	var allSources []string

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
		if len(urls) > 0 {
			if !*plain && !*silent {
				fmt.Println("[+] Received urls from Stdin")
			}
		}
	}

	if *inputFile != "" {
		lines, err := readLines(*inputFile)
		if err != nil {
			log.Fatal(err)
		}
		if !*plain && !*silent {
			fmt.Println("[+] Set url file to", *inputFile)
		}
		urls = append(urls, lines...)
	}

	if *urlArg != "" {
		if !*plain && !*silent {
			fmt.Println("[+] Set url to", *urlArg)
		}
		urls = append(urls, *urlArg)
	}

	if len(urls) == 0 {
		if !*plain && !*silent {
			fmt.Println("[!] No urls supplied")
		}
		os.Exit(3)
	}

	if *resolve && !*complete {
		if !*plain && !*silent {
			fmt.Println("[!] Resolve can only be used in combination with -complete")
		}
		os.Exit(3)
	}

	for _, e := range urls {
		if !*plain && !*silent {
			fmt.Println("[+] Getting sources from", e)
		}
		sources, err := getScriptSrc(e)
		// ToDo: Just skip it. Dont panic. Trow a error in stderr
		if err != nil {
			log.Fatal(err)
		}

		if *complete {
			// ToDo: send copy of sources to completeUrls, and if there was an error. keep the old sources and display the error to stderr
			sources, err = completeUrls(sources, e)
			if err != nil {
				log.Fatal(err)
			}
		}

		if *resolve {
			if *complete {
				if !*plain && !*silent {
					fmt.Println("[+] Resolving files")
				}
				sources, err = resolveUrls(sources)
				if err != nil {
					// ToDo: send copy of sources to resolveUrls, and if there was an error. keep the old sources and display the error to stderr
					log.Fatal(err)
				}
			} else {
				if !*plain && !*silent {
					fmt.Println("[-] Resolve can only be used in combination with -complete")
				}
			}
		}

		if !*silent {
			for _, i := range sources {
				fmt.Println(i)
			}
		}

		if *outputFile != "" {
			allSources = append(allSources, sources...)
		}

	}

	// Save to file
	if *outputFile != "" {
		if !*plain && !*silent {
			fmt.Println("[+] Saving output to ", *outputFile)
		}
		if err := saveToFile(allSources, *outputFile); err != nil {
			log.Fatalf("saveToFile: %s", err)
		}
	}

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
		// ToDo: Change to no panic. only print warning in stderr
		fmt.Fprintln(os.Stderr, url, "didnt resolve/return a 200. StatusCode:", res.StatusCode)
		return nil, err
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
		if src != "" {
			sources = append(sources, src)
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

func resolveUrls(s []string) ([]string, error) {
	for i := len(s) - 1; i >= 0; i-- {
		resp, err := http.Get(s[i])
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 && resp.StatusCode != 304 {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s, nil
}

func completeUrls(s []string, mainUrl string) ([]string, error) {
	u, err := url.Parse(mainUrl)
	if err != nil {
		return nil, err
	}

	for i, _ := range s {
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
