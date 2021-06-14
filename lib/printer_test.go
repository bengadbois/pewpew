package pewpew

import (
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestCreateTextSummary(t *testing.T) {
	tests := []struct {
		name string
		s    RequestStatSummary
	}{
		{
			name: "empty summary",
			s:    RequestStatSummary{},
		},
		{
			name: "valid summary, non-zero values",
			s: RequestStatSummary{
				avgRPS:               12.34,
				avgDuration:          1234,
				minDuration:          1234,
				maxDuration:          1234,
				statusCodes:          map[int]int{100: 1, 200: 2, 300: 3, 400: 4, 500: 5, 0: 1},
				startTime:            time.Now(),
				endTime:              time.Now(),
				avgDataTransferred:   2345,
				maxDataTransferred:   12345,
				minDataTransferred:   1234,
				totalDataTransferred: 123456,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_ = CreateTextSummary(tc.s)
		})
	}
}

func TestPrintStat(t *testing.T) {
	tests := []struct {
		name string
		r    RequestStat
	}{
		{
			name: "empty stat",
			r:    RequestStat{},
		},
		{
			name: "status code 100",
			r:    RequestStat{StatusCode: 100},
		},
		{
			name: "status code 200",
			r:    RequestStat{StatusCode: 200},
		},
		{
			name: "status code 300",
			r:    RequestStat{StatusCode: 300},
		},
		{
			name: "status code 400",
			r:    RequestStat{StatusCode: 400},
		},
		{
			name: "status code 500",
			r:    RequestStat{StatusCode: 500},
		},
		{
			name: "error",
			r:    RequestStat{Error: errors.New("this is an error")},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := printer{output: ioutil.Discard}
			p.printStat(tc.r)

		})
	}
}

func TestPrintVerbose(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		resp *http.Response
	}{
		{
			name: "nil request and response",
			req:  nil,
			resp: nil,
		},
		{
			name: "nil request and empty response",
			req:  nil,
			resp: &http.Response{},
		},
		{
			name: "empty request and nil response",
			req:  &http.Request{},
			resp: nil,
		},
		{
			name: "non-empty request and response",
			req:  &http.Request{},
			resp: &http.Response{Body: http.NoBody},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := printer{output: ioutil.Discard}
			p.printVerbose(tc.req, tc.resp)
		})
	}
}
