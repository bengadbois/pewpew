package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"net/http"
)

type stressRequest interface{}

type finishedStress struct{}

type workerDone struct{}

type requestStat struct {
	duration int64 //nanoseconds
}
type requestStatSummary struct {
	avgDuration int64 //nanoseconds
	maxDuration int64 //nanoseconds
	minDuration int64 //nanoseconds
}

//flags
var (
	numTests      int
	timeout       int
	concurrency   int
	requestMethod string
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

	url := args[0]

	fmt.Println("Stress testing " + url + "...")

	//setup the queue of requests
	requestChan := make(chan stressRequest, numTests+concurrency)
	for i := 0; i < numTests; i++ {
		//TODO optimize by not creating a new http request each time since it's the same thing
		req, err := http.NewRequest(requestMethod, url, nil)
		if err != nil {
			return errors.New("failed to create request: " + err.Error())
		}
		requestChan <- req
	}
	for i := 0; i < concurrency; i++ {
		requestChan <- finishedStress{}
	}

	workerDoneChan := make(chan workerDone)   //workers use this to indicate they are done
	requestStatChan := make(chan requestStat) //workers communicate each requests' info

	//workers
	totalStartTime := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
			for {
				select {
				case req := <-requestChan:
					switch req.(type) {
					case *http.Request:
						//run the acutal request
						reqStartTime := time.Now()
						_, err := client.Do(req.(*http.Request))
						reqEndTime := time.Now()
						if err != nil {
							fmt.Printf(err.Error()) //TODO handle this further up
						}
						reqTimeNs := (reqEndTime.UnixNano() - reqStartTime.UnixNano())
						fmt.Printf("request took %dms\n", reqTimeNs/1000000)
						requestStatChan <- requestStat{duration: reqTimeNs}
					case finishedStress:
						workerDoneChan <- workerDone{}
						return
					}
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

				reqStats := createRequestsStats(allRequestStats)
				totalTimeNs := totalEndTime.UnixNano() - totalStartTime.UnixNano()
				fmt.Println(createTextSummary(reqStats, totalTimeNs))
				return nil
			}
		case requestStat := <-requestStatChan:
			allRequestStats[requestsCompleteCount] = requestStat
			requestsCompleteCount++
		}
	}
}

func createRequestsStats(requestStats []requestStat) requestStatSummary {
	if len(requestStats) == 0 {
		return requestStatSummary{}
	}

	summary := requestStatSummary{maxDuration: requestStats[0].duration, minDuration: requestStats[0].duration}
	var totalDurations int64
	totalDurations = 0
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
	return summary
}

func createTextSummary(reqStatSummary requestStatSummary, totalTimeNs int64) string {
	summary := "\n"
	summary = summary + "Average:    " + strconv.Itoa(int(reqStatSummary.avgDuration/1000000)) + "ms\n"
	summary = summary + "Max:        " + strconv.Itoa(int(reqStatSummary.maxDuration/1000000)) + "ms\n"
	summary = summary + "Min:        " + strconv.Itoa(int(reqStatSummary.minDuration/1000000)) + "ms\n"
	summary = summary + "Total Time: " + strconv.Itoa(int(totalTimeNs/1000000)) + "ms"
	return summary
}
