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
		stressCfg.Count = viper.GetInt("count")
		stressCfg.Concurrency = viper.GetInt("concurrency")

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
				stressCfg.Targets[i].Options.DNSPrefetch = viper.GetBool("dns-prefetch")
				stressCfg.Targets[i].Options.Timeout = viper.GetString("timeout")
				stressCfg.Targets[i].Options.Method = viper.GetString("request-method")
				stressCfg.Targets[i].Options.Body = viper.GetString("body")
				stressCfg.Targets[i].Options.RegexBody = viper.GetBool("body-regex")
				stressCfg.Targets[i].Options.BodyFilename = viper.GetString("body-file")
				stressCfg.Targets[i].Options.Headers = viper.GetString("headers")
				stressCfg.Targets[i].Options.Cookies = viper.GetString("cookies")
				stressCfg.Targets[i].Options.UserAgent = viper.GetString("user-agent")
				stressCfg.Targets[i].Options.BasicAuth = viper.GetString("basic-auth")
				stressCfg.Targets[i].Options.Compress = viper.GetBool("compress")
				stressCfg.Targets[i].Options.KeepAlive = viper.GetBool("keepalive")
				stressCfg.Targets[i].Options.FollowRedirects = viper.GetBool("follow-redirects")
				stressCfg.Targets[i].Options.NoHTTP2 = viper.GetBool("no-http2")
				stressCfg.Targets[i].Options.EnforceSSL = viper.GetBool("enforce-ssl")
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
				if _, set := targetMapVals["DNSPrefetch"]; !set {
					stressCfg.Targets[i].Options.DNSPrefetch = viper.GetBool("dns-prefetch")
				}
				if _, set := targetMapVals["Timeout"]; !set {
					stressCfg.Targets[i].Options.Timeout = viper.GetString("timeout")
				}
				if _, set := targetMapVals["Method"]; !set {
					stressCfg.Targets[i].Options.Method = viper.GetString("request-method")
				}
				if _, set := targetMapVals["Body"]; !set {
					stressCfg.Targets[i].Options.Body = viper.GetString("body")
				}
				if _, set := targetMapVals["RegexBody"]; !set {
					stressCfg.Targets[i].Options.RegexBody = viper.GetBool("body-regex")
				}
				if _, set := targetMapVals["BodyFilename"]; !set {
					stressCfg.Targets[i].Options.BodyFilename = viper.GetString("bodyFile")
				}
				if _, set := targetMapVals["Headers"]; !set {
					stressCfg.Targets[i].Options.Headers = viper.GetString("headers")
				}
				if _, set := targetMapVals["Cookies"]; !set {
					stressCfg.Targets[i].Options.Cookies = viper.GetString("cookies")
				}
				if _, set := targetMapVals["UserAgent"]; !set {
					stressCfg.Targets[i].Options.UserAgent = viper.GetString("userAgent")
				}
				if _, set := targetMapVals["BasicAuth"]; !set {
					stressCfg.Targets[i].Options.BasicAuth = viper.GetString("basicAuth")
				}
				if _, set := targetMapVals["Compress"]; !set {
					stressCfg.Targets[i].Options.Compress = viper.GetBool("compress")
				}
				if _, set := targetMapVals["KeepAlive"]; !set {
					stressCfg.Targets[i].Options.KeepAlive = viper.GetBool("keepalive")
				}
				if _, set := targetMapVals["FollowRedirects"]; !set {
					stressCfg.Targets[i].Options.FollowRedirects = viper.GetBool("followredirects")
				}
				if _, set := targetMapVals["NoHTTP2"]; !set {
					stressCfg.Targets[i].Options.NoHTTP2 = viper.GetBool("no-http2")
				}
				if _, set := targetMapVals["EnforceSSL"]; !set {
					stressCfg.Targets[i].Options.EnforceSSL = viper.GetBool("enforce-ssl")
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
				fmt.Printf("----Target %d: %s %s\n", idx+1, target.Options.Method, target.URL)
				reqStats := pewpew.CreateRequestsStats(targetRequestStats[idx])
				fmt.Println(pewpew.CreateTextSummary(reqStats))
			}
		}

		//combine individual targets to a total one
		globalStats := []pewpew.RequestStat{}
		for i := range stressCfg.Targets {
			globalStats = append(globalStats, targetRequestStats[i]...)
		}
		if len(stressCfg.Targets) > 1 {
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
				return fmt.Errorf("failed to write full result data to %s: %w", filename, err)
			}
			fmt.Println("finished!")
		}
		//write out csv
		if viper.GetString("output-csv") != "" {
			filename := viper.GetString("output-csv")
			fmt.Print("Writing full result data to: " + filename + " ...")
			file, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("failed to write full result data to %s: %w", filename, err)
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
					return fmt.Errorf("failed to write full result data to %s: %w", filename, err)
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
				return fmt.Errorf("failed to write full result data to %s: %w", filename, err)
			}
			fmt.Println("finished!")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(stressCmd)
	stressCmd.Flags().IntP("num", "n", pewpew.DefaultCount, "Number of total requests to make.")
	err := viper.BindPFlag("count", stressCmd.Flags().Lookup("num"))
	if err != nil {
		fmt.Println("failed to configure flags")
		fmt.Println(err)
		os.Exit(-1)
	}

	stressCmd.Flags().IntP("concurrent", "c", pewpew.DefaultConcurrency, "Number of concurrent requests to make.")
	err = viper.BindPFlag("concurrency", stressCmd.Flags().Lookup("concurrent"))
	if err != nil {
		fmt.Println("failed to configure flags")
		fmt.Println(err)
		os.Exit(-1)
	}
}
