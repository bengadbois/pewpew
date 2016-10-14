package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"net/http"
)

type stressRequest interface{}

type finishedStress struct{}

type workerDone struct{}

type requestStat struct {
	duration int64 //milliseconds
}
type requestStatSummary struct {
	avgDuration      int64 //milliseconds
	longestDuration  int64 //milliseconds
	shortestDuration int64 //milliseconds
}

//flags
var (
	numTests    int
	timeout     int
	concurrency int
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
}

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress http[s]://hostname[:port]/path",
	Short: "Run predefined load of requests",
	Long:  `Run predefined load of requests`,
	RunE:  RunStress,
}

func RunStress(cmd *cobra.Command, args []string) error {
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

	fmt.Println("running stress")

	//setup the queue of requests
	requestChan := make(chan stressRequest, numTests+concurrency)
	for i := 0; i < numTests; i++ {
		//TODO optimize by not creating a new http request each time since it's the same thing
		req, err := http.NewRequest("GET", args[0], nil)
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
							fmt.Errorf(err.Error()) //TODO handle this further up
						}
						reqTimeMs := (reqEndTime.UnixNano() - reqStartTime.UnixNano()) / 1000000
						fmt.Printf("request took %dms\n", reqTimeMs)
						requestStatChan <- requestStat{duration: reqTimeMs}
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
				fmt.Println("%+v", createStats(allRequestStats))
				return nil
			}
		case requestStat := <-requestStatChan:
			allRequestStats[requestsCompleteCount] = requestStat
			requestsCompleteCount++
		}
	}

	return nil
}

func createStats(requestStats []requestStat) requestStatSummary {
	if len(requestStats) == 0 {
		return requestStatSummary{}
	}

	summary := requestStatSummary{longestDuration: requestStats[0].duration, shortestDuration: requestStats[0].duration}
	var totalDurations int64
	totalDurations = 0
	for i := 0; i < len(requestStats); i++ {
		if requestStats[i].duration > summary.longestDuration {
			summary.longestDuration = requestStats[i].duration
		}
		if requestStats[i].duration < summary.shortestDuration {
			summary.shortestDuration = requestStats[i].duration
		}
		totalDurations += requestStats[i].duration
	}
	summary.avgDuration = totalDurations / int64(len(requestStats))
	return summary
}
