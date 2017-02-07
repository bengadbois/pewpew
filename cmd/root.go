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

	RootCmd.PersistentFlags().Int("cpu", runtime.GOMAXPROCS(0), "Number of CPUs to use.")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Print extra troubleshooting info.")
	viper.BindPFlags(RootCmd.PersistentFlags())
}
