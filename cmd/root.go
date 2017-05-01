package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pewpew",
	Short: "HTTP(S) & HTTP2 load tester for performance and stress testing",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		runtime.GOMAXPROCS(viper.GetInt("cpu"))
	},
}

// Execute runs the RootCmd and terminates the program if there is an error
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		if _, ok := err.(viper.ConfigParseError); ok {
			fmt.Println("Failed to parse config file " + viper.ConfigFileUsed())
			fmt.Println(err)
			os.Exit(-1)
		}
		fmt.Println("No config file found")
	}

	RootCmd.PersistentFlags().BoolP("regex", "r", false, "Interpret URLs as regular expressions.")
	RootCmd.PersistentFlags().StringP("timeout", "t", "10s", "Maximum seconds to wait for response")
	RootCmd.PersistentFlags().StringP("request-method", "X", "GET", "Request type. GET, HEAD, POST, PUT, etc.")
	RootCmd.PersistentFlags().String("body", "", "String to use as request body e.g. POST body.")
	RootCmd.PersistentFlags().String("body-file", "", "Path to file to use as request body. Will overwrite --body if both are present.")
	RootCmd.PersistentFlags().StringP("headers", "H", "", "Add arbitrary header line, eg. 'Accept-Encoding:gzip, Content-Type:application/json'")
	RootCmd.PersistentFlags().String("cookies", "", "Add request cookies, eg. 'data=123; session=456'")
	RootCmd.PersistentFlags().StringP("user-agent", "A", "pewpew", "Add User-Agent header. Can also be done with the arbitrary header flag.")
	RootCmd.PersistentFlags().String("basic-auth", "", "Add HTTP basic authentication, eg. 'user123:password456'.")
	RootCmd.PersistentFlags().BoolP("compress", "C", true, "Add 'Accept-Encoding: gzip' header if Accept-Encoding is not already present.")
	RootCmd.PersistentFlags().BoolP("keepalive", "k", true, "Enable HTTP KeepAlive.")
	RootCmd.PersistentFlags().Bool("follow-redirects", true, "Follow HTTP redirects.")
	RootCmd.PersistentFlags().Bool("no-http2", false, "Disable HTTP2.")
	RootCmd.PersistentFlags().Bool("enforce-ssl", false, "Enfore SSL certificate correctness.")
	RootCmd.PersistentFlags().String("output-json", "", "Path to file to write full data as JSON")
	RootCmd.PersistentFlags().String("output-csv", "", "Path to file to write full data as CSV")
	RootCmd.PersistentFlags().BoolP("quiet", "q", false, "Do not print while requests are running.")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Print extra troubleshooting info.")
	RootCmd.PersistentFlags().Int("cpu", runtime.GOMAXPROCS(0), "Number of CPUs to use.")

	viper.BindPFlags(RootCmd.PersistentFlags())

}
