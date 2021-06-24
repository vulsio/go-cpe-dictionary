package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/config"
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
		[-dbpath=$PWD/cpe.sqlite3 or connection string]
		[-http-proxy=http://192.168.0.1:8080]
		[-debug]
		[-debug-sql]
		[-log-to-file]
		[-log-dir=/path/to/log]
		[-log-json]
		[-stdout]

For the first time, run the blow command to fetch data. (It takes about 10 minutes)
   $ ./go-cpe-dictionary fetchnvd
   $ ./go-cpe-dictionary fetchnvd --stdout | sort -u > /tmp/nvd.txt
`
}

// SetFlags set flag
func (p *FetchNvdCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&config.Conf.Debug, "debug", false, "debug mode")
	f.BoolVar(&config.Conf.DebugSQL, "debug-sql", false, "SQL debug mode")

	defaultLogDir := util.GetDefaultLogDir()
	f.StringVar(&p.logDir, "log-dir", defaultLogDir, "/path/to/log")
	f.BoolVar(&p.logJSON, "log-json", false, "output log as JSON")
	f.BoolVar(&p.logToFile, "log-to-file", false, "output log to file")
	f.BoolVar(&config.Conf.Stdout, "stdout", false, "display all CPEs to stdout")

	pwd := os.Getenv("PWD")
	f.StringVar(&config.Conf.DBPath, "dbpath", pwd+"/cpe.sqlite3",
		"/path/to/sqlite3 or SQL connection string")

	f.StringVar(&config.Conf.DBType, "dbtype", "sqlite3",
		"Database type to store data in (sqlite3, mysql, postgres or redis supported)")

	f.StringVar(&config.Conf.HTTPProxy, "http-proxy", "", "http://proxy-url:port (default: empty)")
}

// Execute execute
func (p *FetchNvdCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	util.SetLogger(p.logDir, config.Conf.Debug, p.logJSON, p.logToFile)
	if !config.Conf.Validate() {
		return subcommands.ExitUsageError
	}

	log15.Info("Fetch and insert from NVD...")
	cpes, err := nvd.FetchAndInsertCPE()
	if err != nil {
		log15.Crit("Failed to fetch.", "err", err)
		return subcommands.ExitFailure
	}
	if config.Conf.Stdout {
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

	return subcommands.ExitSuccess
}
