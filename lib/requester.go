package pewpew

import (
	"bytes"
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
