package commands

import (
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/fetcher"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fetchJvnCmd = &cobra.Command{
	Use:   "fetchjvn",
	Short: "Fetch CPE from JVN",
	Long:  "Fetch CPE from JVN",
	RunE:  fetchJvn,
}

func init() {
	RootCmd.AddCommand(fetchJvnCmd)

	fetchJvnCmd.PersistentFlags().Bool("stdout", false, "display all CPEs to stdout")
	_ = viper.BindPFlag("stdout", fetchJvnCmd.PersistentFlags().Lookup("stdout"))
}

func fetchJvn(cmd *cobra.Command, args []string) (err error) {
	log15.Info("Initialize Database")
	driver, locked, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"))
	if err != nil {
		if locked {
			log15.Error("Failed to initialize DB. Close DB connection before fetching", "err", err)
		}
		return err
	}

	cpes, err := fetcher.FetchJVN()
	if err != nil {
		log15.Error("Failed to fetch.", "err", err)
		return err
	}
	log15.Info("Fetched", "Number of CPEs", len(cpes))

	if !viper.GetBool("stdout") {
		if err = driver.InsertCpes(models.JVNType, cpes); err != nil {
			log15.Error("Failed to insert.", "err", err)
			return fmt.Errorf("Failed to insert cpes. err : %s", err)
		}
		log15.Info(fmt.Sprintf("Inserted %d CPEs", len(cpes)))
	} else {
		for _, cpe := range cpes {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s%t\n",
				cpe.CpeURI,
				cpe.CpeFS,
				cpe.Part,
				cpe.Vendor,
				cpe.Product,
				cpe.Version,
				cpe.Update,
				cpe.Edition,
				cpe.Language,
				cpe.SoftwareEdition,
				cpe.TargetSoftware,
				cpe.TargetHardware,
				cpe.Other,
				cpe.Deprecated,
			)
		}
	}

	return nil
}
