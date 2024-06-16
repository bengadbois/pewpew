package pewpew

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"

	humanize "github.com/dustin/go-humanize"
	color "github.com/fatih/color"
)

type printer struct {
	//writeLock prevents concurrent messages from being interlaced
	writeLock sync.Mutex

	//output is where the printer writes to
	output io.Writer
}

//CreateTextSummary creates a human friendly summary of entire test
func CreateTextSummary(reqStatSummary RequestStatSummary) string {
	summary := "\n"

	summary += "Timing\n"
	summary += fmt.Sprintf("Mean query speed:     %d ms\n", reqStatSummary.avgDuration/1000000)
	summary += fmt.Sprintf("Fastest query speed:  %d ms\n", reqStatSummary.minDuration/1000000)
	summary += fmt.Sprintf("Slowest query speed:  %d ms\n", reqStatSummary.maxDuration/1000000)
	summary += fmt.Sprintf("Mean RPS:             %.2f req/sec\n", reqStatSummary.avgRPS*1000000000)
	summary += fmt.Sprintf("Total time:           %d ms\n", reqStatSummary.endTime.Sub(reqStatSummary.startTime).Nanoseconds()/1000000)

	summary += "\nData Transferred\n"
	summary += fmt.Sprintf("Mean query:      %s\n", humanize.Bytes(uint64(reqStatSummary.avgDataTransferred)))
	summary += fmt.Sprintf("Largest query:   %s\n", humanize.Bytes(uint64(reqStatSummary.maxDataTransferred)))
	summary += fmt.Sprintf("Smallest query:  %s\n", humanize.Bytes(uint64(reqStatSummary.minDataTransferred)))
	summary += fmt.Sprintf("Total:           %s\n", humanize.Bytes(uint64(reqStatSummary.totalDataTransferred)))

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

	summary += fmt.Sprintf("\nErrors\nFailed requests: %d\n", reqStatSummary.errorCount)
	return summary
}

//print colored single line stats per RequestStat
func (p *printer) printStat(stat RequestStat) {
	p.writeLock.Lock()
	defer p.writeLock.Unlock()

	if stat.Error != nil {
		color.Set(color.FgRed)
		fmt.Fprintln(p.output, "Failed to make request: "+stat.Error.Error())
		color.Unset()
		return
	}

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
	fmt.Fprintf(p.output, "%s %d\t%s \t%d ms\t-> %s %s\n",
		stat.Proto,
		stat.StatusCode,
		humanize.Bytes(uint64(stat.DataTransferred)),
		stat.Duration.Nanoseconds()/1000000,
		stat.Method,
		stat.URL)
	color.Unset()
}

//print tons of info about the request, response and response body
func (p *printer) printVerbose(req *http.Request, response *http.Response) {
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
	p.writeLock.Lock()
	fmt.Fprintln(p.output, requestInfo)
	p.writeLock.Unlock()
}

//writeString is a generic output string printer
func (p *printer) writeString(s string) {
	p.writeLock.Lock()
	fmt.Fprint(p.output, s)
	p.writeLock.Unlock()
}
