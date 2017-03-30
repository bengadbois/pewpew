package pewpew

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCreateRequestsStats(t *testing.T) {
	cases := []struct {
		requestStats []RequestStat
		want         requestStatSummary
	}{
		{requestStats: make([]RequestStat, 0), want: requestStatSummary{}},
		//check basic
		{requestStats: []RequestStat{
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, StatusCode: 200},
		},
			want: requestStatSummary{
				avgRPS:      0.000000000001,
				avgDuration: 1000,
				maxDuration: 1000,
				minDuration: 1000,
				startTime:   time.Unix(1000, 0),
				endTime:     time.Unix(2000, 0),
				statusCodes: map[int]int{200: 1},
			},
		},
		//check multiple
		{requestStats: []RequestStat{
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, StatusCode: 200},
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, StatusCode: 200},
		},
			want: requestStatSummary{
				avgRPS:      0.000000000002,
				avgDuration: 1000,
				maxDuration: 1000,
				minDuration: 1000,
				startTime:   time.Unix(1000, 0),
				endTime:     time.Unix(2000, 0),
				statusCodes: map[int]int{200: 2},
			},
		},
		//checking errors
		{requestStats: []RequestStat{
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, Error: errors.New("test error 1")},
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, Error: errors.New("test error 1")},
		},
			want: requestStatSummary{
				avgRPS:      0,
				avgDuration: 0,
				maxDuration: 0,
				minDuration: 0,
				startTime:   time.Unix(1000, 0),
				endTime:     time.Unix(2000, 0),
				statusCodes: map[int]int{},
			},
		},
		//mix of timings, mix of data transferred, mix of status codes
		{requestStats: []RequestStat{
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, Error: errors.New("test error 1")},
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, StatusCode: 200, DataTransferred: 100},
			{StartTime: time.Unix(2000, 0), EndTime: time.Unix(3000, 0), Duration: 1000, StatusCode: 200, DataTransferred: 200},
			{StartTime: time.Unix(3000, 0), EndTime: time.Unix(4000, 0), Duration: 1000, StatusCode: 400, DataTransferred: 300},
			{StartTime: time.Unix(4000, 0), EndTime: time.Unix(6000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 400},
			{StartTime: time.Unix(5000, 0), EndTime: time.Unix(7000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 500},
			{StartTime: time.Unix(6000, 0), EndTime: time.Unix(7000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 600},
		},
			want: requestStatSummary{
				avgRPS:               0.000000000001,
				avgDuration:          1500,
				maxDuration:          2000,
				minDuration:          1000,
				startTime:            time.Unix(1000, 0),
				endTime:              time.Unix(7000, 0),
				statusCodes:          map[int]int{200: 2, 400: 4},
				avgDataTransferred:   350,
				maxDataTransferred:   600,
				minDataTransferred:   100,
				totalDataTransferred: 2100,
			},
		},
		//like test case from above, but differently ordered
		{requestStats: []RequestStat{
			{StartTime: time.Unix(6000, 0), EndTime: time.Unix(7000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 600},
			{StartTime: time.Unix(5000, 0), EndTime: time.Unix(7000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 500},
			{StartTime: time.Unix(4000, 0), EndTime: time.Unix(6000, 0), Duration: 2000, StatusCode: 400, DataTransferred: 400},
			{StartTime: time.Unix(3000, 0), EndTime: time.Unix(4000, 0), Duration: 1000, StatusCode: 400, DataTransferred: 300},
			{StartTime: time.Unix(2000, 0), EndTime: time.Unix(3000, 0), Duration: 1000, StatusCode: 200, DataTransferred: 200},
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, StatusCode: 200, DataTransferred: 100},
			{StartTime: time.Unix(1000, 0), EndTime: time.Unix(2000, 0), Duration: 1000, Error: errors.New("test error 1")},
		},
			want: requestStatSummary{
				avgRPS:               0.000000000001,
				avgDuration:          1500,
				maxDuration:          2000,
				minDuration:          1000,
				startTime:            time.Unix(1000, 0),
				endTime:              time.Unix(7000, 0),
				statusCodes:          map[int]int{200: 2, 400: 4},
				avgDataTransferred:   350,
				maxDataTransferred:   600,
				minDataTransferred:   100,
				totalDataTransferred: 2100,
			},
		},
	}
	for _, c := range cases {
		summary := CreateRequestsStats(c.requestStats)
		if !reflect.DeepEqual(summary, c.want) {
			t.Errorf("CreateRequestsStats(%+v) == %+v wanted %+v", c.requestStats, summary, c.want)
		}
	}
}
