package pewpew

import (
	"errors"
	"time"
)

//Reasonable default values for a target
const (
	DefaultURL         = "http://localhost"
	DefaultTimeout     = "10s"
	DefaultMethod      = "GET"
	DefaultUserAgent   = "pewpew"
	DefaultCount       = 10
	DefaultConcurrency = 1
	DefaultRPS         = 10
	DefaultDuration    = 15
)

type (
	//Target is location of where send the HTTP request and how to send it.
	Target struct {
		URL string
		//Whether or not to interpret the URL as a regular expression string
		//and generate actual target URLs from that
		RegexURL bool
		//whether or not to resolve hostname to IP address before making request,
		//eliminating that aspect of timing
		DNSPrefetch bool
		Timeout     string
		//A valid HTTP method: GET, HEAD, POST, etc.
		Method string
		//String that is the content of the HTTP body. Empty string is no body.
		Body string
		// Whether or not to interpret the Body as regular expression string
		// and generate actual body from that
		RegexBody bool
		//A location on disk to read the HTTP body from. Empty string means it will not be read.
		BodyFilename    string
		Headers         string
		Cookies         string
		UserAgent       string
		BasicAuth       string
		Compress        bool
		KeepAlive       bool
		FollowRedirects bool
		NoHTTP2         bool
		EnforceSSL      bool
	}
)

func validateTarget(target Target) error {
	if target.URL == "" {
		return errors.New("empty URL")
	}
	if target.Method == "" {
		return errors.New("method cannot be empty string")
	}
	if target.Timeout != "" {
		timeout, err := time.ParseDuration(target.Timeout)
		if err != nil {
			return errors.New("failed to parse timeout: " + target.Timeout)
		}
		if timeout <= time.Millisecond {
			return errors.New("timeout must be greater than one millisecond")
		}
	}
	return nil
}
