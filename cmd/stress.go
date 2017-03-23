package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	pewpew "github.com/bengadbois/pewpew/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var stressCmd = &cobra.Command{
	Use:   "stress URL...",
	Short: "Run stress tests",
	RunE: func(cmd *cobra.Command, args []string) error {

		stressCfg := pewpew.StressConfig{}
		err := viper.Unmarshal(&stressCfg)
		if err != nil {
			fmt.Println(err)
			return errors.New("could not parse config file")
		}

		//global configs
		stressCfg.NoHTTP2 = viper.GetBool("noHTTP2")
		stressCfg.EnforceSSL = viper.GetBool("enforceSSL")
		stressCfg.Quiet = viper.GetBool("quiet")
		stressCfg.Verbose = viper.GetBool("verbose")

		//URLs are handled differently that other config options
		//command line specifying URLs take higher precedence than config URLs

		//check either set via config or command line
		if len(stressCfg.Targets) == 0 && len(args) < 1 {
			return errors.New("requires URL")
		}

		//if URLs are set on command line, use that for Targets instead of config
		if len(args) >= 1 {
			stressCfg.Targets = make([]pewpew.Target, len(args))
			for i := range stressCfg.Targets {
				stressCfg.Targets[i].URL = args[i]
				//use global configs instead of the config file's individual target settings
				stressCfg.Targets[i].RegexURL = viper.GetBool("regex")
				stressCfg.Targets[i].Count = viper.GetInt("count")
				stressCfg.Targets[i].Concurrency = viper.GetInt("concurrency")
				stressCfg.Targets[i].Timeout = viper.GetString("timeout")
				stressCfg.Targets[i].Method = viper.GetString("method")
				stressCfg.Targets[i].Body = viper.GetString("body")
				stressCfg.Targets[i].BodyFilename = viper.GetString("bodyFile")
				stressCfg.Targets[i].Headers = viper.GetString("headers")
				stressCfg.Targets[i].Cookies = viper.GetString("cookies")
				stressCfg.Targets[i].UserAgent = viper.GetString("userAgent")
				stressCfg.Targets[i].BasicAuth = viper.GetString("basicAuth")
				stressCfg.Targets[i].Compress = viper.GetBool("compress")
				stressCfg.Targets[i].KeepAlive = viper.GetBool("keepalive")
				stressCfg.Targets[i].FollowRedirects = viper.GetBool("followredirects")
			}
		} else {
			//set non-URL target settings
			//walk through viper.Get() because that will show which were
			//explictly set instead of guessing at zero-valued defaults
			for i, target := range viper.Get("targets").([]interface{}) {
				targetMapVals := target.(map[string]interface{})
				if _, set := targetMapVals["RegexURL"]; !set {
					stressCfg.Targets[i].RegexURL = viper.GetBool("regex")
				}
				if _, set := targetMapVals["Count"]; !set {
					stressCfg.Targets[i].Count = viper.GetInt("count")
				}
				if _, set := targetMapVals["Concurrency"]; !set {
					stressCfg.Targets[i].Concurrency = viper.GetInt("concurrency")
				}
				if _, set := targetMapVals["Timeout"]; !set {
					stressCfg.Targets[i].Timeout = viper.GetString("timeout")
				}
				if _, set := targetMapVals["Method"]; !set {
					stressCfg.Targets[i].Method = viper.GetString("method")
				}
				if _, set := targetMapVals["Body"]; !set {
					stressCfg.Targets[i].Body = viper.GetString("body")
				}
				if _, set := targetMapVals["BodyFilename"]; !set {
					stressCfg.Targets[i].BodyFilename = viper.GetString("bodyFile")
				}
				if _, set := targetMapVals["Headers"]; !set {
					stressCfg.Targets[i].Headers = viper.GetString("headers")
				}
				if _, set := targetMapVals["Cookies"]; !set {
					stressCfg.Targets[i].Cookies = viper.GetString("cookies")
				}
				if _, set := targetMapVals["UserAgent"]; !set {
					stressCfg.Targets[i].UserAgent = viper.GetString("userAgent")
				}
				if _, set := targetMapVals["BasicAuth"]; !set {
					stressCfg.Targets[i].BasicAuth = viper.GetString("basicAuth")
				}
				if _, set := targetMapVals["Compress"]; !set {
					stressCfg.Targets[i].Compress = viper.GetBool("compress")
				}
				if _, set := targetMapVals["KeepAlive"]; !set {
					stressCfg.Targets[i].KeepAlive = viper.GetBool("keepalive")
				}
				if _, set := targetMapVals["FollowRedirects"]; !set {
					stressCfg.Targets[i].FollowRedirects = viper.GetBool("followredirects")
				}
			}
		}

		targetRequestStats, err := pewpew.RunStress(stressCfg, os.Stdout)
		if err != nil {
			return err
		}

		fmt.Print("\n----Summary----\n\n")

		//only print individual target data if multiple targets
		if len(stressCfg.Targets) > 1 {
			for idx, target := range stressCfg.Targets {
				//info about the request
				fmt.Printf("----Target %d: %s %s\n", idx+1, target.Method, target.URL)
				reqStats := pewpew.CreateRequestsStats(targetRequestStats[idx])
				fmt.Println(pewpew.CreateTextSummary(reqStats))
			}
		}

		//combine individual targets to a total one
		globalStats := []pewpew.RequestStat{}
		for i := range stressCfg.Targets {
			for j := range targetRequestStats[i] {
				globalStats = append(globalStats, targetRequestStats[i][j])
			}
		}
		if len(stressCfg.Targets) > 1 {
			fmt.Println("----Global----")
		}
		reqStats := pewpew.CreateRequestsStats(globalStats)
		fmt.Println(pewpew.CreateTextSummary(reqStats))

		//write out json
		if viper.GetString("ResultFilenameJSON") != "" {
			fmt.Print("Writing full result data to: " + viper.GetString("ResultFilenameJSON") + " ...")
			json, _ := json.MarshalIndent(globalStats, "", "    ")
			err = ioutil.WriteFile(viper.GetString("ResultFilenameJSON"), json, 0644)
			if err != nil {
				return errors.New("failed to write full result data to " +
					viper.GetString("ResultFilenameJSON") + ": " + err.Error())
			}
			fmt.Println("finished!")
		}
		//write out csv
		if viper.GetString("ResultFilenameCSV") != "" {
			fmt.Print("Writing full result data to: " + viper.GetString("ResultFilenameCSV") + " ...")
			file, err := os.Create(viper.GetString("ResultFilenameCSV"))
			if err != nil {
				return errors.New("failed to write full result data to " +
					viper.GetString("ResultFilenameCSV") + ": " + err.Error())
			}
			defer file.Close()

			writer := csv.NewWriter(file)

			for _, req := range globalStats {
				line := []string{
					req.StartTime.String(),
					fmt.Sprintf("%d", req.Duration),
					fmt.Sprintf("%d", req.StatusCode),
					fmt.Sprintf("%d bytes", req.DataTransferred),
				}
				err := writer.Write(line)
				if err != nil {
					return errors.New("failed to write full result data to " +
						viper.GetString("ResultFilenameCSV") + ": " + err.Error())
				}
			}
			defer writer.Flush()
			fmt.Println("finished!")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(stressCmd)
	stressCmd.Flags().BoolP("regex", "r", false, "Interpret URLs as regular expressions.")
	viper.BindPFlag("regex", stressCmd.Flags().Lookup("regex"))

	stressCmd.Flags().IntP("num", "n", 10, "Number of total requests to make.")
	viper.BindPFlag("count", stressCmd.Flags().Lookup("num"))

	stressCmd.Flags().IntP("concurrent", "c", 1, "Number of concurrent requests to make.")
	viper.BindPFlag("concurrency", stressCmd.Flags().Lookup("concurrent"))

	stressCmd.Flags().StringP("timeout", "t", "10s", "Maximum seconds to wait for response")
	viper.BindPFlag("timeout", stressCmd.Flags().Lookup("timeout"))

	stressCmd.Flags().StringP("request-method", "X", "GET", "Request type. GET, HEAD, POST, PUT, etc.")
	viper.BindPFlag("method", stressCmd.Flags().Lookup("request-method"))

	stressCmd.Flags().String("body", "", "String to use as request body e.g. POST body.")
	viper.BindPFlag("body", stressCmd.Flags().Lookup("body"))

	stressCmd.Flags().String("body-file", "", "Path to file to use as request body. Will overwrite --body if both are present.")
	viper.BindPFlag("bodyFile", stressCmd.Flags().Lookup("body-file"))

	stressCmd.Flags().StringP("headers", "H", "", "Add arbitrary header line, eg. 'Accept-Encoding:gzip, Content-Type:application/json'")
	viper.BindPFlag("headers", stressCmd.Flags().Lookup("headers"))

	stressCmd.Flags().String("cookies", "", "Add request cookies, eg. 'data=123; session=456'")
	viper.BindPFlag("cookies", stressCmd.Flags().Lookup("cookies"))

	stressCmd.Flags().StringP("user-agent", "A", "pewpew", "Add User-Agent header. Can also be done with the arbitrary header flag.")
	viper.BindPFlag("userAgent", stressCmd.Flags().Lookup("user-agent"))

	stressCmd.Flags().String("basic-auth", "", "Add HTTP basic authentication, eg. 'user123:password456'.")
	viper.BindPFlag("basicAuth", stressCmd.Flags().Lookup("basic-auth"))

	stressCmd.Flags().BoolP("compress", "C", true, "Add 'Accept-Encoding: gzip' header if Accept-Encoding is not already present.")
	viper.BindPFlag("compress", stressCmd.Flags().Lookup("compress"))

	stressCmd.Flags().BoolP("keepalive", "k", true, "Enable HTTP KeepAlive.")
	viper.BindPFlag("keepalive", stressCmd.Flags().Lookup("keepalive"))

	stressCmd.Flags().Bool("follow-redirects", true, "Follow HTTP redirects.")
	viper.BindPFlag("followredirects", stressCmd.Flags().Lookup("follow-redirects"))

	stressCmd.Flags().Bool("no-http2", false, "Disable HTTP2.")
	viper.BindPFlag("noHTTP2", stressCmd.Flags().Lookup("no-http2"))

	stressCmd.Flags().Bool("ignore-ssl", false, "Enfore SSL certificate/hostname correctness.")
	viper.BindPFlag("enforceSSL", stressCmd.Flags().Lookup("ignore-ssl"))

	stressCmd.Flags().String("output-json", "", "Path to file to write full data as JSON")
	viper.BindPFlag("ResultFilenameJSON", stressCmd.Flags().Lookup("output-json"))

	stressCmd.Flags().String("output-csv", "", "Path to file to write full data as CSV")
	viper.BindPFlag("ResultFilenameCSV", stressCmd.Flags().Lookup("output-csv"))

	stressCmd.Flags().BoolP("quiet", "q", false, "Do not print while requests are running.")
	viper.BindPFlag("quiet", stressCmd.Flags().Lookup("quiet"))

	stressCmd.Flags().BoolP("verbose", "v", false, "Print extra info for debugging.")
	viper.BindPFlag("verbose", stressCmd.Flags().Lookup("verbose"))

}
