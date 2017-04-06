package pewpew

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"

	color "github.com/fatih/color"
)

//CreateTextSummary creates a human friendly summary of entire stress test
func CreateTextSummary(reqStatSummary RequestStatSummary) string {
	summary := "\n"

	summary += "Timing\n"
	summary += "Mean query speed:     " + fmt.Sprintf("%d", reqStatSummary.avgDuration/1000000) + " ms\n"
	summary += "Fastest query speed:  " + fmt.Sprintf("%d", reqStatSummary.minDuration/1000000) + " ms\n"
	summary += "Slowest query speed:  " + fmt.Sprintf("%d", reqStatSummary.maxDuration/1000000) + " ms\n"
	summary += "Mean RPS:             " + fmt.Sprintf("%.2f", reqStatSummary.avgRPS*1000000000) + " req/sec\n"
	summary += "Total time:           " + fmt.Sprintf("%d", reqStatSummary.endTime.Sub(reqStatSummary.startTime).Nanoseconds()/1000000) + " ms\n"

	summary += "\nData Transferred\n"
	summary += "Mean query:      " + fmt.Sprintf("%d", reqStatSummary.avgDataTransferred) + " bytes\n"
	summary += "Largest query:   " + fmt.Sprintf("%d", reqStatSummary.maxDataTransferred) + " bytes\n"
	summary += "Smallest query:  " + fmt.Sprintf("%d", reqStatSummary.minDataTransferred) + " bytes\n"
	summary += "Total:           " + fmt.Sprintf("%d", reqStatSummary.totalDataTransferred) + " bytes\n"

	summary = summary + "\nResponse Codes\n"
	//sort the status codes
	var codes []int
	totalResponses := 0
	for key, val := range reqStatSummary.statusCodes {
		codes = append(codes, key)
		totalResponses += val
	}
	sort.Ints(codes)
	for _, code := range codes {
		if code == 0 {
			summary += "Failed"
		} else {
			summary += fmt.Sprintf("%d", code)
		}
		summary += ": " + fmt.Sprintf("%d", reqStatSummary.statusCodes[code])
		if code == 0 {
			summary += " requests"
		} else {
			summary += " responses"
		}
		summary += " (" + fmt.Sprintf("%.2f", 100*float64(reqStatSummary.statusCodes[code])/float64(totalResponses)) + "%)\n"
	}
	return summary
}

//print colored single line stats per RequestStat
func printStat(stat RequestStat, w io.Writer) {
	if stat.Error != nil {
		color.Set(color.FgRed)
		fmt.Fprintln(w, "Failed to make request: "+stat.Error.Error())
		color.Unset()
	} else {
		if stat.StatusCode >= 100 && stat.StatusCode < 200 {
			color.Set(color.FgBlue)
		} else if stat.StatusCode >= 200 && stat.StatusCode < 300 {
			color.Set(color.FgGreen)
		} else if stat.StatusCode >= 300 && stat.StatusCode < 400 {
			color.Set(color.FgCyan)
		} else if stat.StatusCode >= 400 && stat.StatusCode < 500 {
			color.Set(color.FgMagenta)
		} else {
			color.Set(color.FgRed)
		}
		fmt.Fprintf(w, "%s %d\t%d bytes\t%d ms\t-> %s %s\n",
			stat.Proto,
			stat.StatusCode,
			stat.DataTransferred,
			stat.Duration.Nanoseconds()/1000000,
			stat.Method,
			stat.URL)
		color.Unset()
	}
}

//print tons of info about the request, response and response body
func printVerbose(req *http.Request, response *http.Response, w io.Writer) {
	if req == nil {
		return
	}
	if response == nil {
		return
	}
	var requestInfo string
	//request details
	requestInfo = requestInfo + fmt.Sprintf("Request:\n%+v\n\n", &req)

	//reponse metadata
	requestInfo = requestInfo + fmt.Sprintf("Response:\n%+v\n\n", response)

	//reponse body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		requestInfo = requestInfo + fmt.Sprintf("Failed to read response body: %s\n", err.Error())
	} else {
		requestInfo = requestInfo + fmt.Sprintf("Body:\n%s\n\n", body)
		response.Body.Close()
	}
	fmt.Fprintln(w, requestInfo)
}
