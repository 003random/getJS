package runner

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// Input represents an input source for getJS. The input format is determined by the `Type` property.
type Input struct {
	Type InputType
	Data io.Reader
}

// InputType defines the type of input source for getJS.
type InputType int

const (
	// InputURL defines the input format to line separated, plain text, URLs.
	InputURL InputType = iota
	// InputResponse defines the input format to a HTTP response body.
	InputResponse
)

type runner struct {
	Options Options
	Results chan url.URL
}

// Options represents the configuration options for the runner.
type Options struct {
	Request struct {
		Method             string
		Headers            http.Header
		InsecureSkipVerify bool
		Timeout            time.Duration
	}

	Inputs  []Input
	Outputs []io.Writer

	Complete bool
	Resolve  bool

	Threads int

	Verbose bool
	Colors  bool
}
