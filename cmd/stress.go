package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"net/http"
)

//flags
var (
	numTests int
)

func init() {
	RootCmd.AddCommand(stressCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stressCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stressCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	stressCmd.Flags().IntVarP(&numTests, "num", "n", 100, "number of requests to make")
}

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress [http[s]://]hostname[:port]/path",
	Short: "Run predefined load of requests",
	Long:  `Run number of requests`,
	RunE:  RunStress,
}

func RunStress(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("needs URL")
	}
	fmt.Println("running stress")
	for i := 0; i < numTests; i++ {
		response, err := http.Get(args[0])
		if err != nil {
			fmt.Errorf(err.Error())
		}
		fmt.Printf("%+v", response)
	}
	return nil
}
