package models

import (
	"github.com/jinzhu/gorm"
)

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	gorm.Model      `json:"-" xml:"-"`
	CpeURI          string
	CpeFS           string
	Part            string
	Vendor          string
	Product         string
	Version         string
	Update          string
	Edition         string
	Language        string
	SoftwareEdition string
	TargetSoftware  string
	TargetHardware  string
	Other           string
	Deprecated      bool
}
