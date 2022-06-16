package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/003random/getJS/js"
	"github.com/003random/getJS/logger"
	flag "github.com/spf13/pflag"
)

func main() {
	urlArg := flag.String("url", "", "The url to get the javascript sources from")
	methodArg := flag.String("method", "GET", "The request method. e.g. GET or POST")
	outputFileArg := flag.String("output", "", "Output file to save the results to")
	inputFileArg := flag.String("input", "", "Input file with urls")
	resolveArg := flag.Bool("resolve", false, "Output only existing files")
	completeArg := flag.Bool("complete", false, "Complete the url. e.g. append the domain to the path")
	verboseArg := flag.Bool("verbose", false, "Display info of what is going on")
	noColorsArg := flag.Bool("nocolors", false, "Enable or disable colors")
	HeaderArg := flag.StringArrayP("header", "H", nil, "Any HTTP headers(-H \"Authorization:Bearer token\")")
	insecureArg := flag.Bool("insecure", false, "Check the SSL security checks. Use when the certificate is expired or invalid")
	timeoutArg := flag.Int("timeout", 10, "Max timeout for the requests")
	flag.Parse()

	logger := logger.NewLogger(*verboseArg, !*noColorsArg)

	var urls []string
	var allSources []string

	stat, err := os.Stdin.Stat()
	if err != nil {
		logger.Error("Couldnt read Stdin:", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			logger.Error("Couldnt read Stdin", err)
		}
		if len(urls) > 0 {
			logger.Log("Received urls from Stdin")
		}
	}

	if *inputFileArg != "" {
		f, err := os.Open(*inputFileArg)
		if err != nil {
			logger.Error("Couldn't open input file", err)
			return
		}
		defer f.Close()

		lines, err := readLines(f)
		if err != nil {
			logger.Error("Couldn't read from input file", err)
		}
		logger.Log("Set url file to " + *inputFileArg)
		urls = append(urls, lines...)
	}

	if *urlArg != "" {
		logger.Log(fmt.Sprintf("Set url to %s", *urlArg))
		urls = append(urls, *urlArg)
	}

	if len(urls) == 0 {
		logger.Error("No urls supplied", nil)
		os.Exit(3)
	}

	if *resolveArg && !*completeArg {
		logger.Error("Resolve can only be used in combination with -complete", nil)
		os.Exit(3)
	}

	sourcer := js.NewSourcer(logger, *methodArg, *HeaderArg, *insecureArg, *timeoutArg)

	for _, e := range urls {
		var sourcesBak []string
		var completedSuccessfully = true
		logger.Log("Getting sources from " + e)
		sources, err := sourcer.GetScriptSrc(e)
		if err != nil {
			logger.Error(fmt.Sprintf("Couldn't get sources from %s", e), err)
			continue
		}

		if *completeArg {
			logger.Log("Completing URLs")
			sourcesBak = sources
			sources, err = completeUrls(sources, e)
			if err != nil {
				logger.Error("Couldn't complete URLs", err)
				sources = sourcesBak
				completedSuccessfully = false
			}
		}

		if *resolveArg && *completeArg {
			if completedSuccessfully {
				logger.Log("Resolving files")
				sourcesBak = sources
				sources, err = resolveUrls(sources)
				if err != nil {
					logger.Error("Couldn't resolve URLs", err)
					sources = sourcesBak
				}
			} else {
				logger.Error("Couldn't resolve URLs", nil)
			}
		} else if *resolveArg {
			logger.Error("Resolve can only be used in combination with -complete", nil)
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
		logger.LogF("Saving output to %s", *outputFileArg)
		err := saveToFile(allSources, *outputFileArg)
		if err != nil {
			logger.Error(fmt.Sprintf("Couldn't save to output file %s", *outputFileArg), err)
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

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
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
