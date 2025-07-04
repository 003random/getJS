package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/003random/getJS/v2/runner"
)

func main() {
	options, err := setup()
	if err != nil {
		log.Fatal(fmt.Errorf("parsing flags: %w", err))
	}

	if err := runner.New(options).Run(); err != nil {
		log.Fatal(err)
	}
}

func setup() (options *runner.Options, err error) {
	options = &runner.Options{}

	flag.StringVar(&options.Request.Method, "method", "GET", "The request method that should be used to make fetch the remote contents.")
	flag.DurationVar(&options.Request.Timeout, "timeout", 5*time.Second, "The request timeout used while fetching the remote contents.")
	flag.BoolVar(&options.Request.InsecureSkipVerify, "insecure", true, "Skip certification verification.")
	flag.BoolVar(&options.Complete, "complete", false, "Complete/Autofil relative URLs by adding the current origin.")
	flag.BoolVar(&options.Resolve, "resolve", false, "Resolve the JavaScript files. Can only be used in combination with '--resolve'. Unresolvable hosts are not included in the results.")
	flag.IntVar(&options.Threads, "threads", 2, "The amount of processing threads to spawn.")
	flag.BoolVar(&options.Verbose, "verbose", false, "Print verbose runtime information and errors.")

	var (
		url    string
		input  arrayFlags
		output arrayFlags
		header arrayFlags
	)

	flag.Var(&header, "header", "The optional request headers to add to the requests. This flag can be used multiple times with a new header each time.")
	flag.StringVar(&url, "url", "", "The URL where the JavaScript sources should be extracted from.")
	flag.Var(&input, "input", "The optional URLs input files. Each URL should be on a new line in plain text format. This flag can be used multiple times with different files.")
	flag.Var(&output, "output", "The optional output file where the results are written to.")

	flag.Parse()

	options.Request.Headers = headers(header)

	options.Inputs = inputs(input)
	options.Outputs = outputs(output)

	// Add an input for the single URL option, if set.
	if len(url) > 0 {
		options.Inputs = append(options.Inputs, runner.Input{
			Type: runner.InputURL,
			Data: strings.NewReader(url),
		})
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(fmt.Errorf("error reading stdin: %v", err))
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Read the first line of stdin to detect its format
		reader := bufio.NewReader(os.Stdin)
		firstLine, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Fatal(fmt.Errorf("error reading first line of stdin: %v", err))
		}

		if isURL(strings.TrimSpace(firstLine)) {
			// Treat as URL input.
			options.Inputs = append(options.Inputs, runner.Input{
				Type: runner.InputURL,
				Data: io.MultiReader(strings.NewReader(firstLine), reader),
			})
		} else {
			// Treat as HTTP response body.
			options.Inputs = append(options.Inputs, runner.Input{
				Type: runner.InputResponse,
				Data: io.MultiReader(strings.NewReader(firstLine), reader),
			})
		}
	}

	return
}

func isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

func outputs(names []string) []io.Writer {
	outputs := append([]io.Writer{}, os.Stdout)

	for _, n := range names {
		file, err := os.OpenFile(n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatal(fmt.Errorf("error parsing output file flag: %v", err))
		}

		outputs = append(outputs, file)
	}

	return outputs
}

func inputs(names []string) []runner.Input {
	inputs := []runner.Input{}

	for _, n := range names {
		file, err := os.Open(n)
		if err != nil {
			log.Fatal(fmt.Errorf("error reading from file %s: %v", n, err))
		}

		inputs = append(inputs, runner.Input{Type: runner.InputURL, Data: file})
	}

	return inputs
}

func headers(args []string) http.Header {
	headers := make(http.Header)
	for _, s := range args {
		parts := strings.Split(s, ":")
		if len(parts) <= 1 {
			log.Fatal(fmt.Errorf("invalid header %s", s))
		}

		headers[strings.TrimSpace(parts[0])] = []string{strings.TrimSpace(strings.Join(parts[1:], ":"))}
	}

	return headers
}

type arrayFlags []string

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}
