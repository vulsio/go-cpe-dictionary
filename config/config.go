package config

import (
	valid "github.com/asaskevich/govalidator"
	log "github.com/sirupsen/logrus"
)

// Conf has Configuration
var Conf Config

// Config has config
type Config struct {
	Debug    bool
	DebugSQL bool

	DBPath string
	DBType string

	Bind string `valid:"ipv4"`
	Port string `valid:"port"`

	//TODO Validator
	HTTPProxy string
}

// Validate validates configuration
// TODO test case
func (c *Config) Validate() bool {
	if c.DBType == "sqlite3" {
		if ok, _ := valid.IsFilePath(c.DBPath); !ok {
			log.Fatalf("--dbpath : %s is not valid *Absolute* file path", c.DBPath)
			return false
		}
	}

	_, err := valid.ValidateStruct(c)
	if err != nil {
		log.Fatal("error: " + err.Error())
	}
	return true
}
