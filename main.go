package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"

	config "github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/cpe"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	log "github.com/sirupsen/logrus"
)

//  var debug bool

func main() {
	conf := config.Config{}

	flag.BoolVar(&conf.Debug, "v", false, "Debug mode")
	flag.BoolVar(&conf.DebugSQL, "vv", false, "SQL debug mode")

	pwd := os.Getenv("PWD")
	flag.StringVar(&conf.DumpPath, "dump-path", pwd+"/cpe.json", "/path/to/dump.json")
	flag.BoolVar(&conf.Fetch, "fetch", false, "Fetch CPE data from NVD")
	flag.StringVar(&conf.DBType, "dbtype", "sqlite3",
		"Database type to store data in (sqlite3,  mysql, postgres or redis supported)")
	flag.StringVar(&conf.DBPath, "dbpath", pwd+"/cpe.sqlite3",
		"/path/to/sqlite3 or SQL connection string")
	flag.BoolVar(&conf.Load, "load", false, "load CPE data from dumpfile")
	flag.StringVar(&conf.HTTPProxy, "http-proxy", "", "HTTP Proxy URL (http://proxy-server:8080)")

	flag.Parse()
	conf.Validate()

	if conf.DebugSQL {
		conf.Debug = true
	}
	setupLogger(conf)

	if conf.Fetch {
		log.Infof("Fetching from NVD...")

		cpeList, err := cpe.FetchCPE(conf.HTTPProxy)
		if err != nil {
			log.Fatalf("Failed to fetch. err: %s", err)
			os.Exit(1)
		}
		log.Infof("Dumping XML to %s...", conf.DumpPath)
		b, err := json.Marshal(cpeList)
		if err != nil {
			log.Errorf("Failed to Marshall. err: %s", err)
			os.Exit(1)
		}
		if err := ioutil.WriteFile(conf.DumpPath, b, 0644); err != nil {
			log.Errorf("Failed to dump. dump: %s, err: %s", conf.DumpPath, err)
			os.Exit(1)
		}
		cpes := models.ConvertToModel(cpeList)

		var driver db.DB
		var err error
		if driver, err = db.NewDB(conf.DBType, conf.DBPath, conf.DebugSQL); err != nil {
			log.Errorf("Failed to new db. err : %s", err)
			os.Exit(1)
		}
		defer driver.CloseDB()

		log.Infof("Inserting into DB (%s)", driver.Name())
		if err := driver.InsertCpes(cpes); err != nil {
			log.Fatalf("Failed to insert. dbpath: %s, err: %s", conf.DBPath, err)
			os.Exit(1)
		}
	}

	if conf.Load {
		log.Infof("Loading JSON from %s", conf.DumpPath)
		raw, err := ioutil.ReadFile(conf.DumpPath)
		if err != nil {
			log.Fatalf("Failed to load JSON. pash: %s, err: %s", conf.DumpPath, err)
			os.Exit(1)
		}

		cpeList := cpe.List{}
		if err := json.Unmarshal(raw, &cpeList); err != nil {
			log.Fatalf("Failed to unmarshall JSON. pash: %s, err: %s", conf.DumpPath, err)
			os.Exit(1)
		}
		log.Infof("Success. %d items", len(cpeList.Items))

		cpes := models.ConvertToModel(cpeList)

		var driver db.DB
		if driver, err = db.NewDB(conf.DBType, conf.DBPath, conf.DebugSQL); err != nil {
			log.Errorf("Failed to new db. err : %s", err)
			os.Exit(1)
		}
		defer driver.CloseDB()

		log.Infof("Inserting into DB (%s)", driver.Name())
		if err := driver.InsertCpes(cpes); err != nil {
			log.Fatalf("Failed to insert. dbpath: %s, err: %s", conf.DBPath, err)
			os.Exit(1)
		}
	}
}

func setupLogger(conf config.Config) {
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
