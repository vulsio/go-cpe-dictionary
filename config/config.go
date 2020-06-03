package config

import (
	"fmt"

	valid "github.com/asaskevich/govalidator"
	"github.com/inconshreveable/log15"
)

// Conf has Configuration
var Conf Config

// Config has config
type Config struct {
	Debug    bool
	DebugSQL bool

	DBPath string
	DBType string

	Bind string
	Port string

	//TODO Validator
	HTTPProxy string
}

// Validate validates configuration
// TODO test case
func (c *Config) Validate() bool {
	if c.DBType == "sqlite3" {
		if ok, _ := valid.IsFilePath(c.DBPath); !ok {
			log15.Crit(fmt.Sprintf("--dbpath : %s is not valid *Absolute* file path", c.DBPath))
			return false
		}
	}
	return true
}
