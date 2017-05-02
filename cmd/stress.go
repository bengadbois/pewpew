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
		stressCfg.Quiet = viper.GetBool("quiet")
		stressCfg.Verbose = viper.GetBool("verbose")

		//URLs are handled differently that other config options
		//command line specifying URLs take higher precedence than config URLs

		//check either set via config or command line
		if len(stressCfg.StressTargets) == 0 && len(args) < 1 {
			return errors.New("requires URL")
		}

		//if URLs are set on command line, use that for Targets instead of config
		if len(args) >= 1 {
			stressCfg.StressTargets = make([]pewpew.StressTarget, len(args))
			for i := range stressCfg.StressTargets {
				stressCfg.StressTargets[i].Count = viper.GetInt("count")
				stressCfg.StressTargets[i].Concurrency = viper.GetInt("concurrency")
				stressCfg.StressTargets[i].Target.URL = args[i]
				//use global configs instead of the config file's individual target settings
				stressCfg.StressTargets[i].Target.RegexURL = viper.GetBool("regex")
				stressCfg.StressTargets[i].Target.Timeout = viper.GetString("timeout")
				stressCfg.StressTargets[i].Target.Method = viper.GetString("request-method")
				stressCfg.StressTargets[i].Target.Body = viper.GetString("body")
				stressCfg.StressTargets[i].Target.BodyFilename = viper.GetString("body-file")
				stressCfg.StressTargets[i].Target.Headers = viper.GetString("headers")
				stressCfg.StressTargets[i].Target.Cookies = viper.GetString("cookies")
				stressCfg.StressTargets[i].Target.UserAgent = viper.GetString("user-agent")
				stressCfg.StressTargets[i].Target.BasicAuth = viper.GetString("basic-auth")
				stressCfg.StressTargets[i].Target.Compress = viper.GetBool("compress")
				stressCfg.StressTargets[i].Target.KeepAlive = viper.GetBool("keepalive")
				stressCfg.StressTargets[i].Target.FollowRedirects = viper.GetBool("follow-redirects")
				stressCfg.StressTargets[i].Target.NoHTTP2 = viper.GetBool("no-http2")
				stressCfg.StressTargets[i].Target.EnforceSSL = viper.GetBool("enforce-ssl")
			}
		} else {
			//set non-URL target settings
			//walk through viper.Get() because that will show which were
			//explictly set instead of guessing at zero-valued defaults
			for i, target := range viper.Get("targets").([]interface{}) {
				targetMapVals := target.(map[string]interface{})
				if _, set := targetMapVals["Count"]; !set {
					stressCfg.StressTargets[i].Count = viper.GetInt("count")
				}
				if _, set := targetMapVals["Concurrency"]; !set {
					stressCfg.StressTargets[i].Concurrency = viper.GetInt("concurrency")
				}
				if _, set := targetMapVals["RegexURL"]; !set {
					stressCfg.StressTargets[i].Target.RegexURL = viper.GetBool("regex")
				}
				if _, set := targetMapVals["Timeout"]; !set {
					stressCfg.StressTargets[i].Target.Timeout = viper.GetString("timeout")
				}
				if _, set := targetMapVals["Method"]; !set {
					stressCfg.StressTargets[i].Target.Method = viper.GetString("method")
				}
				if _, set := targetMapVals["Body"]; !set {
					stressCfg.StressTargets[i].Target.Body = viper.GetString("body")
				}
				if _, set := targetMapVals["BodyFilename"]; !set {
					stressCfg.StressTargets[i].Target.BodyFilename = viper.GetString("bodyFile")
				}
				if _, set := targetMapVals["Headers"]; !set {
					stressCfg.StressTargets[i].Target.Headers = viper.GetString("headers")
				}
				if _, set := targetMapVals["Cookies"]; !set {
					stressCfg.StressTargets[i].Target.Cookies = viper.GetString("cookies")
				}
				if _, set := targetMapVals["UserAgent"]; !set {
					stressCfg.StressTargets[i].Target.UserAgent = viper.GetString("userAgent")
				}
				if _, set := targetMapVals["BasicAuth"]; !set {
					stressCfg.StressTargets[i].Target.BasicAuth = viper.GetString("basicAuth")
				}
				if _, set := targetMapVals["Compress"]; !set {
					stressCfg.StressTargets[i].Target.Compress = viper.GetBool("compress")
				}
				if _, set := targetMapVals["KeepAlive"]; !set {
					stressCfg.StressTargets[i].Target.KeepAlive = viper.GetBool("keepalive")
				}
				if _, set := targetMapVals["FollowRedirects"]; !set {
					stressCfg.StressTargets[i].Target.FollowRedirects = viper.GetBool("followredirects")
				}
				if _, set := targetMapVals["NoHTTP2"]; !set {
					stressCfg.StressTargets[i].Target.NoHTTP2 = viper.GetBool("no-http2")
				}
				if _, set := targetMapVals["EnforceSSL"]; !set {
					stressCfg.StressTargets[i].Target.EnforceSSL = viper.GetBool("enforce-ssl")
				}
			}
		}

		targetRequestStats, err := pewpew.RunStress(stressCfg, os.Stdout)
		if err != nil {
			return err
		}

		fmt.Print("\n----Summary----\n\n")

		//only print individual target data if multiple targets
		if len(stressCfg.StressTargets) > 1 {
			for idx, stressTarget := range stressCfg.StressTargets {
				//info about the request
				fmt.Printf("----Target %d: %s %s\n", idx+1, stressTarget.Target.Method, stressTarget.Target.URL)
				reqStats := pewpew.CreateRequestsStats(targetRequestStats[idx])
				fmt.Println(pewpew.CreateTextSummary(reqStats))
			}
		}

		//combine individual targets to a total one
		globalStats := []pewpew.RequestStat{}
		for i := range stressCfg.StressTargets {
			for j := range targetRequestStats[i] {
				globalStats = append(globalStats, targetRequestStats[i][j])
			}
		}
		if len(stressCfg.StressTargets) > 1 {
			fmt.Println("----Global----")
		}
		reqStats := pewpew.CreateRequestsStats(globalStats)
		fmt.Println(pewpew.CreateTextSummary(reqStats))

		//write out json
		if viper.GetString("ResultFilenameJSON") != "" {
			filename := viper.GetString("output-json")
			fmt.Print("Writing full result data to: " + filename + " ...")
			json, _ := json.MarshalIndent(globalStats, "", "    ")
			err = ioutil.WriteFile(filename, json, 0644)
			if err != nil {
				return errors.New("failed to write full result data to " +
					filename + ": " + err.Error())
			}
			fmt.Println("finished!")
		}
		//write out csv
		if viper.GetString("ResultFilenameCSV") != "" {
			filename := viper.GetString("ResultFilenameCSV")
			fmt.Print("Writing full result data to: " + filename + " ...")
			file, err := os.Create(filename)
			if err != nil {
				return errors.New("failed to write full result data to " +
					filename + ": " + err.Error())
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
	stressCmd.Flags().IntP("num", "n", pewpew.DefaultCount, "Number of total requests to make.")
	viper.BindPFlag("count", stressCmd.Flags().Lookup("num"))

	stressCmd.Flags().IntP("concurrent", "c", pewpew.DefaultConcurrency, "Number of concurrent requests to make.")
	viper.BindPFlag("concurrency", stressCmd.Flags().Lookup("concurrent"))
}
