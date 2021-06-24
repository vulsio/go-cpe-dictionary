package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/jvn"
	"github.com/kotakanbe/go-cpe-dictionary/util"
)

// FetchJvnCmd : FetchJvnCmd
type FetchJvnCmd struct {
	logToFile bool
	logDir    string
	logJSON   bool
}

// Name return subcommand name
func (*FetchJvnCmd) Name() string { return "fetchjvn" }

// Synopsis return synopsis
func (*FetchJvnCmd) Synopsis() string { return "Fetch CPE from JVN" }

// Usage return usage
func (*FetchJvnCmd) Usage() string {
	return `fetchjvn:
	fetchjvn
		[-dbtype=mysql|postgres|sqlite3|redis]
		[-dbpath=$PWD/cpe.sqlite3 or connection string]
		[-http-proxy=http://192.168.0.1:8080]
		[-debug]
		[-log-to-file]
		[-log-dir=/path/to/log]
		[-log-json]

   $ go-cpe-dictionary fetchjvn | sort -u > /tmp/jvn.txt
`
}

// SetFlags set flag
func (p *FetchJvnCmd) SetFlags(f *flag.FlagSet) {
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
func (p *FetchJvnCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	util.SetLogger(p.logDir, config.Conf.Debug, p.logJSON, p.logToFile)
	if !config.Conf.Validate() {
		return subcommands.ExitUsageError
	}

	cpes, err := jvn.Fetch()
	if err != nil {
		log15.Crit("Failed to fetch.", "err", err)
		return subcommands.ExitFailure
	}

	if !config.Conf.Stdout {
		if err := jvn.Insert(cpes); err != nil {
			log15.Crit("Failed to insert.", "err", err)
		}
	}

	log15.Info("Fetched", "Number of CPEs", len(cpes))
	if config.Conf.Stdout {
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

	return subcommands.ExitSuccess
}
