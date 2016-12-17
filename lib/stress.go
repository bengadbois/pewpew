package pewpew

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	color "github.com/fatih/color"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

//so concurrent workers don't interlace messages
var writeLock sync.Mutex

type workerDone struct{}

type requestStat struct {
	Duration   int64     `json:"duration"` //nanoseconds
	StartTime  time.Time `json:"startTime"`
	StatusCode int       `json:"statusCode"` //200, 404, etc.
}
type requestStatSummary struct {
	avgQPS      float64     //per nanoseconds
	avgDuration int64       //nanoseconds
	maxDuration int64       //nanoseconds
	minDuration int64       //nanoseconds
	statusCodes map[int]int //counts of each code
}

type (
	//Stress is the top level struct that contains the configuration of stress test
	Stress struct {
		URL                string
		Count              int
		Concurrency        int
		Timeout            time.Duration
		ReqMethod          string
		ReqBody            string
		ReqBodyFilename    string
		ReqHeaders         http.Header
		UserAgent          string
		BasicAuth          BasicAuth
		IgnoreSSL          bool
		Compress           bool
		NoHTTP2            bool
		ResultFilenameJSON string
		Quiet              bool
		Verbose            bool
	}
	//BasicAuth just wraps the user and password in a convenient struct
	BasicAuth struct {
		User     string
		Password string
	}
)

//defaults
const (
	DefaultCount       = 10
	DefaultConcurrency = 1
	DefaultTimeout     = 10 * time.Second
	DefaultReqMethod   = "GET"
	DefaultUserAgent   = "pewpew"
)

//NewStress creates a new Stress object
//with reasonable defaults, but needs URL set
func NewStress() (s *Stress) {
	s = &Stress{
		Count:       DefaultCount,
		Concurrency: DefaultConcurrency,
		Timeout:     DefaultTimeout,
		ReqMethod:   DefaultReqMethod,
		UserAgent:   DefaultUserAgent,
	}
	return
}

//SetURL sets the target URL
func (s *Stress) SetURL(url string) {
	s.URL = url
	return
}

//Run starts the stress tests
func (s *Stress) Run() error {
	//checks
	url, err := url.Parse(s.URL)
	if err != nil || url.String() == "" {
		return errors.New("invalid URL")
	}
	if s.Count <= 0 {
		return errors.New("request count must be greater than zero")
	}
	if s.Concurrency <= 0 {
		return errors.New("concurrency must be greater than zero")
	}
	if s.Timeout <= time.Millisecond {
		return errors.New("timeout must be greater than one millisecond")
	}
	if s.Concurrency > s.Count {
		return errors.New("concurrency must be higher than request count")
	}

	//clean up URL
	//default to http if not specified
	if url.Scheme == "" {
		url.Scheme = "http"
	}

	fmt.Println("Stress testing " + url.String() + "...")
	fmt.Printf("Running %d tests, %d at a time\n", s.Count, s.Concurrency)

	//setup the request
	var req *http.Request
	if s.ReqBodyFilename != "" {
		fileContents, err := ioutil.ReadFile(s.ReqBodyFilename)
		if err != nil {
			return errors.New("failed to read contents of file " + s.ReqBodyFilename + ": " + err.Error())
		}
		req, err = http.NewRequest(s.ReqMethod, url.String(), bytes.NewBuffer(fileContents))
	} else if s.ReqBody != "" {
		req, err = http.NewRequest(s.ReqMethod, url.String(), bytes.NewBuffer([]byte(s.ReqBody)))
	} else {
		req, err = http.NewRequest(s.ReqMethod, url.String(), nil)
	}
	if err != nil {
		return errors.New("failed to create request: " + err.Error())
	}
	//add headers
	req.Header = s.ReqHeaders
	req.Header.Set("User-Agent", s.UserAgent)
	if s.BasicAuth.User != "" {
		req.SetBasicAuth(s.BasicAuth.User, s.BasicAuth.Password)
	}

	//setup the queue of requests
	requestChan := make(chan *http.Request, s.Count)
	for i := 0; i < s.Count; i++ {
		requestChan <- req
	}
	close(requestChan)

	workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
	requestStatChan := make(chan requestStat) //workers communicate each requests' info

	//workers
	totalStartTime := time.Now()
	var totalEndTime time.Time
	workerErrChan := make(chan error)
	for i := 0; i < s.Concurrency; i++ {
		go func(workerErrChan chan error) {
			tr := &http.Transport{}
			if s.NoHTTP2 {
				nilMap := make(map[string](func(authority string, c *tls.Conn) http.RoundTripper))
				tr = &http.Transport{
					TLSNextProto: nilMap,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: s.IgnoreSSL}}
			}
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: s.IgnoreSSL}
			tr.DisableCompression = !s.Compress
			client := &http.Client{Timeout: s.Timeout, Transport: tr}
			for {
				select {
				case req, ok := <-requestChan:
					if !ok {
						workerDoneChan <- workerDone{}
						return
					}
					//run the actual request
					reqStartTime := time.Now()
					response, err := client.Do((*http.Request)(req))
					reqEndTime := time.Now()
					if err != nil {
						workerErrChan <- errors.New("Failed to make request:" + err.Error())
						return
					}
					reqTimeNs := (reqEndTime.UnixNano() - reqStartTime.UnixNano())

					if !s.Quiet {
						writeLock.Lock()
						if response.StatusCode >= 100 && response.StatusCode < 200 {
							color.Set(color.FgBlue)
						} else if response.StatusCode >= 200 && response.StatusCode < 300 {
							color.Set(color.FgGreen)
						} else if response.StatusCode >= 300 && response.StatusCode < 400 {
							color.Set(color.FgCyan)
						} else if response.StatusCode >= 400 && response.StatusCode < 500 {
							color.Set(color.FgMagenta)
						} else {
							color.Set(color.FgRed)
						}
						fmt.Printf("%s %d\t%dms\t-> %s %s\n", response.Proto, response.StatusCode, reqTimeNs/1000000, req.Method, req.URL)
						color.Unset()
						if s.Verbose {
							var requestInfo string
							//request details
							requestInfo = requestInfo + fmt.Sprintf("Request:\n%+v\n\n", *req)

							//reponse metadata
							requestInfo = requestInfo + fmt.Sprintf("Response:\n%+v\n\n", *response)

							//reponse body
							defer response.Body.Close()
							body, err := ioutil.ReadAll(response.Body)
							if err != nil {
								workerErrChan <- errors.New("Failed to read response body:" + err.Error())
								return
							}
							requestInfo = requestInfo + fmt.Sprintf("Body:\n%s\n\n", body)
							fmt.Println(requestInfo)
						}
						writeLock.Unlock()
					}

					requestStatChan <- requestStat{Duration: reqTimeNs, StartTime: reqStartTime, StatusCode: response.StatusCode}
				}
			}
		}(workerErrChan)
	}

	allRequestStats := make([]requestStat, s.Count)
	requestsCompleteCount := 0
	workersDoneCount := 0
	//wait for all workers to finish
WorkerLoop:
	for {
		select {
		case workerErr := <-workerErrChan:
			return workerErr
		case <-workerDoneChan:
			workersDoneCount++
			if workersDoneCount == s.Concurrency {
				//all workers are done
				totalEndTime = time.Now()
				break WorkerLoop
			}
		case requestStat := <-requestStatChan:
			allRequestStats[requestsCompleteCount] = requestStat
			requestsCompleteCount++
		}
	}

	fmt.Print("----Summary----\n\n")

	//info about the request
	fmt.Println("Method: " + req.Method)
	fmt.Println("Host: " + req.Host)

	totalTimeNs := totalEndTime.UnixNano() - totalStartTime.UnixNano()
	reqStats := createRequestsStats(allRequestStats, totalTimeNs)
	fmt.Println(createTextSummary(reqStats, totalTimeNs))

	if s.ResultFilenameJSON != "" {
		fmt.Print("Writing full result data to: " + s.ResultFilenameJSON + " ...")
		json, _ := json.MarshalIndent(allRequestStats, "", "    ")
		err = ioutil.WriteFile(s.ResultFilenameJSON, json, 0644)
		if err != nil {
			return errors.New("failed to write full result data to " + s.ResultFilenameJSON + ": " + err.Error())
		}
		fmt.Println("finished!")
	}

	return nil
}

//create statistical summary of all requests
func createRequestsStats(requestStats []requestStat, totalTimeNs int64) requestStatSummary {
	if len(requestStats) == 0 {
		return requestStatSummary{}
	}

	requestCodes := make(map[int]int)
	summary := requestStatSummary{maxDuration: requestStats[0].Duration, minDuration: requestStats[0].Duration, statusCodes: requestCodes}
	var totalDurations int64
	totalDurations = 0 //total time of all requests (concurrent is counted)
	for i := 0; i < len(requestStats); i++ {
		if requestStats[i].Duration > summary.maxDuration {
			summary.maxDuration = requestStats[i].Duration
		}
		if requestStats[i].Duration < summary.minDuration {
			summary.minDuration = requestStats[i].Duration
		}
		totalDurations += requestStats[i].Duration
		summary.statusCodes[requestStats[i].StatusCode]++
	}
	summary.avgDuration = totalDurations / int64(len(requestStats))
	summary.avgQPS = float64(len(requestStats)) / float64(totalTimeNs)
	return summary
}

//creates nice readable summary of entire stress test
func createTextSummary(reqStatSummary requestStatSummary, totalTimeNs int64) string {
	summary := "\n"

	summary = summary + "Runtime Statistics:\n"
	summary = summary + "Total time:  " + strconv.Itoa(int(totalTimeNs/1000000)) + " ms\n"
	summary = summary + "Mean QPS:    " + fmt.Sprintf("%.2f", reqStatSummary.avgQPS*1000000000) + " req/sec\n"

	summary = summary + "\nQuery Statistics\n"
	summary = summary + "Mean query:     " + strconv.Itoa(int(reqStatSummary.avgDuration/1000000)) + " ms\n"
	summary = summary + "Fastest query:  " + strconv.Itoa(int(reqStatSummary.minDuration/1000000)) + " ms\n"
	summary = summary + "Slowest query:  " + strconv.Itoa(int(reqStatSummary.maxDuration/1000000)) + " ms\n"

	summary = summary + "\nResponse Codes\n"
	//sort the status codes
	var codes []int
	for key := range reqStatSummary.statusCodes {
		codes = append(codes, key)
	}
	sort.Ints(codes)
	for _, code := range codes {
		summary = summary + fmt.Sprintf("%d", code) + ": " + fmt.Sprintf("%d", reqStatSummary.statusCodes[code]) + " responses\n"
	}
	return summary
}
