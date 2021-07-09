package commands

import (
	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start CPE dictionary HTTP server",
	Long:  `Start CPE dictionary HTTP server`,
	RunE:  executeServer,
}

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().String("bind", "127.0.0.1", "HTTP server bind to IP address (default: loop back interface")
	_ = viper.BindPFlag("bind", serverCmd.PersistentFlags().Lookup("bind"))

	serverCmd.PersistentFlags().String("port", "1328", "HTTP server port number (default: 1328")
	_ = viper.BindPFlag("port", serverCmd.PersistentFlags().Lookup("port"))
}

func executeServer(cmd *cobra.Command, args []string) (err error) {
	logDir := viper.GetString("log-dir")
	driver, locked, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"))
	if err != nil {
		if locked {
			log15.Error("Failed to initialize DB. Close DB connection before fetching", "err", err)
		}
		return err
	}

	log15.Info("Starting HTTP Server...")
	if err = server.Start(logDir, driver); err != nil {
		log15.Error("Failed to start server.", "err", err)
		return err
	}

	return nil
}
