package commands

import (
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulsio/go-cpe-dictionary/db"
	"github.com/vulsio/go-cpe-dictionary/fetcher"
	"github.com/vulsio/go-cpe-dictionary/models"
	"github.com/vulsio/go-cpe-dictionary/util"
	"golang.org/x/xerrors"
)

var fetchNvdCmd = &cobra.Command{
	Use:   "nvd",
	Short: "Fetch CPE from NVD",
	Long:  "Fetch CPE from NVD",
	RunE:  fetchNvd,
}

func init() {
	fetchCmd.AddCommand(fetchNvdCmd)
}

func fetchNvd(cmd *cobra.Command, args []string) (err error) {
	if err := util.SetLogger(viper.GetBool("log-to-file"), viper.GetString("log-dir"), viper.GetBool("debug"), viper.GetBool("log-json")); err != nil {
		return xerrors.Errorf("Failed to SetLogger. err: %w", err)
	}

	driver, locked, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"))
	if err != nil {
		if locked {
			return xerrors.Errorf("Failed to initialize DB. Close DB connection before fetching. err: %w", err)
		}
		return err
	}

	fetchMeta, err := driver.GetFetchMeta()
	if err != nil {
		return xerrors.Errorf("Failed to get FetchMeta from DB. err: %w", err)
	}
	if fetchMeta.OutDated() {
		return xerrors.Errorf("Failed to Insert CVEs into DB. SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
	}

	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	cpes, err := fetcher.FetchNVD()
	if err != nil {
		return xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	log15.Info("Fetched", "Number of CPEs", len(cpes))

	if !viper.GetBool("stdout") {
		if err = driver.InsertCpes(models.NVD, cpes); err != nil {
			return xerrors.Errorf("Failed to insert cpes. err: %w", err)
		}
		log15.Info(fmt.Sprintf("Inserted %d CPEs", len(cpes)))
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
