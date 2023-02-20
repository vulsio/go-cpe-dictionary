package commands

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch the data of CPE",
	Long:  `Fetch the data of CPE`,
}

func init() {
	RootCmd.AddCommand(fetchCmd)

	fetchCmd.PersistentFlags().Int("threads", runtime.NumCPU(), "The number of threads to be used")
	_ = viper.BindPFlag("threads", fetchCmd.PersistentFlags().Lookup("threads"))

	fetchCmd.PersistentFlags().Int("batch-size", 100, "The number of batch size to insert.")
	_ = viper.BindPFlag("batch-size", fetchCmd.PersistentFlags().Lookup("batch-size"))
}
