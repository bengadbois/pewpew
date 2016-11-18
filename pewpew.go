package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

var (
	//stress
	stress = kingpin.Command("stress", "Run predefined load of requests.").Alias("s")

	//stress flags
	stressCount       = stress.Flag("num", "Number of requests to make.").Short('n').Default("1").Int()
	stressConcurrency = stress.Flag("concurrent", "Number of multiple requests to make.").Short('c').Default("1").Int()

	//request flags
	stressTimeout   = stress.Flag("timeout", "Maximum seconds to wait for response").Short('t').Default("10s").Duration()
	stressReqMethod = stress.Flag("request-method", "Request type. GET, HEAD, POST, PUT, etc.").Short('X').Default("GET").String()
	stressReqBody   = stress.Flag("body", "String to use as request body e.g. POST body.").String()
	stressHeaders   = HTTPHeader(stress.Flag("header", "Add arbitrary header line, eg. 'Accept-Encoding:gzip'").Short('H'))
	stressUserAgent = stress.Flag("user-agent", "Add User-Agent header.").Short('A').Default("pewpew").String()
	stressBasicAuth = BasicAuth(stress.Flag("basic-auth", "Add HTTP basic authentication, eg. 'user123:password456'"))
	stressHttp2     = stress.Flag("http2", "Use HTTP2.").Bool()

	//url
	stressUrl = stress.Arg("url", "URL to stress, formatted http[s]://hostname[:port][/path]").String()

	//global flags
	verbose  = kingpin.Flag("verbose", "Print extra troubleshooting info").Short('v').Bool()
	cpuCount = kingpin.Flag("cpu", "Number of CPUs to use.").Default(strconv.Itoa(runtime.GOMAXPROCS(0))).Int()
)

func main() {
	kingpin.CommandLine.Help = "HTTP(S) & HTTP2 load tester for performance and stress testing"
	kingpin.CommandLine.HelpFlag.Short('h')

	parseArgs := kingpin.Parse()

	runtime.GOMAXPROCS(*cpuCount)
	if *cpuCount < 1 {
		kingpin.Fatalf("CPU count must be greater or equal to 1")
	}

	switch parseArgs {
	case "stress":
		kingpin.FatalIfError(runStress(), "stress failed")
	}
}

type workerDone struct{}

type requestStat struct {
	duration int64 //nanoseconds
}
type requestStatSummary struct {
	avgQps      float64 //per nanoseconds
	avgDuration int64   //nanoseconds
	maxDuration int64   //nanoseconds
	minDuration int64   //nanoseconds
}

func runStress() error {
	//checks
	if *stressUrl == "" {
		return errors.New("needs URL")
	}
	if *stressCount <= 0 {
		return errors.New("number of requests must be one or more")
	}
	if *stressConcurrency <= 0 {
		return errors.New("concurrency must be one or more")
	}
	if *stressTimeout < 0 {
		return errors.New("timeout must be zero or more")
	}
	if *stressConcurrency > *stressCount {
		return errors.New("concurrency must be higher than number of requests")
	}

	fmt.Println("Stress testing " + *stressUrl + "...")
	fmt.Printf("Running %d tests, %d at a time\n", *stressCount, *stressConcurrency)

	//setup the request
	var req *http.Request
	var err error
	if *stressReqBody != "" {
		req, err = http.NewRequest(*stressReqMethod, *stressUrl, bytes.NewBuffer([]byte(*stressReqBody)))
	} else {
		req, err = http.NewRequest(*stressReqMethod, *stressUrl, nil)
	}
	if err != nil {
		return errors.New("failed to create request: " + err.Error())
	}
	req.Header = *stressHeaders //add headers
	req.Header.Set("User-Agent", *stressUserAgent)
	if (*stressBasicAuth).String() != "" {
		req.SetBasicAuth((*stressBasicAuth).User, (*stressBasicAuth).Password)
	}

	//setup the queue of requests
	requestChan := make(chan *http.Request, *stressCount)
	for i := 0; i < *stressCount; i++ {
		requestChan <- req
	}
	close(requestChan)

	workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
	requestStatChan := make(chan requestStat) //workers communicate each requests' info

	//workers
	totalStartTime := time.Now()
	var totalEndTime time.Time
	for i := 0; i < *stressConcurrency; i++ {
		//TODO handle the returned errors from this
		go func() error {
			tr := &http.Transport{}
			if !*stressHttp2 {
				nilMap := make(map[string](func(authority string, c *tls.Conn) http.RoundTripper))
				tr = &http.Transport{TLSNextProto: nilMap}
			}
			client := &http.Client{Timeout: time.Duration(*stressTimeout) * time.Second, Transport: tr}
			for {
				select {
				case req, ok := <-requestChan:
					if !ok {
						workerDoneChan <- workerDone{}
						return nil
					}
					//run the acutal request
					reqStartTime := time.Now()
					response, err := client.Do((*http.Request)(req))
					reqEndTime := time.Now()
					if err != nil {
						return errors.New("Failed to make request:" + err.Error())
					}
					reqTimeNs := (reqEndTime.UnixNano() - reqStartTime.UnixNano())

					var requestData string
					if *verbose {
						requestData = "----Request----\n\n"

						//request timing
						requestData = requestData + fmt.Sprintf("Request took %dms\n\n", reqTimeNs/1000000)

						//reponse metadata
						requestData = requestData + fmt.Sprintf("Response:\n%+v\n\n", *response)

						//reponse body
						defer response.Body.Close()
						body, err := ioutil.ReadAll(response.Body)
						if err != nil {
							return errors.New("Failed to read response body:" + err.Error())
						}
						requestData = requestData + fmt.Sprintf("Body:\n%s\n\n", body)
					}

					if requestData != "" {
						fmt.Print(requestData)
					}
					requestStatChan <- requestStat{duration: reqTimeNs}
				}
			}
		}()
	}

	allRequestStats := make([]requestStat, *stressCount)
	requestsCompleteCount := 0
	workersDoneCount := 0
	//wait for all workers to finish
WorkerLoop:
	for {
		select {
		case <-workerDoneChan:
			workersDoneCount++
			if workersDoneCount == *stressConcurrency {
				//all workers are done
				totalEndTime = time.Now()
				break WorkerLoop
			}
		case requestStat := <-requestStatChan:
			allRequestStats[requestsCompleteCount] = requestStat
			requestsCompleteCount++
		}
	}

	fmt.Println("----Summary----\n")

	//info about the request
	fmt.Println("Method: " + req.Method)
	fmt.Println("Host: " + req.Host)

	totalTimeNs := totalEndTime.UnixNano() - totalStartTime.UnixNano()
	reqStats := createRequestsStats(allRequestStats, totalTimeNs)
	fmt.Println(createTextSummary(reqStats, totalTimeNs))

	return nil
}

//create statistical summary of all requests
func createRequestsStats(requestStats []requestStat, totalTimeNs int64) requestStatSummary {
	if len(requestStats) == 0 {
		return requestStatSummary{}
	}

	summary := requestStatSummary{maxDuration: requestStats[0].duration, minDuration: requestStats[0].duration}
	var totalDurations int64
	totalDurations = 0 //total time of all requests (concurrent is counted)
	for i := 0; i < len(requestStats); i++ {
		if requestStats[i].duration > summary.maxDuration {
			summary.maxDuration = requestStats[i].duration
		}
		if requestStats[i].duration < summary.minDuration {
			summary.minDuration = requestStats[i].duration
		}
		totalDurations += requestStats[i].duration
	}
	summary.avgDuration = totalDurations / int64(len(requestStats))
	summary.avgQps = float64(len(requestStats)) / float64(totalTimeNs)
	return summary
}

//creates nice readable summary of entire stress test
func createTextSummary(reqStatSummary requestStatSummary, totalTimeNs int64) string {
	summary := "\n"

	summary = summary + "Runtime Statistics:\n"
	summary = summary + "Total time:  " + strconv.Itoa(int(totalTimeNs/1000000)) + " ms\n"
	summary = summary + "Mean QPS:    " + fmt.Sprintf("%.2f", reqStatSummary.avgQps*1000000000) + " req/sec\n"

	summary = summary + "\nQuery Statistics\n"
	summary = summary + "Mean query:     " + strconv.Itoa(int(reqStatSummary.avgDuration/1000000)) + " ms\n"
	summary = summary + "Fastest query:  " + strconv.Itoa(int(reqStatSummary.minDuration/1000000)) + " ms\n"
	summary = summary + "Slowest query:  " + strconv.Itoa(int(reqStatSummary.maxDuration/1000000)) + " ms\n"
	return summary
}
