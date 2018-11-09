package main

import (
        "bufio"
        "flag"
        "fmt"
        "log"
        "net/http"
        urlpkg "net/url"
        "os"
        "strings"

        "github.com/PuerkitoBio/goquery"
)

func main() {
        url := flag.String("url", "", "The url to get the javascript sources from")
        outputFile := flag.String("output", "", "Output file to save the results to")
        inputFile := flag.String("input", "", "Input file with urls")
        resolve := flag.Bool("resolve", false, "Output only existing files")
        complete := flag.Bool("complete", false, "Complete the url. e.g. append the domain to the path")
        plain := flag.Bool("plain", false, "Output only the results")
        silent := flag.Bool("silent", false, "Dont output anything")
        flag.Parse()

        var urls []string
        var allSources []string

        stat, _ := os.Stdin.Stat()
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

        if *url != "" {
                if !*plain && !*silent {
                        fmt.Println("[+] Set url to", *url)
                }
                urls = append(urls, *url)
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
                sources := getScriptSrc(e)

                if *complete {
                        sources = completeUrls(sources, e)
                }

                if *resolve {
                        if *complete {
                                if !*plain && !*silent {
                                        fmt.Println("[+] Resolving files")
                                }
                                sources = resolveUrls(sources)
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

func getScriptSrc(url string) []string {
        // Request the HTML page.
        res, err := http.Get(url)
        if err != nil {
                log.Fatal(err)
        }
        defer res.Body.Close()
        if res.StatusCode != 200 {
                log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
        }

        // Load the HTML document
        doc, err := goquery.NewDocumentFromReader(res.Body)
        if err != nil {
                log.Fatal(err)
        }

        var sources []string

        // Find the script tags, and get the src
        doc.Find("script").Each(func(i int, s *goquery.Selection) {
                src, _ := s.Attr("src")
                if src != "" {
                        sources = append(sources, src)
                }
        })

        return sources
}

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

func resolveUrls(s []string) []string {
        for i := len(s) - 1; i >= 0; i-- {
                resp, err := http.Get(s[i])
                if err != nil {
                        log.Fatal(err)
                }
                if resp.StatusCode != 200 && resp.StatusCode != 304 {
                        s = append(s[:i], s[i+1:]...)
                }
        }
        return s
}

func completeUrls(s []string, url string) []string {
        u, err := urlpkg.Parse(url)
        if err != nil {
                log.Fatal(err)
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
        return s
}
