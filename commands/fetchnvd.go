package commands

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/inconshreveable/log15"
	c "github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/nvd"
	"github.com/kotakanbe/go-cpe-dictionary/util"
)

// FetchNvdCmd : FetchNvdCmd
type FetchNvdCmd struct {
	logToFile bool
	logDir    string
	logJSON   bool
}

// Name return subcommand name
func (*FetchNvdCmd) Name() string { return "fetchnvd" }

// Synopsis return synopsis
func (*FetchNvdCmd) Synopsis() string { return "Fetch CPE from NVD" }

// Usage return usage
func (*FetchNvdCmd) Usage() string {
	return `fetchnvd:
	fetchnvd
		[-dbtype=mysql|postgres|sqlite3|redis]
		[-dbpath=$PWD/cve.sqlite3 or connection string]
		[-http-proxy=http://192.168.0.1:8080]
		[-debug]
		[-debug-sql]
		[-log-to-file]
		[-log-dir=/path/to/log]
		[-log-json]

For the first time, run the blow command to fetch data. (It takes about 10 minutes)
   $ go-cpe-dictionary fetchnvd
`
}

// SetFlags set flag
func (p *FetchNvdCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.Conf.Debug, "debug", false, "debug mode")
	f.BoolVar(&c.Conf.DebugSQL, "debug-sql", false, "SQL debug mode")

	defaultLogDir := util.GetDefaultLogDir()
	f.StringVar(&p.logDir, "log-dir", defaultLogDir, "/path/to/log")
	f.BoolVar(&p.logJSON, "log-json", false, "output log as JSON")
	f.BoolVar(&p.logToFile, "log-to-file", false, "output log to file")

	pwd := os.Getenv("PWD")
	f.StringVar(&c.Conf.DBPath, "dbpath", pwd+"/cpe.sqlite3",
		"/path/to/sqlite3 or SQL connection string")

	f.StringVar(&c.Conf.DBType, "dbtype", "sqlite3",
		"Database type to store data in (sqlite3, mysql, postgres or redis supported)")

	f.StringVar(&c.Conf.HTTPProxy, "http-proxy", "", "http://proxy-url:port (default: empty)")
}

// Execute execute
func (p *FetchNvdCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	util.SetLogger(p.logDir, c.Conf.Debug, p.logJSON, p.logToFile)
	if !c.Conf.Validate() {
		return subcommands.ExitUsageError
	}

	var driver db.DB
	var err error
	if driver, err = db.NewDB(c.Conf.DBType, c.Conf.DBPath, c.Conf.DebugSQL); err != nil {
		log15.Error("Failed to new db.", "err", err)
		return subcommands.ExitFailure
	}
	defer func() {
		_ = driver.CloseDB()
	}()

	log15.Info("Fetch and insert from NVD...")
	if err = nvd.FetchAndInsertCPE(driver); err != nil {
		log15.Crit("Failed to fetch.", "err", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
