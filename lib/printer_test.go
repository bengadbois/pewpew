package pewpew

import (
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestCreateTextSummary(t *testing.T) {
	cases := []struct {
		s RequestStatSummary
	}{
		{RequestStatSummary{}}, //empty
		{RequestStatSummary{
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
		}}, //nonzero values for everything
	}
	for _, c := range cases {
		//could check for the exact string, but that's super tedious and brittle
		_ = CreateTextStressSummary(c.s)
	}
}

func TestPrintStat(t *testing.T) {
	cases := []struct {
		r RequestStat
	}{
		{RequestStat{}}, //empty
		//status codes
		{RequestStat{StatusCode: 100}},
		{RequestStat{StatusCode: 200}},
		{RequestStat{StatusCode: 300}},
		{RequestStat{StatusCode: 400}},
		{RequestStat{StatusCode: 500}},
		//error case
		{RequestStat{Error: errors.New("this is an error")}},
	}
	p := printer{output: ioutil.Discard}
	for _, c := range cases {
		p.printStat(c.r)
	}
}

func TestPrintVerbose(t *testing.T) {
	cases := []struct {
		req  *http.Request
		resp *http.Response
	}{
		{nil, nil},
		{nil, &http.Response{}},
		{&http.Request{}, nil},
		{&http.Request{}, &http.Response{Body: http.NoBody}},
	}
	p := printer{output: ioutil.Discard}
	for _, c := range cases {
		p.printVerbose(c.req, c.resp)
	}
}
