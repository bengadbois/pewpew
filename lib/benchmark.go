package pewpew

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type (
	//BenchmarkConfig is the top level struct that contains the configuration for a benchmark test
	BenchmarkConfig struct {
		Verbose bool
		Quiet   bool

		//RPS is the requests per second rate to make for each Target
		RPS int
		//Duration is the number of seconds to run the benchmark test
		Duration int
		Targets  []Target

		//global target settings

		DNSPrefetch     bool
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
		NoHTTP2         bool
		EnforceSSL      bool
	}
)

//NewBenchmarkConfig creates a new BenchmarkConfig
//with package defaults
func NewBenchmarkConfig() (b *BenchmarkConfig) {
	b = &BenchmarkConfig{
		RPS:      DefaultRPS,
		Duration: DefaultDuration,
		Targets: []Target{
			{
				URL:             DefaultURL,
				Timeout:         DefaultTimeout,
				Method:          DefaultMethod,
				UserAgent:       DefaultUserAgent,
				FollowRedirects: true,
			},
		},
	}
	return
}

//RunBenchmark starts the benchmark tests with the provided BenchmarkConfig.
//Throughout the test, data is sent to w, useful for live updates.
func RunBenchmark(b BenchmarkConfig, w io.Writer) ([][]RequestStat, error) {
	if w == nil {
		return nil, errors.New("nil writer")
	}
	err := validateBenchmarkConfig(b)
	if err != nil {
		return nil, errors.New("invalid configuration: " + err.Error())
	}
	targetCount := len(b.Targets)

	//setup printer
	p := printer{output: w}

	//setup the queue of requests, one queue per target
	requestQueues := make([](chan http.Request), targetCount)
	errChans := make([](chan error), targetCount)
	for idx, target := range b.Targets {
		requestQueue, err := createRequestQueue(b.RPS*b.Duration, target)
		if err != nil {
			return nil, err
		}
		requestQueues[idx] = requestQueue
	}

	if targetCount == 1 {
		fmt.Fprintf(w, "Benchmarking %d target:\n", targetCount)
	} else {
		fmt.Fprintf(w, "Benchmarking %d targets:\n", targetCount)
	}

	//when a target is finished, send all stats into this
	targetStats := make(chan []RequestStat)
	for idx, target := range b.Targets {
		go func(target Target, requestQueue chan http.Request, errChan chan error, targetStats chan []RequestStat) {
			p.writeString(fmt.Sprintf("- Benchmarking %s at %d RSP, for %d seconds\n", target.URL, b.RPS, b.Duration))

			requestStatChan := make(chan RequestStat) //workers communicate each requests' info

			client := createClient(target)

			ticker := time.NewTicker(1 * time.Second)
			secondsLeft := b.Duration
			go func() {
				for {
					<-ticker.C
					secondsLeft--
					if secondsLeft < 0 {
						return
					}
					for i := 0; i < b.RPS; i++ {
						//run all the requests at the start of the second
						//note: this means it's a little bursty, not evenly
						//distributed throughout the 1 second window
						go func() {
							req := <-requestQueue
							response, stat := runRequest(req, client)
							if !b.Quiet {
								p.printStat(stat)
								if b.Verbose {
									p.printVerbose(&req, response)
								}
							}
							requestStatChan <- stat
						}()
					}
				}
			}()

			requestStats := make([]RequestStat, b.RPS*b.Duration)
			requestsCompleteCount := 0
			for {
				stat := <-requestStatChan
				requestStats[requestsCompleteCount] = stat
				requestsCompleteCount++
				if requestsCompleteCount == b.RPS*b.Duration {
					//all requests are finished
					break
				}
			}
			targetStats <- requestStats
		}(target, requestQueues[idx], errChans[idx], targetStats)
	}
	targetRequestStats := make([][]RequestStat, targetCount)
	targetDoneCount := 0
	for reqStats := range targetStats {
		targetRequestStats[targetDoneCount] = reqStats
		targetDoneCount++
		if targetDoneCount == targetCount {
			//all targets are finished
			break
		}
	}

	return targetRequestStats, nil
}

func validateBenchmarkConfig(b BenchmarkConfig) error {
	if len(b.Targets) == 0 {
		return errors.New("zero targets")
	}
	if b.Duration <= 0 {
		return errors.New("duration must be greater than zero")
	}
	if b.RPS <= 0 {
		return errors.New("RPS must be greater than zero")
	}

	for _, target := range b.Targets {
		if err := validateTarget(target); err != nil {
			return err
		}
	}
	return nil
}
