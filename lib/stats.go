package pewpew

import (
	"fmt"
	"time"
)

type requestStatSummary struct {
	avgQPS      float64 //per nanoseconds
	avgDuration time.Duration
	maxDuration time.Duration
	minDuration time.Duration
	statusCodes map[int]int //counts of each code
	startTime   time.Time   //start of first request
	endTime     time.Time   //end of last request
}

//create statistical summary of all requests
func createRequestsStats(requestStats []requestStat) requestStatSummary {
	if len(requestStats) == 0 {
		return requestStatSummary{}
	}

	requestCodes := make(map[int]int)
	summary := requestStatSummary{maxDuration: requestStats[0].Duration,
		minDuration: requestStats[0].Duration,
		statusCodes: requestCodes,
		startTime:   requestStats[0].StartTime,
		endTime:     requestStats[0].EndTime,
	}
	var totalDurations time.Duration //total time of all requests (concurrent is counted)
	for i := 0; i < len(requestStats); i++ {
		if requestStats[i].Duration > summary.maxDuration {
			summary.maxDuration = requestStats[i].Duration
		}
		if requestStats[i].Duration < summary.minDuration {
			summary.minDuration = requestStats[i].Duration
		}
		if requestStats[i].StartTime.Before(summary.startTime) {
			summary.startTime = requestStats[i].StartTime
		}
		if requestStats[i].EndTime.After(summary.endTime) {
			summary.endTime = requestStats[i].EndTime
		}

		totalDurations += requestStats[i].Duration
		summary.statusCodes[requestStats[i].StatusCode]++
	}
	//kinda ugly to calculate average, then convert into nanoseconds
	avgNs := totalDurations.Nanoseconds() / int64(len(requestStats))
	newAvg, _ := time.ParseDuration(fmt.Sprintf("%d", avgNs) + "ns")
	summary.avgDuration = newAvg

	summary.avgQPS = float64(len(requestStats)) / float64(summary.endTime.Sub(summary.startTime))
	return summary
}
