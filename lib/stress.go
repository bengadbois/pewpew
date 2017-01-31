package pewpew

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	color "github.com/fatih/color"
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
	StatusCode int  `json:"statusCode"`
	Error      bool `json:"error"`
}

type (
	//Stress is the top level struct that contains the configuration of stress test
	StressConfig struct {
		Targets            []Target
		ResultFilenameJSON string
		ResultFilenameCSV  string
		Quiet              bool
		Verbose            bool
	}
	Target struct {
		URL             url.URL
		Count           int //how many total requests to make
		Concurrency     int
		Timeout         time.Duration
		ReqMethod       string
		ReqBody         string
		ReqBodyFilename string
		ReqHeaders      http.Header
		UserAgent       string
		BasicAuth       BasicAuth
		IgnoreSSL       bool
		Compress        bool
		NoHTTP2         bool
	}
	//BasicAuth just wraps the user and password in a convenient struct
	BasicAuth struct {
		User     string
		Password string
	}
)

//defaults
var DefaultURL = url.URL{Scheme: "http", Host: "localhost"}

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
		Targets: []Target{
			{
				URL:         DefaultURL,
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Timeout:     DefaultTimeout,
				ReqMethod:   DefaultReqMethod,
				UserAgent:   DefaultUserAgent,
			},
		},
	}
	return
}

//RunStress starts the stress tests
func RunStress(s StressConfig) error {
	err := ValidateTargets(s)
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("invalid configuration")
	}
	targetCount := len(s.Targets)

	//clean up URL
	for i := 0; i < len(s.Targets); i++ {
		//default to http if not specified
		if s.Targets[i].URL.Scheme == "" {
			s.Targets[i].URL.Scheme = "http"
		}
	}

	if targetCount == 1 {
		fmt.Printf("Stress testing %d target\n", targetCount)
	} else {
		fmt.Printf("Stress testing %d targets\n", targetCount)
	}

	requests := make([]*http.Request, targetCount)
	for i, target := range s.Targets {
		req, err := buildRequest(target)
		if err != nil {
			fmt.Println(err.Error())
			return errors.New("failed to create request with target configuration")
		}
		requests[i] = req
	}

	//setup the queue of requests, one per target
	requestQueues := make([](chan *http.Request), targetCount)
	for idx, target := range s.Targets {
		requestQueues[idx] = make(chan *http.Request, target.Count)
		for i := 0; i < target.Count; i++ {
			requestQueues[idx] <- requests[idx]
		}
		close(requestQueues[idx])
	}

	//when a target is finished, send all stats into this
	targetStats := make(chan []requestStat)
	for idx, target := range s.Targets {
		go func(target Target, requestQueue chan *http.Request, targetStats chan []requestStat) {
			fmt.Printf("Running %d tests at %s, %d at a time\n", target.Count, target.URL.String(), target.Concurrency)

			workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
			requestStatChan := make(chan requestStat) //workers communicate each requests' info

			//start up the workers
			for i := 0; i < target.Concurrency; i++ {
				go func() {
					tr := &http.Transport{}
					if target.NoHTTP2 {
						nilMap := make(map[string](func(authority string, c *tls.Conn) http.RoundTripper))
						tr = &http.Transport{
							TLSNextProto: nilMap,
							TLSClientConfig: &tls.Config{
								InsecureSkipVerify: target.IgnoreSSL}}
					}
					tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: target.IgnoreSSL}
					tr.DisableCompression = !target.Compress
					client := &http.Client{Timeout: target.Timeout, Transport: tr}

					for {
						select {
						case req, ok := <-requestQueue:
							if !ok {
								//queue is empty
								workerDoneChan <- workerDone{}
								return
							}
							//run the actual request
							reqStartTime := time.Now()
							response, responseErr := client.Do(req)
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
							if responseErr == nil {
								requestStatChan <- requestStat{
									StartTime:  reqStartTime,
									EndTime:    reqEndTime,
									Duration:   reqEndTime.Sub(reqStartTime),
									StatusCode: response.StatusCode,
									Error:      false,
								}
							} else {
								requestStatChan <- requestStat{
									StartTime:  reqStartTime,
									EndTime:    reqEndTime,
									Duration:   reqEndTime.Sub(reqStartTime),
									StatusCode: 0,
									Error:      true,
								}
							}
						}
					}
				}()
			}
			requestStats := make([]requestStat, target.Count)
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
	targetRequestStats := make([][]requestStat, targetCount)
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

	fmt.Print("\n----Summary----\n\n")

	for idx, target := range s.Targets {
		//info about the request
		fmt.Printf("----Target %d: %s %s\n", idx+1, target.ReqMethod, target.URL.String())
		reqStats := createRequestsStats(targetRequestStats[idx])
		fmt.Println(createTextSummary(reqStats))
	}

	//combine individual targets to a total one
	globalStats := []requestStat{}
	for i := range s.Targets {
		for j := range targetRequestStats[i] {
			globalStats = append(globalStats, targetRequestStats[i][j])
		}
	}
	fmt.Println("----Global----")
	reqStats := createRequestsStats(globalStats)
	fmt.Println(createTextSummary(reqStats))

	//write out json
	if s.ResultFilenameJSON != "" {
		fmt.Print("Writing full result data to: " + s.ResultFilenameJSON + " ...")
		json, _ := json.MarshalIndent(globalStats, "", "    ")
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

		for _, req := range globalStats {
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

func ValidateTargets(s StressConfig) error {
	if len(s.Targets) == 0 {
		return errors.New("zero targets")
	}
	for _, target := range s.Targets {
		//checks
		if target.URL.String() == "" {
			return errors.New("empty URL")
		}
		if target.Count <= 0 {
			return errors.New("request count must be greater than zero")
		}
		if target.Concurrency <= 0 {
			return errors.New("concurrency must be greater than zero")
		}
		if target.Timeout <= time.Millisecond {
			return errors.New("timeout must be greater than one millisecond")
		}
		if target.Concurrency > target.Count {
			return errors.New("concurrency must be higher than request count")
		}
	}
	return nil
}

//build the http request out of the target's config
func buildRequest(t Target) (*http.Request, error) {
	//setup the request
	var req *http.Request
	var err error
	if t.ReqBodyFilename != "" {
		fileContents, err := ioutil.ReadFile(t.ReqBodyFilename)
		if err != nil {
			return nil, errors.New("failed to read contents of file " + t.ReqBodyFilename + ": " + err.Error())
		}
		req, err = http.NewRequest(t.ReqMethod, t.URL.String(), bytes.NewBuffer(fileContents))
	} else if t.ReqBody != "" {
		req, err = http.NewRequest(t.ReqMethod, t.URL.String(), bytes.NewBuffer([]byte(t.ReqBody)))
	} else {
		req, err = http.NewRequest(t.ReqMethod, t.URL.String(), nil)
	}
	if err != nil {
		return nil, errors.New("failed to create request: " + err.Error())
	}
	//add headers
	req.Header = t.ReqHeaders
	req.Header.Set("User-Agent", t.UserAgent)
	if t.BasicAuth.User != "" {
		req.SetBasicAuth(t.BasicAuth.User, t.BasicAuth.Password)
	}
	return req, nil
}
