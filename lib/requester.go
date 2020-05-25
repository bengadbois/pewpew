package pewpew

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

func runRequest(req http.Request, client *http.Client) (response *http.Response, stat RequestStat) {
	reqStartTime := time.Now()

	// get size of request
	reqDump, _ := httputil.DumpRequestOut(&req, false)
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // reset due to read
	}
	totalSizeSentBytes := len(reqDump) + len(reqBody)

	response, responseErr := (*client).Do(&req)
	reqEndTime := time.Now()

	if responseErr != nil {
		stat = RequestStat{
			Proto:           req.Proto,
			URL:             req.URL.String(),
			Method:          req.Method,
			StartTime:       reqStartTime,
			EndTime:         reqEndTime,
			Duration:        reqEndTime.Sub(reqStartTime),
			StatusCode:      0,
			Error:           responseErr,
			DataTransferred: 0,
		}
		return
	}

	// get size of response
	respDump, _ := httputil.DumpResponse(response, false)
	respBody, _ := ioutil.ReadAll(response.Body)
	totalSizeReceivedBytes := len(respDump) + len(respBody)

	stat = RequestStat{
		Proto:           response.Proto,
		URL:             req.URL.String(),
		Method:          req.Method,
		StartTime:       reqStartTime,
		EndTime:         reqEndTime,
		Duration:        reqEndTime.Sub(reqStartTime),
		StatusCode:      response.StatusCode,
		Error:           responseErr,
		DataTransferred: totalSizeSentBytes + totalSizeReceivedBytes,
	}
	return
}

// createRequestQueue creates a channel of http.Requests of size count
func createRequestQueue(count int, target Target) (chan http.Request, error) {
	requestQueue := make(chan http.Request)
	//attempt to build one request - if passes, the rest should too
	_, err := buildRequest(target)
	if err != nil {
		return nil, errors.New("failed to create request with target configuration: " + err.Error())
	}
	go func() {
		for i := 0; i < count; i++ {
			req, err := buildRequest(target)
			if err != nil {
				//this shouldn't happen, but probably should handle for it
				continue
			}
			requestQueue <- req
		}
		close(requestQueue)
	}()
	return requestQueue, nil
}
