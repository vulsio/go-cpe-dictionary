package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
	config "github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/cpe"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/models"
)

//  var debug bool

func main() {
	conf := config.Config{}

	flag.BoolVar(&conf.Debug, "v", false, "Debug mode")
	flag.BoolVar(&conf.DebugSQL, "vv", false, "SQL debug mode")

	pwd := os.Getenv("PWD")
	flag.StringVar(&conf.DumpPath, "dump-path", pwd+"/cpe.json", "/path/to/dump.json")
	flag.BoolVar(&conf.Fetch, "fetch", false, "Fetch CPE data from NVD")
	flag.StringVar(&conf.DBPath, "dbpath", pwd+"/cpe.db", "/path/to/sqlite3/datafile")
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

		log.Infof("Inserting into DB... dbpath: %s", conf.DBPath)
		if err := db.Init(conf); err != nil {
			log.Errorf("Failed to Init DB. err: %s", err)
			os.Exit(1)
		}

		if err := db.InsertCpes(cpes, conf); err != nil {
			log.Errorf("Failed to inserting DB. err: %s", err)
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

		cpeList := cpe.CpeList{}
		if err := json.Unmarshal(raw, &cpeList); err != nil {
			log.Fatalf("Failed to unmarshall JSON. pash: %s, err: %s", conf.DumpPath, err)
			os.Exit(1)
		}
		log.Infof("Success. %d items", len(cpeList.CpeItems))

		cpes := models.ConvertToModel(cpeList)

		log.Infof("Inserting into DB... dbpath: %s", conf.DBPath)
		if err := db.Init(conf); err != nil {
			log.Errorf("Failed to Init DB. err: %s", err)
			os.Exit(1)
		}

		if err := db.InsertCpes(cpes, conf); err != nil {
			log.Errorf("Failed to insert. err: %s", err)
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
