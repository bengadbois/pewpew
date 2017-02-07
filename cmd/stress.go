package cmd

import (
	"errors"
	"fmt"

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
		stressCfg.ResultFilenameJSON = viper.GetString("outputJSON")
		stressCfg.ResultFilenameCSV = viper.GetString("outputCSV")
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
				stressCfg.Targets[i].Count = viper.GetInt("globalCount")
				stressCfg.Targets[i].Concurrency = viper.GetInt("globalConcurrency")
				stressCfg.Targets[i].Timeout = viper.GetString("globalTimeout")
				stressCfg.Targets[i].Method = viper.GetString("globalMethod")
				stressCfg.Targets[i].Body = viper.GetString("globalBody")
				stressCfg.Targets[i].BodyFilename = viper.GetString("globalBodyFile")
				stressCfg.Targets[i].Headers = viper.GetString("globalHeaders")
				stressCfg.Targets[i].UserAgent = viper.GetString("globalUserAgent")
				stressCfg.Targets[i].BasicAuth = viper.GetString("globalBasicAuth")
				stressCfg.Targets[i].Compress = viper.GetBool("globalCompress")
			}
		} else {
			//set non-URL target settings
			//walk through viper.Get() because that will show which were
			//explictly set instead of guessing at zero-valued defaults
			for i, target := range viper.Get("targets").([]interface{}) {
				fmt.Printf("%+v", viper.Get("targets").([]interface{}))
				targetMapVals := target.(map[string]interface{})
				if _, set := targetMapVals["Count"]; !set {
					stressCfg.Targets[i].Count = viper.GetInt("globalCount")
				}
				if _, set := targetMapVals["Concurrency"]; !set {
					stressCfg.Targets[i].Concurrency = viper.GetInt("globalConcurrency")
				}
				if _, set := targetMapVals["Timeout"]; !set {
					stressCfg.Targets[i].Timeout = viper.GetString("globalTimeout")
				}
				if _, set := targetMapVals["Method"]; !set {
					stressCfg.Targets[i].Method = viper.GetString("globalMethod")
				}
				if _, set := targetMapVals["Body"]; !set {
					stressCfg.Targets[i].Body = viper.GetString("globalBody")
				}
				if _, set := targetMapVals["BodyFilename"]; !set {
					stressCfg.Targets[i].BodyFilename = viper.GetString("globalBodyFile")
				}
				if _, set := targetMapVals["Headers"]; !set {
					stressCfg.Targets[i].Headers = viper.GetString("globalHeaders")
				}
				if _, set := targetMapVals["UserAgent"]; !set {
					stressCfg.Targets[i].UserAgent = viper.GetString("globalUserAgent")
				}
				if _, set := targetMapVals["BasicAuth"]; !set {
					stressCfg.Targets[i].BasicAuth = viper.GetString("globalBasicAuth")
				}
				if _, set := targetMapVals["Compress"]; !set {
					stressCfg.Targets[i].Compress = viper.GetBool("globalCompress")
				}
			}
		}

		err = pewpew.RunStress(stressCfg)
		return err
	},
}

func init() {
	RootCmd.AddCommand(stressCmd)
	stressCmd.Flags().IntP("num", "n", 10, "Number of total requests to make.")
	viper.BindPFlag("globalCount", stressCmd.Flags().Lookup("num"))

	stressCmd.Flags().IntP("concurrent", "c", 1, "Number of concurrent requests to make.")
	viper.BindPFlag("globalConcurrency", stressCmd.Flags().Lookup("concurrent"))

	stressCmd.Flags().StringP("timeout", "t", "10s", "Maximum seconds to wait for response")
	viper.BindPFlag("globalTimeout", stressCmd.Flags().Lookup("timeout"))

	stressCmd.Flags().StringP("request-method", "X", "GET", "Request type. GET, HEAD, POST, PUT, etc.")
	viper.BindPFlag("globalMethod", stressCmd.Flags().Lookup("request-method"))

	stressCmd.Flags().String("body", "", "String to use as request body e.g. POST body.")
	viper.BindPFlag("globalBody", stressCmd.Flags().Lookup("body"))

	stressCmd.Flags().String("body-file", "", "Path to file to use as request body. Will overwrite --body if both are present.")
	viper.BindPFlag("globalBodyFile", stressCmd.Flags().Lookup("body-file"))

	stressCmd.Flags().StringP("headers", "H", "", "Add arbitrary header line, eg. 'Accept-Encoding:gzip, Content-Type:application/json'")
	viper.BindPFlag("globalHeaders", stressCmd.Flags().Lookup("headers"))

	stressCmd.Flags().StringP("user-agent", "A", "pewpew", "Add User-Agent header. Can also be done with the arbitrary header flag.")
	viper.BindPFlag("globalUserAgent", stressCmd.Flags().Lookup("user-agent"))

	stressCmd.Flags().String("basic-auth", "", "Add HTTP basic authentication, eg. 'user123:password456'.")
	viper.BindPFlag("globalBasicAuth", stressCmd.Flags().Lookup("basic-auth"))

	stressCmd.Flags().BoolP("compress", "C", true, "Add 'Accept-Encoding: gzip' header if Accept-Encoding is not already present.")
	viper.BindPFlag("globalCompress", stressCmd.Flags().Lookup("compress"))

	stressCmd.Flags().Bool("no-http2", false, "Disable HTTP2.")
	viper.BindPFlag("noHTTP2", stressCmd.Flags().Lookup("no-http2"))

	stressCmd.Flags().Bool("ignore-ssl", false, "Enfore SSL certificate/hostname correctness.")
	viper.BindPFlag("enforceSSL", stressCmd.Flags().Lookup("ignore-ssl"))

	stressCmd.Flags().String("output-json", "", "Path to file to write full data as JSON")
	viper.BindPFlag("outputJSON", stressCmd.Flags().Lookup("output-json"))

	stressCmd.Flags().String("output-csv", "", "Path to file to write full data as CSV")
	viper.BindPFlag("outputCSV", stressCmd.Flags().Lookup("output-csv"))

	stressCmd.Flags().BoolP("quiet", "q", false, "Do not print while requests are running.")
	viper.BindPFlag("quiet", stressCmd.Flags().Lookup("quiet"))

	stressCmd.Flags().BoolP("verbose", "v", false, "Print extra info for debugging.")
	viper.BindPFlag("verbose", stressCmd.Flags().Lookup("verbose"))

}
