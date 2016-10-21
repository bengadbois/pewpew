package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

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

//verbose levels
const (
	VerboseNone = iota
	VerboseLow
	VerboseMedium
	VerboseHigh
)

//flags
var (
	numTests      int
	timeout       int
	concurrency   int
	requestMethod string
	verboseLevel  int
)

func init() {
	RootCmd.AddCommand(stressCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stressCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stressCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	stressCmd.Flags().IntVarP(&numTests, "num", "n", 100, "Number of requests to make")
	stressCmd.Flags().IntVarP(&concurrency, "concurrent", "c", 1, "Number of multiple requests to make")
	stressCmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "Maximum seconds to wait for response. 0 means unlimited")
	stressCmd.Flags().StringVarP(&requestMethod, "requestMethod", "X", "GET", "Request type. GET, HEAD, POST, PUT, etc.")
	stressCmd.Flags().IntVarP(&verboseLevel, "verbose", "v", 0, "Level of verbosity ("+strconv.Itoa(VerboseNone)+"-"+strconv.Itoa(VerboseHigh)+")")
}

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress http[s]://hostname[:port]/path",
	Short: "Run predefined load of requests",
	Long:  `Run predefined load of requests`,
	RunE:  runStress,
}

func runStress(cmd *cobra.Command, args []string) error {
	//checks
	if len(args) != 1 {
		return errors.New("needs URL")
	}
	if numTests <= 0 {
		return errors.New("number of requests must be one or more")
	}
	if concurrency <= 0 {
		return errors.New("concurrency must be one or more")
	}
	if timeout < 0 {
		return errors.New("timeout must be zero or more")
	}
	if concurrency > numTests {
		return errors.New("concurrency must be higher than number of requests")
	}
	if verboseLevel < VerboseNone || verboseLevel > VerboseHigh {
		return errors.New("verbose level must be between " + strconv.Itoa(VerboseNone) + " and " + strconv.Itoa(VerboseHigh))
	}

	url := args[0]

	fmt.Println("Stress testing " + url + "...")

	//setup the request
	req, err := http.NewRequest(requestMethod, url, nil)
	if err != nil {
		return errors.New("failed to create request: " + err.Error())
	}

	//info about the request
	if verboseLevel >= VerboseMedium {
		fmt.Println(req.Proto)
		fmt.Println("Method: " + req.Method)
		fmt.Println("Host: " + req.Host)
	}

	//setup the queue of requests
	requestChan := make(chan *http.Request, numTests)
	for i := 0; i < numTests; i++ {
		requestChan <- req
	}
	close(requestChan)

	workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
	requestStatChan := make(chan requestStat) //workers communicate each requests' info

	//workers
	totalStartTime := time.Now()
	for i := 0; i < concurrency; i++ {
		//TODO handle the returned errors from this
		go func() error {
			client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
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
					if verboseLevel >= VerboseLow {
						requestData = "--------\n\n"
					}
					if verboseLevel >= VerboseLow {
						//request timing
						requestData = requestData + fmt.Sprintf("Request took %dms\n\n", reqTimeNs/1000000)
					}
					if verboseLevel >= VerboseMedium {
						//reponse metadata
						requestData = requestData + fmt.Sprintf("Response:\n%+v\n\n", response)
					}
					if verboseLevel >= VerboseHigh {
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

	allRequestStats := make([]requestStat, numTests)
	requestsCompleteCount := 0
	workersDoneCount := 0
	//wait for all workers to finish
	for {
		select {
		case <-workerDoneChan:
			workersDoneCount++
			if workersDoneCount == concurrency {
				//all workers are done
				totalEndTime := time.Now()

				totalTimeNs := totalEndTime.UnixNano() - totalStartTime.UnixNano()
				reqStats := createRequestsStats(allRequestStats, totalTimeNs)
				fmt.Println(createTextSummary(reqStats, totalTimeNs))
				return nil
			}
		case requestStat := <-requestStatChan:
			allRequestStats[requestsCompleteCount] = requestStat
			requestsCompleteCount++
		}
	}
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
