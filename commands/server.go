package commands

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/server"
	"github.com/kotakanbe/go-cpe-dictionary/util"
)

// ServerCmd : ServerCmd
type ServerCmd struct {
	logToFile bool
	logDir    string
	logJSON   bool
}

// Name return subcommand name
func (*ServerCmd) Name() string { return "server" }

// Synopsis return synopsis
func (*ServerCmd) Synopsis() string { return "Start CPE dictionary HTTP server" }

// Usage return usage
func (*ServerCmd) Usage() string {
	return `server:
	server
		[-bind=127.0.0.1]
		[-port=8000]
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
   $ ./go-cpe-dictionary server
`
}

// SetFlags set flag
func (p *ServerCmd) SetFlags(f *flag.FlagSet) {
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

	f.StringVar(&config.Conf.Bind,
		"bind",
		"127.0.0.1",
		"HTTP server bind to IP address (default: loop back interface)")
	f.StringVar(&config.Conf.Port, "port", "1323",
		"HTTP server port number")
}

// Execute execute
func (p *ServerCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	util.SetLogger(p.logDir, config.Conf.Debug, p.logJSON, p.logToFile)
	if !config.Conf.Validate() {
		return subcommands.ExitUsageError
	}

	driver, err := db.NewDB(config.Conf.DBType, config.Conf.DBPath, config.Conf.DebugSQL)
	if err != nil {
		log15.Error("Failed to new DB.", "err", err)
		return subcommands.ExitFailure
	}

	log15.Info("Starting HTTP Server...")
	if err := server.Start(p.logDir, driver); err != nil {
		log15.Error("Failed to start server", "err", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
