package commands

import (
	"fmt"
	"runtime"

	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/fetcher"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
)

var fetchNvdCmd = &cobra.Command{
	Use:   "fetchnvd",
	Short: "Fetch CPE from NVD",
	Long:  "Fetch CPE from NVD",
	RunE:  fetchNvd,
}

func init() {
	RootCmd.AddCommand(fetchNvdCmd)

	fetchNvdCmd.PersistentFlags().Bool("stdout", false, "display all CPEs to stdout")
	_ = viper.BindPFlag("stdout", fetchNvdCmd.PersistentFlags().Lookup("stdout"))

	fetchNvdCmd.PersistentFlags().Int("wait", 0, "Interval between fetch (seconds)")
	_ = viper.BindPFlag("wait", fetchNvdCmd.PersistentFlags().Lookup("wait"))

	fetchNvdCmd.PersistentFlags().Int("threads", runtime.NumCPU(), "The number of threads to be used")
	_ = viper.BindPFlag("threads", fetchNvdCmd.PersistentFlags().Lookup("threads"))
}

func fetchNvd(cmd *cobra.Command, args []string) (err error) {
	log15.Info("Initialize Database")
	driver, locked, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"))
	if err != nil {
		if locked {
			log15.Error("Failed to initialize DB. Close DB connection before fetching", "err", err)
		}
		return err
	}

	fetchMeta, err := driver.GetFetchMeta()
	if err != nil {
		log15.Error("Failed to get FetchMeta from DB.", "err", err)
		return err
	}
	if fetchMeta.OutDated() {
		log15.Error("Failed to Insert CVEs into DB. SchemaVersion is old", "SchemaVersion", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
		return xerrors.New("Failed to Insert CVEs into DB. SchemaVersion is old")
	}

	cpes, err := fetcher.FetchNVD()
	if err != nil {
		log15.Error("Failed to fetch.", "err", err)
		return err
	}
	log15.Info("Fetched", "Number of CPEs", len(cpes))

	if !viper.GetBool("stdout") {
		if err = driver.InsertCpes(models.NVD, cpes); err != nil {
			log15.Error("Failed to insert.", "err", err)
			return fmt.Errorf("Failed to insert cpes. err : %s", err)
		}
		log15.Info(fmt.Sprintf("Inserted %d CPEs", len(cpes)))

		if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
			log15.Error("Failed to upsert FetchMeta to DB.", "err", err)
			return err
		}
	} else {
		for _, cpe := range cpes {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%t\n",
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
