package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pewpew",
	Short: "HTTP(S) & HTTP2 load tester for performance and stress testing",
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().IntVar(&cpuFlag, "cpu", runtime.GOMAXPROCS(0), "Number of CPUs to use.")
	RootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Print extra troubleshooting info.")
}

var cpuFlag int
var verboseFlag bool
