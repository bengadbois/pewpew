package cmd

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	pewpew "github.com/bengadbois/pewpew/lib"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var benchmarkCmd = &cobra.Command{
	Use:     "benchmark URL...",
	Aliases: []string{"bench"},
	Short:   "Run benchmark tests",
	RunE: func(cmd *cobra.Command, args []string) error {

		benchmarkCfg := pewpew.BenchmarkConfig{}
		err := viper.Unmarshal(&benchmarkCfg)
		if err != nil {
			fmt.Println(err)
			return errors.New("could not parse config file")
		}

		//global configs
		benchmarkCfg.Quiet = viper.GetBool("quiet")
		benchmarkCfg.Verbose = viper.GetBool("verbose")
		benchmarkCfg.RPS = viper.GetInt("rps")
		benchmarkCfg.Duration = viper.GetInt("duration")

		//URLs are handled differently that other config options
		//command line specifying URLs take higher precedence than config URLs

		//check either set via config or command line
		if len(benchmarkCfg.Targets) == 0 && len(args) < 1 {
			return errors.New("requires URL")
		}

		//if URLs are set on command line, use that for Targets instead of config
		if len(args) >= 1 {
			benchmarkCfg.Targets = make([]pewpew.Target, len(args))
			for i := range benchmarkCfg.Targets {
				benchmarkCfg.Targets[i].URL = args[i]
				//use global configs instead of the config file's individual target settings
				benchmarkCfg.Targets[i].RegexURL = viper.GetBool("regex")
				benchmarkCfg.Targets[i].Options.DNSPrefetch = viper.GetBool("dns-prefetch")
				benchmarkCfg.Targets[i].Options.Timeout = viper.GetString("timeout")
				benchmarkCfg.Targets[i].Options.Method = viper.GetString("request-method")
				benchmarkCfg.Targets[i].Options.Body = viper.GetString("body")
				benchmarkCfg.Targets[i].Options.BodyFilename = viper.GetString("body-file")
				benchmarkCfg.Targets[i].Options.Headers = viper.GetString("headers")
				benchmarkCfg.Targets[i].Options.Cookies = viper.GetString("cookies")
				benchmarkCfg.Targets[i].Options.UserAgent = viper.GetString("user-agent")
				benchmarkCfg.Targets[i].Options.BasicAuth = viper.GetString("basic-auth")
				benchmarkCfg.Targets[i].Options.Compress = viper.GetBool("compress")
				benchmarkCfg.Targets[i].Options.KeepAlive = viper.GetBool("keepalive")
				benchmarkCfg.Targets[i].Options.FollowRedirects = viper.GetBool("follow-redirects")
				benchmarkCfg.Targets[i].Options.NoHTTP2 = viper.GetBool("no-http2")
				benchmarkCfg.Targets[i].Options.EnforceSSL = viper.GetBool("enforce-ssl")
			}
		} else {
			//set non-URL target settings
			//walk through viper.Get() because that will show which were
			//explictly set instead of guessing at zero-valued defaults
			for i, target := range viper.Get("targets").([]interface{}) {
				targetMapVals := target.(map[string]interface{})

				if _, set := targetMapVals["RegexURL"]; !set {
					benchmarkCfg.Targets[i].RegexURL = viper.GetBool("regex")
				}
				if _, set := targetMapVals["DNSPrefetch"]; !set {
					benchmarkCfg.Targets[i].Options.DNSPrefetch = viper.GetBool("dns-prefetch")
				}
				if _, set := targetMapVals["Timeout"]; !set {
					benchmarkCfg.Targets[i].Options.Timeout = viper.GetString("timeout")
				}
				if _, set := targetMapVals["Method"]; !set {
					benchmarkCfg.Targets[i].Options.Method = viper.GetString("request-method")
				}
				if _, set := targetMapVals["Body"]; !set {
					benchmarkCfg.Targets[i].Options.Body = viper.GetString("body")
				}
				if _, set := targetMapVals["BodyFilename"]; !set {
					benchmarkCfg.Targets[i].Options.BodyFilename = viper.GetString("bodyFile")
				}
				if _, set := targetMapVals["Headers"]; !set {
					benchmarkCfg.Targets[i].Options.Headers = viper.GetString("headers")
				}
				if _, set := targetMapVals["Cookies"]; !set {
					benchmarkCfg.Targets[i].Options.Cookies = viper.GetString("cookies")
				}
				if _, set := targetMapVals["UserAgent"]; !set {
					benchmarkCfg.Targets[i].Options.UserAgent = viper.GetString("userAgent")
				}
				if _, set := targetMapVals["BasicAuth"]; !set {
					benchmarkCfg.Targets[i].Options.BasicAuth = viper.GetString("basicAuth")
				}
				if _, set := targetMapVals["Compress"]; !set {
					benchmarkCfg.Targets[i].Options.Compress = viper.GetBool("compress")
				}
				if _, set := targetMapVals["KeepAlive"]; !set {
					benchmarkCfg.Targets[i].Options.KeepAlive = viper.GetBool("keepalive")
				}
				if _, set := targetMapVals["FollowRedirects"]; !set {
					benchmarkCfg.Targets[i].Options.FollowRedirects = viper.GetBool("followredirects")
				}
				if _, set := targetMapVals["NoHTTP2"]; !set {
					benchmarkCfg.Targets[i].Options.NoHTTP2 = viper.GetBool("no-http2")
				}
				if _, set := targetMapVals["EnforceSSL"]; !set {
					benchmarkCfg.Targets[i].Options.EnforceSSL = viper.GetBool("enforce-ssl")
				}
			}
		}

		targetRequestStats, err := pewpew.RunBenchmark(benchmarkCfg, os.Stdout)
		if err != nil {
			return err
		}

		fmt.Print("\n----Summary----\n\n")

		//only print individual target data if multiple targets
		if len(benchmarkCfg.Targets) > 1 {
			for idx, target := range benchmarkCfg.Targets {
				//info about the request
				fmt.Printf("----Target %d: %s %s\n", idx+1, target.Options.Method, target.URL)
				reqStats := pewpew.CreateRequestsStats(targetRequestStats[idx])
				fmt.Println(pewpew.CreateTextSummary(reqStats))
			}
		}

		//combine individual targets to a total one
		globalStats := []pewpew.RequestStat{}
		for i := range benchmarkCfg.Targets {
			for j := range targetRequestStats[i] {
				globalStats = append(globalStats, targetRequestStats[i][j])
			}
		}
		if len(benchmarkCfg.Targets) > 1 {
			fmt.Println("----Global----")
		}
		reqStats := pewpew.CreateRequestsStats(globalStats)
		fmt.Println(pewpew.CreateTextSummary(reqStats))

		if viper.GetString("output-json") != "" {
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
		if viper.GetString("output-csv") != "" {
			filename := viper.GetString("output-csv")
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
					humanize.Bytes(uint64(req.DataTransferred)),
				}
				err := writer.Write(line)
				if err != nil {
					return errors.New("failed to write full result data to " +
						filename + ": " + err.Error())
				}
			}
			defer writer.Flush()
			fmt.Println("finished!")
		}
		//write out xml
		if viper.GetString("output-xml") != "" {
			filename := viper.GetString("output-xml")
			fmt.Print("Writing full result data to: " + filename + " ...")
			xml, _ := xml.MarshalIndent(globalStats, "", "    ")
			err = ioutil.WriteFile(viper.GetString("output-xml"), xml, 0644)
			if err != nil {
				return errors.New("failed to write full result data to " +
					filename + ": " + err.Error())
			}
			fmt.Println("finished!")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.Flags().Int("rps", pewpew.DefaultRPS, "Requests per second to make.")
	err := viper.BindPFlag("rps", benchmarkCmd.Flags().Lookup("rps"))
	if err != nil {
		fmt.Println("failed to configure flags")
		fmt.Println(err)
		os.Exit(-1)
	}

	benchmarkCmd.Flags().IntP("duration", "d", pewpew.DefaultConcurrency, "Number of seconds to send requests. Total benchmark test duration will be longer due to waiting for requests to finish.")
	err = viper.BindPFlag("duration", benchmarkCmd.Flags().Lookup("duration"))
	if err != nil {
		fmt.Println("failed to configure flags")
		fmt.Println(err)
		os.Exit(-1)
	}
}
