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
	if len(args) != 1 {
		return errors.New("needs URL")
	}
	fmt.Println("running stress")

	//checks
	if numTests <= 0 {
		return errors.New("number of requests must be one or more")
	}
	if concurrency <= 0 {
		return errors.New("concurrency must be one or more")
	}
	if timeout < 0 {
		return errors.New("timeout must be zero or more")
	}

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

	workerDoneChan := make(chan workerDone)

	fmt.Println("workers")
	//workers
	for i := 0; i < concurrency; i++ {
		go func() {
			client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
			for {
				select {
				case req := <-requestChan:
					switch req.(type) {
					case *http.Request:
						fmt.Println("http request")
						resp, err := client.Do(req.(*http.Request))
						if err != nil {
							fmt.Errorf(err.Error())
						}
						fmt.Printf("%+v\n", resp)
					case finishedStress:
						fmt.Println("worker ending")
						workerDoneChan <- workerDone{}
						return
					}
				}
			}
		}()
	}

	workersDoneCount := 0
	//wait for all workers to finish
	for {
		select {
		case <-workerDoneChan:
			workersDoneCount++
			if workersDoneCount == concurrency {
				return nil
			}
		}
	}

	return nil
}
