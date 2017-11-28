package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildTime = ""
var version = ""

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if version == "" {
			fmt.Println("pewpew not official build")
		} else {
			fmt.Println("pewpew " + version)
		}
		if buildTime != "" {
			fmt.Println("Built at " + buildTime)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
