package pewpew

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	reggen "github.com/lucasjones/reggen"
	http2 "golang.org/x/net/http2"
)

//so concurrent workers don't interlace messages
var writeLock sync.Mutex

type workerDone struct{}

//RequestStat is the saved information about an individual completed HTTP request
type RequestStat struct {
	Proto     string
	URL       string
	Method    string
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	//equivalent to the difference between StartTime and EndTime
	Duration time.Duration `json:"duration"`
	//HTTP Status Code, e.g. 200, 404, 503
	StatusCode      int   `json:"statusCode"`
	Error           error `json:"error"`
	DataTransferred int   //bytes
}

type (
	//Stress is the top level struct that contains the configuration for a stress test
	StressConfig struct {
		Targets    []Target
		Verbose    bool
		Quiet      bool
		NoHTTP2    bool
		EnforceSSL bool

		//global target settings

		Count           int
		Concurrency     int
		Timeout         string
		Method          string
		Body            string
		BodyFilename    string
		Headers         string
		Cookies         string
		UserAgent       string
		BasicAuth       string
		Compress        bool
		KeepAlive       bool
		FollowRedirects bool
	}
	//Target is location to send the HTTP request.
	Target struct {
		URL string
		//Whether or not to interpret the URL as a regular expression string
		//and generate actual target URLs from that
		RegexURL bool
		//How many total requests to make
		Count int
		//How many requests can be happening simultaneously for this Target
		Concurrency int
		Timeout     string
		//A valid HTTP method: GET, HEAD, POST, etc.
		Method string
		//String that is the content of the HTTP body. Empty string is no body.
		Body string
		//A location on disk to read the HTTP body from. Empty string means it will not be read.
		BodyFilename    string
		Headers         string
		Cookies         string
		UserAgent       string
		BasicAuth       string
		Compress        bool
		KeepAlive       bool
		FollowRedirects bool
	}
)

const (
	DefaultURL         = "http://localhost"
	DefaultCount       = 10
	DefaultConcurrency = 1
	DefaultTimeout     = "10s"
	DefaultMethod      = "GET"
	DefaultUserAgent   = "pewpew"
)

//NewStress creates a new Stress object
//with reasonable defaults
func NewStressConfig() (s *StressConfig) {
	s = &StressConfig{
		Targets: []Target{
			{
				URL:             DefaultURL,
				Count:           DefaultCount,
				Concurrency:     DefaultConcurrency,
				Timeout:         DefaultTimeout,
				Method:          DefaultMethod,
				UserAgent:       DefaultUserAgent,
				FollowRedirects: true,
			},
		},
	}
	return
}

//RunStress starts the stress tests with the provided StressConfig.
//Throughout the test, data is sent to w, useful for live updates.
func RunStress(s StressConfig, w io.Writer) ([][]RequestStat, error) {
	if w == nil {
		return nil, errors.New("nil writer")
	}
	err := validateTargets(s)
	if err != nil {
		return nil, errors.New("invalid configuration: " + err.Error())
	}
	targetCount := len(s.Targets)

	//setup the queue of requests, one queue per target
	requestQueues := make([](chan http.Request), targetCount)
	for idx, target := range s.Targets {
		requestQueues[idx] = make(chan http.Request, target.Count)
		for i := 0; i < target.Count; i++ {
			req, err := buildRequest(target)
			if err != nil {
				return nil, errors.New("failed to create request with target configuration: " + err.Error())
			}
			requestQueues[idx] <- req
		}
		close(requestQueues[idx])
	}

	if targetCount == 1 {
		fmt.Fprintf(w, "Stress testing %d target:\n", targetCount)
	} else {
		fmt.Fprintf(w, "Stress testing %d targets:\n", targetCount)
	}

	//when a target is finished, send all stats into this
	targetStats := make(chan []RequestStat)
	for idx, target := range s.Targets {
		go func(target Target, requestQueue chan http.Request, targetStats chan []RequestStat) {
			writeLock.Lock()
			fmt.Fprintf(w, "- Running %d tests at %s, %d at a time\n", target.Count, target.URL, target.Concurrency)
			writeLock.Unlock()

			workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
			requestStatChan := make(chan RequestStat) //workers communicate each requests' info

			tr := &http.Transport{}
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: !s.EnforceSSL}
			tr.DisableCompression = !target.Compress
			tr.DisableKeepAlives = !target.KeepAlive
			if s.NoHTTP2 {
				tr.TLSNextProto = make(map[string](func(string, *tls.Conn) http.RoundTripper))
			} else {
				http2.ConfigureTransport(tr)
			}
			var timeout time.Duration
			if target.Timeout != "" {
				timeout, _ = time.ParseDuration(target.Timeout)
			} else {
				timeout = time.Duration(0)
			}
			client := &http.Client{Timeout: timeout, Transport: tr}
			if !target.FollowRedirects {
				client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}
			}

			//start up the workers
			for i := 0; i < target.Concurrency; i++ {
				go func() {
					for {
						select {
						case req, ok := <-requestQueue:
							if !ok {
								//queue is empty
								workerDoneChan <- workerDone{}
								return
							}

							response, stat := runRequest(req, client)
							if !s.Quiet {
								writeLock.Lock()
								printStat(stat, w)
								if s.Verbose {
									printVerbose(&req, response, w)
								}
								writeLock.Unlock()
							}

							requestStatChan <- stat
						}
					}
				}()
			}
			requestStats := make([]RequestStat, target.Count)
			requestsCompleteCount := 0
			workersDoneCount := 0
			//wait for all workers to finish
			for {
				select {
				case <-workerDoneChan:
					workersDoneCount++
				case stat := <-requestStatChan:
					requestStats[requestsCompleteCount] = stat
					requestsCompleteCount++
				}
				if workersDoneCount == target.Concurrency {
					//all workers are finished
					break
				}
			}
			targetStats <- requestStats
		}(target, requestQueues[idx], targetStats)
	}
	targetRequestStats := make([][]RequestStat, targetCount)
	targetDoneCount := 0
	for {
		select {
		case reqStats := <-targetStats:
			targetRequestStats[targetDoneCount] = reqStats
			targetDoneCount++
		}
		if targetDoneCount == targetCount {
			//all targets are finished
			break
		}
	}

	return targetRequestStats, nil
}

func validateTargets(s StressConfig) error {
	if len(s.Targets) == 0 {
		return errors.New("zero targets")
	}
	for _, target := range s.Targets {
		//checks
		if target.URL == "" {
			return errors.New("empty URL")
		}
		if target.Count <= 0 {
			return errors.New("request count must be greater than zero")
		}
		if target.Concurrency <= 0 {
			return errors.New("concurrency must be greater than zero")
		}
		if target.Method == "" {
			return errors.New("method cannot be empty string")
		}
		if target.Timeout != "" {
			//TODO should save this parsed duration so don't have to inefficiently reparse later
			timeout, err := time.ParseDuration(target.Timeout)
			if err != nil {
				return errors.New("failed to parse timeout: " + target.Timeout)
			}
			if timeout <= time.Millisecond {
				return errors.New("timeout must be greater than one millisecond")
			}
		}
		if target.Concurrency > target.Count {
			return errors.New("concurrency must be higher than request count")
		}
	}
	return nil
}

//build the http request out of the target's config
func buildRequest(t Target) (http.Request, error) {
	if t.URL == "" {
		return http.Request{}, errors.New("empty URL")
	}
	if len(t.URL) < 8 {
		return http.Request{}, errors.New("URL too short")
	}
	//prepend "http://" if scheme not provided
	//maybe a cleaner way to do this via net.url?
	if t.URL[:7] != "http://" && t.URL[:8] != "https://" {
		t.URL = "http://" + t.URL
	}
	var urlStr string
	var err error
	//when regex set, generate urls
	if t.RegexURL {
		urlStr, err = reggen.Generate(t.URL, 10)
		if err != nil {
			return http.Request{}, errors.New("failed to parse regex: " + err.Error())
		}
	} else {
		urlStr = t.URL
	}
	URL, err := url.Parse(urlStr)
	if err != nil {
		return http.Request{}, errors.New("failed to parse URL " + urlStr + " : " + err.Error())
	}
	if URL.Host == "" {
		return http.Request{}, errors.New("empty hostname")
	}

	//setup the request
	var req *http.Request
	if t.BodyFilename != "" {
		fileContents, err := ioutil.ReadFile(t.BodyFilename)
		if err != nil {
			return http.Request{}, errors.New("failed to read contents of file " + t.BodyFilename + ": " + err.Error())
		}
		req, err = http.NewRequest(t.Method, URL.String(), bytes.NewBuffer(fileContents))
	} else if t.Body != "" {
		req, err = http.NewRequest(t.Method, URL.String(), bytes.NewBuffer([]byte(t.Body)))
	} else {
		req, err = http.NewRequest(t.Method, URL.String(), nil)
	}
	if err != nil {
		return http.Request{}, errors.New("failed to create request: " + err.Error())
	}
	//add headers
	if t.Headers != "" {
		headerMap, err := parseKeyValString(t.Headers, ",", ":")
		if err != nil {
			return http.Request{}, errors.New("could not parse headers: " + err.Error())
		}
		for key, val := range headerMap {
			req.Header.Add(key, val)
		}
	}

	req.Header.Set("User-Agent", t.UserAgent)

	//add cookies
	if t.Cookies != "" {
		cookieMap, err := parseKeyValString(t.Cookies, ";", "=")
		if err != nil {
			return http.Request{}, errors.New("could not parse cookies: " + err.Error())
		}
		for key, val := range cookieMap {
			req.AddCookie(&http.Cookie{Name: key, Value: val})
		}
	}

	if t.BasicAuth != "" {
		authMap, err := parseKeyValString(t.BasicAuth, ",", ":")
		if err != nil {
			return http.Request{}, errors.New("could not parse basic auth: " + err.Error())
		}
		for key, val := range authMap {
			req.SetBasicAuth(key, val)
			break
		}
	}
	return *req, nil
}

//splits on delim into parts and trims whitespace
//delim1 splits the pairs, delim2 splits amongst the pairs
//like parseKeyValString("key1: val2, key3 : val4,key5:val6 ", ",", ":") becomes
//["key1"]->"val2"
//["key3"]->"val4"
//["key5"]->"val6"
func parseKeyValString(keyValStr, delim1, delim2 string) (map[string]string, error) {
	m := make(map[string]string)
	if delim1 == delim2 {
		return m, errors.New("delimiters can't be equal")
	}
	pairs := strings.SplitN(keyValStr, delim1, -1)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, delim2, 2)
		if len(parts) != 2 {
			return m, errors.New("failed to parse into two parts")
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return m, errors.New("key or value is empty")
		}
		m[key] = val
	}
	return m, nil
}
