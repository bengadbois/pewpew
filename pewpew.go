package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	color "github.com/fatih/color"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//so concurrent workers don't interlace messages
var writeLock sync.Mutex

var (
	//stress
	stress = kingpin.Command("stress", "Run predefined load of requests.").Alias("s")

	//stress flags
	stressCount       = stress.Flag("num", "Number of requests to make.").Short('n').Default("1").Int()
	stressConcurrency = stress.Flag("concurrent", "Number of concurrent requests to make.").Short('c').Default("1").Int()

	//request flags
	stressTimeout         = stress.Flag("timeout", "Maximum seconds to wait for response").Short('t').Default("10s").Duration()
	stressReqMethod       = stress.Flag("request-method", "Request type. GET, HEAD, POST, PUT, etc.").Short('X').Default("GET").String()
	stressReqBody         = stress.Flag("body", "String to use as request body e.g. POST body.").String()
	stressReqBodyFilename = stress.Flag("body-file", "Path to file to use as request body. Will overwrite --body if both are present.").String()
	stressHeaders         = HTTPHeader(stress.Flag("header", "Add arbitrary header line, eg. 'Accept-Encoding:gzip'. Repeatable.").Short('H'))
	stressUserAgent       = stress.Flag("user-agent", "Add User-Agent header. Can also be done with the arbitrary header flag.").Short('A').Default("pewpew").String()
	stressBasicAuth       = BasicAuth(stress.Flag("basic-auth", "Add HTTP basic authentication, eg. 'user123:password456'."))
	stressIgnoreSSL       = stress.Flag("ignore-ssl", "Ignore SSL certificate/hostname issues.").Bool()
	stressCompress        = stress.Flag("compress", "Add 'Accept-Encoding: gzip' header if Accept-Encoding is not already present.").Short('C').Bool()
	stressNoH2            = stress.Flag("no-http2", "Disable HTTP2.").Bool()
	stressQuiet           = kingpin.Flag("quiet", "Do not print while requests are running.").Short('q').Bool()

	//url
	stressUrl = stress.Arg("url", "URL to stress, formatted [http[s]://]hostname[:port][/path]").String()

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
	url, err := url.Parse(*stressUrl)
	if err != nil || url.String() == "" {
		return errors.New("invalid URL")
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

	//clean up URL
	//default to http if not specified
	if url.Scheme == "" {
		url.Scheme = "http"
	}

	fmt.Println("Stress testing " + url.String() + "...")
	fmt.Printf("Running %d tests, %d at a time\n", *stressCount, *stressConcurrency)

	//setup the request
	var req *http.Request
	if *stressReqBodyFilename != "" {
		fileContents, err := ioutil.ReadFile(*stressReqBodyFilename)
		if err != nil {
			return errors.New("failed to read contents of file " + *stressReqBodyFilename + ": " + err.Error())
		}
		req, err = http.NewRequest(*stressReqMethod, url.String(), bytes.NewBuffer(fileContents))
	} else if *stressReqBody != "" {
		req, err = http.NewRequest(*stressReqMethod, url.String(), bytes.NewBuffer([]byte(*stressReqBody)))
	} else {
		req, err = http.NewRequest(*stressReqMethod, url.String(), nil)
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
	workerErrChan := make(chan error)
	for i := 0; i < *stressConcurrency; i++ {
		go func(workerErrChan chan error) {
			tr := &http.Transport{}
			if *stressNoH2 {
				nilMap := make(map[string](func(authority string, c *tls.Conn) http.RoundTripper))
				tr = &http.Transport{
					TLSNextProto: nilMap,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: *stressIgnoreSSL}}
			}
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: *stressIgnoreSSL}
			tr.DisableCompression = !*stressCompress
			client := &http.Client{Timeout: *stressTimeout, Transport: tr}
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

					if !*stressQuiet {
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
						if *verbose {
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

					requestStatChan <- requestStat{duration: reqTimeNs}
				}
			}
		}(workerErrChan)
	}

	allRequestStats := make([]requestStat, *stressCount)
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

	fmt.Print("----Summary----\n\n")

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
