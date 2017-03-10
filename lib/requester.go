package pewpew

import (
	"net/http"
	"net/http/httputil"
	"time"
)

func runRequest(req http.Request, client *http.Client) (response *http.Response, stat RequestStat) {
	reqStartTime := time.Now()
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

	//get size of request
	reqDump, _ := httputil.DumpRequestOut(&req, true)
	respDump, _ := httputil.DumpResponse(response, true)
	totalSizeSentBytes := len(reqDump)
	totalSizeReceivedBytes := len(respDump)
	totalSizeBytes := totalSizeSentBytes + totalSizeReceivedBytes

	stat = RequestStat{
		Proto:           response.Proto,
		URL:             req.URL.String(),
		Method:          req.Method,
		StartTime:       reqStartTime,
		EndTime:         reqEndTime,
		Duration:        reqEndTime.Sub(reqStartTime),
		StatusCode:      response.StatusCode,
		Error:           responseErr,
		DataTransferred: totalSizeBytes,
	}
	return
}
