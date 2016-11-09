package main

import (
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"strings"
)

type HTTPHeaderValue http.Header

func (h *HTTPHeaderValue) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected HEADER:VALUE got '%s'", value)
	}
	(*http.Header)(h).Add(parts[0], parts[1])
	return nil
}

//make it repeatable
func (h *HTTPHeaderValue) IsCumulative() bool {
	return true
}

func (h *HTTPHeaderValue) String() string {
	return ""
}

func HTTPHeader(s kingpin.Settings) (target *http.Header) {
	target = &http.Header{}
	s.SetValue((*HTTPHeaderValue)(target))
	return
}
