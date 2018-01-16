package config

import (
	"os"

	valid "github.com/asaskevich/govalidator"
	"github.com/prometheus/common/log"
)

// Config has config
type Config struct {
	Debug    bool
	DebugSQL bool

	Load     bool
	Fetch    bool
	DumpPath string
	DBPath   string
	DBType   string

	Server bool
	Bind   string `valid:"ipv4"`
	Port   string `valid:"port"`

	//TODO Validator
	HTTPProxy string
}

// Validate validates configuration
// TODO test case
func (c *Config) Validate() bool {
	if c.Load || c.Fetch {
		if ok, _ := valid.IsFilePath(c.DumpPath); !ok {
			log.Fatalf("--dumpPath: %s is not valid *Absolute* file path", c.DumpPath)
			os.Exit(1)
		}
	}

	if c.DBType == "sqlite3" {
		if ok, _ := valid.IsFilePath(c.DBPath); !ok {
			log.Fatalf("--dbpath : %s is not valid *Absolute* file path", c.DBPath)
			os.Exit(1)
		}
	}

	if !(c.Load || c.Fetch) {
		c.Fetch = true
	}

	if c.Fetch && c.Load {
		log.Fatalf("--fetch and --load are not specified at the same time")
		os.Exit(1)
	}

	_, err := valid.ValidateStruct(c)
	if err != nil {
		log.Fatal("error: " + err.Error())
	}
	return false
}
