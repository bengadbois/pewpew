package pewpew

import (
	"fmt"
	"sort"
)

//creates nice readable summary of entire stress test
func createTextSummary(reqStatSummary requestStatSummary) string {
	summary := "\n"

	summary = summary + "Runtime Statistics:\n"
	summary = summary + "Total time:  " + fmt.Sprintf("%d", reqStatSummary.endTime.Sub(reqStatSummary.startTime).Nanoseconds()/1000000) + " ms\n"
	summary = summary + "Mean RPS:    " + fmt.Sprintf("%.2f", reqStatSummary.avgRPS*1000000000) + " req/sec\n"

	summary = summary + "\nQuery Statistics\n"
	summary = summary + "Mean query:     " + fmt.Sprintf("%d", reqStatSummary.avgDuration/1000000) + " ms\n"
	summary = summary + "Fastest query:  " + fmt.Sprintf("%d", reqStatSummary.minDuration/1000000) + " ms\n"
	summary = summary + "Slowest query:  " + fmt.Sprintf("%d", reqStatSummary.maxDuration/1000000) + " ms\n"

	summary = summary + "Total Data Transferred: " + fmt.Sprintf("%d", reqStatSummary.totalDataTransferred) + " bytes\n"
	summary = summary + "Average Data Transferred:  " + fmt.Sprintf("%d", reqStatSummary.avgDataTransferred) + " bytes\n"

	summary = summary + "\nResponse Codes\n"
	//sort the status codes
	var codes []int
	for key := range reqStatSummary.statusCodes {
		codes = append(codes, key)
	}
	sort.Ints(codes)
	for _, code := range codes {
		if code == 0 {
			continue
		}
		summary = summary + fmt.Sprintf("%d", code) + ": " + fmt.Sprintf("%d", reqStatSummary.statusCodes[code]) + " responses\n"
	}
	if reqStatSummary.statusCodes[0] > 0 {
		summary = summary + "Failed: " + fmt.Sprintf("%d", reqStatSummary.statusCodes[0]) + " requests\n"
	}
	return summary
}
