package main

import (
	"fmt"
	"net/http"

	"github.com/fatih/color"
)

var (
	boldred   = color.New(color.FgHiRed, color.Bold).SprintFunc()
	grey      = color.New(color.FgHiBlack).SprintFunc()
	boldwhite = color.New(color.FgHiWhite).SprintFunc()
	warn      = color.New(color.FgYellow, color.Bold).SprintFunc()
	alarm     = color.New(color.FgHiWhite, color.BgHiRed, color.Bold).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()

	logLevel = 1
)

func logVerbose(a ...any) {
	if logLevel >= 3 {
		fmt.Println(a...)
	}
}

// applyVerbose sets logLevel from the global -v flag count.
func applyVerbose() {
	if len(globalOpts.Verbose) >= 1 {
		logLevel = 3
	}
}

// loggingTransport wraps an http.RoundTripper and logs each request/response.
type loggingTransport struct {
	base http.RoundTripper
	logf func(format string, args ...any)
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.logf("[ERR] %s %s: %v", req.Method, req.URL, err)
		return nil, err
	}
	t.logf("[%d] %s %s", resp.StatusCode, req.Method, req.URL)
	return resp, nil
}

// newLoggingHTTPClient returns an *http.Client that logs requests via logf.
func newLoggingHTTPClient(logf func(string, ...any)) *http.Client {
	return &http.Client{
		Transport: &loggingTransport{
			base: http.DefaultTransport,
			logf: logf,
		},
	}
}

func init() {
	color.NoColor = false
}
