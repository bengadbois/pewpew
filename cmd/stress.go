package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	pewpew "github.com/bengadbois/pewpew/lib"
	"github.com/spf13/cobra"
)

var stressCmd = &cobra.Command{
	Use:   "stress URL...",
	Short: "Run stress tests",
	RunE: func(cmd *cobra.Command, args []string) error {

		if len(args) < 1 {
			return errors.New("requires URL")
		}

		stressCfg := pewpew.NewStressConfig()
		stressCfg.ResultFilenameJSON = resultFileJSONFlag
		stressCfg.ResultFilenameCSV = resultFileCSVFlag
		stressCfg.Quiet = quietFlag
		stressCfg.Verbose = verboseFlag

		//per target config
		stressCfg.Targets = make([]pewpew.Target, len(args))
		for i := range stressCfg.Targets {
			parsedURL, err := url.Parse(args[i])
			if err != nil {
				return errors.New("cannot parse url " + args[i])
			}
			stressCfg.Targets[i].URL = *parsedURL
			stressCfg.Targets[i].Count = numFlag
			stressCfg.Targets[i].Concurrency = concurrentFlag
			stressCfg.Targets[i].Timeout = (time.Duration(timeoutFlag*1000) * time.Millisecond) //preserve float's decimal
			stressCfg.Targets[i].ReqMethod = requestMethodFlag
			stressCfg.Targets[i].ReqBody = bodyFlag
			stressCfg.Targets[i].ReqBodyFilename = bodyFileFlag
			stressCfg.Targets[i].ReqHeaders = headerFlag.Header
			stressCfg.Targets[i].UserAgent = userAgentFlag
			if basicAuthFlag != "" {
				key, val, err := parseKeyValString(basicAuthFlag, ":")
				if err != nil {
					return errors.New("failed to parse basic auth")
				}
				stressCfg.Targets[i].BasicAuth = pewpew.BasicAuth{User: key, Password: val}
			}
			stressCfg.Targets[i].IgnoreSSL = ignoreSSLFlag
			stressCfg.Targets[i].Compress = compressFlag
			stressCfg.Targets[i].NoHTTP2 = ignoreSSLFlag
		}

		err := pewpew.RunStress(*stressCfg)
		return err
	},
}

func init() {
	headerFlag = headers{http.Header{}}
	RootCmd.AddCommand(stressCmd)
	stressCmd.Flags().IntVarP(&numFlag, "num", "n", 10, "Number of total requests to make.")
	stressCmd.Flags().IntVarP(&concurrentFlag, "concurrent", "c", 1, "Number of concurrent requests to make.")
	stressCmd.Flags().Float64VarP(&timeoutFlag, "timeout", "t", 10, "Maximum seconds to wait for response")
	stressCmd.Flags().StringVarP(&requestMethodFlag, "request-method", "X", "GET", "Request type. GET, HEAD, POST, PUT, etc.")
	stressCmd.Flags().StringVar(&bodyFlag, "body", "", "String to use as request body e.g. POST body.")
	stressCmd.Flags().StringVar(&bodyFileFlag, "body-file", "", "Path to file to use as request body. Will overwrite --body if both are present.")
	stressCmd.Flags().VarP(&headerFlag, "header", "H", "Add arbitrary header line, eg. 'Accept-Encoding:gzip'. Repeatable.")
	stressCmd.Flags().StringVarP(&userAgentFlag, "user-agent", "A", "pewpew", "Add User-Agent header. Can also be done with the arbitrary header flag.")
	stressCmd.Flags().StringVar(&basicAuthFlag, "basic-auth", "", "Add HTTP basic authentication, eg. 'user123:password456'.")

	stressCmd.Flags().BoolVar(&ignoreSSLFlag, "ignore-ssl", true, "Ignore SSL certificate/hostname issues.")
	stressCmd.Flags().BoolVarP(&compressFlag, "compress", "C", true, "Add 'Accept-Encoding: gzip' header if Accept-Encoding is not already present.")
	stressCmd.Flags().BoolVar(&noHTTP2Flag, "no-http2", false, "Disable HTTP2.")
	stressCmd.Flags().StringVar(&resultFileJSONFlag, "output-json", "", "Path to file to write full data as JSON")
	stressCmd.Flags().StringVar(&resultFileCSVFlag, "output-csv", "", "Path to file to write full data as CSV")
	stressCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Do not print while requests are running.")
}

var numFlag int
var concurrentFlag int
var timeoutFlag float64
var requestMethodFlag string
var bodyFlag string
var bodyFileFlag string
var headerFlag headers
var userAgentFlag string
var basicAuthFlag string
var ignoreSSLFlag bool
var compressFlag bool
var noHTTP2Flag bool
var resultFileJSONFlag string
var resultFileCSVFlag string
var quietFlag bool

// custom implementation of repeated header flag parsing

type headers struct{ http.Header }

func (h *headers) String() string {
	buf := &bytes.Buffer{}
	if err := h.Write(buf); err != nil {
		return ""
	}
	return buf.String()
}

func (h *headers) Set(headerString string) error {
	key, val, err := parseKeyValString(headerString, ":")
	if err != nil {
		return fmt.Errorf("invalid header %s: %s", headerString, err.Error())
	}
	h.Add(key, val)
	return nil
}

//required by pflag.Value interface
func (h *headers) Type() string {
	return "headers"
}

//splits on delim into parts, trims whitespace,
//like "key:val", "key: val", "key : val ", etc.
func parseKeyValString(keyValStr, delim string) (string, string, error) {
	parts := strings.SplitN(keyValStr, delim, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("failed to parse into two parts")
	}
	key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if key == "" || val == "" {
		return "", "", fmt.Errorf("key or value is empty")
	}
	return key, val, nil
}
