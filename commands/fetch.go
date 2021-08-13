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

	fetchCmd.PersistentFlags().Bool("stdout", false, "display all CPEs to stdout")
	_ = viper.BindPFlag("stdout", fetchCmd.PersistentFlags().Lookup("stdout"))

	fetchCmd.PersistentFlags().Int("wait", 0, "Interval between fetch (seconds)")
	_ = viper.BindPFlag("wait", fetchCmd.PersistentFlags().Lookup("wait"))

	fetchCmd.PersistentFlags().Int("threads", runtime.NumCPU(), "The number of threads to be used")
	_ = viper.BindPFlag("threads", fetchCmd.PersistentFlags().Lookup("threads"))

	fetchCmd.PersistentFlags().Uint("expire", 0, "timeout to set for Redis keys in seconds. If set to 0, the key is persistent.")
	_ = viper.BindPFlag("expire", fetchCmd.PersistentFlags().Lookup("expire"))
}
