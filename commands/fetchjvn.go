package commands

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/db"
	"github.com/vulsio/go-cpe-dictionary/fetcher"
	"github.com/vulsio/go-cpe-dictionary/models"
	"github.com/vulsio/go-cpe-dictionary/util"
)

var fetchJvnCmd = &cobra.Command{
	Use:   "jvn",
	Short: "Fetch CPE from JVN",
	Long:  "Fetch CPE from JVN",
	RunE:  fetchJvn,
}

func init() {
	fetchCmd.AddCommand(fetchJvnCmd)

	fetchJvnCmd.PersistentFlags().Int("wait", 0, "Interval between fetch (seconds)")
	_ = viper.BindPFlag("wait", fetchJvnCmd.PersistentFlags().Lookup("wait"))
}

func fetchJvn(_ *cobra.Command, _ []string) (err error) {
	if err := util.SetLogger(viper.GetBool("log-to-file"), viper.GetString("log-dir"), viper.GetBool("debug"), viper.GetBool("log-json")); err != nil {
		return xerrors.Errorf("Failed to SetLogger. err: %w", err)
	}

	driver, locked, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"), db.Option{})
	if err != nil {
		if locked {
			return xerrors.Errorf("Failed to initialize DB. Close DB connection before fetching. err: %w", err)
		}
		return xerrors.Errorf("Failed to open DB. err: %w", err)
	}

	fetchMeta, err := driver.GetFetchMeta()
	if err != nil {
		return xerrors.Errorf("Failed to get FetchMeta from DB. err: %w", err)
	}
	if fetchMeta.OutDated() {
		return xerrors.Errorf("Failed to Insert CVEs into DB. err: SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
	}
	// If the fetch fails the first time (without SchemaVersion), the DB needs to be cleaned every time, so insert SchemaVersion.
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	cpes, err := fetcher.FetchJVN()
	if err != nil {
		return xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	if err := driver.InsertCpes(models.JVN, cpes); err != nil {
		return xerrors.Errorf("Failed to insert cpes. err: %w", err)
	}

	fetchMeta.LastFetchedAt = time.Now()
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. dbpath: %s, err: %w", viper.GetString("dbpath"), err)
	}

	return nil
}
