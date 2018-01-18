package commands

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	c "github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/nvd"
	"github.com/kotakanbe/go-cpe-dictionary/util"
	log "github.com/sirupsen/logrus"
)

// FetchNvdCmd : FetchNvdCmd
type FetchNvdCmd struct {
	debug    bool
	debugSQL bool
	quiet    bool
	logDir   string

	dbpath string
	dbtype string

	httpProxy string
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
		[-log-dir=/path/to/log]

For the first time, run the blow command to fetch data. (It takes about 10 minutes)
   $ go-cpe-dictionary fetchnvd
`
}

// SetFlags set flag
func (p *FetchNvdCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.debug, "debug", false, "debug mode")
	f.BoolVar(&p.debugSQL, "debug-sql", false, "SQL debug mode")

	defaultLogDir := util.GetDefaultLogDir()
	f.StringVar(&p.logDir, "log-dir", defaultLogDir, "/path/to/log")

	pwd := os.Getenv("PWD")
	f.StringVar(&p.dbpath, "dbpath", pwd+"/cpe.sqlite3",
		"/path/to/sqlite3 or SQL connection string")

	f.StringVar(&p.dbtype, "dbtype", "sqlite3",
		"Database type to store data in (sqlite3, mysql, postgres or redis supported)")

	f.StringVar(
		&p.httpProxy,
		"http-proxy",
		"",
		"http://proxy-url:port (default: empty)",
	)
}

// Execute execute
func (p *FetchNvdCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	c.Conf.DebugSQL = p.debugSQL
	c.Conf.Debug = p.debug
	c.Conf.DBPath = p.dbpath
	c.Conf.DBType = p.dbtype
	c.Conf.HTTPProxy = p.httpProxy

	if c.Conf.Debug {
		log.SetLevel(log.DebugLevel)
	}

	if !c.Conf.Validate() {
		return subcommands.ExitUsageError
	}

	log.Infof("Fetching from NVD...")

	var driver db.DB
	var err error
	if driver, err = db.NewDB(c.Conf.DBType, c.Conf.DBPath, c.Conf.DebugSQL); err != nil {
		log.Errorf("Failed to new db. err : %s", err)
		return subcommands.ExitFailure
	}
	defer driver.CloseDB()

	log.Infof("Inserting into DB (%s)", driver.Name())
	if err = nvd.FetchAndInsertCPE(driver); err != nil {
		log.Fatalf("Failed to fetch. err: %s", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
