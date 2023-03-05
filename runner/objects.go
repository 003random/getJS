package runner

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

type Input struct {
	Type inputType
	Data io.Reader
}

type inputType int

const (
	InputURL inputType = iota
	InputResponse
)

type runner struct {
	Options Options
	Results chan url.URL
}

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
