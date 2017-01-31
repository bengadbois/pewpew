package pewpew

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	color "github.com/fatih/color"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

//so concurrent workers don't interlace messages
var writeLock sync.Mutex

type workerDone struct{}

type requestStat struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	//equivalent to the difference between StartTime and EndTime
	Duration time.Duration `json:"duration"`
	//HTTP Status Code, e.g. 200, 404, 503
	StatusCode int `json:"statusCode"`
	Error      bool
}

type (
	//Stress is the top level struct that contains the configuration of stress test
	StressConfig struct {
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
		ResultFilenameCSV  string
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
func NewStressConfig() (s *StressConfig) {
	s = &StressConfig{
		Count:       DefaultCount,
		Concurrency: DefaultConcurrency,
		Timeout:     DefaultTimeout,
		ReqMethod:   DefaultReqMethod,
		UserAgent:   DefaultUserAgent,
	}
	return
}

//Run starts the stress tests
func RunStress(s StressConfig) error {
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
						//queue is empty
						workerDoneChan <- workerDone{}
						return
					}
					//run the actual request
					reqStartTime := time.Now()
					response, responseErr := client.Do((*http.Request)(req))
					reqEndTime := time.Now()

					if !s.Quiet {
						writeLock.Lock()
						if responseErr != nil {
							color.Set(color.FgRed)
							fmt.Println("Failed to make request: " + responseErr.Error())
							response = &http.Response{StatusCode: 0}
							color.Unset()
						} else {
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
							fmt.Printf("%s %d\t%dms\t-> %s %s\n",
								response.Proto,
								response.StatusCode,
								reqEndTime.Sub(reqStartTime).Nanoseconds()/1000000,
								req.Method,
								req.URL)
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
									requestInfo = requestInfo + fmt.Sprintf("Body: Failed to read response body: %s\n", err.Error())
								} else {
									requestInfo = requestInfo + fmt.Sprintf("Body:\n%s\n\n", body)
								}
								fmt.Println(requestInfo)
							}
						}
						writeLock.Unlock()
					}

					requestStatChan <- requestStat{
						StartTime:  reqStartTime,
						EndTime:    reqEndTime,
						Duration:   reqEndTime.Sub(reqStartTime),
						StatusCode: response.StatusCode,
						Error:      responseErr != nil,
					}
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
		case <-workerDoneChan:
			workersDoneCount++
			if workersDoneCount == s.Concurrency {
				//all workers are done
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

	reqStats := createRequestsStats(allRequestStats)
	fmt.Println(createTextSummary(reqStats))

	//write out json
	if s.ResultFilenameJSON != "" {
		fmt.Print("Writing full result data to: " + s.ResultFilenameJSON + " ...")
		json, _ := json.MarshalIndent(allRequestStats, "", "    ")
		err = ioutil.WriteFile(s.ResultFilenameJSON, json, 0644)
		if err != nil {
			return errors.New("failed to write full result data to " +
				s.ResultFilenameJSON + ": " + err.Error())
		}
		fmt.Println("finished!")
	}
	//write out csv
	if s.ResultFilenameCSV != "" {
		fmt.Print("Writing full result data to: " + s.ResultFilenameCSV + " ...")
		file, err := os.Create(s.ResultFilenameCSV)
		if err != nil {
			return errors.New("failed to write full result data to " +
				s.ResultFilenameCSV + ": " + err.Error())
		}
		defer file.Close()

		writer := csv.NewWriter(file)

		for _, req := range allRequestStats {
			line := []string{
				req.StartTime.String(),
				fmt.Sprintf("%d", req.Duration),
				fmt.Sprintf("%d", req.StatusCode)}
			err := writer.Write(line)
			if err != nil {
				return errors.New("failed to write full result data to " +
					s.ResultFilenameCSV + ": " + err.Error())
			}
		}
		defer writer.Flush()
		fmt.Println("finished!")
	}

	return nil
}
