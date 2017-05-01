package pewpew

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

//TODO move to other file
//so concurrent workers don't interlace messages
var writeLock sync.Mutex

//TODO move to other file
type workerDone struct{}

type (
	//StressConfig is the top level struct that contains the configuration for a stress test
	StressConfig struct {
		StressTargets []StressTarget
		Verbose       bool
		Quiet         bool

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
		NoHTTP2         bool
		EnforceSSL      bool
	}
	//StressTarget combines stress related configuration with a Target configuration
	StressTarget struct {
		//How many total requests to make for this Target
		Count int
		//How many requests can be happening simultaneously for this Target
		Concurrency int
		Target      Target
	}
)

//NewStressConfig creates a new StressConfig
//with package defaults
func NewStressConfig() (s *StressConfig) {
	s = &StressConfig{
		StressTargets: []StressTarget{
			{
				Count:       DefaultCount,
				Concurrency: DefaultConcurrency,
				Target: Target{
					URL:             DefaultURL,
					Timeout:         DefaultTimeout,
					Method:          DefaultMethod,
					UserAgent:       DefaultUserAgent,
					FollowRedirects: true,
				},
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
	err := validateStressConfig(s)
	if err != nil {
		return nil, errors.New("invalid configuration: " + err.Error())
	}
	targetCount := len(s.StressTargets)

	//setup the queue of requests, one queue per target
	requestQueues := make([](chan http.Request), targetCount)
	for idx, stressTarget := range s.StressTargets {
		requestQueues[idx] = make(chan http.Request, stressTarget.Count)
		for i := 0; i < stressTarget.Count; i++ {
			req, err := buildRequest(stressTarget.Target)
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
	for idx, stressTarget := range s.StressTargets {
		go func(target StressTarget, requestQueue chan http.Request, targetStats chan []RequestStat) {
			writeLock.Lock()
			fmt.Fprintf(w, "- Running %d tests at %s, %d at a time\n", target.Count, target.Target.URL, target.Concurrency)
			writeLock.Unlock()

			workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
			requestStatChan := make(chan RequestStat) //workers communicate each requests' info

			client := createClient(target.Target)

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
		}(stressTarget, requestQueues[idx], targetStats)
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

func validateStressConfig(s StressConfig) error {
	if len(s.StressTargets) == 0 {
		return errors.New("zero targets")
	}
	for _, stressTarget := range s.StressTargets {
		if err := validateStressTarget(stressTarget); err != nil {
			return err
		}
	}
	return nil
}

func validateStressTarget(stressTarget StressTarget) error {
	if stressTarget.Count <= 0 {
		return errors.New("request count must be greater than zero")
	}
	if stressTarget.Concurrency <= 0 {
		return errors.New("concurrency must be greater than zero")
	}
	if stressTarget.Concurrency > stressTarget.Count {
		return errors.New("concurrency must be higher than request count")
	}

	if err := validateTarget(stressTarget.Target); err != nil {
		return err
	}
	return nil
}
